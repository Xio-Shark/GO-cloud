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

type ExecutionRepository struct {
	db *gorm.DB
}

func NewExecutionRepository(db *gorm.DB) repository.ExecutionRepository {
	return &ExecutionRepository{db: db}
}

func (r *ExecutionRepository) Create(ctx context.Context, execution *domain.TaskExecution) error {
	record := executionToModel(*execution)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return err
	}
	execution.ID = record.ID
	return nil
}

func (r *ExecutionRepository) DeleteByExecutionNo(ctx context.Context, executionNo string) error {
	return r.db.WithContext(ctx).Where("execution_no = ?", executionNo).Delete(&model.TaskExecution{}).Error
}

func (r *ExecutionRepository) GetByExecutionNo(ctx context.Context, executionNo string) (*domain.TaskExecution, error) {
	var record model.TaskExecution
	err := r.db.WithContext(ctx).Where("execution_no = ?", executionNo).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := executionFromModel(record)
	return &result, nil
}

func (r *ExecutionRepository) List(ctx context.Context, filter repository.ExecutionListFilter) ([]domain.TaskExecution, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.TaskExecution{})
	if filter.TaskID != nil {
		query = query.Where("task_id = ?", *filter.TaskID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page := normalizePage(filter.Page)
	pageSize := normalizePageSize(filter.PageSize)
	var records []model.TaskExecution
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	items := make([]domain.TaskExecution, 0, len(records))
	for _, record := range records {
		items = append(items, executionFromModel(record))
	}
	return items, total, nil
}

func (r *ExecutionRepository) ListByTaskID(ctx context.Context, taskID int64, page int, pageSize int) ([]domain.TaskExecution, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.TaskExecution{}).Where("task_id = ?", taskID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page = normalizePage(page)
	pageSize = normalizePageSize(pageSize)
	var records []model.TaskExecution
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	items := make([]domain.TaskExecution, 0, len(records))
	for _, record := range records {
		items = append(items, executionFromModel(record))
	}
	return items, total, nil
}

func (r *ExecutionRepository) UpdateStatus(ctx context.Context, executionNo string, status domain.ExecutionStatus, workerID string) error {
	now := time.Now().UTC()
	updates := map[string]any{
		"status":     string(status),
		"worker_id":  workerID,
		"updated_at": now,
	}
	if status == domain.ExecutionStatusRunning {
		updates["start_time"] = now
	}
	return r.db.WithContext(ctx).Model(&model.TaskExecution{}).Where("execution_no = ?", executionNo).Updates(updates).Error
}

func (r *ExecutionRepository) Finish(ctx context.Context, executionNo string, status domain.ExecutionStatus, durationMs int64, exitCode *int, errorMessage *string, outputLog *string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&model.TaskExecution{}).Where("execution_no = ?", executionNo).Updates(map[string]any{
		"status":        string(status),
		"duration_ms":   durationMs,
		"exit_code":     exitCode,
		"error_message": errorMessage,
		"output_log":    outputLog,
		"end_time":      now,
		"updated_at":    now,
	}).Error
}

func executionToModel(execution domain.TaskExecution) model.TaskExecution {
	return model.TaskExecution{
		ID:           execution.ID,
		TaskID:       execution.TaskID,
		ExecutionNo:  execution.ExecutionNo,
		TriggerType:  string(execution.TriggerType),
		WorkerID:     execution.WorkerID,
		Status:       string(execution.Status),
		StartTime:    execution.StartTime,
		EndTime:      execution.EndTime,
		DurationMs:   execution.DurationMs,
		RetryCount:   execution.RetryCount,
		ExitCode:     execution.ExitCode,
		ErrorMessage: execution.ErrorMessage,
		OutputLog:    execution.OutputLog,
		CreatedAt:    execution.CreatedAt,
		UpdatedAt:    execution.UpdatedAt,
	}
}

func executionFromModel(record model.TaskExecution) domain.TaskExecution {
	return domain.TaskExecution{
		ID:           record.ID,
		TaskID:       record.TaskID,
		ExecutionNo:  record.ExecutionNo,
		TriggerType:  domain.TriggerType(record.TriggerType),
		WorkerID:     record.WorkerID,
		Status:       domain.ExecutionStatus(record.Status),
		StartTime:    record.StartTime,
		EndTime:      record.EndTime,
		DurationMs:   record.DurationMs,
		RetryCount:   record.RetryCount,
		ExitCode:     record.ExitCode,
		ErrorMessage: record.ErrorMessage,
		OutputLog:    record.OutputLog,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}
