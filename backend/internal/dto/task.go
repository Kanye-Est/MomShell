package dto

import "time"

type UserTaskItem struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Category      string     `json:"category"`
	Difficulty    int        `json:"difficulty"`
	Priority      string     `json:"priority"`
	Status        string     `json:"status"`
	Source        string     `json:"source"`
	Score         *int       `json:"score"`
	Comment       *string    `json:"comment"`
	ProofPhotoURL *string    `json:"proof_photo_url"`
	CompletedAt   *time.Time `json:"completed_at"`
	ScoredAt      *time.Time `json:"scored_at"`
	Date          time.Time  `json:"date"`
}

type CompleteTaskRequest struct {
	ProofPhotoURL *string `json:"proof_photo_url" binding:"omitempty,max=500"`
}

type TaskScore struct {
	Score   int    `json:"score" binding:"required,min=1,max=5"`
	Comment string `json:"comment" binding:"max=500"`
}

type TaskStats struct {
	XP    int `json:"xp"`
	Level int `json:"level"`
}

type SetBabyAgeRequest struct {
	AgeStage string `json:"age_stage" binding:"required,oneof=pregnancy 0-3m 3-6m 6-12m 1-2y 2-3y 3-4y 4-5y"`
}

type BabyAgeResponse struct {
	AgeStage string `json:"age_stage"`
	Source   string `json:"source"` // "manual", "partner", "memory", ""
}
