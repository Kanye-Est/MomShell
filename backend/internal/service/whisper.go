package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/momshell/backend/internal/dto"
	"github.com/momshell/backend/internal/model"
	"github.com/momshell/backend/internal/repository"
	"github.com/momshell/backend/pkg/openai"
)

const (
	errWhisperUserNotFound    = "用户不存在"
	errWhisperPartnerRequired = "请先完成伴侣绑定"
)

type WhisperService struct {
	whisperRepo *repository.WhisperRepo
	userRepo    *repository.UserRepo
	aiClient    *openai.Client
}

func NewWhisperService(
	whisperRepo *repository.WhisperRepo,
	userRepo *repository.UserRepo,
	aiClient *openai.Client,
) *WhisperService {
	return &WhisperService{
		whisperRepo: whisperRepo,
		userRepo:    userRepo,
		aiClient:    aiClient,
	}
}

// CreateWhisper creates a new whisper from a mom user.
func (s *WhisperService) CreateWhisper(authorID, content string) (*dto.WhisperItem, error) {
	user, err := s.userRepo.FindByID(authorID)
	if err != nil {
		return nil, errors.New(errWhisperUserNotFound)
	}
	if user.Role != model.RoleMom {
		return nil, fmt.Errorf("只有妈妈角色可以写心语")
	}
	if user.PartnerID == nil {
		return nil, errors.New(errWhisperPartnerRequired)
	}

	w := &model.Whisper{
		AuthorID: authorID,
		Content:  content,
	}
	if err := s.whisperRepo.Create(w); err != nil {
		return nil, err
	}

	return &dto.WhisperItem{
		ID:        w.ID,
		Content:   w.Content,
		CreatedAt: w.CreatedAt,
	}, nil
}

// GetWhispers returns whispers for the partner to read.
// If the caller is Dad, returns Mom's whispers.
func (s *WhisperService) GetWhispers(callerID string) ([]dto.WhisperItem, error) {
	user, err := s.userRepo.FindByID(callerID)
	if err != nil {
		return nil, errors.New(errWhisperUserNotFound)
	}
	if user.PartnerID == nil {
		return nil, errors.New(errWhisperPartnerRequired)
	}

	// Determine whose whispers to fetch
	var targetID string
	if user.Role == model.RoleDad {
		targetID = *user.PartnerID // Dad reads Mom's whispers
	} else {
		targetID = user.ID // Mom reads her own whispers
	}

	whispers, err := s.whisperRepo.FindByAuthorID(targetID, 20)
	if err != nil {
		return nil, err
	}

	items := make([]dto.WhisperItem, len(whispers))
	for i, w := range whispers {
		items[i] = dto.WhisperItem{
			ID:        w.ID,
			Content:   w.Content,
			CreatedAt: w.CreatedAt,
		}
	}
	return items, nil
}

const whisperTipsPrompt = `你是「小石光」，一位温柔的家庭关系顾问。
以下是一位妈妈最近写下的心语（心情、感受或愿望）：

%s

请根据这些心语，给她的伴侣（爸爸）一些温暖、实用的提示和建议：
1. 先简要分析妈妈目前的情绪状态
2. 给出 2-3 条具体的行动建议（比如可以做什么、说什么）
3. 语气温暖鼓励，像朋友之间的建议

用纯文本回复，不要使用 JSON 格式，1-3 段话即可。`

// GetWhisperTips generates AI tips for Dad based on Mom's whispers.
func (s *WhisperService) GetWhisperTips(callerID string) (*dto.WhisperTips, error) {
	user, err := s.userRepo.FindByID(callerID)
	if err != nil {
		return nil, errors.New(errWhisperUserNotFound)
	}
	if user.Role != model.RoleDad {
		return nil, fmt.Errorf("只有爸爸角色可以获取提示")
	}
	if user.PartnerID == nil {
		return nil, errors.New(errWhisperPartnerRequired)
	}

	whispers, err := s.whisperRepo.FindByAuthorID(*user.PartnerID, 10)
	if err != nil {
		return nil, err
	}

	items := make([]dto.WhisperItem, len(whispers))
	for i, w := range whispers {
		items[i] = dto.WhisperItem{
			ID:        w.ID,
			Content:   w.Content,
			CreatedAt: w.CreatedAt,
		}
	}

	if len(whispers) == 0 {
		return &dto.WhisperTips{
			Tips:     "她还没有写下心语，也许你可以主动关心一下她今天过得怎么样。",
			Whispers: items,
		}, nil
	}
	if !s.aiClient.Enabled() {
		return &dto.WhisperTips{Tips: openai.ErrNotConfigured.Error(), Whispers: items}, nil
	}

	// Build whisper context
	var sb strings.Builder
	for _, w := range whispers {
		fmt.Fprintf(&sb, "- %s（%s）\n", w.Content, w.CreatedAt.Format("01-02 15:04"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages := []openai.Message{
		{Role: "system", Content: fmt.Sprintf(whisperTipsPrompt, sb.String())},
		{Role: "user", Content: "请给出建议。"},
	}

	tips, err := s.aiClient.Chat(ctx, messages)
	if err != nil {
		log.Printf("[WhisperService] AI tips generation failed: %v", err)
		message := "暂时无法生成建议，请稍后再试。"
		if errors.Is(err, openai.ErrAuthentication) {
			message = openai.ErrAuthentication.Error()
		}
		return &dto.WhisperTips{
			Tips:     message,
			Whispers: items,
		}, nil
	}

	return &dto.WhisperTips{
		Tips:     strings.TrimSpace(tips),
		Whispers: items,
	}, nil
}
