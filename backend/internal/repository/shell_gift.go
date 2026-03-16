package repository

import (
	"time"

	"github.com/momshell/backend/internal/model"
	"gorm.io/gorm"
)

type ShellGiftRepo struct {
	db *gorm.DB
}

func NewShellGiftRepo(db *gorm.DB) *ShellGiftRepo {
	return &ShellGiftRepo{db: db}
}

func (r *ShellGiftRepo) FindByID(id string) (*model.ShellGift, error) {
	var g model.ShellGift
	if err := r.db.First(&g, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *ShellGiftRepo) FindByTaskID(taskID string) (*model.ShellGift, error) {
	var g model.ShellGift
	if err := r.db.First(&g, "task_id = ?", taskID).Error; err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *ShellGiftRepo) FindByRecipient(userID string, limit int) ([]model.ShellGift, error) {
	if limit <= 0 {
		limit = 50
	}
	var gifts []model.ShellGift
	if err := r.db.
		Where("to_user_id = ?", userID).
		Order("created_at desc").
		Limit(limit).
		Find(&gifts).Error; err != nil {
		return nil, err
	}
	return gifts, nil
}

func (r *ShellGiftRepo) Create(g *model.ShellGift) error {
	return r.db.Create(g).Error
}

func (r *ShellGiftRepo) MarkOpened(id string, openedAt time.Time) error {
	return r.db.Model(&model.ShellGift{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_opened": true,
			"opened_at": openedAt,
		}).Error
}
