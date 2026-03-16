package database

import (
	"github.com/momshell/backend/internal/model"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
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
		&model.DailyTask{},
		&model.UserTask{},
		&model.AIGeneratedTask{},
	); err != nil {
		return err
	}

	// Seed default daily tasks if none exist
	seedDailyTasks(db)

	// Migrate legacy role='admin' users to is_admin flag
	db.Model(&model.User{}).Where("role = ?", "admin").Updates(map[string]interface{}{
		"is_admin": true,
		"role":     "mom",
	})

	// Backfill OwnerUserID for existing ChatMemoryFacts
	db.Exec("UPDATE chat_memory_facts SET owner_user_id = user_id WHERE owner_user_id IS NULL OR owner_user_id = ''")

	// Drop legacy NOT NULL and FK constraint on user_tasks.task_id (now nullable for AI tasks)
	db.Exec("ALTER TABLE user_tasks ALTER COLUMN task_id DROP NOT NULL")
	db.Exec("ALTER TABLE user_tasks DROP CONSTRAINT IF EXISTS fk_user_tasks_task")

	// Fix column names from GORM's default naming (a_idescription → ai_description)
	db.Exec("ALTER TABLE user_tasks RENAME COLUMN a_idescription TO ai_description")
	db.Exec("ALTER TABLE user_tasks RENAME COLUMN a_idifficulty TO ai_difficulty")
	return nil
}

func seedDailyTasks(db *gorm.DB) {
	var count int64
	db.Model(&model.DailyTask{}).Count(&count)
	if count > 0 {
		return
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

	for i := range tasks {
		if tasks[i].Priority == "" {
			tasks[i].Priority = model.PriorityT2
		}
		db.Create(&tasks[i])
	}
}
