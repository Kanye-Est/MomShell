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
	"gorm.io/gorm"
)

const (
	errUserNotFound    = "用户不存在"
	errPartnerRequired = "请先完成伴侣绑定"
	errTaskNotFound    = "任务不存在"
	errTaskChanged     = "任务状态已更新，请刷新后重试"
	dailyTaskCount     = 4
)

type TaskService struct {
	taskRepo           *repository.TaskRepo
	userRepo           *repository.UserRepo
	chatRepo           *repository.ChatRepo
	whisperRepo        *repository.WhisperRepo
	photoRepo          *repository.PhotoRepo
	openaiClient       *openai.Client
	imageModel         string
	achievementService *AchievementService
}

func NewTaskService(
	taskRepo *repository.TaskRepo,
	userRepo *repository.UserRepo,
	chatRepo *repository.ChatRepo,
	whisperRepo *repository.WhisperRepo,
	photoRepo *repository.PhotoRepo,
	openaiClient *openai.Client,
	imageModel string,
	achievementService *AchievementService,
) *TaskService {
	return &TaskService{
		taskRepo:           taskRepo,
		userRepo:           userRepo,
		chatRepo:           chatRepo,
		whisperRepo:        whisperRepo,
		photoRepo:          photoRepo,
		openaiClient:       openaiClient,
		imageModel:         imageModel,
		achievementService: achievementService,
	}
}

func today() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// GetDailyTasks returns today's whisper-intel tasks for a Dad user.
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

	return toTaskItems(filterWhisperDrivenTasks(tasks)), nil
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

	if err := s.updateTaskStatus(ut, model.TaskPending); err != nil {
		return nil, err
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

	// Filter out pending tasks — mom only sees completed/verified
	var visible []model.UserTask
	for _, t := range filterWhisperDrivenTasks(tasks) {
		if t.Status != model.TaskPending {
			visible = append(visible, t)
		}
	}

	return toTaskItems(visible), nil
}

// requireMomWithPartner validates that the caller is a Mom with a linked partner.
func (s *TaskService) requireMomWithPartner(callerID string) (*model.User, error) {
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
	return caller, nil
}

// findPartnerTask finds a task and validates it belongs to the caller's partner with the expected status.
func (s *TaskService) findPartnerTask(partnerID, taskID string, expectedStatus model.TaskStatus) (*model.UserTask, error) {
	ut, err := s.taskRepo.FindUserTaskByID(taskID)
	if err != nil {
		return nil, errors.New(errTaskNotFound)
	}
	if ut.UserID != partnerID {
		return nil, fmt.Errorf("只能验收伴侣的任务")
	}
	if ut.Status != expectedStatus {
		if expectedStatus == model.TaskCompleted {
			return nil, fmt.Errorf("只能验收已完成的任务")
		}
		return nil, fmt.Errorf("任务状态不符合要求")
	}
	return ut, nil
}

// ScoreTask allows Mom to verify and score a completed task.
func (s *TaskService) ScoreTask(callerID, taskID string, req dto.TaskScore) (*dto.UserTaskItem, error) {
	caller, err := s.requireMomWithPartner(callerID)
	if err != nil {
		return nil, err
	}

	ut, err := s.findPartnerTask(*caller.PartnerID, taskID, model.TaskCompleted)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	ut.Status = model.TaskVerified
	ut.Score = &req.Score
	if req.Comment != "" {
		ut.Comment = &req.Comment
	}
	ut.ScoredByID = &callerID
	ut.ScoredAt = &now

	if err := s.updateTaskStatus(ut, model.TaskCompleted); err != nil {
		return nil, err
	}

	if s.achievementService != nil {
		if err := s.achievementService.CheckAndUnlock(ut.UserID); err != nil {
			log.Printf("[TaskService] achievement check failed: %v", err)
		}
	}

	if s.openaiClient.Enabled() && s.photoRepo != nil && s.imageModel != "" {
		verifiedCopy := *ut
		go func() {
			if err := s.generateVerifiedTaskCardPhoto(callerID, verifiedCopy); err != nil {
				log.Printf("[TaskService] failed to generate verified task card: %v", err)
			}
		}()
	}

	item := toTaskItem(*ut)
	return &item, nil
}

// RegenerateTaskCard allows Mom to regenerate the memory card for a verified task.
func (s *TaskService) RegenerateTaskCard(callerID, taskID string) error {
	caller, err := s.requireMomWithPartner(callerID)
	if err != nil {
		return err
	}

	ut, err := s.findPartnerTask(*caller.PartnerID, taskID, model.TaskVerified)
	if err != nil {
		return err
	}

	if !s.openaiClient.Enabled() || s.photoRepo == nil || s.imageModel == "" {
		return errors.New("AI 绘图服务未配置，无法生成卡片")
	}

	verifiedCopy := *ut
	go func() {
		if err := s.generateVerifiedTaskCardPhoto(callerID, verifiedCopy); err != nil {
			log.Printf("[TaskService] failed to regenerate verified task card: %v", err)
		}
	}()

	return nil
}

// RejectTask allows Mom to reject a completed task back to pending.
func (s *TaskService) RejectTask(callerID, taskID string, comment string) (*dto.UserTaskItem, error) {
	caller, err := s.requireMomWithPartner(callerID)
	if err != nil {
		return nil, err
	}

	ut, err := s.findPartnerTask(*caller.PartnerID, taskID, model.TaskCompleted)
	if err != nil {
		return nil, err
	}

	ut.Status = model.TaskPending
	ut.CompletedAt = nil
	ut.ProofPhotoURL = nil
	if comment != "" {
		ut.Comment = &comment
	}

	if err := s.updateTaskStatus(ut, model.TaskCompleted); err != nil {
		return nil, err
	}

	item := toTaskItem(*ut)
	return &item, nil
}

func (s *TaskService) updateTaskStatus(task *model.UserTask, previousStatus model.TaskStatus) error {
	err := s.taskRepo.UpdateUserTask(task, previousStatus)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New(errTaskChanged)
	}
	return err
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
		ID:             ut.ID,
		Priority:       string(ut.Priority),
		Status:         string(ut.Status),
		Source:         string(ut.Source),
		Score:          ut.Score,
		Comment:        ut.Comment,
		ProofPhotoURL:  ut.ProofPhotoURL,
		MemoryPhotoURL: ut.MemoryPhotoURL,
		CompletedAt:    ut.CompletedAt,
		ScoredAt:       ut.ScoredAt,
		Date:           ut.Date,
	}

	if ut.Source != model.TaskSourceTemplate {
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

func filterWhisperDrivenTasks(tasks []model.UserTask) []model.UserTask {
	filtered := make([]model.UserTask, 0, len(tasks))
	for _, task := range tasks {
		if task.Source == model.TaskSourceWhisper {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

func resolveTaskContent(ut model.UserTask) (title string, description string, category string, difficulty int) {
	if ut.Source != model.TaskSourceTemplate {
		return ut.AITitle, ut.AIDescription, ut.AICategory, ut.AIDifficulty
	}
	if ut.Task != nil {
		return ut.Task.Title, ut.Task.Description, string(ut.Task.Category), ut.Task.Difficulty
	}
	return "", "", "", 0
}

// SetBabyAge sets the baby age stage for the user.
func (s *TaskService) SetBabyAge(userID, ageStage string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New(errUserNotFound)
	}

	taskOwner := userID
	family := []string{userID}
	if user.PartnerID != nil {
		family = append(family, *user.PartnerID)
		if user.Role == model.RoleMom {
			taskOwner = *user.PartnerID
		}
	}

	if err := s.taskRepo.WithDailyTaskLock(taskOwner, func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("id IN ?", family).Update("baby_age_stage", ageStage).Error; err != nil {
			return err
		}
		if user.PartnerID == nil {
			return nil
		}
		key := coupleKey(userID, *user.PartnerID)
		return repository.NewTaskRepo(tx).DeleteAICacheByCouple(key, today().Format("2006-01-02"), "future-letter-template")
	}); err != nil {
		log.Printf("[TaskService] Could not update baby age: %v", err)
		return fmt.Errorf("保存失败，请重试")
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

// RegenerateTasks returns the current whisper-driven tasks.
func (s *TaskService) RegenerateTasks(userID string) ([]dto.UserTaskItem, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New(errUserNotFound)
	}
	if user.Role != model.RoleDad {
		return nil, fmt.Errorf("只有爸爸角色可以重新生成任务")
	}
	if user.PartnerID == nil {
		return nil, errors.New(errPartnerRequired)
	}

	date := today()
	tasks, err := s.taskRepo.FindUserTasksByDate(userID, date)
	if err != nil {
		return nil, err
	}
	return toTaskItems(filterWhisperDrivenTasks(tasks)), nil
}
