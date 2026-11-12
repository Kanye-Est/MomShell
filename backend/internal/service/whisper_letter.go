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
	"github.com/momshell/backend/pkg/llmvalidate"
	"github.com/momshell/backend/pkg/openai"
	"gorm.io/gorm"
)

var errFutureLetterChanged = errors.New("心语情报已更新，请刷新后再试")

const (
	futureLetterCodeOverview   = "future-self-v1"
	futureLetterCodePrenatal   = "future-prenatal-v1"
	futureLetterCodeVaccine    = "future-vaccine-v1"
	futureLetterCodeSafetySeat = "future-safety-seat-v1"
)

type futureLetterOptionDef struct {
	ID    string
	Label string
	Hint  string
	Tag   string
}

type futureLetterTemplate struct {
	Code       string
	Title      string
	Intro      string
	Outro      string
	Signature  string
	WishPrompt string
	SceneHint  string
	Questions  []dto.FutureLetterQuestion
}

type dadMissionTaskDef struct {
	Title       string
	Description string
	Category    string
	Priority    string
	Difficulty  int
}

type dadMissionTemplate struct {
	Code      string
	Title     string
	Headline  string
	Summary   string
	Tasks     []dadMissionTaskDef
	ImageTone string
}

type futureLetterAIMission struct {
	Title    string       `json:"title"`
	Headline string       `json:"headline"`
	Summary  string       `json:"summary"`
	Tasks    []AITaskData `json:"tasks"`
}

type dadAdviceItemDef struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
}

type dadAdviceSourceDef struct {
	Label  string `json:"label"`
	Detail string `json:"detail"`
}

type dadAdviceTemplate struct {
	Code     string
	Title    string
	Headline string
	Summary  string
	Sources  []dadAdviceSourceDef
	Items    []dadAdviceItemDef
}

type futureLetterAIAdvice struct {
	Title    string               `json:"title"`
	Headline string               `json:"headline"`
	Summary  string               `json:"summary"`
	Sources  []dadAdviceSourceDef `json:"sources"`
	Items    []dadAdviceItemDef   `json:"items"`
}

type futureLetterAdviceContext struct {
	Template       futureLetterTemplate
	AgeStage       string
	AgeLabel       string
	PrimaryTag     string
	PrimaryLabel   string
	SecondaryTag   string
	SecondaryLabel string
	Wish           string
	MemorySummary  string
	TurnLines      []string
	FactLines      []string
	WhisperLines   []string
	ResponseLines  []string
}

var overviewStageOptions = []futureLetterOptionDef{
	{ID: "pregnancy", Label: "还在“合体”中，数着 TA 的胎动", Hint: "孕期", Tag: "PREGNANCY"},
	{ID: "infant_0_12m", Label: "TA 还很软糯，世界中心就是我的怀抱", Hint: "0-1岁", Tag: "INFANT_0_12M"},
	{ID: "toddler_2y", Label: "已是“满地跑”的探险家，每天奇思妙想", Hint: "2岁+", Tag: "TODDLER_2Y"},
}

var overviewStateOptions = []futureLetterOptionDef{
	{ID: "physical_pain", Label: "像生锈的机器，只想关机重启", Hint: "身体疼痛/需修复", Tag: "PHYSICAL_PAIN"},
	{ID: "exhausted", Label: "整体还行，但想舒展下筋骨，喘口气", Hint: "极度疲惫/需补觉", Tag: "EXHAUSTED"},
	{ID: "mental_space", Label: "状态不错，但想脱下“妈妈”的身份，去透透气", Hint: "精神压力/需社交空间", Tag: "MENTAL_SPACE"},
}

var vaccineProgressOptions = []futureLetterOptionDef{
	{ID: "vaccine_observing", Label: "正在按部就班：刚打完上一针，正在观察期", Hint: "确定当前进度", Tag: "VACCINE_ON_TRACK"},
	{ID: "vaccine_key_node", Label: "到了关键节点：下周似乎有好几个“大家伙”要接种", Hint: "识别近期任务", Tag: "VACCINE_KEY_NODE"},
	{ID: "vaccine_confused", Label: "略显迷茫：疫苗本密密麻麻，还没理清接下来该去哪", Hint: "触发整理需求", Tag: "VACCINE_CONFUSED"},
}

var vaccineSupportOptions = []futureLetterOptionDef{
	{ID: "vaccine_support", Label: "物理防御：现场人多手乱，我需要能够带娃、排队、拿药的辅助", Hint: "接种日陪同", Tag: "VACCINE_PHYSICAL_SUPPORT"},
	{ID: "vaccine_booking", Label: "脑力搜索：我对疫苗预约的注意事项很困惑", Hint: "预约和资料整理", Tag: "VACCINE_BOOKING_CONFUSION"},
}

var prenatalProgressOptions = []futureLetterOptionDef{
	{ID: "prenatal_regular", Label: "按节奏推进：产检在计划里，只是每次准备都很碎", Hint: "常规产检支持", Tag: "PRENATAL_REGULAR"},
	{ID: "prenatal_key_node", Label: "到了关键节点：最近有重要产检或检查要做", Hint: "临近关键检查", Tag: "PRENATAL_KEY_NODE"},
	{ID: "prenatal_bag", Label: "开始慌了：越靠近生产，待产包和证件越理不清", Hint: "待产准备混乱", Tag: "PRENATAL_BAG_CONFUSED"},
}

var prenatalSupportOptions = []futureLetterOptionDef{
	{ID: "prenatal_companion", Label: "我需要有人替我记路线、排队、盯流程，别让我一个人跑", Hint: "现场陪同", Tag: "PRENATAL_COMPANION_SUPPORT"},
	{ID: "prenatal_prepare", Label: "我更需要有人把证件、物品和注意事项先整理好", Hint: "资料和物品整理", Tag: "PRENATAL_PREP_SUPPORT"},
}

var safetySeatProgressOptions = []futureLetterOptionDef{
	{ID: "seat_not_bought", Label: "还没买定：参数太多，越看越没有把握", Hint: "选购阶段", Tag: "SAFETY_SEAT_NOT_BOUGHT"},
	{ID: "seat_installed", Label: "已经买了，但总担心安装角度和固定方式不对", Hint: "安装阶段", Tag: "SAFETY_SEAT_INSTALLING"},
	{ID: "seat_short_trip", Label: "短途偶尔会松懈，想着抱一会儿也许没事", Hint: "执行松动", Tag: "SAFETY_SEAT_ROUTINE_LOOSE"},
}

var safetySeatSupportOptions = []futureLetterOptionDef{
	{ID: "seat_research", Label: "我需要有人把车型、接口和孩子体型对应的参数研究明白", Hint: "选购研究", Tag: "SAFETY_SEAT_RESEARCH_SUPPORT"},
	{ID: "seat_install", Label: "我需要有人亲自装好，并把每次上车前检查做成固定动作", Hint: "安装和执行", Tag: "SAFETY_SEAT_INSTALL_SUPPORT"},
}

var infantAgeStages = map[string]bool{
	"0-3m":  true,
	"3-6m":  true,
	"6-12m": true,
}

var toddlerAgeStages = map[string]bool{
	"1-2y": true,
	"2-3y": true,
	"3-4y": true,
	"4-5y": true,
}

func (s *WhisperService) GetFutureLetterView(callerID string) (*dto.FutureLetterView, error) {
	user, err := s.userRepo.FindByID(callerID)
	if err != nil {
		return nil, errors.New(errWhisperUserNotFound)
	}

	var items []model.FutureLetterResponse
	if s.futureLetterRepo != nil {
		if user.Role == model.RoleDad {
			items, err = s.futureLetterRepo.FindRecentByRecipientID(user.ID, 30)
		} else {
			items, err = s.futureLetterRepo.FindRecentByAuthorID(user.ID, 30)
		}
		if err != nil {
			items = nil
		}
	}

	currentAgeStage := resolvedFutureLetterAgeStage(user, s.userRepo, s.chatRepo)

	// Use AI to generate template dynamically (skip for Dad, cache for Mom)
	var template futureLetterTemplate
	if user.Role == model.RoleMom && s.aiClient.Enabled() && user.PartnerID != nil && s.taskRepo != nil {
		ck := coupleKey(user.ID, *user.PartnerID)
		date := time.Now().Format("2006-01-02")
		if cache, err := s.taskRepo.FindAICache(ck, date, "future-letter-template"); err == nil && cache != nil && cache.AgeStage == currentAgeStage {
			_ = json.Unmarshal([]byte(cache.Content), &template)
		}

		if template.Code == "" {
			prompt := fmt.Sprintf(`[MomShell 垂直场景“时空关怀”脚本引擎]
Role:
你是一位走过育儿长路、温柔睿智的“未来的自己”。你正在给过去那个正处于育儿难关中的自己写信.

Objective:
通过一封带有感官温度的信件，引导用户完成一次特定场景下的需求调研。我们需要通过互动，识别用户在该任务中的当前进度和核心卡点（瓶颈）。

Core Logic (核心逻辑):
1. 场景化切入：抛弃大而全的问候，直接锁定一个具体的“育儿关卡”。
2. 双维度数据采集：
   - 维度一（进度）：用户目前在这个任务里走到哪一步了？（刚开始/进行中/迷茫中）
   - 维度二（痛点）：用户最怕的是什么？是体力上的累，还是认知上的盲点？
3. 未来的视角：用“过来人”的淡定去稀释用户当下的焦虑，让调研看起来像是一份关怀清单。

Task:
请根据以上逻辑，针对妈妈当前所处的阶段或育儿场景：“%s”，生成一段具体的互动脚本。

Structure (必须遵循的模板结构) 并严格输出为以下JSON格式：
{
  "title": "信件标题",
  "intro": "【场景共情】：描述一个该场景下的典型细节（如：疫苗本的厚度、拍嗝的方法等），引出核心话题。温暖贴心。",
  "outro": "【治愈收尾】：给自己一个肯定的心理暗示。【署名】：统一为“——未来的自己”。",
  "scene_hint": "场景提示，比如：海风吹过信角...",
  "wish_prompt": "引导她写下一句心愿的话，如：补一句最想被接住的心愿。",
  "questions": [
    {
      "id": "stage",
      "prompt": "【进度盘点】：针对该话题询问当前状态/进度",
      "options": [
        {"id": "stage_1", "label": "选项1（感性描述+具体状态）", "hint": "简短提示"},
        {"id": "stage_2", "label": "选项2（感性描述+具体状态）", "hint": "简短提示"},
        {"id": "stage_3", "label": "选项3（感性描述+具体状态）", "hint": "简短提示"}
      ]
    },
    {
      "id": "state",
      "prompt": "【深度摸排】：询问在这个环节中“最头疼/最担心”的是什么",
      "options": [
        {"id": "state_1", "label": "选项1（物理操作难/信息获取难/情绪压力大等维度）", "hint": "简短提示"},
        {"id": "state_2", "label": "选项2（其他痛点维度）", "hint": "简短提示"},
        {"id": "state_3", "label": "选项3（其他痛点维度）", "hint": "简短提示"}
      ]
    }
  ]
}`, currentAgeStage)

			respStr, err := s.aiClient.Chat(context.Background(), []openai.Message{
				{Role: "user", Content: prompt},
			})
			if err == nil {
				respStr = llmvalidate.Sanitize(respStr)

				// Cross-validate: structure + content safety review
				ltVR, cvErr := llmvalidate.CrossValidate(context.Background(), s.aiClient, llmvalidate.TypeLetterTemplate, respStr, "")
				if cvErr != nil {
					log.Printf("[WhisperLetter] template cross-validate error: %v", cvErr)
				}
				if ltVR != nil && !ltVR.Valid {
					log.Printf("[WhisperLetter] template structure invalid: %v", ltVR.StructErrors)
				}

				var aiTemplate struct {
					Title      string                     `json:"title"`
					Intro      string                     `json:"intro"`
					Outro      string                     `json:"outro"`
					SceneHint  string                     `json:"scene_hint"`
					WishPrompt string                     `json:"wish_prompt"`
					Questions  []dto.FutureLetterQuestion `json:"questions"`
				}
				if err := json.Unmarshal([]byte(llmvalidate.StripCodeFence(respStr)), &aiTemplate); err == nil {
					template = futureLetterTemplate{
						Code:       fmt.Sprintf("ai-dynamic-%d", time.Now().Unix()),
						Title:      aiTemplate.Title,
						Intro:      aiTemplate.Intro,
						Outro:      aiTemplate.Outro,
						Signature:  "——未来的自己",
						WishPrompt: aiTemplate.WishPrompt,
						SceneHint:  aiTemplate.SceneHint,
						Questions:  aiTemplate.Questions,
					}
					// Cache it
					templateJSON, _ := json.Marshal(template)
					_ = s.taskRepo.SaveAICache(&model.AIGeneratedTask{
						CoupleKey: ck,
						Date:      date,
						Type:      "future-letter-template",
						AgeStage:  currentAgeStage,
						Content:   string(templateJSON),
					})
				}
			}
		}
	}

	if template.Code == "" {
		template = buildFutureLetterTemplate(selectFutureLetterCode(items, currentAgeStage, time.Now()))
	}

	view := &dto.FutureLetterView{
		LetterCode:      template.Code,
		Title:           template.Title,
		Intro:           template.Intro,
		Outro:           template.Outro,
		Signature:       template.Signature,
		WishPrompt:      template.WishPrompt,
		PaperTheme:      "paper",
		SceneHint:       template.SceneHint,
		Questions:       template.Questions,
		RecentResponses: []dto.FutureLetterResponseItem{},
	}

	if len(items) > 0 {
		view.RecentResponses = toFutureLetterResponseItems(items)
		view.LatestResponse = &view.RecentResponses[0]
	}

	return view, nil
}

func (s *WhisperService) RespondFutureLetter(authorID string, req dto.FutureLetterRespondRequest) (*dto.FutureLetterResponseItem, error) {
	author, err := s.userRepo.FindByID(authorID)
	if err != nil {
		return nil, errors.New(errWhisperUserNotFound)
	}
	if author.Role != model.RoleMom {
		return nil, fmt.Errorf("只有妈妈角色可以发送心语情报")
	}
	if author.PartnerID == nil {
		return nil, errors.New(errWhisperPartnerRequired)
	}
	if s.futureLetterRepo == nil || s.taskRepo == nil {
		return nil, fmt.Errorf("心语情报系统暂未就绪，请稍后再试")
	}

	wish := strings.TrimSpace(req.WishContent)
	mission, err := s.buildDadMission(
		author,
		req.LetterCode,
		req.StageOptionID,
		req.StageOptionLabel,
		req.StateOptionID,
		req.StateOptionLabel,
		wish,
	)
	if err != nil {
		if errors.Is(err, openai.ErrNotConfigured) || errors.Is(err, openai.ErrAuthentication) {
			return nil, err
		}
		return nil, fmt.Errorf("心语情报生成失败，请稍后再试")
	}
	advice := s.buildDadAdvice(
		author,
		req.LetterCode,
		req.StageOptionID,
		req.StageOptionLabel,
		req.StateOptionID,
		req.StateOptionLabel,
		wish,
		mission,
	)

	record := buildFutureLetterResponseRecord(
		author,
		req.LetterCode,
		req.StageOptionID,
		req.StageOptionLabel,
		req.StateOptionID,
		req.StateOptionLabel,
		advice,
		buildFutureLetterImagePrompt(req.LetterCode, req.StageOptionLabel, req.StateOptionLabel, mission.ImageTone, wish),
		wish,
	)

	if err := s.saveFutureLetterMission(record, mission); err != nil {
		log.Printf("[WhisperService] save future letter: %v", err)
		return nil, fmt.Errorf("保存回信失败，请稍后再试")
	}
	item := toFutureLetterResponseItem(*record)
	return &item, nil
}

// Removing obsolete switch-case respond methods:

func buildFutureLetterResponseRecord(
	author *model.User,
	letterCode string,
	primaryTag string,
	primaryLabel string,
	secondaryTag string,
	secondaryLabel string,
	advice dadAdviceTemplate,
	imagePrompt string,
	wish string,
) *model.FutureLetterResponse {
	itemsJSON, _ := json.Marshal(advice.Items)
	sourcesJSON, _ := json.Marshal(advice.Sources)
	record := &model.FutureLetterResponse{
		AuthorID:             author.ID,
		RecipientID:          *author.PartnerID,
		LetterCode:           letterCode,
		StageTag:             primaryTag,
		StageLabel:           primaryLabel,
		StateTag:             secondaryTag,
		StateLabel:           secondaryLabel,
		DadPlanCode:          advice.Code,
		DadPlanTitle:         advice.Title,
		DadHeadline:          advice.Headline,
		DadSummary:           advice.Summary,
		DadTasksJSON:         string(itemsJSON),
		DadAdviceSourcesJSON: string(sourcesJSON),
		ImagePrompt:          imagePrompt,
	}
	if wish != "" {
		record.WishContent = &wish
	}
	return record
}

func (s *WhisperService) buildDadMission(
	author *model.User,
	letterCode string,
	primaryTag string,
	primaryLabel string,
	secondaryTag string,
	secondaryLabel string,
	wish string,
) (dadMissionTemplate, error) {
	ageStage := resolvedFutureLetterAgeStage(author, s.userRepo, s.chatRepo)
	mission, err := s.translateDadMissionWithAI(
		author,
		letterCode,
		ageStage,
		primaryTag,
		primaryLabel,
		secondaryTag,
		secondaryLabel,
		wish,
	)
	if err != nil {
		log.Printf("[WhisperService] AI dad mission translation failed: %v", err)
		return dadMissionTemplate{}, err
	}
	return mission, nil
}

func (s *WhisperService) buildDadAdvice(
	author *model.User,
	letterCode string,
	primaryTag string,
	primaryLabel string,
	secondaryTag string,
	secondaryLabel string,
	wish string,
	mission dadMissionTemplate,
) dadAdviceTemplate {
	ctx := s.collectFutureLetterAdviceContext(
		author,
		letterCode,
		primaryTag,
		primaryLabel,
		secondaryTag,
		secondaryLabel,
		wish,
	)

	advice, err := s.translateDadAdviceWithAI(ctx, mission)
	if err != nil {
		log.Printf("[WhisperService] AI dad advice generation failed: %v", err)
		return buildFallbackDadAdvice(ctx)
	}

	return advice
}

func (s *WhisperService) collectFutureLetterAdviceContext(
	author *model.User,
	letterCode string,
	primaryTag string,
	primaryLabel string,
	secondaryTag string,
	secondaryLabel string,
	wish string,
) futureLetterAdviceContext {
	ageStage := resolvedFutureLetterAgeStage(author, s.userRepo, s.chatRepo)
	ageLabel := ageStageLabels[ageStage]
	if ageLabel == "" {
		ageLabel = ageStage
	}

	ctx := futureLetterAdviceContext{
		Template:       buildFutureLetterTemplate(letterCode),
		AgeStage:       ageStage,
		AgeLabel:       ageLabel,
		PrimaryTag:     primaryTag,
		PrimaryLabel:   primaryLabel,
		SecondaryTag:   secondaryTag,
		SecondaryLabel: secondaryLabel,
		Wish:           wish,
	}

	if s.chatRepo != nil {
		if memory, err := s.chatRepo.FindByUserID(author.ID); err == nil {
			ctx.MemorySummary = strings.TrimSpace(memory.ConversationSummary)
			turns := memory.GetTurns()
			start := 0
			if len(turns) > 6 {
				start = len(turns) - 6
			}
			for _, turn := range turns[start:] {
				role, _ := turn["role"].(string)
				content, _ := turn["content"].(string)
				if strings.TrimSpace(content) == "" {
					continue
				}
				ctx.TurnLines = append(ctx.TurnLines, fmt.Sprintf("- [%s] %s", role, strings.TrimSpace(content)))
			}
		}

		if facts, err := s.chatRepo.FindFactsByUserID(author.ID); err == nil {
			limit := min(len(facts), 12)
			for _, fact := range facts[:limit] {
				ctx.FactLines = append(ctx.FactLines, fmt.Sprintf("- [%s] %s", fact.Category, fact.Content))
			}
		}
	}

	if s.whisperRepo != nil {
		if whispers, err := s.whisperRepo.FindByAuthorID(author.ID, 8); err == nil {
			for _, whisper := range whispers {
				ctx.WhisperLines = append(ctx.WhisperLines, fmt.Sprintf("- %s（%s）", whisper.Content, whisper.CreatedAt.Format("01-02 15:04")))
			}
		}
	}

	if s.futureLetterRepo != nil {
		if responses, err := s.futureLetterRepo.FindRecentByAuthorID(author.ID, 4); err == nil {
			for _, item := range responses {
				ctx.ResponseLines = append(ctx.ResponseLines, fmt.Sprintf("- [%s] %s / %s -> %s", item.LetterCode, item.StageLabel, item.StateLabel, item.DadPlanTitle))
			}
		}
	}

	return ctx
}

func (s *WhisperService) translateDadAdviceWithAI(
	ctx futureLetterAdviceContext,
	mission dadMissionTemplate,
) (dadAdviceTemplate, error) {
	if !s.aiClient.Enabled() {
		return dadAdviceTemplate{}, fmt.Errorf("ai unavailable")
	}

	systemPrompt := `你是 MomShell 的“Dad 端心语情报顾问”。
你的任务是读取妈妈本次问卷、她补充的心愿、她与 AI 的聊天记录和记忆，生成一份给爸爸看的陪伴建议卡。

严格要求：
- 语气必须温和、细腻、富有同理心。仿佛一个懂她的老友在提醒你。
- 禁止使用“工单、工单号、指令、任务、同步、执行、系统、T0/T1/T2、优先级、反馈”等词汇。
- 必须紧密结合妈妈在问卷中回答的【具体问题】和【答案】来进行解读，切勿只关注她补充的心愿。
- 确保每次生成的建议具有独特性，避免同质化，要结合当次问卷的具体场景。
- 输出 4 条建议，必须且只能包含 1 条 decode、1 条 opening、1 条 observe、1 条 avoid。
- decode：帮爸爸解读她行为背后的情绪，如“她其实是想说...”。
- opening：给出一句温暖、不生硬的开场白。
- observe：提醒爸爸观察她的一个小动作或表情变化。
- avoid：提醒爸爸今天避开哪个雷区。
- title 必须短，4 到 8 个中文字符，不要有系统标签感。
- summary 是一句总结性的叮嘱，温暖有力，不要有指令感。
- headline 是一句温和的现状判断。
- sources 对应你看到的真实依据。

只返回 JSON 对象，不要附加解释，不要使用 Markdown 代码块。
JSON 格式：
{
  "title": "情报主题",
  "headline": "温和的现状判断",
  "summary": "一句温暖的总结性叮嘱",
  "sources": [{"label": "依据来源", "detail": "具体内容"}],
  "items": [
    {
      "title": "解析/开口/观察/避坑",
      "description": "具体的温暖建议",
      "kind": "decode/opening/observe/avoid"
    }
  ]
}`

	var sb strings.Builder
	fmt.Fprintf(&sb, "今天投放给妈妈的情报主题：%s\n", ctx.Template.Title)
	fmt.Fprintf(&sb, "情报开场：%s\n", ctx.Template.Intro)
	if ctx.AgeLabel != "" {
		fmt.Fprintf(&sb, "宝宝当前年龄阶段：%s\n", ctx.AgeLabel)
	}
	if len(ctx.Template.Questions) > 0 {
		fmt.Fprintf(&sb, "问卷问题一：%s\n", ctx.Template.Questions[0].Prompt)
	}
	fmt.Fprintf(&sb, "妈妈的回答（第一层选择）：%s（%s）\n", ctx.PrimaryLabel, ctx.PrimaryTag)
	if len(ctx.Template.Questions) > 1 {
		fmt.Fprintf(&sb, "问卷问题二：%s\n", ctx.Template.Questions[1].Prompt)
	}
	fmt.Fprintf(&sb, "妈妈的回答（第二层选择）：%s（%s）\n", ctx.SecondaryLabel, ctx.SecondaryTag)
	if ctx.Wish != "" {
		fmt.Fprintf(&sb, "妈妈额外写下的心愿：%s\n", ctx.Wish)
	}
	if ctx.MemorySummary != "" {
		fmt.Fprintf(&sb, "\n妈妈与 AI 的聊天摘要：\n%s\n", ctx.MemorySummary)
	}
	if len(ctx.TurnLines) > 0 {
		sb.WriteString("\n妈妈与 AI 最近几轮聊天记录：\n")
		sb.WriteString(strings.Join(ctx.TurnLines, "\n"))
		sb.WriteString("\n")
	}
	if len(ctx.FactLines) > 0 {
		sb.WriteString("\n妈妈与 AI 聊天沉淀下来的记忆事实：\n")
		sb.WriteString(strings.Join(ctx.FactLines, "\n"))
		sb.WriteString("\n")
	}
	if len(ctx.WhisperLines) > 0 {
		sb.WriteString("\n妈妈最近留下的心愿/心声记录：\n")
		sb.WriteString(strings.Join(ctx.WhisperLines, "\n"))
		sb.WriteString("\n")
	}
	if len(ctx.ResponseLines) > 0 {
		sb.WriteString("\n历史情报记录：\n")
		sb.WriteString(strings.Join(ctx.ResponseLines, "\n"))
		sb.WriteString("\n")
	}
	if len(mission.Tasks) > 0 {
		sb.WriteString("\n已经同步到任务系统的行动计划（请不要重复这些动作）：\n")
		for _, task := range mission.Tasks {
			fmt.Fprintf(&sb, "- %s: %s\n", task.Title, task.Description)
		}

		sb.WriteString("\n请基于这些信息，为 Dad 端生成一份非指令化的陪伴建议卡。")
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := s.aiClient.Chat(reqCtx, []openai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: sb.String()},
	})
	if err != nil {
		return dadAdviceTemplate{}, err
	}

	cleaned := llmvalidate.StripCodeFence(llmvalidate.Sanitize(resp))

	// Cross-validate: structure + content safety review
	avr, cvErr := llmvalidate.CrossValidate(reqCtx, s.aiClient, llmvalidate.TypeAdviceResponse, cleaned, "")
	if cvErr != nil {
		log.Printf("[WhisperLetter] advice cross-validate error: %v", cvErr)
	}
	if avr != nil && !avr.Valid {
		log.Printf("[WhisperLetter] advice structure invalid: %v", avr.StructErrors)
	}

	var payload futureLetterAIAdvice
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
		return dadAdviceTemplate{}, fmt.Errorf("failed to parse ai advice: %w", err)
	}
	if strings.TrimSpace(payload.Title) == "" || strings.TrimSpace(payload.Headline) == "" || len(payload.Items) == 0 {
		return dadAdviceTemplate{}, fmt.Errorf("ai advice payload incomplete")
	}

	items := normalizeDadAdviceItems(payload.Items)
	sources := normalizeDadAdviceSources(payload.Sources)
	if len(items) == 0 {
		return dadAdviceTemplate{}, fmt.Errorf("ai advice items empty")
	}
	if len(sources) == 0 {
		sources = buildAdviceSourceNotes(ctx)
	}

	return dadAdviceTemplate{
		Code:     "advice-" + ctx.Template.Code,
		Title:    strings.TrimSpace(payload.Title),
		Headline: strings.TrimSpace(payload.Headline),
		Summary:  strings.TrimSpace(payload.Summary),
		Sources:  sources,
		Items:    items,
	}, nil
}

func (s *WhisperService) translateDadMissionWithAI(author *model.User, letterCode string, ageStage string, primaryTag string, primaryLabel string, secondaryTag string, secondaryLabel string, wish string) (dadMissionTemplate, error) {
	if !s.aiClient.Enabled() {
		return dadMissionTemplate{}, openai.ErrNotConfigured
	}

	template := buildFutureLetterTemplate(letterCode)
	ageLabel := ageStageLabels[ageStage]
	if ageLabel == "" {
		ageLabel = ageStage
	}

	var memorySummary string
	var turnLines []string
	if s.chatRepo != nil {
		if memory, err := s.chatRepo.FindByUserID(author.ID); err == nil {
			memorySummary = strings.TrimSpace(memory.ConversationSummary)
			turns := memory.GetTurns()
			start := 0
			if len(turns) > 6 {
				start = len(turns) - 6
			}
			for _, turn := range turns[start:] {
				role, _ := turn["role"].(string)
				content, _ := turn["content"].(string)
				if strings.TrimSpace(content) == "" {
					continue
				}
				turnLines = append(turnLines, fmt.Sprintf("- [%s] %s", role, strings.TrimSpace(content)))
			}
		}
	}

	var factLines []string
	if s.chatRepo != nil {
		if facts, err := s.chatRepo.FindFactsByUserID(author.ID); err == nil {
			limit := min(len(facts), 12)
			for _, fact := range facts[:limit] {
				factLines = append(factLines, fmt.Sprintf("- [%s] %s", fact.Category, fact.Content))
			}
		}
	}

	var whisperLines []string
	if s.whisperRepo != nil {
		if whispers, err := s.whisperRepo.FindByAuthorID(author.ID, 8); err == nil {
			for _, whisper := range whispers {
				whisperLines = append(whisperLines, fmt.Sprintf("- %s（%s）", whisper.Content, whisper.CreatedAt.Format("01-02 15:04")))
			}
		}
	}

	var responseLines []string
	if s.futureLetterRepo != nil {
		if responses, err := s.futureLetterRepo.FindRecentByAuthorID(author.ID, 4); err == nil {
			for _, item := range responses {
				responseLines = append(responseLines, fmt.Sprintf("- [%s] %s / %s -> %s", item.LetterCode, item.StageLabel, item.StateLabel, item.DadPlanTitle))
			}
		}
	}

	systemPrompt := `你是 MomShell 的“心语情报编译器”。
你的任务是：读取妈妈今天在心语情报问卷中的选择、她补充的心愿、她与 AI 的聊天记忆、过往心愿记录和历史情报，把这些信息转译成给爸爸的精准行动指令。

我们需要你生成极具落地感的行动，直接告诉爸爸他今晚该做什么来支持老婆。

严格要求：
- 指令要非常具体、有画面感、有步骤。
- 输出 1 到 3 个任务。
- 必须基于妈妈在问卷中回答的【具体问题】和【答案】进行生成，避免每次生成的任务都同质化。每次生成的任务都需要有针对性和新鲜感，切勿只关注她补充的心愿。
- 禁止使用“工单、执行、T0/T1/T2、优先级、反馈、汇报”等机械化的系统词汇。
- title 要短（4到10个字），有行动感。
- description 指导性强，不要说废话，不要讲大道理。
- category 只能是 housework、parenting、health、emotional。
- priority 依然返回 T0、T1、T2 格式（供后端使用），但在展示字段中不要体现这个词。
- headline 是一句掷地有声的行动口号，不要以“[核心指令]”开头，可以尝试用“[今日侧重]”或直接描述目标。
- summary 是一句解释为什么要这么做的理由，温暖且有说服力。

只返回 JSON 对象，不要附加解释，不要使用 Markdown 代码块。
JSON 格式：
{
  "title": "行动主题",
  "headline": "[今日侧重] ...",
  "summary": "...",
  "tasks": [
    {
      "title": "具体任务名",
      "description": "怎么做...",
      "category": "housework",
      "priority": "T0",
      "difficulty": 1
    }
  ]
}`

	var sb strings.Builder
	fmt.Fprintf(&sb, "今天投放给妈妈的情报主题：%s\n", template.Title)
	fmt.Fprintf(&sb, "情报开场：%s\n", template.Intro)
	if ageLabel != "" {
		fmt.Fprintf(&sb, "宝宝当前年龄阶段：%s\n", ageLabel)
	}
	if len(template.Questions) > 0 {
		fmt.Fprintf(&sb, "问卷问题一：%s\n", template.Questions[0].Prompt)
	}
	fmt.Fprintf(&sb, "妈妈的回答（第一层选择）：%s（%s）\n", primaryLabel, primaryTag)
	if len(template.Questions) > 1 {
		fmt.Fprintf(&sb, "问卷问题二：%s\n", template.Questions[1].Prompt)
	}
	fmt.Fprintf(&sb, "妈妈的回答（第二层选择）：%s（%s）\n", secondaryLabel, secondaryTag)
	if wish != "" {
		fmt.Fprintf(&sb, "妈妈额外写下的心愿：%s\n", wish)
	}
	if memorySummary != "" {
		fmt.Fprintf(&sb, "\n妈妈与 AI 的聊天摘要：\n%s\n", memorySummary)
	}
	if len(turnLines) > 0 {
		sb.WriteString("\n妈妈与 AI 最近几轮聊天记录：\n")
		sb.WriteString(strings.Join(turnLines, "\n"))
		sb.WriteString("\n")
	}
	if len(factLines) > 0 {
		sb.WriteString("\n妈妈与 AI 聊天沉淀下来的记忆事实：\n")
		sb.WriteString(strings.Join(factLines, "\n"))
		sb.WriteString("\n")
	}
	if len(whisperLines) > 0 {
		sb.WriteString("\n妈妈最近留下的心愿/心声记录：\n")
		sb.WriteString(strings.Join(whisperLines, "\n"))
		sb.WriteString("\n")
	}
	if len(responseLines) > 0 {
		sb.WriteString("\n历史情报与已生成行动建议记录：\n")
		sb.WriteString(strings.Join(responseLines, "\n"))
		sb.WriteString("\n")
	}

	sb.WriteString("\n请把这些信息编译成爸爸今天的具体行动建议。")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := s.aiClient.Chat(ctx, []openai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: sb.String()},
	})
	if err != nil {
		return dadMissionTemplate{}, err
	}

	cleaned := llmvalidate.StripCodeFence(llmvalidate.Sanitize(resp))

	// Cross-validate: structure + content safety review
	mvr, cvErr := llmvalidate.CrossValidate(ctx, s.aiClient, llmvalidate.TypeMissionResponse, cleaned, "")
	if cvErr != nil {
		log.Printf("[WhisperLetter] mission cross-validate error: %v", cvErr)
	}
	if mvr != nil && !mvr.Valid {
		log.Printf("[WhisperLetter] mission structure invalid: %v", mvr.StructErrors)
	}

	var payload futureLetterAIMission
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
		return dadMissionTemplate{}, fmt.Errorf("failed to parse ai mission: %w", err)
	}
	if strings.TrimSpace(payload.Title) == "" || strings.TrimSpace(payload.Headline) == "" || len(payload.Tasks) == 0 {
		return dadMissionTemplate{}, fmt.Errorf("ai mission payload incomplete")
	}

	tasks := normalizeAITasks(payload.Tasks)
	if len(tasks) == 0 {
		return dadMissionTemplate{}, fmt.Errorf("ai mission returned no usable tasks")
	}
	missionTasks := make([]dadMissionTaskDef, 0, len(tasks))
	for _, task := range tasks {
		missionTasks = append(missionTasks, dadMissionTaskDef{
			Title:       strings.TrimSpace(task.Title),
			Description: strings.TrimSpace(task.Description),
			Category:    task.Category,
			Priority:    task.Priority,
			Difficulty:  task.Difficulty,
		})
	}

	return dadMissionTemplate{
		Code:      "ai-" + letterCode,
		Title:     strings.TrimSpace(payload.Title),
		Headline:  strings.TrimSpace(payload.Headline),
		Summary:   strings.TrimSpace(payload.Summary),
		Tasks:     missionTasks,
		ImageTone: "",
	}, nil
}

func normalizeDadAdviceItems(items []dadAdviceItemDef) []dadAdviceItemDef {
	validKinds := map[string]bool{
		"decode":  true,
		"opening": true,
		"observe": true,
		"avoid":   true,
	}

	normalized := make([]dadAdviceItemDef, 0, len(items))
	for _, item := range items {
		title := strings.TrimSpace(item.Title)
		description := strings.TrimSpace(item.Description)
		kind := strings.TrimSpace(strings.ToLower(item.Kind))
		if title == "" || description == "" {
			continue
		}
		if !validKinds[kind] {
			kind = "decode"
		}
		normalized = append(normalized, dadAdviceItemDef{
			Title:       truncateRunes(title, 12),
			Description: description,
			Kind:        kind,
		})
	}
	return reorderDadAdviceItems(normalized)
}

func normalizeDadAdviceSources(sources []dadAdviceSourceDef) []dadAdviceSourceDef {
	normalized := make([]dadAdviceSourceDef, 0, len(sources))
	for _, source := range sources {
		label := strings.TrimSpace(source.Label)
		detail := strings.TrimSpace(source.Detail)
		if label == "" || detail == "" {
			continue
		}
		normalized = append(normalized, dadAdviceSourceDef{
			Label:  truncateRunes(label, 12),
			Detail: truncateRunes(detail, 72),
		})
		if len(normalized) == 4 {
			break
		}
	}
	return normalized
}

func buildFallbackDadAdvice(ctx futureLetterAdviceContext) dadAdviceTemplate {
	return dadAdviceTemplate{
		Code:     "fallback-" + ctx.Template.Code,
		Title:    "小石光建议卡",
		Headline: fallbackAdviceHeadline(ctx.SecondaryTag),
		Summary:  fallbackAdviceSummary(ctx),
		Sources:  buildAdviceSourceNotes(ctx),
		Items:    buildFallbackAdviceItems(ctx),
	}
}

func (s *WhisperService) RegenerateFutureLetterForDad(callerID string) (*dto.FutureLetterResponseItem, error) {
	user, err := s.userRepo.FindByID(callerID)
	if err != nil {
		return nil, errors.New(errWhisperUserNotFound)
	}
	if user.Role != model.RoleDad {
		return nil, fmt.Errorf("只有爸爸角色可以重新生成情报")
	}
	if user.PartnerID == nil {
		return nil, errors.New(errWhisperPartnerRequired)
	}
	if s.futureLetterRepo == nil || s.taskRepo == nil {
		return nil, fmt.Errorf("心语情报系统暂未就绪，请稍后再试")
	}

	items, err := s.futureLetterRepo.FindRecentByRecipientID(user.ID, 1)
	if err != nil {
		return nil, fmt.Errorf("读取历史情报失败，请稍后再试")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("还没有可重新生成的情报")
	}

	record := items[0]
	author, err := s.userRepo.FindByID(record.AuthorID)
	if err != nil {
		return nil, errors.New(errWhisperUserNotFound)
	}

	wish := ""
	if record.WishContent != nil {
		wish = strings.TrimSpace(*record.WishContent)
	}

	mission, err := s.buildDadMission(
		author,
		record.LetterCode,
		record.StageTag,
		record.StageLabel,
		record.StateTag,
		record.StateLabel,
		wish,
	)
	if err != nil {
		if errors.Is(err, openai.ErrNotConfigured) || errors.Is(err, openai.ErrAuthentication) {
			return nil, err
		}
		return nil, fmt.Errorf("重新生成任务失败，请稍后再试")
	}

	advice := s.buildDadAdvice(
		author,
		record.LetterCode,
		record.StageTag,
		record.StageLabel,
		record.StateTag,
		record.StateLabel,
		wish,
		mission,
	)

	itemsJSON, _ := json.Marshal(advice.Items)
	sourcesJSON, _ := json.Marshal(advice.Sources)
	record.DadPlanCode = advice.Code
	record.DadPlanTitle = advice.Title
	record.DadHeadline = advice.Headline
	record.DadSummary = advice.Summary
	record.DadTasksJSON = string(itemsJSON)
	record.DadAdviceSourcesJSON = string(sourcesJSON)
	record.ImagePrompt = buildFutureLetterImagePrompt(
		record.LetterCode,
		record.StageLabel,
		record.StateLabel,
		mission.ImageTone,
		wish,
	)

	if err := s.saveFutureLetterMission(&record, mission); err != nil {
		if errors.Is(err, errFutureLetterChanged) {
			return nil, err
		}
		log.Printf("[WhisperService] save regenerated letter: %v", err)
		return nil, fmt.Errorf("保存重生成情报失败，请稍后再试")
	}

	item := toFutureLetterResponseItem(record)
	return &item, nil
}

func buildAdviceSourceNotes(ctx futureLetterAdviceContext) []dadAdviceSourceDef {
	sources := []dadAdviceSourceDef{
		{
			Label:  "问卷信号",
			Detail: truncateRunes(fmt.Sprintf("%s / %s", ctx.PrimaryLabel, ctx.SecondaryLabel), 72),
		},
	}

	if ctx.Wish != "" {
		sources = append(sources, dadAdviceSourceDef{
			Label:  "她的心愿",
			Detail: truncateRunes(ctx.Wish, 72),
		})
	}
	if ctx.MemorySummary != "" {
		sources = append(sources, dadAdviceSourceDef{
			Label:  "聊天摘要",
			Detail: truncateRunes(firstNonEmptyLine(ctx.MemorySummary), 72),
		})
	} else if len(ctx.TurnLines) > 0 {
		sources = append(sources, dadAdviceSourceDef{
			Label:  "最近对话",
			Detail: truncateRunes(trimContextBullet(ctx.TurnLines[len(ctx.TurnLines)-1]), 72),
		})
	}
	if len(ctx.FactLines) > 0 {
		sources = append(sources, dadAdviceSourceDef{
			Label:  "AI 记忆",
			Detail: truncateRunes(trimContextBullet(ctx.FactLines[0]), 72),
		})
	}
	if len(ctx.WhisperLines) > 0 && ctx.Wish == "" {
		sources = append(sources, dadAdviceSourceDef{
			Label:  "最近心语",
			Detail: truncateRunes(trimContextBullet(ctx.WhisperLines[0]), 72),
		})
	}

	return normalizeDadAdviceSources(sources)
}

func buildFallbackAdviceItems(ctx futureLetterAdviceContext) []dadAdviceItemDef {
	items := []dadAdviceItemDef{
		{
			Title:       "她在表达什么",
			Description: fallbackDecodeAdvice(ctx),
			Kind:        "decode",
		},
		{
			Title:       "先这样开口",
			Description: fallbackOpeningAdvice(ctx),
			Kind:        "opening",
		},
		{
			Title:       "留意这个信号",
			Description: fallbackObserveAdvice(ctx),
			Kind:        "observe",
		},
		{
			Title:       "今天先别这样",
			Description: fallbackAvoidAdvice(ctx),
			Kind:        "avoid",
		},
	}

	return normalizeDadAdviceItems(items)
}

func reorderDadAdviceItems(items []dadAdviceItemDef) []dadAdviceItemDef {
	order := []string{"decode", "opening", "observe", "avoid"}
	byKind := make(map[string]dadAdviceItemDef, len(items))
	for _, item := range items {
		if _, exists := byKind[item.Kind]; !exists {
			byKind[item.Kind] = item
		}
	}

	reordered := make([]dadAdviceItemDef, 0, len(order))
	for _, kind := range order {
		if item, ok := byKind[kind]; ok {
			reordered = append(reordered, item)
		}
	}
	return reordered
}

func fallbackAdviceHeadline(stateTag string) string {
	switch stateTag {
	case "PHYSICAL_PAIN":
		return "她现在更需要被轻一点地对待，而不是再多解释一次自己的难受。"
	case "EXHAUSTED":
		return "她眼下最缺的不是安排，而是一段不用继续撑着运转的缓冲。"
	case "MENTAL_SPACE":
		return "她并不是想退开你，而是想先从持续被需要的状态里松一口气。"
	default:
		return "先别急着解决问题，先让她感觉到你真的看见了她今天的状态。"
	}
}

func fallbackAdviceSummary(ctx futureLetterAdviceContext) string {
	summary := "今天这份建议先围绕她刚在问卷里表达的状态来走。你要做的不是把事情做满，而是用更低负担的方式靠近她。"
	if ctx.Wish != "" {
		summary = "今天这份建议同时参考了她在问卷里的状态和她补充写下的心愿。先让她少解释一点、少承担一点，比急着给方案更有效。"
	}
	if ctx.MemorySummary != "" || len(ctx.FactLines) > 0 {
		summary = "今天这份建议除了参考问卷和心愿，也结合了她与 AI 的聊天摘要和记忆。重点不是重复说教，而是用她更能接住的方式靠近她。"
	}
	return summary
}

func fallbackDecodeAdvice(ctx futureLetterAdviceContext) string {
	switch ctx.SecondaryTag {
	case "PHYSICAL_PAIN":
		return "她今天更想被理解的，不是“事情太多”，而是身体已经在持续发出吃不消的信号。先把她的不舒服当成真实状态，而不是普通疲惫。"
	case "EXHAUSTED":
		return "她今天真正想被接住的，通常不是某一件具体小事，而是那种一直不能停下来的消耗感。她更需要的是有人替她关掉一点噪音，而不是继续追加决定。"
	case "MENTAL_SPACE":
		return "她今天表达的重点不是拒绝靠近，而是想暂时从“随时被需要”的位置退开一点。你越少追问她为什么，她越容易慢慢放松下来。"
	default:
		return "她今天更需要的不是被安排，而是先被读懂。先接住她眼前最重的感受，再谈你们接下来怎么配合。"
	}
}

func fallbackOpeningAdvice(ctx futureLetterAdviceContext) string {
	base := fmt.Sprintf("开口先复述她今天的状态，例如：“我看到你现在更像是在%s，我先不催你，你告诉我今天最想少做哪一件。”", ctx.SecondaryLabel)
	if ctx.Wish != "" {
		base = fmt.Sprintf("开口先接住她补充写下的那句心愿，例如：“你前面提到%s，这件事我记住了，今晚我们先把你最不想扛的那一段放下来。”", truncateRunes(ctx.Wish, 24))
	}
	return base
}

func fallbackObserveAdvice(ctx futureLetterAdviceContext) string {
	switch ctx.SecondaryTag {
	case "PHYSICAL_PAIN":
		return "留意她是不是连坐下、起身、弯腰这种小动作都明显变慢了。如果是，就别再把她的难受归到“普通累了”。"
	case "EXHAUSTED":
		return "留意她是不是已经开始用“都行”“随便”来结束对话。很多时候这不是没想法，而是已经没有多余力气继续做决定。"
	case "MENTAL_SPACE":
		return "留意她是不是刚安静一会儿就又被新的问题、安排或照看需求拉回去。如果她总是被打断，说明她还没真正离开高压位。"
	default:
		return "留意她今天最容易沉默的那一刻，那通常就是她最需要被接住的时候。"
	}
}

func fallbackAvoidAdvice(ctx futureLetterAdviceContext) string {
	switch ctx.SecondaryTag {
	case "PHYSICAL_PAIN":
		return "今天先别用“再坚持一下”“多休息就好了”这类话轻轻带过她的身体感受。先承认她真的不舒服，再决定你接下来做什么。"
	case "EXHAUSTED":
		return "今天先别把沟通变成流程复盘，比如追问她哪里没做好、为什么不早点说。她现在更需要被减负，而不是再负责说明。"
	case "MENTAL_SPACE":
		return "今天先别把她想安静一会儿理解成情绪疏远。给她一点不必解释的空间，反而更容易让她重新靠近。"
	default:
		return "今天先别急着给判断和大道理，先让她少解释一点，再谈你准备怎么帮。"
	}
}

func trimContextBullet(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "- ")
	return strings.TrimSpace(trimmed)
}

func firstNonEmptyLine(value string) string {
	for _, line := range strings.Split(value, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return strings.TrimSpace(value)
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "..."
}

func selectFutureLetterCode(items []model.FutureLetterResponse, ageStage string, now time.Time) string {
	if ageStage == "" || needsStageRecheck(items, now) {
		return futureLetterCodeOverview
	}

	switch {
	case ageStage == "pregnancy":
		return futureLetterCodePrenatal
	case infantAgeStages[ageStage]:
		return futureLetterCodeVaccine
	case toddlerAgeStages[ageStage]:
		return futureLetterCodeSafetySeat
	default:
		return futureLetterCodeOverview
	}
}

func resolvedFutureLetterAgeStage(
	user *model.User,
	userRepo *repository.UserRepo,
	chatRepo *repository.ChatRepo,
) string {
	if stage := deriveAgeStageFromBirthDate(user.BabyBirthDate); stage != "" {
		return stage
	}

	if user.PartnerID != nil {
		if partner, err := userRepo.FindByID(*user.PartnerID); err == nil {
			if stage := deriveAgeStageFromBirthDate(partner.BabyBirthDate); stage != "" {
				return stage
			}
		}
	}

	stage, _ := resolveAgeStage(user, userRepo, chatRepo)
	return stage
}

func needsStageRecheck(items []model.FutureLetterResponse, now time.Time) bool {
	var lastOverview *time.Time
	for _, item := range items {
		if item.LetterCode == futureLetterCodeOverview {
			ts := item.CreatedAt
			lastOverview = &ts
			break
		}
	}

	if lastOverview == nil {
		return true
	}

	return now.Sub(*lastOverview) >= 30*24*time.Hour
}

func deriveAgeStageFromBirthDate(birthDate *time.Time) string {
	if birthDate == nil {
		return ""
	}

	now := time.Now()
	if birthDate.After(now) {
		return ""
	}

	days := int(now.Sub(*birthDate).Hours() / 24)
	switch {
	case days < 90:
		return "0-3m"
	case days < 180:
		return "3-6m"
	case days < 365:
		return "6-12m"
	case days < 730:
		return "1-2y"
	case days < 1095:
		return "2-3y"
	case days < 1460:
		return "3-4y"
	default:
		return "4-5y"
	}
}

func buildFutureLetterTemplate(code string) futureLetterTemplate {
	switch code {
	case futureLetterCodePrenatal:
		return futureLetterTemplate{
			Code:       futureLetterCodePrenatal,
			Title:      "心语情报：产检和待产准备",
			Intro:      "产检和待产准备最磨人的，往往不是某一件大事，而是那些会在一天里反复打断你的细碎安排。",
			Outro:      "把最耗神的那部分提前说出来，才能让他知道今天该先接走什么。",
			Signature:  "小石光",
			WishPrompt: "如果你还惦记着某个待办，比如证件、路线、待产包，也可以顺手补一句。",
			SceneHint:  "海风吹过信角，把临近产检和待产前的碎事慢慢摊平",
			Questions: []dto.FutureLetterQuestion{
				buildFutureLetterQuestion("stage", "最近的产检或待产准备，走到哪一步了？", prenatalProgressOptions),
				buildFutureLetterQuestion("state", "眼下最想有人替你接走的是哪一部分？", prenatalSupportOptions),
			},
		}
	case futureLetterCodeVaccine:
		return futureLetterTemplate{
			Code:       futureLetterCodeVaccine,
			Title:      "心语情报：疫苗小关卡",
			Intro:      "接种这件事看起来只是跑一次流程，但真正累人的，是预约、排队、留观和事后记录这些碎得停不下来的环节。",
			Outro:      "把眼下最头疼的那部分交代清楚，他才知道该在哪一段真正顶上来。",
			Signature:  "小石光",
			WishPrompt: "如果你对接种日还有一句担心，也可以顺手写下来。",
			SceneHint:  "海风翻动信纸，像把那些接种日前的慌乱一页页理顺",
			Questions: []dto.FutureLetterQuestion{
				buildFutureLetterQuestion("stage", "想问问此刻的你，关于宝宝的疫苗情况：", vaccineProgressOptions),
				buildFutureLetterQuestion("state", "关于接种这件事，你对什么最头疼？", vaccineSupportOptions),
			},
		}
	case futureLetterCodeSafetySeat:
		return futureLetterTemplate{
			Code:       futureLetterCodeSafetySeat,
			Title:      "心语情报：安全座椅这件事",
			Intro:      "孩子越长越大，越容易让人对“就这一小段路”放松警惕。真正让人安心的，不是运气，而是一次次被认真执行的安全动作。",
			Outro:      "把你最担心的那一段说清楚，他才知道该把哪一步做扎实。",
			Signature:  "小石光",
			WishPrompt: "如果你对出门、上车或安装还有额外担心，也可以补一句。",
			SceneHint:  "海边风很轻，像提醒人把每次出发前的安全动作都做扎实",
			Questions: []dto.FutureLetterQuestion{
				buildFutureLetterQuestion("stage", "关于安全座椅，现在最真实的状态是？", safetySeatProgressOptions),
				buildFutureLetterQuestion("state", "你更希望他接手哪一类支持？", safetySeatSupportOptions),
			},
		}
	default:
		return futureLetterTemplate{
			Code:       futureLetterCodeOverview,
			Title:      "本轮心语情报",
			Intro:      "先告诉我你现在正走到哪一阶段、最想被接住的是什么，我会把这些整理成一份更容易被看懂的情报。",
			Outro:      "别一个人硬撑。先把今天最想被接走的那部分说出来。",
			Signature:  "小石光",
			WishPrompt: "如果还想补一句最想让他接手的事，也可以写在这里。",
			SceneHint:  "海边散步、暖风、被晚霞轻轻托住的信纸",
			Questions: []dto.FutureLetterQuestion{
				buildFutureLetterQuestion("stage", "想问问当下的你：正牵着哪一阶段的小手呢？", overviewStageOptions),
				buildFutureLetterQuestion("state", "身体总会发出些诚实的信号。此刻的你，感觉还好吗？", overviewStateOptions),
			},
		}
	}
}

func buildFutureLetterQuestion(id, prompt string, defs []futureLetterOptionDef) dto.FutureLetterQuestion {
	options := make([]dto.FutureLetterOption, 0, len(defs))
	for _, item := range defs {
		options = append(options, dto.FutureLetterOption{
			ID:    item.ID,
			Label: item.Label,
			Hint:  item.Hint,
		})
	}

	return dto.FutureLetterQuestion{
		ID:      id,
		Prompt:  prompt,
		Options: options,
	}
}

func (s *WhisperService) saveFutureLetterMission(record *model.FutureLetterResponse, mission dadMissionTemplate) error {
	date := today()
	dateStr := date.Format("2006-01-02")
	key := coupleKey(record.RecipientID, record.AuthorID)
	tasks := make([]model.UserTask, 0, len(mission.Tasks))
	for _, task := range mission.Tasks {
		priority := model.TaskPriority(task.Priority)
		if priority == "" {
			priority = model.PriorityT1
		}
		tasks = append(tasks, model.UserTask{
			UserID:        record.RecipientID,
			Date:          date,
			Status:        model.TaskPending,
			Source:        model.TaskSourceWhisper,
			AITitle:       task.Title,
			AIDescription: task.Description,
			AICategory:    task.Category,
			AIDifficulty:  task.Difficulty,
			Priority:      priority,
		})
	}

	var wish *model.Whisper
	err := s.taskRepo.WithDailyTaskLock(record.RecipientID, func(tx *gorm.DB) error {
		letters := repository.NewFutureLetterRepo(tx)
		if record.ID == "" {
			if err := letters.Create(record); err != nil {
				return err
			}
			if s.whisperRepo != nil && record.WishContent != nil {
				wish = &model.Whisper{AuthorID: record.AuthorID, Content: *record.WishContent}
				if err := repository.NewWhisperRepo(tx).Create(wish); err != nil {
					return err
				}
			}
		} else {
			latest, err := letters.FindRecentByRecipientID(record.RecipientID, 1)
			if err != nil {
				return err
			}
			if len(latest) == 0 || latest[0].ID != record.ID {
				return errFutureLetterChanged
			}
			if err := letters.Update(record); err != nil {
				return err
			}
		}

		repo := repository.NewTaskRepo(tx)
		if err := repo.DeletePendingUserTasksByDate(record.RecipientID, date); err != nil {
			return err
		}
		for _, cacheType := range []string{"task", "tips", "future-letter-template"} {
			if err := repo.DeleteAICacheByCouple(key, dateStr, cacheType); err != nil {
				return err
			}
		}
		return repo.CreateUserTasks(tasks)
	})
	if err != nil {
		return err
	}
	if wish != nil && s.ragService != nil {
		go func() {
			_ = s.ragService.IndexText(context.Background(), model.SourceWhisper, wish.ID, &wish.AuthorID, wish.Content)
		}()
	}
	return nil
}

func buildFutureLetterImagePrompt(letterCode, primaryLabel, secondaryLabel, missionTone, wish string) string {
	var sb strings.Builder
	switch letterCode {
	case futureLetterCodePrenatal:
		sb.WriteString("一张带着海风质感的中文信纸，孕期的妈妈站在海边，脑海里装着产检、待产包和临近生产前的琐碎准备。")
		fmt.Fprintf(&sb, "画面里要能感受到“%s”和“%s”这两种现实压力。", primaryLabel, secondaryLabel)
	case futureLetterCodeVaccine:
		sb.WriteString("一张带着海风质感的中文信纸，妈妈在沙滩边想着宝宝即将面对的疫苗小关卡。")
		fmt.Fprintf(&sb, "画面里要能感受到“%s”和“%s”这两种现实压力。", primaryLabel, secondaryLabel)
	case futureLetterCodeSafetySeat:
		sb.WriteString("一张带着海风质感的中文信纸，妈妈想到每一次出门上车前的安全动作，画面里有孩子、车门和被认真系好的安全感。")
		fmt.Fprintf(&sb, "画面里要能感受到“%s”和“%s”这两种现实压力。", primaryLabel, secondaryLabel)
	default:
		sb.WriteString("一张带着海风质感的中文信纸，漫步沙滩的妈妈被晚霞和海浪轻轻包住。")
		fmt.Fprintf(&sb, "她正处在“%s”的阶段，身体和情绪状态是“%s”。", primaryLabel, secondaryLabel)
	}
	if missionTone != "" {
		fmt.Fprintf(&sb, "画面气质要呈现%s。", missionTone)
	}
	if wish != "" {
		fmt.Fprintf(&sb, "把她此刻的小心愿也悄悄藏进画面里：%s。", wish)
	}
	sb.WriteString("整体温暖、细腻、轻盈，像被理解后终于能松一口气的瞬间。")
	return sb.String()
}

func toFutureLetterResponseItems(items []model.FutureLetterResponse) []dto.FutureLetterResponseItem {
	resp := make([]dto.FutureLetterResponseItem, len(items))
	for i, item := range items {
		resp[i] = toFutureLetterResponseItem(item)
	}
	return resp
}

func toFutureLetterResponseItem(item model.FutureLetterResponse) dto.FutureLetterResponseItem {
	adviceItems := make([]dto.FutureLetterAdviceItem, 0)
	if item.DadTasksJSON != "" {
		var raw []dadAdviceItemDef
		if err := json.Unmarshal([]byte(item.DadTasksJSON), &raw); err == nil {
			adviceItems = make([]dto.FutureLetterAdviceItem, 0, len(raw))
			for _, advice := range raw {
				adviceItems = append(adviceItems, dto.FutureLetterAdviceItem{
					Title:       advice.Title,
					Description: advice.Description,
					Kind:        advice.Kind,
				})
			}
		}
	}
	adviceSources := make([]dto.FutureLetterAdviceSource, 0)
	if item.DadAdviceSourcesJSON != "" {
		var raw []dadAdviceSourceDef
		if err := json.Unmarshal([]byte(item.DadAdviceSourcesJSON), &raw); err == nil {
			adviceSources = make([]dto.FutureLetterAdviceSource, 0, len(raw))
			for _, source := range raw {
				adviceSources = append(adviceSources, dto.FutureLetterAdviceSource{
					Label:  source.Label,
					Detail: source.Detail,
				})
			}
		}
	}

	return dto.FutureLetterResponseItem{
		ID:               item.ID,
		LetterCode:       item.LetterCode,
		StageTag:         item.StageTag,
		StageLabel:       item.StageLabel,
		StateTag:         item.StateTag,
		StateLabel:       item.StateLabel,
		WishContent:      item.WishContent,
		DadPlanCode:      item.DadPlanCode,
		DadPlanTitle:     item.DadPlanTitle,
		DadHeadline:      item.DadHeadline,
		DadSummary:       item.DadSummary,
		DadAdviceItems:   adviceItems,
		DadAdviceSources: adviceSources,
		ImagePrompt:      item.ImagePrompt,
		CreatedAt:        item.CreatedAt,
	}
}

// PreGenerateDailyLetters pre-generates future letter templates for all bound moms.
// Call this from a background goroutine on server startup and daily.
func (s *WhisperService) PreGenerateDailyLetters() {
	if !s.aiClient.Enabled() {
		return
	}
	moms, err := s.userRepo.FindBoundMoms()
	if err != nil {
		log.Printf("[WhisperService] PreGenerateDailyLetters: failed to find bound moms: %v", err)
		return
	}
	for _, mom := range moms {
		s.PreGenerateForUser(mom.ID)
	}
	log.Printf("[WhisperService] PreGenerateDailyLetters: completed for %d moms", len(moms))
}

// PreGenerateForUser pre-generates the daily letter template for a single user in background.
func (s *WhisperService) PreGenerateForUser(userID string) {
	if !s.aiClient.Enabled() {
		return
	}
	go func() {
		if _, err := s.GetFutureLetterView(userID); err != nil {
			log.Printf("[WhisperService] PreGenerateForUser: failed for user %s: %v", userID, err)
		}
	}()
}
