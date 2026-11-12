package repository

import (
	"time"

	"github.com/momshell/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type TaskRepo struct {
	db *gorm.DB
}

func NewTaskRepo(db *gorm.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

// DailyTask template operations

func (r *TaskRepo) CountTemplates() (int64, error) {
	var count int64
	err := r.db.Model(&model.DailyTask{}).Count(&count).Error
	return count, err
}

func (r *TaskRepo) CreateTemplate(t *model.DailyTask) error {
	return r.db.Create(t).Error
}

func (r *TaskRepo) FindRandomTemplates(count int, excludeIDs []string) ([]model.DailyTask, error) {
	var tasks []model.DailyTask
	query := r.db.Order("RANDOM()").Limit(count)
	if len(excludeIDs) > 0 {
		query = query.Where("id NOT IN ?", excludeIDs)
	}
	err := query.Find(&tasks).Error
	return tasks, err
}

// UserTask operations

func (r *TaskRepo) WithDailyTaskLock(userID string, run func(tx *gorm.DB) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(589023418, hashtext(?))", userID).Error; err != nil {
			return err
		}
		return run(tx)
	})
}

func (r *TaskRepo) CreateUserTasks(tasks []model.UserTask) error {
	return r.db.Create(&tasks).Error
}

func (r *TaskRepo) FindUserTasksByDate(userID string, date time.Time) ([]model.UserTask, error) {
	var tasks []model.UserTask
	err := r.db.Preload("Task").
		Where("user_id = ? AND date = ?", userID, date.Format("2006-01-02")).
		Order("created_at asc").
		Find(&tasks).Error
	return tasks, err
}

func (r *TaskRepo) FindUserTaskByID(id string) (*model.UserTask, error) {
	var task model.UserTask
	err := r.db.Preload("Task").Preload("User").First(&task, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepo) UpdateUserTask(ut *model.UserTask, previousStatus model.TaskStatus) error {
	result := r.db.Model(ut).
		Where("status = ?", previousStatus).
		Select("status", "completed_at", "proof_photo_url", "score", "comment", "scored_by_id", "scored_at").
		Updates(ut)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *TaskRepo) SumScoreByUserID(userID string) (int, error) {
	var total *int
	err := r.db.Model(&model.UserTask{}).
		Where("user_id = ? AND status = ? AND score IS NOT NULL", userID, model.TaskVerified).
		Select("COALESCE(SUM(score), 0)").
		Scan(&total).Error
	if err != nil {
		return 0, err
	}
	if total == nil {
		return 0, nil
	}
	return *total, nil
}

func (r *TaskRepo) CountVerifiedTasksByUserID(userID string) (int, error) {
	var count int64
	err := r.db.Model(&model.UserTask{}).
		Where("user_id = ? AND status = ?", userID, model.TaskVerified).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r *TaskRepo) FindVerifiedTasksByUserID(userID string) ([]model.UserTask, error) {
	var tasks []model.UserTask
	err := r.db.Preload("Task").
		Where("user_id = ? AND status = ? AND score IS NOT NULL", userID, model.TaskVerified).
		Order("scored_at asc").
		Find(&tasks).Error
	return tasks, err
}

// AI cache operations

func (r *TaskRepo) FindAICache(coupleKey, date string) (*model.AIGeneratedTask, error) {
	var cache model.AIGeneratedTask
	err := r.db.Session(&gorm.Session{Logger: r.db.Logger.LogMode(logger.Silent)}).
		Where("couple_key = ? AND date = ?", coupleKey, date).First(&cache).Error
	if err != nil {
		return nil, err
	}
	return &cache, nil
}

func (r *TaskRepo) SaveAICache(cache *model.AIGeneratedTask) error {
	return r.db.Create(cache).Error
}

func (r *TaskRepo) DeleteAICacheByCouple(coupleKey, date string) error {
	return r.db.Where("couple_key = ? AND date = ?", coupleKey, date).
		Delete(&model.AIGeneratedTask{}).Error
}

func (r *TaskRepo) DeletePendingUserTasksByDate(userIDs []string, date time.Time) error {
	return r.db.Where("user_id IN ? AND date = ? AND status = ?", userIDs, date.Format("2006-01-02"), model.TaskPending).
		Delete(&model.UserTask{}).Error
}
