package repository

import (
	"context"
	"time"

	"go-cloud/internal/domain"
)

type TaskListFilter struct {
	Status   *string
	TaskType *string
	Page     int
	PageSize int
}

type ExecutionListFilter struct {
	TaskID   *int64
	Status   *string
	Page     int
	PageSize int
}

type ReleaseListFilter struct {
	Environment *string
	Status      *string
	Page        int
	PageSize    int
}

type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, id int64) (*domain.Task, error)
	List(ctx context.Context, filter TaskListFilter) ([]domain.Task, int64, error)
	Update(ctx context.Context, task *domain.Task) error
	UpdateStatus(ctx context.Context, id int64, status domain.TaskStatus, updatedBy string) error
	ListDueTasks(ctx context.Context, now time.Time, limit int) ([]domain.Task, error)
}

type ExecutionRepository interface {
	Create(ctx context.Context, execution *domain.TaskExecution) error
	DeleteByExecutionNo(ctx context.Context, executionNo string) error
	GetByExecutionNo(ctx context.Context, executionNo string) (*domain.TaskExecution, error)
	List(ctx context.Context, filter ExecutionListFilter) ([]domain.TaskExecution, int64, error)
	ListByTaskID(ctx context.Context, taskID int64, page int, pageSize int) ([]domain.TaskExecution, int64, error)
	UpdateStatus(ctx context.Context, executionNo string, status domain.ExecutionStatus, workerID string) error
	Finish(ctx context.Context, executionNo string, status domain.ExecutionStatus, durationMs int64, exitCode *int, errorMessage *string, outputLog *string) error
}

type ReleaseRepository interface {
	Create(ctx context.Context, release *domain.ReleaseRecord) error
	GetByID(ctx context.Context, id int64) (*domain.ReleaseRecord, error)
	List(ctx context.Context, filter ReleaseListFilter) ([]domain.ReleaseRecord, int64, error)
}

type QueueRepository interface {
	EnqueueTask(ctx context.Context, payload []byte) error
	DequeueTask(ctx context.Context, timeout time.Duration) ([]byte, error)
	EnqueueNotification(ctx context.Context, payload []byte) error
	DequeueNotification(ctx context.Context, timeout time.Duration) ([]byte, error)
}

type LockRepository interface {
	AcquireTaskDispatchLock(ctx context.Context, taskID int64, ttl time.Duration) (bool, error)
}
