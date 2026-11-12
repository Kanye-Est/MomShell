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
	"github.com/momshell/backend/pkg/llmvalidate"
	"github.com/momshell/backend/pkg/openai"
)

const (
	errWhisperUserNotFound    = "用户不存在"
	errWhisperPartnerRequired = "请先完成伴侣绑定"
)

type WhisperService struct {
	whisperRepo      *repository.WhisperRepo
	futureLetterRepo *repository.FutureLetterRepo
	userRepo         *repository.UserRepo
	chatRepo         *repository.ChatRepo
	taskRepo         *repository.TaskRepo
	aiClient         *openai.Client
	ragService       *RAGService
}

func NewWhisperService(
	whisperRepo *repository.WhisperRepo,
	futureLetterRepo *repository.FutureLetterRepo,
	userRepo *repository.UserRepo,
	chatRepo *repository.ChatRepo,
	taskRepo *repository.TaskRepo,
	aiClient *openai.Client,
	ragService *RAGService,
) *WhisperService {
	return &WhisperService{
		whisperRepo:      whisperRepo,
		futureLetterRepo: futureLetterRepo,
		userRepo:         userRepo,
		chatRepo:         chatRepo,
		taskRepo:         taskRepo,
		aiClient:         aiClient,
		ragService:       ragService,
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

	// Index for RAG in background with UserID for privacy
	if s.ragService != nil {
		go func() {
			ctx := context.Background()
			_ = s.ragService.IndexText(ctx, model.SourceWhisper, w.ID, &authorID, content)
		}()
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

以下是一些参考语录，你可以从中汲取灵感，但不要直接照搬：
- 父亲能为孩子做的最重要的事情，就是爱他们的母亲。
- 陪伴是最长情的告白。
- 最好的教育，是言传身教。
- 真正的强大，是温柔地对待最亲近的人。
- 你不必完美，但你需要在场。
- 用心倾听，是最好的回应。
- 爱是动词，需要每天练习。

请根据这些心语，给她的伴侣（爸爸）一条非常简短、温暖且实用的每日建议。
要求：不要列点，必须在1-2句话内结束，语气像朋友之间简短的日常提醒。
禁止使用”建议、执行、任务、指令、系统、提示、反馈”等词汇，直接说出你对他的叮嘱。`

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

	// Check cache
	ck := coupleKey(user.ID, *user.PartnerID)
	date := time.Now().Format("2006-01-02")
	if cache, err := s.taskRepo.FindAICache(ck, date, "tips"); err == nil && cache != nil {
		return &dto.WhisperTips{
			Tips:     cache.Content,
			Whispers: items,
		}, nil
	}

	if !s.aiClient.Enabled() {
		return &dto.WhisperTips{
			Tips:     "她最近留下了一些心声。先别急着解决问题，今晚可以先问问她最想让你接走哪一件事。",
			Whispers: items,
		}, nil
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

	tips = llmvalidate.Sanitize(tips)

	// Cross-validate: content safety review
	tvr, cvErr := llmvalidate.CrossValidate(ctx, s.aiClient, llmvalidate.TypePlainText, tips, "")
	if cvErr != nil {
		log.Printf("[WhisperService] cross-validate error: %v", cvErr)
	}

	finalTips := strings.TrimSpace(tips)

	// Apply safety appendix if needed
	if tvr != nil {
		finalTips = llmvalidate.ApplyAppendix(finalTips, tvr)
	}

	// Save to cache
	_ = s.taskRepo.SaveAICache(&model.AIGeneratedTask{
		CoupleKey: ck,
		Date:      date,
		Type:      "tips",
		Content:   finalTips,
	})

	return &dto.WhisperTips{
		Tips:     finalTips,
		Whispers: items,
	}, nil
}
