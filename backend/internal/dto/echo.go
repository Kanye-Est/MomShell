package dto

import "github.com/momshell/backend/internal/model"

type IdentityTagCreateRequest struct {
	TagType string `json:"tag_type" binding:"required,oneof=music sound literature memory"`
	Content string `json:"content" binding:"required,min=1,max=200"`
}

type IdentityTagListResponse struct {
	Music      []model.IdentityTag `json:"music"`
	Sound      []model.IdentityTag `json:"sound"`
	Literature []model.IdentityTag `json:"literature"`
	Memory     []model.IdentityTag `json:"memory"`
}

type GenerateMemoirRequest struct {
	Theme *string `json:"theme" binding:"omitempty,max=100"`
}

type GenerateMemoirResponse struct {
	*model.Memoir
	ImageGenerationEnabled bool `json:"image_generation_enabled"`
}

type MemoirListResponse struct {
	Memoirs []model.Memoir `json:"memoirs"`
	Total   int64          `json:"total"`
}

type RateMemoirRequest struct {
	Rating int `json:"rating" binding:"required,min=1,max=5"`
}
