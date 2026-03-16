package dto

import "time"

type CreatePerkCardRequest struct {
	Title       string     `json:"title" binding:"required,min=1,max=100"`
	Description string     `json:"description" binding:"omitempty,max=500"`
	IconType    string     `json:"icon_type" binding:"omitempty,max=50"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

type PerkCardItem struct {
	ID          string     `json:"id"`
	FromUserID  string     `json:"from_user_id"`
	ToUserID    string     `json:"to_user_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	IconType    string     `json:"icon_type"`
	Status      string     `json:"status"`
	UsedAt      *time.Time `json:"used_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
}
