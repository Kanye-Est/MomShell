package dto

import "time"

type SkillRadar struct {
	Nutrition int `json:"nutrition"`
	Cleaning  int `json:"cleaning"`
	Emotional int `json:"emotional"`
	Logistics int `json:"logistics"`
	Health    int `json:"health"`
	Playtime  int `json:"playtime"`
}

type AchievementItem struct {
	ID          string     `json:"id"`
	Code        string     `json:"code"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	IconURL     string     `json:"icon_url"`
	Unlocked    bool       `json:"unlocked"`
	UnlockedAt  *time.Time `json:"unlocked_at"`
}
