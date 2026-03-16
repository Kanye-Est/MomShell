package dto

import "time"

type ShellGiftItem struct {
	ID         string `json:"id"`
	TaskID     string `json:"task_id"`
	FromUserID string `json:"from_user_id"`
	ToUserID   string `json:"to_user_id"`

	AITitle   string `json:"ai_title"`
	AIContent string `json:"ai_content"`

	CoverURL string  `json:"cover_url"`
	PhotoURL *string `json:"photo_url"`

	IsOpened  bool       `json:"is_opened"`
	OpenedAt  *time.Time `json:"opened_at"`
	CreatedAt time.Time  `json:"created_at"`
}
