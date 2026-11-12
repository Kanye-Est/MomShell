package service

import (
	"context"
	"fmt"
	"time"

	"github.com/momshell/backend/internal/model"
)

// buildTaskCardPrompt creates an image generation prompt for a task memory card.
func buildTaskCardPrompt(title, description string) string {
	scene := title
	if description != "" {
		runes := []rune(description)
		if len(runes) > 60 {
			runes = runes[:60]
		}
		scene = title + "，" + string(runes)
	}

	return fmt.Sprintf(
		"一张充满温度的家庭时光拍立得照片，定格了一个温馨瞬间：%s。"+
			"画面氛围：治愈、松弛感的日系清新漫画风格或绘本插画风格，绝对不要写实风格、不要3D渲染。"+
			"如果有家庭成员（爸爸、妈妈、宝宝），他们的互动要自然且充满爱意，人物造型要是可爱的漫画感；如果不包含人物，则通过物品光影和环境展现出温暖的居家氛围。"+
			"画面有留白，就像一张珍贵的家庭记忆卡片。"+
			"禁止任何文字、字母或水印。",
		scene,
	)
}

func buildVerifiedTaskCardPrompt(title, description string, score *int, comment *string) string {
	prompt := buildTaskCardPrompt(title, description)
	if score == nil && (comment == nil || *comment == "") {
		return prompt + " 这是一段被用心珍藏的家庭生活记忆。"
	}

	var details string
	if score != nil {
		if *score == 5 {
			details = " 充满极大的惊喜与感动，画面格外明亮欢乐。"
		} else if *score >= 3 {
			details = " 充满平淡的踏实与温馨。"
		} else {
			details = " 带着些许波折但依然真实的家庭日常。"
		}
	}
	if comment != nil && *comment != "" {
		details += fmt.Sprintf(" 照片背后的情绪线索是：“%s”。请将这种情绪自然融入画面的氛围中。", *comment)
	}
	return prompt + details
}

func (s *TaskService) generateVerifiedTaskCardPhoto(momID string, ut model.UserTask) error {
	if !s.openaiClient.Enabled() || s.imageModel == "" || s.photoRepo == nil {
		return nil
	}

	title, desc, _, _ := resolveTaskContent(ut)
	if title == "" {
		title = "任务纪念卡"
	}
	prompt := buildVerifiedTaskCardPrompt(title, desc, ut.Score, ut.Comment)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	imgResp, err := s.openaiClient.GenerateImage(ctx, s.imageModel, prompt)
	if err != nil {
		return fmt.Errorf("image generation failed: %w", err)
	}

	imageURL, err := saveGeneratedImage(imgResp)
	if err != nil {
		return fmt.Errorf("image download failed: %w", err)
	}

	description := "妈妈已验收通过的任务纪念卡"
	if ut.Comment != nil && *ut.Comment != "" {
		description = *ut.Comment
	}

	photo := &model.Photo{
		UserID:      momID,
		Title:       title,
		Description: description,
		ImageURL:    imageURL,
		Source:      "task_card",
	}

	if err := s.photoRepo.Create(photo); err != nil {
		return err
	}

	return s.taskRepo.SetTaskMemoryPhoto(ut.ID, photo.ImageURL)
}
