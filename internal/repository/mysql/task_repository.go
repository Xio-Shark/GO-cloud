package mysqlrepo

import (
	"context"
	"errors"
	"time"

	"go-cloud/internal/domain"
	"go-cloud/internal/repository"
	"go-cloud/internal/repository/model"

	"gorm.io/gorm"
)

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) repository.TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, task *domain.Task) error {
	record := taskToModel(*task)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return err
	}
	task.ID = record.ID
	return nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	var record model.Task
	err := r.db.WithContext(ctx).
		Where("id = ? AND status <> ?", id, string(domain.TaskStatusDeleted)).
		First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := taskFromModel(record)
	return &result, nil
}

func (r *TaskRepository) List(ctx context.Context, filter repository.TaskListFilter) ([]domain.Task, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Task{})
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	} else {
		query = query.Where("status <> ?", string(domain.TaskStatusDeleted))
	}
	if filter.TaskType != nil {
		query = query.Where("task_type = ?", *filter.TaskType)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page := normalizePage(filter.Page)
	pageSize := normalizePageSize(filter.PageSize)
	var records []model.Task
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	items := make([]domain.Task, 0, len(records))
	for _, record := range records {
		items = append(items, taskFromModel(record))
	}
	return items, total, nil
}

func (r *TaskRepository) Update(ctx context.Context, task *domain.Task) error {
	record := taskToModel(*task)
	return r.db.WithContext(ctx).Model(&model.Task{}).Where("id = ?", task.ID).Updates(&record).Error
}

func (r *TaskRepository) UpdateStatus(ctx context.Context, id int64, status domain.TaskStatus, updatedBy string) error {
	return r.db.WithContext(ctx).Model(&model.Task{}).Where("id = ?", id).Updates(map[string]any{
		"status":     string(status),
		"updated_by": updatedBy,
		"updated_at": time.Now().UTC(),
	}).Error
}

func (r *TaskRepository) ListDueTasks(ctx context.Context, now time.Time, limit int) ([]domain.Task, error) {
	if limit <= 0 {
		limit = 100
	}
	var records []model.Task
	err := r.db.WithContext(ctx).
		Where("status = ? AND next_run_time IS NOT NULL AND next_run_time <= ?", string(domain.TaskStatusActive), now).
		Order("next_run_time ASC").
		Limit(limit).
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	items := make([]domain.Task, 0, len(records))
	for _, record := range records {
		items = append(items, taskFromModel(record))
	}
	return items, nil
}

func taskToModel(task domain.Task) model.Task {
	return model.Task{
		ID:             task.ID,
		Name:           task.Name,
		Description:    task.Description,
		TaskType:       string(task.TaskType),
		ScheduleType:   string(task.ScheduleType),
		CronExpr:       task.CronExpr,
		Payload:        task.Payload,
		TimeoutSeconds: task.TimeoutSeconds,
		RetryTimes:     task.RetryTimes,
		Status:         string(task.Status),
		CallbackURL:    task.CallbackURL,
		CreatedBy:      task.CreatedBy,
		UpdatedBy:      task.UpdatedBy,
		LastRunTime:    task.LastRunTime,
		NextRunTime:    task.NextRunTime,
		CreatedAt:      task.CreatedAt,
		UpdatedAt:      task.UpdatedAt,
	}
}

func taskFromModel(record model.Task) domain.Task {
	return domain.Task{
		ID:             record.ID,
		Name:           record.Name,
		Description:    record.Description,
		TaskType:       domain.TaskType(record.TaskType),
		ScheduleType:   domain.ScheduleType(record.ScheduleType),
		CronExpr:       record.CronExpr,
		Payload:        record.Payload,
		TimeoutSeconds: record.TimeoutSeconds,
		RetryTimes:     record.RetryTimes,
		Status:         domain.TaskStatus(record.Status),
		CallbackURL:    record.CallbackURL,
		CreatedBy:      record.CreatedBy,
		UpdatedBy:      record.UpdatedBy,
		LastRunTime:    record.LastRunTime,
		NextRunTime:    record.NextRunTime,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
	}
}

func normalizePage(page int) int {
	if page <= 0 {
		return 1
	}
	return page
}

func normalizePageSize(pageSize int) int {
	if pageSize <= 0 || pageSize > 100 {
		return 20
	}
	return pageSize
}
