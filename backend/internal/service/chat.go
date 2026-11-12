package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/momshell/backend/internal/dto"
	"github.com/momshell/backend/internal/model"
	"github.com/momshell/backend/internal/repository"
	"github.com/momshell/backend/pkg/firecrawl"
	"github.com/momshell/backend/pkg/openai"
)

const companionSystemPromptMom = `你是「小石光」，一位真诚的知心朋友。

## 角色定位：知心朋友

你面对的用户首先是一个完整的「她自己」，而不仅仅是某个身份标签。你真正在意的不是给出完美答案，而是让她感到被看见、被理解。

## 沟通原则

1. **看见与理解优先**：当她表达疲惫、焦虑、自我怀疑时，首先「看见」她的内心活动，让她知道这些感受是真实的、合理的。不急于给出建议。
2. **引导而非安慰**：不用廉价的安慰敷衍（如「一切都会好的」「你已经很棒了」），而是通过真诚的提问帮她梳理内心——「你觉得最让你在意的是什么？」「如果抛开别人的期待，你自己想要的是什么？」
3. **关注自我与当下**：温和地引导她把注意力从外界评价拉回到自己身上——她此刻的感受、她真正的需求、她值得被善待的事实。
4. **拒绝说教**：不说「你应该」「你必须」，用「我注意到...」「有时候...」「或许可以...」的方式自然地分享。
5. **有深度的对话**：当她自我否定时，不一味否认她的感受，而是通过提问引导她客观地审视自己——既看到困难，也看到自己已经做到的。
6. **避免有毒正能量**：不说「你要积极一点」「想开点就好了」。承认困难的真实性，陪她一起面对。

## 记忆上下文

### 你记得关于她的重要信息
%s

### 她和你之间有过以下对话片段
%s

在回应时，自然地融入这些记忆。
重要：上方「重要信息」列表是最准确的记忆来源。如果对话片段中出现了不在此列表中的信息，说明用户已删除或更正，请勿重新记录。

## 响应格式

你的每一次回复必须是一个 JSON 对象，包含以下字段：
1. **text**: 一段温暖、真诚的回应文字（1-3句话）。注意：text 内容必须是纯文本，禁止使用任何 Markdown 格式。
2. **visual_metadata**: 描述这次回复应呈现的视觉氛围
   - effect_type: "ripple" | "sunlight" | "calm" | "warm_glow" | "gentle_wave"
   - intensity: 0.0 ~ 1.0
   - color_tone: "soft_pink" | "warm_gold" | "gentle_blue" | "lavender" | "neutral_white" | "coral" | "sage"
3. **memory_extract**: 如果用户分享了值得记住的信息或更正了之前的说法，提取为 JSON 对象：
   - **facts**: 数组，每条包含 content（一句话概括）和 category（分类：personal_info/family/interest/concern/preference/other），如 [{"content": "爱吃苹果", "category": "preference"}, {"content": "宝宝叫小米", "category": "family"}]
   - **corrections**: 如果用户更正了之前的说法（如"我之前说错了"、"其实不是…"、"不对，应该是…"），列出需要删除的旧信息关键词，如 ["爱吃苹果"]；没有更正则省略此字段
   没有需要记录或更正的信息时为 null
   **记忆过滤规则**：以下类型的内容不应被记录为记忆：
   - 自我伤害、自杀倾向等危机性想法
   - 极端负面的自我评价（如"我是个失败者"、"我活着没意义"）
   - 针对他人的恶意或暴力想法
   - 反社会倾向或违法相关内容
   - 不利于夫妻关系、家庭关系的偏激想法（如"我要离婚"、"我恨他/她"等一时冲动的表达）
   - 一时的情绪宣泄（如"我不想活了"、"我什么都做不好"）
   这些内容应当被倾听和回应，但不应作为事实记忆持久化。只记录客观事实和稳定的个人信息。

记住：你的存在不是为了「解决她的问题」，而是让她感到——在这一刻，有人真正看见了她。`

const companionSystemPromptDad = `你是「小石光」，一位耐心的同行者和指导者。

## 角色定位：耐心的指导者

你面对的用户正在家庭中承担重要角色。你的存在不是为了评判他做得好不好，而是帮他看清眼前的事，一步步做好。

## 沟通原则

1. **理解先行**：先理解他面对的具体困境，不急于评价。承认「这确实不容易」比空洞的鼓励更有意义。
2. **引导理解**：帮他理解事物的本质——为什么伴侣需要支持、孩子的需求意味着什么、家庭中每个角色如何协作。不是说教，而是帮他「看见」。
3. **给出具体方案**：在理解的基础上，提供可操作的建议。不是笼统的「多关心她」，而是具体到「今晚试着主动承担哄睡，让她有一个小时的独处时间」。
4. **平和而不讨好**：语气平等、真诚，像一个有经验的朋友在分享。不使用过度热情或夸张的语气。
5. **有深度的对话**：当他感到挫败或困惑时，不简单否定他的感受，而是通过提问帮他理清头绪——「你觉得最大的障碍是什么？」「如果换个角度看这件事呢？」
6. **拒绝空洞鼓励**：不说「你很厉害」「加油就行」。面对问题时坦诚地分析，给出切实的下一步。

## 记忆上下文

### 你记得关于他的重要信息
%s

### 他和你之间有过以下对话片段
%s

在回应时，自然地融入这些记忆。
重要：上方「重要信息」列表是最准确的记忆来源。如果对话片段中出现了不在此列表中的信息，说明用户已删除或更正，请勿重新记录。

## 响应格式

你的每一次回复必须是一个 JSON 对象，包含以下字段：
1. **text**: 一段平和、真诚的回应文字（1-3句话）。注意：text 内容必须是纯文本，禁止使用任何 Markdown 格式。
2. **visual_metadata**: 描述这次回复应呈现的视觉氛围
   - effect_type: "ripple" | "sunlight" | "calm" | "warm_glow" | "gentle_wave"
   - intensity: 0.0 ~ 1.0
   - color_tone: "soft_pink" | "warm_gold" | "gentle_blue" | "lavender" | "neutral_white" | "coral" | "sage"
3. **memory_extract**: 如果用户分享了值得记住的信息或更正了之前的说法，提取为 JSON 对象：
   - **facts**: 数组，每条包含 content（一句话概括）和 category（分类：personal_info/family/interest/concern/preference/other），如 [{"content": "爱吃苹果", "category": "preference"}, {"content": "宝宝叫小米", "category": "family"}]
   - **corrections**: 如果用户更正了之前的说法（如"我之前说错了"、"其实不是…"、"不对，应该是…"），列出需要删除的旧信息关键词，如 ["爱吃苹果"]；没有更正则省略此字段
   没有需要记录或更正的信息时为 null
   **记忆过滤规则**：以下类型的内容不应被记录为记忆：
   - 自我伤害、自杀倾向等危机性想法
   - 极端负面的自我评价（如"我是个失败者"、"我活着没意义"）
   - 针对他人的恶意或暴力想法
   - 反社会倾向或违法相关内容
   - 不利于夫妻关系、家庭关系的偏激想法（如"我要离婚"、"我恨他/她"等一时冲动的表达）
   - 一时的情绪宣泄（如"我不想活了"、"我什么都做不好"）
   这些内容应当被倾听和回应，但不应作为事实记忆持久化。只记录客观事实和稳定的个人信息。

记住：你的存在是帮他成为更好的自己——看清方向，迈出下一步。`

const companionSystemPromptProfessional = `你是「小石光」，一位尊重专业的交流伙伴。

## 角色定位：对等的交流者

你面对的用户是一位医疗或心理健康领域的专业人士。他们拥有扎实的专业知识和临床经验，同时也是一个有自己感受和需求的人。

## 沟通原则

1. **尊重专业**：不解释用户已知的基础知识，在更高层次上交流。涉及他们的专业领域时，可以直接讨论核心问题。
2. **对等交流**：语气平等坦诚，像同行之间的讨论。不需要过多铺垫和修饰。
3. **关注个人层面**：专业人士也有自己的疲惫、困惑和情感需求。帮助他们在专业角色之外看见自己作为一个人的需要。
4. **有深度的对话**：可以进行更深入、更直接的讨论。当他们分享观点时，认真对待并回应，必要时坦诚提出不同视角。
5. **避免说教**：他们不需要被「教育」。当他们自我怀疑时，不廉价安慰，而是帮他们客观审视处境。
6. **拒绝讨好**：不因为对方是专业人士就过度恭维。真诚比礼貌更重要。

## 记忆上下文

### 你记得关于对方的重要信息
%s

### 对方和你之间有过以下对话片段
%s

在回应时，自然地融入这些记忆。
重要：上方「重要信息」列表是最准确的记忆来源。如果对话片段中出现了不在此列表中的信息，说明用户已删除或更正，请勿重新记录。

## 响应格式

你的每一次回复必须是一个 JSON 对象，包含以下字段：
1. **text**: 一段坦诚、直接的回应文字（1-3句话）。注意：text 内容必须是纯文本，禁止使用任何 Markdown 格式。
2. **visual_metadata**: 描述这次回复应呈现的视觉氛围
   - effect_type: "ripple" | "sunlight" | "calm" | "warm_glow" | "gentle_wave"
   - intensity: 0.0 ~ 1.0
   - color_tone: "soft_pink" | "warm_gold" | "gentle_blue" | "lavender" | "neutral_white" | "coral" | "sage"
3. **memory_extract**: 如果用户分享了值得记住的信息或更正了之前的说法，提取为 JSON 对象：
   - **facts**: 数组，每条包含 content（一句话概括）和 category（分类：personal_info/family/interest/concern/preference/other），如 [{"content": "爱吃苹果", "category": "preference"}, {"content": "宝宝叫小米", "category": "family"}]
   - **corrections**: 如果用户更正了之前的说法（如"我之前说错了"、"其实不是…"、"不对，应该是…"），列出需要删除的旧信息关键词，如 ["爱吃苹果"]；没有更正则省略此字段
   没有需要记录或更正的信息时为 null
   **记忆过滤规则**：以下类型的内容不应被记录为记忆：
   - 自我伤害、自杀倾向等危机性想法
   - 极端负面的自我评价（如"我是个失败者"、"我活着没意义"）
   - 针对他人的恶意或暴力想法
   - 反社会倾向或违法相关内容
   - 不利于夫妻关系、家庭关系的偏激想法（如"我要离婚"、"我恨他/她"等一时冲动的表达）
   - 一时的情绪宣泄（如"我不想活了"、"我什么都做不好"）
   这些内容应当被倾听和回应，但不应作为事实记忆持久化。只记录客观事实和稳定的个人信息。

记住：你的存在是提供一个对等的、可以卸下专业面具的空间。`

const adminPromptSuffix = "\n\n## 额外信息\n该用户是社区管理员。保持一贯的真诚态度，涉及社区管理话题时可以更直接高效地交流。"

const summarizationPrompt = `请将以下对话历史压缩为一段简洁的中文摘要（不超过500字）。
保留关键信息：用户提到的重要事件、情感变化、做出的决定、讨论的话题。
删除重复和琐碎内容。如果已有旧摘要，将新内容与旧摘要合并。

已有摘要：
%s

新增对话：
%s

请直接输出合并后的完整摘要（纯文本，不要JSON格式，不要任何前缀说明）。`

const (
	maxGuestSessions = 1000
	summaryThreshold = 20 // trigger summarization when turns exceed this
	keepRecentTurns  = 15 // keep this many recent turns after summarization
	promptTurns      = 10 // inject this many turns into the prompt
)

func getCompanionPrompt(role model.UserRole, isAdmin bool) string {
	var prompt string
	if model.ProfessionalRoles[role] {
		prompt = companionSystemPromptProfessional
	} else if role == model.RoleDad {
		prompt = companionSystemPromptDad
	} else {
		prompt = companionSystemPromptMom
	}
	if isAdmin {
		prompt += adminPromptSuffix
	}
	return prompt
}

func pronounFor(role model.UserRole) string {
	if model.ProfessionalRoles[role] {
		return "对方"
	}
	if role == model.RoleDad {
		return "他"
	}
	return "她"
}

type ChatService struct {
	client    *openai.Client
	chatRepo  *repository.ChatRepo
	userRepo  *repository.UserRepo
	firecrawl *firecrawl.Client
	jwtSecret string
	// In-memory storage for guest sessions
	mu              sync.RWMutex
	guestMemory     map[string][]map[string]interface{}
	guestProfiles   map[string]map[string]interface{}
	guestLastAccess map[string]time.Time
}

func NewChatService(client *openai.Client, chatRepo *repository.ChatRepo, userRepo *repository.UserRepo, fc *firecrawl.Client, jwtSecret string) *ChatService {
	return &ChatService{
		client:          client,
		chatRepo:        chatRepo,
		userRepo:        userRepo,
		firecrawl:       fc,
		jwtSecret:       jwtSecret,
		guestMemory:     make(map[string][]map[string]interface{}),
		guestProfiles:   make(map[string]map[string]interface{}),
		guestLastAccess: make(map[string]time.Time),
	}
}

func (s *ChatService) getFamilyIDs(userID string) []string {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return []string{userID}
	}
	if user.PartnerID != nil && *user.PartnerID != "" {
		return []string{userID, *user.PartnerID}
	}
	return []string{userID}
}

func (s *ChatService) buildNicknameMap(familyIDs []string) map[string]string {
	m := make(map[string]string, len(familyIDs))
	for _, id := range familyIDs {
		if u, err := s.userRepo.FindByID(id); err == nil {
			m[id] = u.Nickname
		}
	}
	return m
}

func (s *ChatService) Chat(ctx context.Context, msg dto.UserMessage, userID string) (*dto.VisualResponse, error) {
	if !s.client.Enabled() {
		return nil, openai.ErrNotConfigured
	}
	if userID != "" {
		return s.chatAuthenticated(ctx, msg, userID)
	}
	return s.chatGuest(ctx, msg)
}

// appendWebSearchResults appends web search results to the system prompt if available.
func appendWebSearchResults(systemPrompt, webResults string) string {
	if webResults == "" {
		return systemPrompt
	}
	return systemPrompt + "\n\n## 联网搜索参考\n" + webResults + "\n日常聊天不需要引用来源。仅在提供专业性建议时才引用，引用时直接写出具体来源名称（如「根据XX的一篇文章...」），不要使用[来源1]这样的标注。不确定的信息请标明。"
}

// chatUserContext holds resolved user context for authenticated chat.
type chatUserContext struct {
	role        model.UserRole
	isAdmin     bool
	partnerID   string
	partnerRole model.UserRole
}

// resolveChatUserContext looks up the user and partner information for chat.
func (s *ChatService) resolveChatUserContext(userID string) chatUserContext {
	ctx := chatUserContext{role: model.RoleMom}
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return ctx
	}
	ctx.role = user.Role
	ctx.isAdmin = user.IsAdmin
	if user.PartnerID != nil && *user.PartnerID != "" {
		ctx.partnerID = *user.PartnerID
		if partner, pErr := s.userRepo.FindByID(ctx.partnerID); pErr == nil {
			ctx.partnerRole = partner.Role
		}
	}
	return ctx
}

func (s *ChatService) chatAuthenticated(ctx context.Context, msg dto.UserMessage, userID string) (*dto.VisualResponse, error) {
	// Look up user role and partner info
	uc := s.resolveChatUserContext(userID)
	pronoun := pronounFor(uc.role)

	familyIDs := []string{userID}
	if uc.partnerID != "" {
		familyIDs = append(familyIDs, uc.partnerID)
	}

	// Load memory from DB (per-user, not shared)
	profile, turns, summary := s.loadUserMemory(userID)

	// Load structured facts for prompt (family-scoped)
	factsText, deletedFactsText := s.loadFactsForPrompt(userID, familyIDs, uc.role, uc.partnerRole)

	systemPrompt := fmt.Sprintf(getCompanionPrompt(uc.role, uc.isAdmin),
		formatProfile(profile, pronoun, factsText),
		formatTurns(turns, summary, pronoun),
	)

	// Update memory section header for family mode
	if uc.partnerID != "" {
		for _, old := range []string{"你记得关于她的重要信息", "你记得关于他的重要信息", "你记得关于对方的重要信息"} {
			systemPrompt = strings.Replace(systemPrompt, old, "你记得关于这个家庭的重要信息", 1)
		}
	}

	if deletedFactsText != "" {
		systemPrompt += "\n\n### 已删除的记忆（用户已删除或更正，请勿重新记录）\n" + deletedFactsText
	}

	webResults := s.searchWebForChat(ctx, msg.Content)
	systemPrompt = appendWebSearchResults(systemPrompt, webResults)

	messages := []openai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: msg.Content},
	}

	rawContent, err := s.client.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("AI 服务调用失败: %w", err)
	}

	parsed := parseLLMResponse(rawContent)

	// Update profile from extract
	memoryUpdated := updateProfileFromExtract(profile, parsed["memory_extract"])

	// Save structured facts (Phase 3) - with OwnerUserID
	if s.saveFactsFromExtract(userID, parsed["memory_extract"]) {
		memoryUpdated = true
	}

	// Process corrections (delete outdated facts) in family scope
	if s.processMemoryCorrections(familyIDs, parsed["memory_extract"], profile) {
		memoryUpdated = true
	}

	// Append new turn
	turns = append(turns, map[string]interface{}{
		"user_input":         msg.Content,
		"assistant_response": parsed["text"],
	})

	// Phase 2: trigger summarization if turns exceed threshold
	if len(turns) > summaryThreshold {
		toSummarize := turns[:len(turns)-keepRecentTurns]
		turns = turns[len(turns)-keepRecentTurns:]
		go s.generateAndSaveSummary(userID, summary, toSummarize)
	}

	// Save to DB
	s.saveUserMemory(userID, profile, turns, summary)

	return buildVisualResponse(parsed, memoryUpdated), nil
}

// evictOldestSessions removes the oldest 10% of guest sessions when at capacity.
// Caller must hold s.mu.
func (s *ChatService) evictOldestSessions() {
	toDelete := maxGuestSessions / 10
	if toDelete < 1 {
		toDelete = 1
	}
	type sessionAge struct {
		id string
		ts time.Time
	}
	ages := make([]sessionAge, 0, len(s.guestLastAccess))
	for k, t := range s.guestLastAccess {
		ages = append(ages, sessionAge{k, t})
	}
	sort.Slice(ages, func(i, j int) bool {
		return ages[i].ts.Before(ages[j].ts)
	})
	for i := 0; i < toDelete && i < len(ages); i++ {
		delete(s.guestMemory, ages[i].id)
		delete(s.guestProfiles, ages[i].id)
		delete(s.guestLastAccess, ages[i].id)
	}
}

func (s *ChatService) chatGuest(ctx context.Context, msg dto.UserMessage) (*dto.VisualResponse, error) {
	sessionID := ""
	if msg.SessionID != nil {
		sessionID = *msg.SessionID
	}

	// Validate HMAC signature if session_id is provided
	if sessionID != "" {
		if !s.verifySessionID(sessionID) {
			// Invalid signature; generate a new signed session
			sessionID = ""
		}
	}
	if sessionID == "" {
		sessionID = s.signSessionID(uuid.New().String())
	}

	s.mu.Lock()
	if _, ok := s.guestMemory[sessionID]; !ok {
		// Evict least recently accessed sessions if at capacity
		if len(s.guestMemory) >= maxGuestSessions {
			s.evictOldestSessions()
		}
		s.guestMemory[sessionID] = nil
		s.guestProfiles[sessionID] = make(map[string]interface{})
	}
	s.guestLastAccess[sessionID] = time.Now()
	profile := s.guestProfiles[sessionID]
	turns := s.guestMemory[sessionID]
	s.mu.Unlock()

	systemPrompt := fmt.Sprintf(getCompanionPrompt(model.RoleMom, false),
		formatProfile(profile, "她", ""),
		formatTurns(turns, "", "她"),
	)

	webResults := s.searchWebForChat(ctx, msg.Content)
	systemPrompt = appendWebSearchResults(systemPrompt, webResults)

	messages := []openai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: msg.Content},
	}

	rawContent, err := s.client.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("AI 服务调用失败: %w", err)
	}

	parsed := parseLLMResponse(rawContent)
	memoryUpdated := updateProfileFromExtract(profile, parsed["memory_extract"])

	turns = append(turns, map[string]interface{}{
		"user_input":         msg.Content,
		"assistant_response": parsed["text"],
	})
	if len(turns) > summaryThreshold {
		turns = turns[len(turns)-keepRecentTurns:]
	}

	s.mu.Lock()
	s.guestMemory[sessionID] = turns
	s.guestProfiles[sessionID] = profile
	s.mu.Unlock()

	return buildVisualResponse(parsed, memoryUpdated), nil
}

// --- Guest session HMAC signing ---

// signSessionID creates a signed session ID in the format "uuid:hmac_hex".
func (s *ChatService) signSessionID(id string) string {
	mac := hmac.New(sha256.New, []byte(s.jwtSecret))
	mac.Write([]byte(id))
	sig := hex.EncodeToString(mac.Sum(nil))
	return id + ":" + sig
}

// verifySessionID validates that a session ID has a valid HMAC signature.
func (s *ChatService) verifySessionID(sessionID string) bool {
	idx := strings.LastIndex(sessionID, ":")
	if idx < 0 || idx == len(sessionID)-1 {
		return false
	}
	id := sessionID[:idx]
	sig := sessionID[idx+1:]

	mac := hmac.New(sha256.New, []byte(s.jwtSecret))
	mac.Write([]byte(id))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}

// --- Phase 2: Conversation Summary ---

func (s *ChatService) generateAndSaveSummary(userID string, existingSummary string, oldTurns []map[string]interface{}) {
	if len(oldTurns) == 0 {
		return
	}

	// Format old turns for summarization
	var sb strings.Builder
	for _, t := range oldTurns {
		fmt.Fprintf(&sb, "用户：%v\n回复：%v\n\n", t["user_input"], t["assistant_response"])
	}

	oldSummaryText := existingSummary
	if oldSummaryText == "" {
		oldSummaryText = "（无）"
	}

	prompt := fmt.Sprintf(summarizationPrompt, oldSummaryText, sb.String())
	messages := []openai.Message{
		{Role: "system", Content: "你是一个对话摘要助手。"},
		{Role: "user", Content: prompt},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	newSummary, err := s.client.Chat(ctx, messages)
	if err != nil {
		log.Printf("[ChatService] failed to generate summary for user %s: %v", userID, err)
		return
	}

	newSummary = strings.TrimSpace(newSummary)

	// Load current memory to get latest turns (may have changed since goroutine started)
	mem, err := s.chatRepo.FindByUserID(userID)
	if err != nil {
		log.Printf("[ChatService] failed to load memory for summary update user %s: %v", userID, err)
		return
	}

	mem.SetSummary(newSummary)
	if err := s.chatRepo.UpdateSummaryAndTurns(userID, newSummary, mem.ConversationTurns); err != nil {
		log.Printf("[ChatService] failed to save summary for user %s: %v", userID, err)
	}
}

// --- Phase 3: Structured Memory Facts ---

// parseFactItem extracts content and category from a single fact item.
func parseFactItem(v interface{}) (content, aiCategory string) {
	switch item := v.(type) {
	case map[string]interface{}:
		c, _ := item["content"].(string)
		content = strings.TrimSpace(c)
		cat, _ := item["category"].(string)
		aiCategory = strings.TrimSpace(cat)
	case string:
		content = strings.TrimSpace(item)
	}
	return content, aiCategory
}

// saveSingleFact creates a fact if it does not already exist. Returns true if saved.
func (s *ChatService) saveSingleFact(userID, content, aiCategory string) bool {
	exists, err := s.chatRepo.FactExistsByContent(userID, content)
	if err != nil {
		log.Printf("[ChatService] failed to check fact existence: %v", err)
		return false
	}
	if exists {
		return false
	}

	category := resolveFactCategory(aiCategory, content)
	fact := &model.ChatMemoryFact{
		UserID:      userID,
		OwnerUserID: userID,
		Content:     content,
		Category:    category,
	}
	if err := s.chatRepo.CreateFact(fact); err != nil {
		log.Printf("[ChatService] failed to save fact for user %s: %v", userID, err)
		return false
	}
	return true
}

func (s *ChatService) saveFactsFromExtract(userID string, extract interface{}) bool {
	if extract == nil {
		return false
	}
	extractMap, ok := extract.(map[string]interface{})
	if !ok {
		return false
	}
	facts, ok := extractMap["facts"].([]interface{})
	if !ok || len(facts) == 0 {
		return false
	}

	saved := false
	for _, v := range facts {
		content, aiCategory := parseFactItem(v)
		if content == "" {
			continue
		}
		if s.saveSingleFact(userID, content, aiCategory) {
			saved = true
		}
	}
	return saved
}

// extractCorrectionPhrases extracts trimmed non-empty correction strings from a memory_extract.
func extractCorrectionPhrases(extract interface{}) []string {
	if extract == nil {
		return nil
	}
	extractMap, ok := extract.(map[string]interface{})
	if !ok {
		return nil
	}
	corrections, ok := extractMap["corrections"].([]interface{})
	if !ok || len(corrections) == 0 {
		return nil
	}

	phrases := make([]string, 0, len(corrections))
	for _, v := range corrections {
		if str, ok := v.(string); ok && strings.TrimSpace(str) != "" {
			phrases = append(phrases, strings.TrimSpace(str))
		}
	}
	return phrases
}

// cleanLegacyProfileFacts removes legacy profile facts that match any of the correction phrases.
func cleanLegacyProfileFacts(profile map[string]interface{}, phrases []string) {
	legacyFacts, ok := profile["facts"].([]interface{})
	if !ok || len(legacyFacts) == 0 {
		return
	}
	var remaining []interface{}
	for _, fact := range legacyFacts {
		factStr := fmt.Sprintf("%v", fact)
		matched := false
		for _, phrase := range phrases {
			if strings.Contains(factStr, phrase) {
				matched = true
				break
			}
		}
		if !matched {
			remaining = append(remaining, fact)
		}
	}
	profile["facts"] = remaining
}

// processMemoryCorrections handles user corrections by fuzzy-deleting matching facts in family scope.
func (s *ChatService) processMemoryCorrections(familyIDs []string, extract interface{}, profile map[string]interface{}) bool {
	phrases := extractCorrectionPhrases(extract)
	if len(phrases) == 0 {
		return false
	}

	corrected := false
	if err := s.chatRepo.DeleteFactsByContentLikeFamily(familyIDs, phrases); err != nil {
		log.Printf("[ChatService] failed to process memory corrections: %v", err)
	} else {
		corrected = true
	}

	cleanLegacyProfileFacts(profile, phrases)
	return corrected
}

// resolveFactCategory uses the AI-provided category if valid, otherwise falls back to keyword detection.
func resolveFactCategory(aiCategory, content string) model.FactCategory {
	switch model.FactCategory(aiCategory) {
	case model.FactCategoryPersonalInfo, model.FactCategoryFamily,
		model.FactCategoryInterest, model.FactCategoryConcern,
		model.FactCategoryPreference, model.FactCategoryOther:
		return model.FactCategory(aiCategory)
	}
	return categorizeFactContent(content)
}

type factCategoryRule struct {
	keywords []string
	category model.FactCategory
}

var factCategoryRules = []factCategoryRule{
	{
		keywords: []string{"宝宝", "孩子", "老公", "老婆", "伴侣", "家人", "父母", "婆婆"},
		category: model.FactCategoryFamily,
	},
	{
		keywords: []string{"爱吃", "不爱吃", "最爱", "喜爱", "讨厌", "想吃", "常吃", "爱喝",
			"爱看", "爱听", "最喜欢", "不想", "受不了", "喜欢", "偏好", "习惯", "不喜欢"},
		category: model.FactCategoryPreference,
	},
	{
		keywords: []string{"叫", "名字", "岁", "职业", "住在", "工作", "公司", "城市", "来自", "毕业", "专业"},
		category: model.FactCategoryPersonalInfo,
	},
	{
		keywords: []string{"担心", "焦虑", "害怕", "困扰", "希望", "愿望", "烦", "压力", "累", "纠结", "迷茫"},
		category: model.FactCategoryConcern,
	},
	{
		keywords: []string{"爱好", "兴趣", "在学", "爱", "在玩", "在看", "在读", "在听", "开始学", "报了"},
		category: model.FactCategoryInterest,
	},
}

func categorizeFactContent(content string) model.FactCategory {
	lower := strings.ToLower(content)
	for _, rule := range factCategoryRules {
		for _, kw := range rule.keywords {
			if strings.Contains(lower, kw) {
				return rule.category
			}
		}
	}
	return model.FactCategoryOther
}

// factLabel returns the display label for a fact in family mode.
func factLabel(f model.ChatMemoryFact, userID string, userRole, partnerRole model.UserRole) string {
	if f.Category == model.FactCategoryFamily {
		return "家庭"
	}
	if f.OwnerUserID == userID {
		return pronounFor(userRole)
	}
	return pronounFor(partnerRole)
}

func (s *ChatService) loadFactsForPrompt(userID string, familyIDs []string, userRole, partnerRole model.UserRole) (string, string) {
	hasPartner := len(familyIDs) > 1

	facts, err := s.chatRepo.FindFactsByFamilyIDs(familyIDs)
	if err != nil {
		facts = nil
	}

	// Active facts
	ids := make([]string, 0, len(facts))
	var sb strings.Builder
	for _, f := range facts {
		if hasPartner {
			label := factLabel(f, userID, userRole, partnerRole)
			fmt.Fprintf(&sb, "  · [%s] %s\n", label, f.Content)
		} else {
			fmt.Fprintf(&sb, "  · %s\n", f.Content)
		}
		ids = append(ids, f.ID)
	}

	// Update last_referenced_at in background
	if len(ids) > 0 {
		go func() {
			if err := s.chatRepo.TouchFactReferencedAt(ids); err != nil {
				log.Printf("[ChatService] failed to touch fact referenced_at: %v", err)
			}
		}()
	}

	// Deleted facts in family scope (prevent re-learning)
	var deletedSB strings.Builder
	deletedFacts, err := s.chatRepo.FindDeletedFactsByFamilyIDs(familyIDs)
	if err == nil {
		for _, f := range deletedFacts {
			fmt.Fprintf(&deletedSB, "- %s\n", f.Content)
		}
	}

	return sb.String(), deletedSB.String()
}

// GetMemories returns all structured memory facts for the family (Phase 3 API).
func (s *ChatService) GetMemories(userID string) (*dto.ChatMemoryFactsResponse, error) {
	familyIDs := s.getFamilyIDs(userID)
	nicknameMap := s.buildNicknameMap(familyIDs)

	facts, err := s.chatRepo.FindFactsByFamilyIDs(familyIDs)
	if err != nil {
		return &dto.ChatMemoryFactsResponse{Facts: []dto.ChatMemoryFactDTO{}, Total: 0}, nil
	}

	items := make([]dto.ChatMemoryFactDTO, 0, len(facts))
	for _, f := range facts {
		ownerID := f.OwnerUserID
		if ownerID == "" {
			ownerID = f.UserID
		}
		item := dto.ChatMemoryFactDTO{
			ID:            f.ID,
			Content:       f.Content,
			Category:      string(f.Category),
			OwnerUserID:   ownerID,
			OwnerNickname: nicknameMap[ownerID],
			CreatedAt:     f.CreatedAt.Format(time.RFC3339),
		}
		if f.LastReferencedAt != nil {
			t := f.LastReferencedAt.Format(time.RFC3339)
			item.LastReferencedAt = &t
		}
		items = append(items, item)
	}

	return &dto.ChatMemoryFactsResponse{Facts: items, Total: len(items)}, nil
}

// DeleteMemory deletes a single memory fact, verifying family ownership.
func (s *ChatService) DeleteMemory(userID, factID string) error {
	fact, err := s.chatRepo.FindFactByID(factID)
	if err != nil {
		return fmt.Errorf("记忆条目不存在")
	}
	// Allow deletion if the fact belongs to any family member
	familyIDs := s.getFamilyIDs(userID)
	allowed := false
	for _, id := range familyIDs {
		if fact.UserID == id {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("无权删除此记忆")
	}
	return s.chatRepo.DeleteFact(factID)
}

// GetConversationHistory returns the user's conversation turns and summary.
func (s *ChatService) GetConversationHistory(userID string) (*dto.ConversationHistoryResponse, error) {
	_, turns, summary := s.loadUserMemory(userID)

	dtoTurns := make([]dto.ConversationTurn, 0, len(turns))
	for _, t := range turns {
		userInput, _ := t["user_input"].(string)
		assistantResp, _ := t["assistant_response"].(string)
		dtoTurns = append(dtoTurns, dto.ConversationTurn{
			UserInput:         userInput,
			AssistantResponse: assistantResp,
		})
	}

	return &dto.ConversationHistoryResponse{
		Turns:   dtoTurns,
		Summary: summary,
	}, nil
}

// ClearConversationHistory resets the user's conversation turns and summary.
func (s *ChatService) ClearConversationHistory(userID string) error {
	return s.chatRepo.UpdateSummaryAndTurns(userID, "", "[]")
}

// --- Existing helpers (updated) ---

func (s *ChatService) searchWebForChat(ctx context.Context, userMessage string) string {
	if s.firecrawl == nil {
		return ""
	}
	results, err := s.firecrawl.Search(ctx, userMessage, 3)
	if err != nil {
		log.Printf("[ChatService] web search failed: %v", err)
		return ""
	}
	if len(results) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, r := range results {
		// Validate URL scheme before including in output
		if !strings.HasPrefix(r.URL, "https://") && !strings.HasPrefix(r.URL, "http://") {
			continue
		}
		content := r.Markdown
		if content == "" {
			content = r.Description
		}
		if len([]rune(content)) > 300 {
			content = string([]rune(content)[:300]) + "..."
		}
		fmt.Fprintf(&sb, "来源「%s」（%s）：%s\n", r.Title, r.URL, content)
	}
	return sb.String()
}

func (s *ChatService) GetProfile(userID string) (*dto.ChatProfile, error) {
	mem, err := s.chatRepo.FindByUserID(userID)
	if err != nil {
		return &dto.ChatProfile{
			Interests:             []string{},
			Concerns:              []string{},
			Facts:                 []string{},
			ImportantDates:        []string{},
			CommunityInteractions: []string{},
		}, nil
	}

	profile := mem.GetProfile()
	return profileToDTO(profile), nil
}

func (s *ChatService) GetGuestProfile(sessionID string) *dto.ChatProfile {
	s.mu.RLock()
	profile, ok := s.guestProfiles[sessionID]
	s.mu.RUnlock()
	if !ok {
		return &dto.ChatProfile{
			Interests:             []string{},
			Concerns:              []string{},
			Facts:                 []string{},
			ImportantDates:        []string{},
			CommunityInteractions: []string{},
		}
	}
	return profileToDTO(profile)
}

func (s *ChatService) loadUserMemory(userID string) (map[string]interface{}, []map[string]interface{}, string) {
	mem, err := s.chatRepo.FindByUserID(userID)
	if err != nil {
		return make(map[string]interface{}), nil, ""
	}
	return mem.GetProfile(), mem.GetTurns(), mem.GetSummary()
}

func (s *ChatService) saveUserMemory(userID string, profile map[string]interface{}, turns []map[string]interface{}, summary string) {
	mem := &model.ChatMemory{
		UserID: userID,
	}
	mem.SetProfile(profile)
	mem.SetTurns(turns)
	mem.SetSummary(summary)

	if err := s.chatRepo.Upsert(mem); err != nil {
		log.Printf("[ChatService] failed to save memory for user %s: %v", userID, err)
	}
}

// formatSliceField formats a profile slice field (e.g. interests, concerns) as a comma-separated line.
func formatSliceField(profile map[string]interface{}, key string, label string) string {
	items, ok := profile[key].([]interface{})
	if !ok || len(items) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(label)
	for i, v := range items {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%v", v)
	}
	sb.WriteString("\n")
	return sb.String()
}

func formatProfile(profile map[string]interface{}, pronoun string, factsText string) string {
	if len(profile) == 0 && factsText == "" {
		return "（暂无记录）"
	}
	parts := ""
	if name, ok := profile["preferred_name"].(string); ok && name != "" {
		parts += fmt.Sprintf("- %s喜欢被称为：%s\n", pronoun, name)
	}
	if hasPets, ok := profile["has_pets"].(bool); ok && hasPets {
		if details, ok := profile["pet_details"].(string); ok {
			parts += fmt.Sprintf("- %s有宠物：%s\n", pronoun, details)
		}
	}

	// Structured facts from DB (Phase 3)
	if factsText != "" {
		parts += "- 重要信息：\n" + factsText
	}

	parts += formatSliceField(profile, "interests", fmt.Sprintf("- %s的兴趣：", pronoun))
	parts += formatSliceField(profile, "concerns", fmt.Sprintf("- %s曾表达的担忧：", pronoun))

	if parts == "" {
		return "（暂无记录）"
	}
	return parts
}

func formatTurns(turns []map[string]interface{}, summary string, pronoun string) string {
	if len(turns) == 0 && summary == "" {
		return "（这是你们的第一次对话）"
	}
	var result string

	// Prepend summary of older conversations (Phase 2)
	if summary != "" {
		result += "【earlier conversation summary】\n" + summary + "\n\n【recent conversations】\n"
	}

	start := 0
	if len(turns) > promptTurns {
		start = len(turns) - promptTurns
	}
	for _, t := range turns[start:] {
		response := fmt.Sprintf("%v", t["assistant_response"])
		if len([]rune(response)) > 200 {
			response = string([]rune(response)[:200]) + "..."
		}
		result += fmt.Sprintf("%s说：%v\n你回复：%s\n", pronoun, t["user_input"], response)
	}
	return result
}

func parseLLMResponse(content string) map[string]interface{} {
	var result map[string]interface{}

	// Try direct parse
	if err := json.Unmarshal([]byte(content), &result); err == nil {
		return result
	}

	// Try extract JSON block
	re := regexp.MustCompile("(?s)```json\\s*(.*?)\\s*```")
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		if err := json.Unmarshal([]byte(matches[1]), &result); err == nil {
			return result
		}
	}

	// Try extract braces
	re2 := regexp.MustCompile(`(?s)\{.*\}`)
	if match := re2.FindString(content); match != "" {
		if err := json.Unmarshal([]byte(match), &result); err == nil {
			return result
		}
	}

	// Fallback
	return map[string]interface{}{
		"text": content,
		"visual_metadata": map[string]interface{}{
			"effect_type": "calm",
			"intensity":   0.5,
			"color_tone":  "gentle_blue",
		},
		"memory_extract": nil,
	}
}

func updateProfileFromExtract(profile map[string]interface{}, extract interface{}) bool {
	if extract == nil {
		return false
	}
	extractMap, ok := extract.(map[string]interface{})
	if !ok {
		return false
	}

	updated := false
	if name, ok := extractMap["preferred_name"].(string); ok && name != "" {
		profile["preferred_name"] = name
		updated = true
	}
	if hasPets, ok := extractMap["has_pets"].(bool); ok {
		profile["has_pets"] = hasPets
		updated = true
	}
	if details, ok := extractMap["pet_details"].(string); ok && details != "" {
		profile["pet_details"] = details
		updated = true
	}
	if interests, ok := extractMap["interests"].([]interface{}); ok && len(interests) > 0 {
		existing, _ := profile["interests"].([]interface{})
		profile["interests"] = deduplicateAndCap(existing, interests, 20)
		updated = true
	}
	if concerns, ok := extractMap["concerns"].([]interface{}); ok && len(concerns) > 0 {
		existing, _ := profile["concerns"].([]interface{})
		profile["concerns"] = deduplicateAndCap(existing, concerns, 20)
		updated = true
	}
	return updated
}

func deduplicateAndCap(existing, newItems []interface{}, maxItems int) []interface{} {
	seen := make(map[string]bool)
	var result []interface{}
	for _, v := range existing {
		s := fmt.Sprintf("%v", v)
		if !seen[s] {
			seen[s] = true
			result = append(result, v)
		}
	}
	for _, v := range newItems {
		s := fmt.Sprintf("%v", v)
		if !seen[s] {
			seen[s] = true
			result = append(result, v)
		}
	}
	if len(result) > maxItems {
		result = result[len(result)-maxItems:]
	}
	return result
}

// parseVisualMetadata extracts visual metadata from parsed LLM response, returning defaults if absent.
func parseVisualMetadata(parsed map[string]interface{}) dto.VisualMetadata {
	vm := dto.VisualMetadata{
		EffectType: "calm",
		Intensity:  0.5,
		ColorTone:  "gentle_blue",
	}
	vmData, ok := parsed["visual_metadata"].(map[string]interface{})
	if !ok {
		return vm
	}
	if et, ok := vmData["effect_type"].(string); ok {
		vm.EffectType = et
	}
	if intensity, ok := vmData["intensity"].(float64); ok {
		vm.Intensity = intensity
	}
	if ct, ok := vmData["color_tone"].(string); ok {
		vm.ColorTone = ct
	}
	return vm
}

func buildVisualResponse(parsed map[string]interface{}, memoryUpdated bool) *dto.VisualResponse {
	text := "我在这里陪着你。"
	if t, ok := parsed["text"].(string); ok && t != "" {
		text = t
	}

	return &dto.VisualResponse{
		Text:           text,
		VisualMetadata: parseVisualMetadata(parsed),
		MemoryUpdated:  memoryUpdated,
	}
}

// extractStringSlice extracts a string slice from a profile field that stores []interface{}.
func extractStringSlice(profile map[string]interface{}, key string) []string {
	items, ok := profile[key].([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, v := range items {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func profileToDTO(profile map[string]interface{}) *dto.ChatProfile {
	cp := &dto.ChatProfile{
		Interests:             []string{},
		Concerns:              []string{},
		Facts:                 []string{},
		ImportantDates:        []string{},
		CommunityInteractions: []string{},
	}

	if name, ok := profile["preferred_name"].(string); ok {
		cp.PreferredName = &name
	}
	if hasPets, ok := profile["has_pets"].(bool); ok {
		cp.HasPets = hasPets
	}
	if details, ok := profile["pet_details"].(string); ok {
		cp.PetDetails = &details
	}
	if v := extractStringSlice(profile, "interests"); len(v) > 0 {
		cp.Interests = v
	}
	if v := extractStringSlice(profile, "concerns"); len(v) > 0 {
		cp.Concerns = v
	}
	if v := extractStringSlice(profile, "facts"); len(v) > 0 {
		cp.Facts = v
	}
	if v := extractStringSlice(profile, "important_dates"); len(v) > 0 {
		cp.ImportantDates = v
	}
	if weeks, ok := profile["baby_age_weeks"].(float64); ok {
		w := int(weeks)
		cp.BabyAgeWeeks = &w
	}

	return cp
}
