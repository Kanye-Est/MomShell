package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SkillDimension represents one axis on the Dad skill radar.
type SkillDimension string

const (
	SkillNutrition SkillDimension = "nutrition" // 营养喂养
	SkillCleaning  SkillDimension = "cleaning"  // 清洁护理
	SkillEmotional SkillDimension = "emotional" // 情绪安抚
	SkillLogistics SkillDimension = "logistics" // 后勤保障
	SkillHealth    SkillDimension = "health"    // 健康护理
	SkillPlaytime  SkillDimension = "playtime"  // 亲子陪伴
)

type Achievement struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	Code        string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	Title       string    `gorm:"type:varchar(100);not null" json:"title"`
	Description string    `gorm:"type:varchar(255);not null" json:"description"`
	IconURL     string    `gorm:"type:text" json:"icon_url"`
	Condition   string    `gorm:"type:text" json:"condition"` // JSON string describing unlock condition
	CreatedAt   time.Time `json:"created_at"`
}

func (a *Achievement) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

type UserAchievement struct {
	ID            string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID        string    `gorm:"type:varchar(36);not null;index;uniqueIndex:idx_user_achievement" json:"user_id"`
	AchievementID string    `gorm:"type:varchar(36);not null;index;uniqueIndex:idx_user_achievement" json:"achievement_id"`
	UnlockedAt    time.Time `json:"unlocked_at"`

	Achievement Achievement `gorm:"foreignKey:AchievementID" json:"achievement,omitempty"`
}

func (ua *UserAchievement) BeforeCreate(tx *gorm.DB) error {
	if ua.ID == "" {
		ua.ID = uuid.New().String()
	}
	if ua.UnlockedAt.IsZero() {
		ua.UnlockedAt = time.Now()
	}
	return nil
}
