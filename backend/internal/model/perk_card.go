package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PerkCardStatus string

const (
	PerkActive  PerkCardStatus = "active"
	PerkUsed    PerkCardStatus = "used"
	PerkExpired PerkCardStatus = "expired"
)

type PerkCard struct {
	ID string `gorm:"type:varchar(36);primaryKey" json:"id"`

	FromUserID string `gorm:"type:varchar(36);not null;index" json:"from_user_id"` // 妈妈
	ToUserID   string `gorm:"type:varchar(36);not null;index" json:"to_user_id"`   // 爸爸

	Title       string         `gorm:"type:varchar(100);not null" json:"title"`
	Description string         `gorm:"type:varchar(500)" json:"description"`
	IconType    string         `gorm:"type:varchar(50)" json:"icon_type"`
	Status      PerkCardStatus `gorm:"type:varchar(20);default:'active'" json:"status"`

	UsedAt    *time.Time `json:"used_at"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

func (c *PerkCard) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}
