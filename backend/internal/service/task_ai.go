package service

import (
	"strings"
	"unicode/utf8"

	"github.com/momshell/backend/internal/model"
	"github.com/momshell/backend/internal/repository"
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

func normalizeAITasks(tasks []AITaskData) []AITaskData {
	// Validate categories
	validCategories := map[string]bool{
		"housework": true, "parenting": true,
		"health": true, "emotional": true,
	}
	validPriorities := map[string]bool{
		"T0": true, "T1": true, "T2": true,
	}

	normalized := make([]AITaskData, 0, min(len(tasks), 5))
	seen := make(map[string]bool)
	for _, task := range tasks {
		task.Title = strings.TrimSpace(task.Title)
		if task.Title == "" || utf8.RuneCountInString(task.Title) > 200 || seen[task.Title] {
			continue
		}
		seen[task.Title] = true
		task.Category = strings.ToLower(strings.TrimSpace(task.Category))
		if !validCategories[task.Category] {
			task.Category = "parenting"
		}
		task.Difficulty = max(1, min(5, task.Difficulty))
		task.Priority = strings.ToUpper(strings.TrimSpace(task.Priority))
		if !validPriorities[task.Priority] {
			task.Priority = "T2"
		}
		normalized = append(normalized, task)
		if len(normalized) == 5 {
			break
		}
	}
	return normalized
}
