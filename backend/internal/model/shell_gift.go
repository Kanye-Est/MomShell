package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ShellGift is a "blind box shell" generated from a Dad's completed task and delivered to Mom.
type ShellGift struct {
	ID         string `gorm:"type:varchar(36);primaryKey" json:"id"`
	TaskID     string `gorm:"type:varchar(36);not null;uniqueIndex" json:"task_id"`
	FromUserID string `gorm:"type:varchar(36);not null;index" json:"from_user_id"`
	ToUserID   string `gorm:"type:varchar(36);not null;index" json:"to_user_id"`

	AITitle   string `gorm:"type:varchar(200);not null" json:"ai_title"`
	AIContent string `gorm:"type:text;not null" json:"ai_content"`

	CoverURL string  `gorm:"type:text" json:"cover_url"`
	PhotoURL *string `gorm:"type:varchar(500)" json:"photo_url"`

	IsOpened bool       `gorm:"default:false" json:"is_opened"`
	OpenedAt *time.Time `json:"opened_at"`

	CreatedAt time.Time `json:"created_at"`
}

func (s *ShellGift) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}
