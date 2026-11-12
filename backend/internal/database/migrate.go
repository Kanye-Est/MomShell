package database

import (
	"fmt"
	"log"

	"github.com/momshell/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Migrate(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Serialize schema changes and seeding when two instances start together.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(589023417)").Error; err != nil {
			return err
		}
		if err := migrateLegacyTaskColumns(tx); err != nil {
			return fmt.Errorf("migrate legacy task columns: %w", err)
		}
		return migrateSchema(tx)
	})
}

func migrateSchema(db *gorm.DB) error {
	// A savepoint keeps the main migration transaction usable when vector is unavailable.
	vectorErr := db.Transaction(func(tx *gorm.DB) error {
		return tx.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error
	})
	hasPgvector := vectorErr == nil
	if vectorErr != nil {
		log.Printf("[migrate] pgvector extension unavailable: %v", vectorErr)
	}

	models := []any{
		&model.User{},
		&model.UserCertification{},
		&model.Tag{},
		&model.Question{},
		&model.QuestionTag{},
		&model.Answer{},
		&model.Comment{},
		&model.Like{},
		&model.Collection{},
		&model.ModerationLog{},
		&model.ChatMemory{},
		&model.ChatMemoryFact{},
		&model.IdentityTag{},
		&model.Memoir{},
		&model.Photo{},
		&model.Whisper{},
		&model.FutureLetterResponse{},
		&model.DailyTask{},
		&model.UserTask{},
		&model.AIGeneratedTask{},
		&model.Achievement{},
		&model.UserAchievement{},
		&model.PerkCard{},
	}
	if hasPgvector {
		models = append(models, &model.KnowledgeEmbedding{})
	}

	if err := db.AutoMigrate(models...); err != nil {
		return err
	}

	if hasPgvector {
		if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_knowledge_embeddings_content_fts ON knowledge_embeddings USING GIN (to_tsvector('simple', content))").Error; err != nil {
			return err
		}
	}

	if err := seedDailyTasks(db); err != nil {
		return fmt.Errorf("seed daily tasks: %w", err)
	}
	if err := seedAchievements(db); err != nil {
		return fmt.Errorf("seed achievements: %w", err)
	}

	// Migrate legacy role='admin' users to is_admin flag
	if err := db.Model(&model.User{}).Where("role = ?", "admin").Updates(map[string]interface{}{
		"is_admin": true,
		"role":     "mom",
	}).Error; err != nil {
		return err
	}

	// Backfill dad chat style for legacy users after the column is introduced.
	if err := db.Exec("UPDATE users SET dad_chat_style = 'terminal' WHERE dad_chat_style IS NULL OR dad_chat_style = ''").Error; err != nil {
		return err
	}

	// Backfill OwnerUserID for existing ChatMemoryFacts
	if err := db.Exec("UPDATE chat_memory_facts SET owner_user_id = user_id WHERE owner_user_id IS NULL OR owner_user_id = ''").Error; err != nil {
		return err
	}

	// Drop legacy NOT NULL and FK constraint on user_tasks.task_id (now nullable for AI tasks)
	return db.Exec(`ALTER TABLE user_tasks
		ALTER COLUMN task_id DROP NOT NULL,
		DROP CONSTRAINT IF EXISTS fk_user_tasks_task`).Error
}

func migrateLegacyTaskColumns(db *gorm.DB) error {
	migrator := db.Migrator()
	columns := []struct {
		old, current, backfill string
	}{
		{"a_idescription", "ai_description", `UPDATE user_tasks SET ai_description = a_idescription
			WHERE (ai_description IS NULL OR ai_description = '') AND a_idescription <> ''`},
		{"a_idifficulty", "ai_difficulty", `UPDATE user_tasks SET ai_difficulty = a_idifficulty
			WHERE (ai_difficulty IS NULL OR ai_difficulty = 0) AND a_idifficulty <> 0`},
	}
	for _, col := range columns {
		if !migrator.HasColumn("user_tasks", col.old) {
			continue
		}
		if !migrator.HasColumn("user_tasks", col.current) {
			if err := migrator.RenameColumn("user_tasks", col.old, col.current); err != nil {
				return err
			}
			continue
		}
		// Older builds could add the new column before failing to rename the old one.
		if err := db.Exec(col.backfill).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedDailyTasks(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.DailyTask{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	tasks := []model.DailyTask{
		// Housework
		{Title: "做一顿早餐", Description: "为家人准备一份营养早餐", Category: model.TaskCategoryHousework, Difficulty: 2},
		{Title: "整理卧室", Description: "整理床铺、叠好衣物", Category: model.TaskCategoryHousework, Difficulty: 1},
		{Title: "打扫客厅", Description: "扫地、拖地、整理杂物", Category: model.TaskCategoryHousework, Difficulty: 2},
		{Title: "清洁厨房", Description: "洗碗、擦台面、清理灶台", Category: model.TaskCategoryHousework, Difficulty: 2},
		{Title: "洗一次衣服", Description: "把脏衣服分类洗好、晾晒", Category: model.TaskCategoryHousework, Difficulty: 2},
		{Title: "倒垃圾", Description: "把垃圾分类打包扔掉", Category: model.TaskCategoryHousework, Difficulty: 1},
		// Parenting
		{Title: "学习婴儿急救知识", Description: "看一篇婴儿急救相关的文章或视频", Category: model.TaskCategoryParenting, Difficulty: 3},
		{Title: "了解宝宝辅食添加", Description: "学习适龄辅食的种类和注意事项", Category: model.TaskCategoryParenting, Difficulty: 2},
		{Title: "学习安抚哭闹技巧", Description: "学习如何科学安抚宝宝", Category: model.TaskCategoryParenting, Difficulty: 2},
		{Title: "阅读一篇育儿文章", Description: "选一篇科学育儿知识文章阅读", Category: model.TaskCategoryParenting, Difficulty: 1},
		{Title: "陪宝宝互动 15 分钟", Description: "放下手机，全身心陪伴宝宝玩耍", Category: model.TaskCategoryParenting, Difficulty: 2},
		// Health
		{Title: "陪妈妈散步 30 分钟", Description: "陪伴一起户外走走，呼吸新鲜空气", Category: model.TaskCategoryHealth, Difficulty: 2},
		{Title: "提醒妈妈按时吃饭", Description: "确保她三餐按时吃、营养均衡", Category: model.TaskCategoryHealth, Difficulty: 1},
		{Title: "帮妈妈做肩颈按摩", Description: "帮她放松一下肩颈，缓解疲劳", Category: model.TaskCategoryHealth, Difficulty: 2},
		{Title: "提醒妈妈喝水", Description: "关心她的饮水量，适时提醒", Category: model.TaskCategoryHealth, Difficulty: 1},
		// Emotional
		{Title: "给妈妈写一段鼓励的话", Description: "用文字表达你对她的感谢和支持", Category: model.TaskCategoryEmotional, Difficulty: 3},
		{Title: "主动询问她今天的感受", Description: "认真倾听她的心情，不急着给建议", Category: model.TaskCategoryEmotional, Difficulty: 2},
		{Title: "准备一个小惊喜", Description: "一杯热饮、一束花、或一句暖心的话", Category: model.TaskCategoryEmotional, Difficulty: 3},
		{Title: "主动承担一次夜间照顾", Description: "让妈妈好好休息一晚", Category: model.TaskCategoryEmotional, Difficulty: 4},
		{Title: "表达一次感谢", Description: "告诉她你看到了她的付出和辛苦", Category: model.TaskCategoryEmotional, Difficulty: 1},
	}

	return db.Create(&tasks).Error
}

func seedAchievements(db *gorm.DB) error {
	achievements := []model.Achievement{
		{
			Code:        "aap_apprentice",
			Title:       "AAP见习奶爸",
			Description: "累计完成并验收 10 个任务",
			Condition:   `{"type":"task_count","min":10}`,
		},
		{
			Code:        "sleep_master",
			Title:       "哄睡大师",
			Description: "情绪安抚维度达到 20",
			Condition:   `{"type":"dimension_min","dimension":"emotional","min":20}`,
		},
		{
			Code:        "logistics_pro",
			Title:       "后勤达人",
			Description: "后勤保障维度达到 30",
			Condition:   `{"type":"dimension_min","dimension":"logistics","min":30}`,
		},
	}

	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoNothing: true,
	}).Create(&achievements).Error
}
