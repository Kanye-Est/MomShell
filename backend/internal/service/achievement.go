package service

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/momshell/backend/internal/dto"
	"github.com/momshell/backend/internal/model"
	"github.com/momshell/backend/internal/repository"
	"gorm.io/gorm"
)

type AchievementService struct {
	taskRepo        *repository.TaskRepo
	achievementRepo *repository.AchievementRepo
	userRepo        *repository.UserRepo
}

func NewAchievementService(
	taskRepo *repository.TaskRepo,
	achievementRepo *repository.AchievementRepo,
	userRepo *repository.UserRepo,
) *AchievementService {
	return &AchievementService{
		taskRepo:        taskRepo,
		achievementRepo: achievementRepo,
		userRepo:        userRepo,
	}
}

func (s *AchievementService) GetSkillRadar(callerID string) (*dto.SkillRadar, error) {
	user, err := s.userRepo.FindByID(callerID)
	if err != nil {
		return &dto.SkillRadar{}, errors.New(errUserNotFound)
	}

	targetID := callerID
	if user.Role == model.RoleMom {
		if user.PartnerID == nil {
			return &dto.SkillRadar{}, nil
		}
		targetID = *user.PartnerID
	}

	tasks, err := s.taskRepo.FindVerifiedTasksByUserID(targetID)
	if err != nil {
		return nil, err
	}
	radar := computeSkillRadar(tasks)
	return &radar, nil
}

func (s *AchievementService) GetAchievements(callerID string) ([]dto.AchievementItem, error) {
	user, err := s.userRepo.FindByID(callerID)
	if err != nil {
		return nil, errors.New(errUserNotFound)
	}

	targetID := callerID
	if user.Role == model.RoleMom {
		if user.PartnerID == nil {
			return []dto.AchievementItem{}, nil
		}
		targetID = *user.PartnerID
	}

	all, err := s.achievementRepo.FindAll()
	if err != nil {
		return nil, err
	}

	unlocked, err := s.achievementRepo.FindUserAchievements(targetID)
	if err != nil {
		return nil, err
	}

	unlockedMap := make(map[string]timePair, len(unlocked))
	for _, ua := range unlocked {
		unlockedMap[ua.AchievementID] = timePair{t: ua.UnlockedAt, ok: true}
	}

	items := make([]dto.AchievementItem, 0, len(all))
	for _, a := range all {
		var unlockedAt *time.Time
		isUnlocked := false
		if pair, ok := unlockedMap[a.ID]; ok && pair.ok {
			isUnlocked = true
			unlockedAt = &pair.t
		}
		items = append(items, dto.AchievementItem{
			ID:          a.ID,
			Code:        a.Code,
			Title:       a.Title,
			Description: a.Description,
			IconURL:     a.IconURL,
			Unlocked:    isUnlocked,
			UnlockedAt:  unlockedAt,
		})
	}
	return items, nil
}

// CheckAndUnlock evaluates all achievements for the given user and unlocks the missing ones.
func (s *AchievementService) CheckAndUnlock(userID string) error {
	all, err := s.achievementRepo.FindAll()
	if err != nil {
		return err
	}

	unlocked, err := s.achievementRepo.FindUserAchievements(userID)
	if err != nil {
		return err
	}
	unlockedSet := make(map[string]bool, len(unlocked))
	for _, ua := range unlocked {
		unlockedSet[ua.AchievementID] = true
	}

	verifiedCount, err := s.taskRepo.CountVerifiedTasksByUserID(userID)
	if err != nil {
		return err
	}
	verifiedTasks, err := s.taskRepo.FindVerifiedTasksByUserID(userID)
	if err != nil {
		return err
	}
	radar := computeSkillRadar(verifiedTasks)

	for _, a := range all {
		if unlockedSet[a.ID] {
			continue
		}
		cond, ok := parseAchievementCondition(a.Condition)
		if !ok {
			continue
		}

		if isConditionSatisfied(cond, verifiedCount, radar) {
			ua := &model.UserAchievement{
				UserID:        userID,
				AchievementID: a.ID,
			}
			if err := s.achievementRepo.CreateUserAchievement(ua); err != nil {
				// Unique index makes this idempotent; ignore duplicates
				if errors.Is(err, gorm.ErrDuplicatedKey) ||
					strings.Contains(strings.ToLower(err.Error()), "duplicate") ||
					strings.Contains(strings.ToLower(err.Error()), "unique") {
					continue
				}
				log.Printf("[Achievement] unlock failed for %s/%s: %v", userID, a.Code, err)
				continue
			}
		}
	}

	return nil
}

type achievementCondition struct {
	Type      string `json:"type"` // task_count | dimension_min
	Min       int    `json:"min"`
	Dimension string `json:"dimension"`
}

func parseAchievementCondition(raw string) (achievementCondition, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return achievementCondition{}, false
	}
	var cond achievementCondition
	if err := json.Unmarshal([]byte(raw), &cond); err != nil {
		return achievementCondition{}, false
	}
	cond.Type = strings.TrimSpace(cond.Type)
	cond.Dimension = strings.TrimSpace(cond.Dimension)
	if cond.Min <= 0 || cond.Type == "" {
		return achievementCondition{}, false
	}
	return cond, true
}

func isConditionSatisfied(cond achievementCondition, verifiedCount int, radar dto.SkillRadar) bool {
	switch cond.Type {
	case "task_count":
		return verifiedCount >= cond.Min
	case "dimension_min":
		switch strings.ToLower(cond.Dimension) {
		case string(model.SkillNutrition):
			return radar.Nutrition >= cond.Min
		case string(model.SkillCleaning):
			return radar.Cleaning >= cond.Min
		case string(model.SkillEmotional):
			return radar.Emotional >= cond.Min
		case string(model.SkillLogistics):
			return radar.Logistics >= cond.Min
		case string(model.SkillHealth):
			return radar.Health >= cond.Min
		case string(model.SkillPlaytime):
			return radar.Playtime >= cond.Min
		default:
			return false
		}
	default:
		return false
	}
}

func computeSkillRadar(tasks []model.UserTask) dto.SkillRadar {
	var radar dto.SkillRadar

	for _, ut := range tasks {
		if ut.Score == nil {
			continue
		}
		score := *ut.Score
		cat := resolveUserTaskCategory(ut)
		switch cat {
		case string(model.TaskCategoryHousework):
			radar.Cleaning += score
			radar.Logistics += score
		case string(model.TaskCategoryParenting):
			radar.Nutrition += score
			radar.Playtime += score
		case string(model.TaskCategoryHealth):
			radar.Health += score
		case string(model.TaskCategoryEmotional):
			radar.Emotional += score
		}
	}

	return radar
}

func resolveUserTaskCategory(ut model.UserTask) string {
	if ut.Source == model.TaskSourceAI {
		return strings.ToLower(strings.TrimSpace(ut.AICategory))
	}
	if ut.Task != nil {
		return string(ut.Task.Category)
	}
	return ""
}

type timePair struct {
	t  time.Time
	ok bool
}
