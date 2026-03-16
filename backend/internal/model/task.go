package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskCategory enum
type TaskCategory string

const (
	TaskCategoryHousework TaskCategory = "housework"
	TaskCategoryParenting TaskCategory = "parenting"
	TaskCategoryHealth    TaskCategory = "health"
	TaskCategoryEmotional TaskCategory = "emotional"
)

// TaskPriority enum
type TaskPriority string

const (
	PriorityT0 TaskPriority = "T0" // 突发/情绪干预
	PriorityT1 TaskPriority = "T1" // 关键里程碑
	PriorityT2 TaskPriority = "T2" // 日常守护
)

// TaskStatus enum
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskCompleted TaskStatus = "completed"
	TaskVerified  TaskStatus = "verified"
)

// DailyTask is a task template from which user tasks are generated.
type DailyTask struct {
	ID          string       `gorm:"type:varchar(36);primaryKey" json:"id"`
	Title       string       `gorm:"type:varchar(200);not null" json:"title"`
	Description string       `gorm:"type:text" json:"description"`
	Category    TaskCategory `gorm:"type:varchar(50);not null;index" json:"category"`
	Difficulty  int          `gorm:"default:1" json:"difficulty"` // 1-5
	Priority    TaskPriority `gorm:"type:varchar(10);default:'T2'" json:"priority"`
	CreatedAt   time.Time    `json:"created_at"`
}

func (t *DailyTask) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

// TaskSource indicates how a UserTask was created.
type TaskSource string

const (
	TaskSourceTemplate TaskSource = "template"
	TaskSourceAI       TaskSource = "ai"
)

// UserTask is a concrete task assigned to a user on a given date.
type UserTask struct {
	ID     string  `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID string  `gorm:"type:varchar(36);not null;index" json:"user_id"`
	TaskID *string `gorm:"type:varchar(36)" json:"task_id"` // nil for AI tasks

	Date   time.Time  `gorm:"type:date;not null;index" json:"date"`
	Status TaskStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`

	// Priority level
	Priority TaskPriority `gorm:"type:varchar(10);default:'T2'" json:"priority"`

	// AI-generated task fields (populated when Source == "ai")
	Source        TaskSource `gorm:"type:varchar(20);default:'template'" json:"source"`
	AITitle       string     `gorm:"column:ai_title;type:varchar(200)" json:"ai_title,omitempty"`
	AIDescription string     `gorm:"column:ai_description;type:text" json:"ai_description,omitempty"`
	AICategory    string     `gorm:"column:ai_category;type:varchar(50)" json:"ai_category,omitempty"`
	AIDifficulty  int        `gorm:"column:ai_difficulty;default:0" json:"ai_difficulty,omitempty"`

	// Verification by partner
	Score      *int       `gorm:"type:int" json:"score"`
	Comment    *string    `gorm:"type:text" json:"comment"`
	ScoredByID *string    `gorm:"type:varchar(36)" json:"scored_by_id"`
	ScoredAt   *time.Time `json:"scored_at"`

	// Proof photo (optional, attached when completing)
	ProofPhotoURL *string `gorm:"type:varchar(500)" json:"proof_photo_url"`

	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Relationships
	User     User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Task     *DailyTask `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	ScoredBy *User      `gorm:"foreignKey:ScoredByID" json:"scored_by,omitempty"`
}

// AIGeneratedTask caches the AI-generated task set per couple per day.
type AIGeneratedTask struct {
	ID        uint      `gorm:"primaryKey"`
	CoupleKey string    `gorm:"type:varchar(80);uniqueIndex:idx_couple_date" json:"couple_key"`
	Date      string    `gorm:"type:varchar(10);uniqueIndex:idx_couple_date" json:"date"`
	AgeStage  string    `gorm:"type:varchar(20)" json:"age_stage"`
	TasksJSON string    `gorm:"type:text" json:"tasks_json"`
	CreatedAt time.Time `json:"created_at"`
}

func (ut *UserTask) BeforeCreate(tx *gorm.DB) error {
	if ut.ID == "" {
		ut.ID = uuid.New().String()
	}
	return nil
}
