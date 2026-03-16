package service

import (
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/momshell/backend/internal/dto"
	"github.com/momshell/backend/internal/model"
	"github.com/momshell/backend/internal/repository"
	"github.com/momshell/backend/pkg/openai"
)

const (
	errUserNotFound    = "用户不存在"
	errPartnerRequired = "请先完成伴侣绑定"
	errTaskNotFound    = "任务不存在"
)

type TaskService struct {
	taskRepo           *repository.TaskRepo
	userRepo           *repository.UserRepo
	chatRepo           *repository.ChatRepo
	openaiClient       *openai.Client
	shellGiftService   *ShellGiftService
	achievementService *AchievementService
}

func NewTaskService(
	taskRepo *repository.TaskRepo,
	userRepo *repository.UserRepo,
	chatRepo *repository.ChatRepo,
	openaiClient *openai.Client,
	shellGiftService *ShellGiftService,
	achievementService *AchievementService,
) *TaskService {
	return &TaskService{
		taskRepo:           taskRepo,
		userRepo:           userRepo,
		chatRepo:           chatRepo,
		openaiClient:       openaiClient,
		shellGiftService:   shellGiftService,
		achievementService: achievementService,
	}
}

func today() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// GetDailyTasks returns today's tasks for a Dad user, lazily creating them if needed.
func (s *TaskService) GetDailyTasks(userID string) ([]dto.UserTaskItem, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New(errUserNotFound)
	}
	if user.Role != model.RoleDad {
		return nil, fmt.Errorf("只有爸爸角色可以查看任务")
	}
	if user.PartnerID == nil {
		return nil, errors.New(errPartnerRequired)
	}

	date := today()
	tasks, err := s.taskRepo.FindUserTasksByDate(userID, date)
	if err != nil {
		return nil, err
	}

	// Lazy initialization: assign tasks for today
	if len(tasks) == 0 {
		if err := s.createTasksForUser(user, userID, date); err != nil {
			return []dto.UserTaskItem{}, nil
		}
		tasks, err = s.taskRepo.FindUserTasksByDate(userID, date)
		if err != nil {
			return nil, err
		}
	}

	return toTaskItems(tasks), nil
}

// CompleteTask marks a task as completed by the Dad user.
// proofPhotoURL is optional.
func (s *TaskService) CompleteTask(userID, taskID string, proofPhotoURL *string) (*dto.UserTaskItem, error) {
	ut, err := s.taskRepo.FindUserTaskByID(taskID)
	if err != nil {
		return nil, errors.New(errTaskNotFound)
	}
	if ut.UserID != userID {
		return nil, fmt.Errorf("无权操作此任务")
	}
	if ut.Status != model.TaskPending {
		return nil, fmt.Errorf("任务已完成或已验收")
	}

	now := time.Now()
	ut.Status = model.TaskCompleted
	ut.CompletedAt = &now
	if proofPhotoURL != nil && *proofPhotoURL != "" {
		ut.ProofPhotoURL = proofPhotoURL
	}

	if err := s.taskRepo.UpdateUserTask(ut); err != nil {
		return nil, err
	}

	if s.shellGiftService != nil {
		// Fire-and-forget: completion should not be blocked by AI generation.
		utCopy := *ut
		go func() {
			if err := s.shellGiftService.CreateFromTask(utCopy); err != nil {
				log.Printf("[TaskService] shell gift generation failed: %v", err)
			}
		}()
	}

	item := toTaskItem(*ut)
	return &item, nil
}

// GetPartnerTasks returns the partner (Dad)'s tasks for Mom to review.
func (s *TaskService) GetPartnerTasks(callerID string) ([]dto.UserTaskItem, error) {
	user, err := s.userRepo.FindByID(callerID)
	if err != nil {
		return nil, errors.New(errUserNotFound)
	}
	if user.Role != model.RoleMom {
		return nil, fmt.Errorf("只有妈妈角色可以查看伴侣任务")
	}
	if user.PartnerID == nil {
		return nil, errors.New(errPartnerRequired)
	}

	date := today()
	tasks, err := s.taskRepo.FindUserTasksByDate(*user.PartnerID, date)
	if err != nil {
		return nil, err
	}

	// Lazily assign tasks for partner if none exist today
	if len(tasks) == 0 {
		partner, pErr := s.userRepo.FindByID(*user.PartnerID)
		if pErr != nil {
			return []dto.UserTaskItem{}, nil
		}
		if err := s.createTasksForUser(partner, *user.PartnerID, date); err != nil {
			return []dto.UserTaskItem{}, nil
		}
		tasks, err = s.taskRepo.FindUserTasksByDate(*user.PartnerID, date)
		if err != nil {
			return nil, err
		}
	}

	return toTaskItems(tasks), nil
}

// ScoreTask allows Mom to verify and score a completed task.
func (s *TaskService) ScoreTask(callerID, taskID string, req dto.TaskScore) (*dto.UserTaskItem, error) {
	caller, err := s.userRepo.FindByID(callerID)
	if err != nil {
		return nil, errors.New(errUserNotFound)
	}
	if caller.Role != model.RoleMom {
		return nil, fmt.Errorf("只有妈妈角色可以验收任务")
	}
	if caller.PartnerID == nil {
		return nil, errors.New(errPartnerRequired)
	}

	ut, err := s.taskRepo.FindUserTaskByID(taskID)
	if err != nil {
		return nil, errors.New(errTaskNotFound)
	}
	if ut.UserID != *caller.PartnerID {
		return nil, fmt.Errorf("只能验收伴侣的任务")
	}
	if ut.Status != model.TaskCompleted {
		return nil, fmt.Errorf("只能验收已完成的任务")
	}

	now := time.Now()
	ut.Status = model.TaskVerified
	ut.Score = &req.Score
	if req.Comment != "" {
		ut.Comment = &req.Comment
	}
	ut.ScoredByID = &callerID
	ut.ScoredAt = &now

	if err := s.taskRepo.UpdateUserTask(ut); err != nil {
		return nil, err
	}

	if s.achievementService != nil {
		if err := s.achievementService.CheckAndUnlock(ut.UserID); err != nil {
			log.Printf("[TaskService] achievement check failed: %v", err)
		}
	}

	item := toTaskItem(*ut)
	return &item, nil
}

// RejectTask allows Mom to reject a completed task back to pending.
func (s *TaskService) RejectTask(callerID, taskID string, comment string) (*dto.UserTaskItem, error) {
	caller, err := s.userRepo.FindByID(callerID)
	if err != nil {
		return nil, errors.New(errUserNotFound)
	}
	if caller.Role != model.RoleMom {
		return nil, fmt.Errorf("只有妈妈角色可以验收任务")
	}
	if caller.PartnerID == nil {
		return nil, errors.New(errPartnerRequired)
	}

	ut, err := s.taskRepo.FindUserTaskByID(taskID)
	if err != nil {
		return nil, errors.New(errTaskNotFound)
	}
	if ut.UserID != *caller.PartnerID {
		return nil, fmt.Errorf("只能验收伴侣的任务")
	}
	if ut.Status != model.TaskCompleted {
		return nil, fmt.Errorf("只能驳回已完成的任务")
	}

	ut.Status = model.TaskPending
	ut.CompletedAt = nil
	ut.ProofPhotoURL = nil
	if comment != "" {
		ut.Comment = &comment
	}

	if err := s.taskRepo.ResetUserTaskToPending(ut); err != nil {
		return nil, err
	}

	item := toTaskItem(*ut)
	return &item, nil
}

// GetTaskStats returns XP and level for a user (Dad).
func (s *TaskService) GetTaskStats(userID string) (*dto.TaskStats, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New(errUserNotFound)
	}
	if user.PartnerID == nil {
		return &dto.TaskStats{XP: 0, Level: 1}, nil
	}

	// For Mom, show partner's stats
	targetID := userID
	if user.Role == model.RoleMom {
		targetID = *user.PartnerID
	}

	xp, err := s.taskRepo.SumScoreByUserID(targetID)
	if err != nil {
		return nil, err
	}

	// Level formula: level = floor(sqrt(xp / 10)) + 1
	level := int(math.Floor(math.Sqrt(float64(xp)/10))) + 1

	return &dto.TaskStats{XP: xp, Level: level}, nil
}

func toTaskItem(ut model.UserTask) dto.UserTaskItem {
	item := dto.UserTaskItem{
		ID:            ut.ID,
		Priority:      string(ut.Priority),
		Status:        string(ut.Status),
		Source:        string(ut.Source),
		Score:         ut.Score,
		Comment:       ut.Comment,
		ProofPhotoURL: ut.ProofPhotoURL,
		CompletedAt:   ut.CompletedAt,
		ScoredAt:      ut.ScoredAt,
		Date:          ut.Date,
	}

	if ut.Source == model.TaskSourceAI {
		item.Title = ut.AITitle
		item.Description = ut.AIDescription
		item.Category = ut.AICategory
		item.Difficulty = ut.AIDifficulty
	} else if ut.Task != nil {
		item.Title = ut.Task.Title
		item.Description = ut.Task.Description
		item.Category = string(ut.Task.Category)
		item.Difficulty = ut.Task.Difficulty
	}

	if item.Priority == "" {
		if ut.Task != nil && ut.Task.Priority != "" {
			item.Priority = string(ut.Task.Priority)
		} else {
			item.Priority = string(model.PriorityT2)
		}
	}

	if item.Source == "" {
		item.Source = "template"
	}

	return item
}

func toTaskItems(tasks []model.UserTask) []dto.UserTaskItem {
	items := make([]dto.UserTaskItem, len(tasks))
	for i, t := range tasks {
		items[i] = toTaskItem(t)
	}
	return items
}

// createTasksForUser creates today's tasks for a user.
// Uses AI generation if age stage is available, otherwise falls back to templates.
func (s *TaskService) createTasksForUser(user *model.User, userID string, date time.Time) error {
	// Try AI generation
	if s.openaiClient != nil && user.PartnerID != nil {
		stage, _ := resolveAgeStage(user, s.userRepo, s.chatRepo)
		if stage != "" {
			ck := coupleKey(userID, *user.PartnerID)
			dateStr := date.Format("2006-01-02")
			aiTasks, err := getOrGenerateAITasks(s.openaiClient, s.taskRepo, ck, dateStr, stage)
			if err == nil && len(aiTasks) > 0 {
				for _, at := range aiTasks {
					ut := &model.UserTask{
						UserID:        userID,
						Date:          date,
						Status:        model.TaskPending,
						Source:        model.TaskSourceAI,
						AITitle:       at.Title,
						AIDescription: at.Description,
						AICategory:    at.Category,
						AIDifficulty:  at.Difficulty,
						Priority:      model.TaskPriority(at.Priority),
					}
					if ut.Priority == "" {
						ut.Priority = model.PriorityT2
					}
					_ = s.taskRepo.CreateUserTask(ut)
				}
				return nil
			}
			// AI failed — fall through to template
		}
	}

	// Fallback: random templates
	templates, err := s.taskRepo.FindRandomTemplates(4)
	if err != nil || len(templates) == 0 {
		return fmt.Errorf("no templates available")
	}
	for _, tmpl := range templates {
		tid := tmpl.ID
		priority := tmpl.Priority
		if priority == "" {
			priority = model.PriorityT2
		}
		ut := &model.UserTask{
			UserID:   userID,
			TaskID:   &tid,
			Date:     date,
			Status:   model.TaskPending,
			Source:   model.TaskSourceTemplate,
			Priority: priority,
		}
		_ = s.taskRepo.CreateUserTask(ut)
	}
	return nil
}

// SetBabyAge sets the baby age stage for the user and immediately regenerates tasks.
func (s *TaskService) SetBabyAge(userID, ageStage string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New(errUserNotFound)
	}

	user.BabyAgeStage = &ageStage
	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("保存失败，请重试")
	}

	// Immediate regeneration: delete today's pending tasks and AI cache
	date := today()
	_ = s.taskRepo.DeletePendingUserTasksByDate(userID, date)

	if user.PartnerID != nil {
		ck := coupleKey(userID, *user.PartnerID)
		_ = s.taskRepo.DeleteAICacheByCouple(ck, date.Format("2006-01-02"))
		// Also clear partner's pending tasks so they get regenerated too
		_ = s.taskRepo.DeletePendingUserTasksByDate(*user.PartnerID, date)
	}

	return nil
}

// GetBabyAge returns the resolved baby age stage and its source.
func (s *TaskService) GetBabyAge(userID string) (*dto.BabyAgeResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New(errUserNotFound)
	}

	stage, source := resolveAgeStage(user, s.userRepo, s.chatRepo)
	return &dto.BabyAgeResponse{
		AgeStage: stage,
		Source:   source,
	}, nil
}
