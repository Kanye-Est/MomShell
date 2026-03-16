package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/momshell/backend/internal/model"
	"github.com/momshell/backend/internal/repository"
	"github.com/momshell/backend/pkg/openai"
)

// AITaskData represents a single AI-generated task before being persisted.
type AITaskData struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Difficulty  int    `json:"difficulty"`
	Priority    string `json:"priority"`
}

// ageStageLabels maps backend keys to Chinese display labels for the prompt.
var ageStageLabels = map[string]string{
	"pregnancy": "孕期",
	"0-3m":      "0-3个月",
	"3-6m":      "3-6个月",
	"6-12m":     "6-12个月",
	"1-2y":      "1-2岁",
	"2-3y":      "2-3岁",
	"3-4y":      "3-4岁",
	"4-5y":      "4-5岁",
}

// coupleKey produces a consistent key from two user IDs.
func coupleKey(a, b string) string {
	if a < b {
		return a + "-" + b
	}
	return b + "-" + a
}

// resolveAgeStageFromChatMemory tries to infer the age stage from chat memory facts.
func resolveAgeStageFromChatMemory(chatRepo *repository.ChatRepo, familyIDs []string) (string, bool) {
	facts, err := chatRepo.FindFactsByFamilyIDs(familyIDs)
	if err != nil {
		return "", false
	}
	for _, f := range facts {
		lower := strings.ToLower(f.Content)
		if strings.Contains(lower, "宝宝") || strings.Contains(lower, "baby") || strings.Contains(lower, "孩子") {
			stage := inferAgeStageFromMemory(f.Content)
			if stage != "" {
				return stage, true
			}
		}
	}
	return "", false
}

// resolveAgeStage returns the baby age stage for the couple, checking:
// 1. The user's own BabyAgeStage
// 2. The partner's BabyAgeStage
// 3. Chat memory facts (baby_age or baby_age_weeks)
func resolveAgeStage(
	user *model.User,
	userRepo *repository.UserRepo,
	chatRepo *repository.ChatRepo,
) (stage string, source string) {
	// 1. User's own setting
	if user.BabyAgeStage != nil && *user.BabyAgeStage != "" {
		return *user.BabyAgeStage, "manual"
	}

	// 2. Partner's setting
	if user.PartnerID != nil {
		partner, err := userRepo.FindByID(*user.PartnerID)
		if err == nil && partner.BabyAgeStage != nil && *partner.BabyAgeStage != "" {
			return *partner.BabyAgeStage, "partner"
		}
	}

	// 3. Chat memory: look for baby_age or age-related facts
	if chatRepo != nil {
		familyIDs := []string{user.ID}
		if user.PartnerID != nil {
			familyIDs = append(familyIDs, *user.PartnerID)
		}
		if s, found := resolveAgeStageFromChatMemory(chatRepo, familyIDs); found {
			return s, "memory"
		}
	}

	return "", ""
}

// agePattern pairs an age stage key with its keyword patterns.
type agePattern struct {
	stage    string
	patterns []string
}

// agePatterns is an ordered list so matching is deterministic.
var agePatterns = []agePattern{
	{"pregnancy", []string{"怀孕", "孕期", "pregnant", "孕"}},
	{"0-3m", []string{"新生儿", "月子", "0个月", "1个月", "2个月", "3个月", "刚出生"}},
	{"3-6m", []string{"4个月", "5个月", "6个月", "半岁"}},
	{"6-12m", []string{"7个月", "8个月", "9个月", "10个月", "11个月"}},
	{"1-2y", []string{"1岁", "一岁", "12个月"}},
	{"2-3y", []string{"2岁", "两岁"}},
	{"3-4y", []string{"3岁", "三岁"}},
	{"4-5y", []string{"4岁", "四岁", "5岁", "五岁"}},
}

// inferAgeStageFromMemory attempts to extract an age stage from a memory fact.
func inferAgeStageFromMemory(content string) string {
	lower := strings.ToLower(content)

	for _, ap := range agePatterns {
		for _, p := range ap.patterns {
			if strings.Contains(lower, p) {
				return ap.stage
			}
		}
	}

	return ""
}

// generateAITasks calls the OpenAI API to generate tasks for a given age stage.
func generateAITasks(
	client *openai.Client,
	ageStage string,
) ([]AITaskData, error) {
	label := ageStageLabels[ageStage]
	if label == "" {
		label = ageStage
	}

	systemPrompt := `你是一个专业的育儿任务顾问。请为一位爸爸生成每日育儿任务。

要求：
- 生成4到5个任务
- 任务要与孩子当前年龄阶段匹配
- 任务类别包括：housework（家务）、parenting（育儿）、health（健康）、emotional（情感）
- 每个任务难度1-5分
- 每个任务需要标注优先级 priority：T0（突发/情绪干预）、T1（关键里程碑）、T2（日常守护）
- 任务要具体可执行，不要太笼统
- 包含直接照顾宝宝的任务，也要包含关心妈妈的任务
- 用中文描述

请以JSON数组格式返回，每个元素包含：title, description, category, difficulty, priority
不要返回任何其他内容，只返回JSON数组。`

	userPrompt := fmt.Sprintf("孩子当前年龄阶段：%s\n请生成今日任务。", label)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, []openai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return nil, fmt.Errorf("AI task generation failed: %w", err)
	}

	// Parse JSON response — strip markdown code fences if present
	cleaned := strings.TrimSpace(resp)
	if strings.HasPrefix(cleaned, "```") {
		if idx := strings.Index(cleaned[3:], "\n"); idx >= 0 {
			cleaned = cleaned[3+idx+1:]
		}
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	}

	var tasks []AITaskData
	if err := json.Unmarshal([]byte(cleaned), &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse AI tasks: %w (response: %s)", err, cleaned[:min(len(cleaned), 200)])
	}

	return normalizeAITasks(tasks), nil
}

func normalizeAITasks(tasks []AITaskData) []AITaskData {
	// Validate categories
	validCategories := map[string]bool{
		"housework": true, "parenting": true,
		"health": true, "emotional": true,
	}
	validPriorities := map[string]bool{
		"T0": true, "T1": true, "T2": true,
	}

	for i := range tasks {
		tasks[i].Category = strings.ToLower(strings.TrimSpace(tasks[i].Category))
		if !validCategories[tasks[i].Category] {
			tasks[i].Category = "parenting"
		}
		if tasks[i].Difficulty < 1 {
			tasks[i].Difficulty = 1
		}
		if tasks[i].Difficulty > 5 {
			tasks[i].Difficulty = 5
		}

		tasks[i].Priority = strings.ToUpper(strings.TrimSpace(tasks[i].Priority))
		if !validPriorities[tasks[i].Priority] {
			tasks[i].Priority = "T2"
		}
	}

	return tasks
}

// getOrGenerateAITasks checks cache first, then generates if needed.
func getOrGenerateAITasks(
	client *openai.Client,
	taskRepo *repository.TaskRepo,
	ck string,
	date string,
	ageStage string,
) ([]AITaskData, error) {
	// Check cache
	cache, err := taskRepo.FindAICache(ck, date)
	if err == nil && cache != nil {
		var tasks []AITaskData
		if err := json.Unmarshal([]byte(cache.TasksJSON), &tasks); err == nil {
			return normalizeAITasks(tasks), nil
		}
		// Bad cache entry — delete it so the unique index won't block re-save
		log.Printf("[TaskAI] cache parse error for %s/%s, deleting and regenerating", ck, date)
		_ = taskRepo.DeleteAICacheByCouple(ck, date)
	}

	// Generate
	tasks, err := generateAITasks(client, ageStage)
	if err != nil {
		return nil, err
	}

	// Save to cache
	tasksJSON, _ := json.Marshal(tasks)
	cacheEntry := &model.AIGeneratedTask{
		CoupleKey: ck,
		Date:      date,
		AgeStage:  ageStage,
		TasksJSON: string(tasksJSON),
	}
	if err := taskRepo.SaveAICache(cacheEntry); err != nil {
		log.Printf("[TaskAI] failed to cache AI tasks: %v", err)
	}

	return tasks, nil
}
