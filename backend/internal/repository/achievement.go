package repository

import (
	"github.com/momshell/backend/internal/model"
	"gorm.io/gorm"
)

type AchievementRepo struct {
	db *gorm.DB
}

func NewAchievementRepo(db *gorm.DB) *AchievementRepo {
	return &AchievementRepo{db: db}
}

func (r *AchievementRepo) FindAll() ([]model.Achievement, error) {
	var achievements []model.Achievement
	if err := r.db.Order("created_at asc").Find(&achievements).Error; err != nil {
		return nil, err
	}
	return achievements, nil
}

func (r *AchievementRepo) FindUserAchievements(userID string) ([]model.UserAchievement, error) {
	var unlocked []model.UserAchievement
	if err := r.db.Preload("Achievement").
		Where("user_id = ?", userID).
		Order("unlocked_at asc").
		Find(&unlocked).Error; err != nil {
		return nil, err
	}
	return unlocked, nil
}

func (r *AchievementRepo) CreateUserAchievement(ua *model.UserAchievement) error {
	return r.db.Create(ua).Error
}
