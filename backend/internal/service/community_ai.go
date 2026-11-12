package service

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/momshell/backend/internal/model"
	"github.com/momshell/backend/internal/repository"
	"github.com/momshell/backend/pkg/firecrawl"
	"github.com/momshell/backend/pkg/llmvalidate"
	"github.com/momshell/backend/pkg/openai"
)

var mentionPattern = regexp.MustCompile(`@小石光`)

const logFindQuestionFailed = "[CommunityAI] failed to find question %s: %v"

type CommunityAIService struct {
	client       *openai.Client
	firecrawl    *firecrawl.Client
	questionRepo *repository.QuestionRepo
	answerRepo   *repository.AnswerRepo
	commentRepo  *repository.CommentRepo
	userRepo     *repository.UserRepo
	aiUserID     string
}

func NewCommunityAIService(
	client *openai.Client,
	fc *firecrawl.Client,
	questionRepo *repository.QuestionRepo,
	answerRepo *repository.AnswerRepo,
	commentRepo *repository.CommentRepo,
	userRepo *repository.UserRepo,
	aiUserID string,
) *CommunityAIService {
	return &CommunityAIService{
		client:       client,
		firecrawl:    fc,
		questionRepo: questionRepo,
		answerRepo:   answerRepo,
		commentRepo:  commentRepo,
		userRepo:     userRepo,
		aiUserID:     aiUserID,
	}
}

// ContainsMention checks if content mentions @小石光.
func ContainsMention(content string) bool {
	return mentionPattern.MatchString(content)
}

// IsMentioned checks if the given content mentions @小石光.
func (s *CommunityAIService) IsMentioned(content string) bool {
	return s.client.Enabled() && ContainsMention(content)
}

// ShouldReplyToComment returns true if AI should reply to this comment:
// - comment mentions @小石光, OR
// - comment is on an AI-authored answer, OR
// - comment is a reply to an AI-authored comment.
func (s *CommunityAIService) ShouldReplyToComment(content, answerID string, parentID *string) bool {
	if !s.client.Enabled() {
		return false
	}
	if ContainsMention(content) {
		return true
	}
	answer, err := s.answerRepo.FindByID(answerID)
	if err == nil && answer.AuthorID == s.aiUserID {
		return true
	}
	if parentID != nil {
		parent, err := s.commentRepo.FindByID(*parentID)
		if err == nil && parent.AuthorID == s.aiUserID {
			return true
		}
	}
	return false
}

// HandleNewQuestion generates an AI answer for a question that mentions @小石光.
func (s *CommunityAIService) HandleNewQuestion(questionID string) {
	if !s.client.Enabled() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	q, err := s.questionRepo.FindByID(questionID)
	if err != nil {
		log.Printf(logFindQuestionFailed, questionID, err)
		return
	}

	threadCtx := fmt.Sprintf("帖子标题：%s\n帖子内容：%s", q.Title, q.Content)
	searchCtx, sources := s.searchWeb(ctx, q.Title)

	authorRole, authorIsAdmin := s.lookupUserRole(q.AuthorID)
	reply, err := s.generateReply(ctx, threadCtx, searchCtx, sources, authorRole, authorIsAdmin)
	if err != nil {
		log.Printf("[CommunityAI] failed to generate reply for question %s: %v", questionID, err)
		return
	}

	answer := &model.Answer{
		QuestionID:     questionID,
		AuthorID:       s.aiUserID,
		Content:        reply,
		AuthorRole:     model.RoleAIAssistant,
		IsProfessional: false,
		Status:         model.StatusPublished,
	}

	if err := s.answerRepo.Create(answer); err != nil {
		log.Printf("[CommunityAI] failed to create answer for question %s: %v", questionID, err)
		return
	}

	_ = s.questionRepo.IncrementAnswerCount(questionID)
	log.Printf("[CommunityAI] AI answered question %s", questionID)
}

// HandleNewAnswer generates an AI comment on an answer that mentions @小石光.
func (s *CommunityAIService) HandleNewAnswer(questionID, answerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	q, err := s.questionRepo.FindByID(questionID)
	if err != nil {
		log.Printf(logFindQuestionFailed, questionID, err)
		return
	}

	answer, err := s.answerRepo.FindByID(answerID)
	if err != nil {
		log.Printf("[CommunityAI] failed to find answer %s: %v", answerID, err)
		return
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "帖子标题：%s\n帖子内容：%s\n\n", q.Title, q.Content)
	fmt.Fprintf(&sb, "用户回答：%s", answer.Content)

	searchCtx, sources := s.searchWeb(ctx, q.Title)

	authorRole, authorIsAdmin := s.lookupUserRole(answer.AuthorID)
	reply, err := s.generateReply(ctx, sb.String(), searchCtx, sources, authorRole, authorIsAdmin)
	if err != nil {
		log.Printf("[CommunityAI] failed to generate reply for answer %s: %v", answerID, err)
		return
	}

	comment := &model.Comment{
		AnswerID:      answerID,
		AuthorID:      s.aiUserID,
		ReplyToUserID: &answer.AuthorID,
		Content:       reply,
		Status:        model.StatusPublished,
	}

	if err := s.commentRepo.Create(comment); err != nil {
		log.Printf("[CommunityAI] failed to create comment on answer %s: %v", answerID, err)
		return
	}

	_ = s.answerRepo.IncrementCommentCount(answerID)
	log.Printf("[CommunityAI] AI commented on answer %s", answerID)
}

// HandleNewComment generates an AI comment reply when a comment mentions @小石光.
func (s *CommunityAIService) HandleNewComment(answerID, commentID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	answer, err := s.answerRepo.FindByID(answerID)
	if err != nil {
		log.Printf("[CommunityAI] failed to find answer %s: %v", answerID, err)
		return
	}

	q, err := s.questionRepo.FindByID(answer.QuestionID)
	if err != nil {
		log.Printf(logFindQuestionFailed, answer.QuestionID, err)
		return
	}

	comments, err := s.commentRepo.FindByAnswerID(answerID)
	if err != nil {
		log.Printf("[CommunityAI] failed to find comments for answer %s: %v", answerID, err)
		return
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "帖子标题：%s\n帖子内容：%s\n\n", q.Title, q.Content)
	fmt.Fprintf(&sb, "当前回答：%s\n\n评论区对话：\n", answer.Content)
	for _, c := range comments {
		role := "用户"
		if c.AuthorID == s.aiUserID {
			role = "小石光"
		}
		fmt.Fprintf(&sb, "%s：%s\n", role, c.Content)
	}

	searchCtx, sources := s.searchWeb(ctx, q.Title)

	triggerComment, err := s.commentRepo.FindByID(commentID)
	if err != nil {
		log.Printf("[CommunityAI] failed to find trigger comment %s: %v", commentID, err)
		return
	}

	authorRole, authorIsAdmin := s.lookupUserRole(triggerComment.AuthorID)
	reply, err := s.generateReply(ctx, sb.String(), searchCtx, sources, authorRole, authorIsAdmin)
	if err != nil {
		log.Printf("[CommunityAI] failed to generate comment reply: %v", err)
		return
	}

	comment := &model.Comment{
		AnswerID:      answerID,
		AuthorID:      s.aiUserID,
		ParentID:      &commentID,
		ReplyToUserID: &triggerComment.AuthorID,
		Content:       reply,
		Status:        model.StatusPublished,
	}

	if err := s.commentRepo.Create(comment); err != nil {
		log.Printf("[CommunityAI] failed to create comment reply: %v", err)
		return
	}

	_ = s.answerRepo.IncrementCommentCount(answerID)
	log.Printf("[CommunityAI] AI replied to comment %s on answer %s", commentID, answerID)
}

type sourceRef struct {
	index int
	title string
	url   string
}

func (s *CommunityAIService) searchWeb(ctx context.Context, query string) (string, []sourceRef) {
	if s.firecrawl == nil {
		return "", nil
	}

	results, err := s.firecrawl.Search(ctx, query, 3)
	if err != nil {
		log.Printf("[CommunityAI] web search failed: %v", err)
		return "", nil
	}
	if len(results) == 0 {
		return "", nil
	}

	var sources []sourceRef
	var sb strings.Builder
	for i, r := range results {
		content := r.Markdown
		if content == "" {
			content = r.Description
		}
		if len([]rune(content)) > 500 {
			content = string([]rune(content)[:500]) + "..."
		}
		fmt.Fprintf(&sb, "来源「%s」（%s）：\n%s\n\n", r.Title, r.URL, content)
		sources = append(sources, sourceRef{index: i + 1, title: r.Title, url: r.URL})
	}
	return sb.String(), sources
}

func (s *CommunityAIService) lookupUserRole(userID string) (model.UserRole, bool) {
	if s.userRepo == nil {
		return model.RoleMom, false
	}
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return model.RoleMom, false
	}
	return user.Role, user.IsAdmin
}

const communityAISystemPromptMom = `你是「小石光」，一位真诚的知心朋友，正在社区帖子中回复用户。

## 角色定位：知心朋友
你面对的用户首先是一个完整的「她自己」。你真正在意的不是给出完美答案，而是让她感到被看见、被理解。

## 回复原则
1. **看见与理解优先**：先看见和理解她的内心活动，让她知道这些感受是真实的、合理的
2. **引导而非安慰**：不用「一切都会好的」「你已经很棒了」这类廉价安慰，而是通过提问帮她梳理内心——「你觉得最在意的是什么？」
3. **关注自我与当下**：温和地引导她把注意力拉回自己身上——她此刻的感受、她真正的需求
4. **拒绝说教**：用「我了解到...」「有研究表明...」「或许可以试试...」的方式分享
5. **有深度的回应**：当她自我否定时，不一味否认，而是通过提问引导她客观审视自己——既看到困难，也看到已经做到的
6. **避免有毒正能量**：承认困难的真实性

## 帖子上下文
%s

## 联网搜索结果
%s

## 引用与防幻觉规则（严格遵守）
1. 只基于上述帖子上下文和搜索结果回答
2. 日常共情、鼓励、生活建议不需要添加引用来源
3. 仅在提供专业性建议时（如医学知识、研究数据、权威指南）才引用来源
4. 引用来源时，直接在回复中写出具体来源名称和链接（如「根据XX的一篇文章（链接）...」），不要使用[来源1]这样的标注方式
5. 如果不确定，明确说「关于这一点我不太确定，建议咨询专业人士」
6. 绝不编造医疗数据、药物剂量、具体治疗方案
7. 涉及医疗问题时，始终建议咨询专业医生

## 回复要求
- 用纯文本回复，不要使用 JSON 格式
- 语气温暖自然，像朋友聊天
- 1-3段话即可，不要过长
- 不要重复用户说过的话
- 移除@提及，直接回复内容`

const communityAISystemPromptDad = `你是「小石光」，一位耐心的同行者和指导者，正在社区帖子中回复用户。

## 角色定位：耐心的指导者
你面对的用户正在家庭中承担重要角色。你的存在不是为了评判他做得好不好，而是帮他看清眼前的事，一步步做好。

## 回复原则
1. **理解先行**：先理解他面对的具体困境，不急于评价。承认「这确实不容易」比空洞的鼓励更有意义
2. **引导理解**：帮他理解事物的本质——伴侣的需求、孩子的成长、家庭角色的协作。不是说教，而是帮他「看见」
3. **给出具体方案**：在理解的基础上，提供可操作的建议。不是笼统的「多关心她」，而是具体的行动
4. **平和而不讨好**：语气平等、真诚，像一个有经验的朋友在分享
5. **有深度的回应**：当他感到挫败时，不简单否定感受，而是通过提问帮他理清头绪——「你觉得最大的障碍是什么？」
6. **拒绝空洞鼓励**：不说「你很厉害」「加油就行」。面对问题时坦诚分析，给出切实的下一步

## 帖子上下文
%s

## 联网搜索结果
%s

## 引用与防幻觉规则（严格遵守）
1. 只基于上述帖子上下文和搜索结果回答
2. 日常共情、鼓励、生活建议不需要添加引用来源
3. 仅在提供专业性建议时（如医学知识、研究数据、权威指南）才引用来源
4. 引用来源时，直接在回复中写出具体来源名称和链接（如「根据XX的一篇文章（链接）...」），不要使用[来源1]这样的标注方式
5. 如果不确定，明确说「关于这一点我不太确定，建议咨询专业人士」
6. 绝不编造医疗数据、药物剂量、具体治疗方案
7. 涉及医疗问题时，始终建议咨询专业医生

## 回复要求
- 用纯文本回复，不要使用 JSON 格式
- 语气平和真诚
- 1-3段话即可，不要过长
- 不要重复用户说过的话
- 移除@提及，直接回复内容`

const communityAISystemPromptProfessional = `你是「小石光」，正在社区帖子中与一位医疗/心理健康领域的专业人士交流。

## 角色定位：对等的交流者
你面对的用户拥有专业知识和临床经验。尊重他们的专业背景，在更高层次上交流。

## 回复原则
1. **尊重专业**：不解释基础知识，直接讨论核心问题
2. **对等交流**：语气平等坦诚，像同行之间的讨论
3. **关注个人层面**：即使是专业人士，分享困惑时也需要被认真对待
4. **有深度的回应**：可以更直接、更深入地讨论，不需要简化
5. **避免讨好**：不因对方是专业人士就过度恭维，真诚比礼貌更重要
6. **坦诚交流**：如果信息超出范围，直接说明

## 帖子上下文
%s

## 联网搜索结果
%s

## 引用与防幻觉规则（严格遵守）
1. 只基于上述帖子上下文和搜索结果回答
2. 日常交流不需要添加引用来源
3. 仅在提供专业性建议时（如引用研究数据、临床指南）才引用来源
4. 引用来源时，直接在回复中写出具体来源名称和链接（如「根据XX的一篇文章（链接）...」），不要使用[来源1]这样的标注方式
5. 如果不确定，坦诚说明
6. 绝不编造医疗数据、药物剂量、具体治疗方案
7. 涉及具体诊疗方案时，建议结合临床实际判断

## 回复要求
- 用纯文本回复，不要使用 JSON 格式
- 语气平等直接
- 1-3段话即可，不要过长
- 不要重复用户说过的话
- 移除@提及，直接回复内容`

func getCommunityAIPrompt(role model.UserRole, isAdmin bool) string {
	var prompt string
	if model.ProfessionalRoles[role] {
		prompt = communityAISystemPromptProfessional
	} else if role == model.RoleDad {
		prompt = communityAISystemPromptDad
	} else {
		prompt = communityAISystemPromptMom
	}
	if isAdmin {
		prompt += "\n\n## 额外信息\n该用户是社区管理员。保持真诚态度，涉及社区管理话题时可以更直接高效地交流。"
	}
	return prompt
}

func (s *CommunityAIService) generateReply(ctx context.Context, threadContext, searchContext string, sources []sourceRef, role model.UserRole, isAdmin bool) (string, error) {
	if searchContext == "" {
		searchContext = "（无搜索结果）"
	}

	systemPrompt := fmt.Sprintf(getCommunityAIPrompt(role, isAdmin), threadContext, searchContext)

	messages := []openai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: "请根据帖子上下文，给出一个温暖、有帮助的回复。"},
	}

	reply, err := s.client.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("AI 服务调用失败: %w", err)
	}

	reply = llmvalidate.Sanitize(reply)

	// Cross-validate: content safety review
	cvr, cvErr := llmvalidate.CrossValidate(ctx, s.client, llmvalidate.TypePlainText, reply, "")
	if cvErr != nil {
		log.Printf("[CommunityAI] cross-validate error: %v", cvErr)
	}

	reply = mentionPattern.ReplaceAllString(reply, "")
	reply = strings.TrimSpace(reply)

	if reply == "" {
		reply = "谢谢你的分享。如果需要更多帮助，随时可以找我聊聊。"
	}

	// Apply safety appendix if needed
	if cvr != nil {
		reply = llmvalidate.ApplyAppendix(reply, cvr)
	}

	// Replace [来源N] markers with actual source names (fallback)
	reply = replaceSourceReferences(reply, sources)

	return reply, nil
}

var sourceRefPattern = regexp.MustCompile(`\[来源(\d+)\]`)

// replaceSourceReferences replaces any [来源N] markers in the reply with
// the actual source title and URL inline, as a fallback in case the AI
// still uses the old numbered citation format.
func replaceSourceReferences(reply string, sources []sourceRef) string {
	if len(sources) == 0 {
		return reply
	}

	matches := sourceRefPattern.FindAllStringSubmatch(reply, -1)
	if len(matches) == 0 {
		return reply
	}

	// Build source index map for quick lookup
	sourceMap := make(map[int]sourceRef)
	for _, src := range sources {
		sourceMap[src.index] = src
	}

	// Replace [来源N] with actual source name and URL inline
	return sourceRefPattern.ReplaceAllStringFunc(reply, func(match string) string {
		sub := sourceRefPattern.FindStringSubmatch(match)
		var idx int
		if _, err := fmt.Sscanf(sub[1], "%d", &idx); err == nil {
			if src, ok := sourceMap[idx]; ok {
				return fmt.Sprintf("（来源：%s %s）", src.title, src.url)
			}
		}
		return match
	})
}
