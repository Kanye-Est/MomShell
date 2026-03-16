package repository

import (
	"time"

	"github.com/momshell/backend/internal/model"
	"gorm.io/gorm"
)

type PerkCardRepo struct {
	db *gorm.DB
}

func NewPerkCardRepo(db *gorm.DB) *PerkCardRepo {
	return &PerkCardRepo{db: db}
}

func (r *PerkCardRepo) Create(card *model.PerkCard) error {
	return r.db.Create(card).Error
}

func (r *PerkCardRepo) FindByID(id string) (*model.PerkCard, error) {
	var card model.PerkCard
	if err := r.db.First(&card, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &card, nil
}

func (r *PerkCardRepo) FindByRecipient(userID string, limit int) ([]model.PerkCard, error) {
	if limit <= 0 {
		limit = 100
	}
	var cards []model.PerkCard
	if err := r.db.
		Where("to_user_id = ?", userID).
		Order("created_at desc").
		Limit(limit).
		Find(&cards).Error; err != nil {
		return nil, err
	}
	return cards, nil
}

func (r *PerkCardRepo) FindByIssuer(fromUserID, toUserID string, limit int) ([]model.PerkCard, error) {
	if limit <= 0 {
		limit = 100
	}
	var cards []model.PerkCard
	if err := r.db.
		Where("from_user_id = ? AND to_user_id = ?", fromUserID, toUserID).
		Order("created_at desc").
		Limit(limit).
		Find(&cards).Error; err != nil {
		return nil, err
	}
	return cards, nil
}

func (r *PerkCardRepo) MarkUsed(id string, usedAt time.Time) error {
	return r.db.Model(&model.PerkCard{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":  model.PerkUsed,
			"used_at": usedAt,
		}).Error
}

func (r *PerkCardRepo) MarkExpired(id string) error {
	return r.db.Model(&model.PerkCard{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status": model.PerkExpired,
		}).Error
}
