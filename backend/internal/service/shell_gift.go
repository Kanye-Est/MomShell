package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/momshell/backend/internal/dto"
	"github.com/momshell/backend/internal/model"
	"github.com/momshell/backend/internal/repository"
	"github.com/momshell/backend/pkg/openai"
	"gorm.io/gorm"
)

const (
	errShellGiftUserNotFound = "用户不存在"
	errShellGiftNotFound     = "贝壳不存在"
)

type ShellGiftService struct {
	shellGiftRepo *repository.ShellGiftRepo
	userRepo      *repository.UserRepo
	aiClient      *openai.Client
}

func NewShellGiftService(
	shellGiftRepo *repository.ShellGiftRepo,
	userRepo *repository.UserRepo,
	aiClient *openai.Client,
) *ShellGiftService {
	return &ShellGiftService{
		shellGiftRepo: shellGiftRepo,
		userRepo:      userRepo,
		aiClient:      aiClient,
	}
}

// CreateFromTask generates a ShellGift for Mom after Dad completed a task.
// It is safe to call multiple times for the same task (idempotent by task_id unique index).
func (s *ShellGiftService) CreateFromTask(ut model.UserTask) error {
	// Ensure we don't create duplicates
	if _, err := s.shellGiftRepo.FindByTaskID(ut.ID); err == nil {
		return nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Determine sender/recipient
	dadID := ut.UserID
	dad, err := s.userRepo.FindByID(dadID)
	if err != nil {
		return errors.New(errShellGiftUserNotFound)
	}
	if dad.PartnerID == nil || *dad.PartnerID == "" {
		// No partner bound; nothing to deliver
		return nil
	}
	momID := *dad.PartnerID

	// Build task display fields
	title, desc, category := ut.AITitle, ut.AIDescription, ut.AICategory
	difficulty := ut.AIDifficulty
	priority := string(ut.Priority)
	if ut.Source != model.TaskSourceAI && ut.Task != nil {
		title = ut.Task.Title
		desc = ut.Task.Description
		category = string(ut.Task.Category)
		difficulty = ut.Task.Difficulty
		if priority == "" {
			priority = string(ut.Task.Priority)
		}
	}
	if priority == "" {
		priority = string(model.PriorityT2)
	}

	aiTitle, aiContent := s.generateGiftText(title, desc, category, difficulty, priority, ut.ProofPhotoURL != nil)

	seed := ut.ID
	cover := generateMemoirCoverDataURI(aiTitle, &seed)

	gift := &model.ShellGift{
		TaskID:     ut.ID,
		FromUserID: dadID,
		ToUserID:   momID,
		AITitle:    aiTitle,
		AIContent:  aiContent,
		CoverURL:   cover,
		PhotoURL:   ut.ProofPhotoURL,
		IsOpened:   false,
	}

	if err := s.shellGiftRepo.Create(gift); err != nil {
		// Unique constraint (task_id) makes this idempotent; ignore duplicates
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil
		}
		return err
	}

	return nil
}

func (s *ShellGiftService) generateGiftText(
	taskTitle, taskDesc, category string,
	difficulty int,
	priority string,
	hasProofPhoto bool,
) (title string, content string) {
	// Fallback when AI is not configured
	fallback := func() (string, string) {
		baseTitle := "今日战绩"
		if taskTitle != "" {
			baseTitle = truncateText(taskTitle, 16)
		}
		photoLine := ""
		if hasProofPhoto {
			photoLine = "他还上传了证明照片。"
		}
		body := fmt.Sprintf("他完成了「%s」。%s谢谢你们一起把这件事做扎实。", taskTitle, photoLine)
		return baseTitle, strings.TrimSpace(body)
	}

	if s.aiClient == nil {
		return fallback()
	}

	systemPrompt := `你是「小石光」，一位家庭协作记录官。
根据爸爸完成的任务与证明信息，为妈妈生成一段“既有科学意义、又有情感价值”的评价，用作盲盒贝壳内容。

要求：
- 用中文
- 输出 JSON：{"title":"不超过16字", "content":"50-120字"}
- content 结构建议：1句科学解释/原则 + 1句情感肯定/感谢 + 1句可选的下一步建议
- 不要使用 emoji
- 不要输出除 JSON 以外的任何内容`

	proofHint := "无"
	if hasProofPhoto {
		proofHint = "有（照片内容无需描述具体物体，重点写“完成证明”即可）"
	}

	userPrompt := fmt.Sprintf(
		"任务标题：%s\n任务说明：%s\n类别：%s\n难度：%d/5\n优先级：%s\n证明照片：%s\n请生成盲盒贝壳内容。",
		taskTitle, taskDesc, category, difficulty, priority, proofHint,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := s.aiClient.Chat(ctx, []openai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		log.Printf("[ShellGift] AI generation failed: %v", err)
		return fallback()
	}

	cleaned := strings.TrimSpace(resp)
	if strings.HasPrefix(cleaned, "```") {
		if idx := strings.Index(cleaned[3:], "\n"); idx >= 0 {
			cleaned = cleaned[3+idx+1:]
		}
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	}

	var parsed struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		log.Printf("[ShellGift] AI response parse failed: %v", err)
		return fallback()
	}

	parsed.Title = strings.TrimSpace(parsed.Title)
	parsed.Content = strings.TrimSpace(parsed.Content)
	if parsed.Title == "" || parsed.Content == "" {
		return fallback()
	}
	return truncateText(parsed.Title, 16), parsed.Content
}

// ListForMom returns shell gifts for the caller (Mom). Unopened gifts do not include AIContent.
func (s *ShellGiftService) ListForMom(callerID string) ([]dto.ShellGiftItem, error) {
	user, err := s.userRepo.FindByID(callerID)
	if err != nil {
		return nil, errors.New(errShellGiftUserNotFound)
	}
	if user.Role != model.RoleMom {
		return nil, fmt.Errorf("只有妈妈角色可以查看贝壳")
	}

	gifts, err := s.shellGiftRepo.FindByRecipient(callerID, 50)
	if err != nil {
		return nil, err
	}

	items := make([]dto.ShellGiftItem, 0, len(gifts))
	for _, g := range gifts {
		item := toShellGiftItem(g)
		if !g.IsOpened {
			item.AIContent = ""
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *ShellGiftService) OpenForMom(callerID, giftID string) (*dto.ShellGiftItem, error) {
	user, err := s.userRepo.FindByID(callerID)
	if err != nil {
		return nil, errors.New(errShellGiftUserNotFound)
	}
	if user.Role != model.RoleMom {
		return nil, fmt.Errorf("只有妈妈角色可以打开贝壳")
	}

	g, err := s.shellGiftRepo.FindByID(giftID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(errShellGiftNotFound)
		}
		return nil, err
	}
	if g.ToUserID != callerID {
		return nil, fmt.Errorf("只能打开自己的贝壳")
	}

	if !g.IsOpened {
		now := time.Now()
		if err := s.shellGiftRepo.MarkOpened(g.ID, now); err != nil {
			return nil, err
		}
		g.IsOpened = true
		g.OpenedAt = &now
	}

	item := toShellGiftItem(*g)
	return &item, nil
}

func toShellGiftItem(g model.ShellGift) dto.ShellGiftItem {
	return dto.ShellGiftItem{
		ID:         g.ID,
		TaskID:     g.TaskID,
		FromUserID: g.FromUserID,
		ToUserID:   g.ToUserID,
		AITitle:    g.AITitle,
		AIContent:  g.AIContent,
		CoverURL:   g.CoverURL,
		PhotoURL:   g.PhotoURL,
		IsOpened:   g.IsOpened,
		OpenedAt:   g.OpenedAt,
		CreatedAt:  g.CreatedAt,
	}
}
