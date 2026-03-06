package service

import (
	"context"
	"encoding/json"
	"time"

	"go-cloud/internal/domain"
	"go-cloud/internal/queue"
	"go-cloud/internal/repository"
)

type ExecutionService interface {
	GetExecution(ctx context.Context, executionNo string) (*domain.TaskExecution, error)
	ListExecutions(ctx context.Context, filter repository.ExecutionListFilter) ([]domain.TaskExecution, int64, error)
	ListTaskExecutions(ctx context.Context, taskID int64, page int, pageSize int) ([]domain.TaskExecution, int64, error)
	GetExecutionLogs(ctx context.Context, executionNo string) (*string, error)
	RetryExecution(ctx context.Context, executionNo string, triggerBy string) (string, error)
	CancelExecution(ctx context.Context, executionNo string, cancelledBy string) error
}

type executionService struct {
	taskRepo      repository.TaskRepository
	executionRepo repository.ExecutionRepository
	queueRepo     repository.QueueRepository
}

func NewExecutionService(taskRepo repository.TaskRepository, executionRepo repository.ExecutionRepository, queueRepo repository.QueueRepository) ExecutionService {
	return &executionService{
		taskRepo:      taskRepo,
		executionRepo: executionRepo,
		queueRepo:     queueRepo,
	}
}

func (s *executionService) GetExecution(ctx context.Context, executionNo string) (*domain.TaskExecution, error) {
	execution, err := s.executionRepo.GetByExecutionNo(ctx, executionNo)
	if err != nil {
		return nil, err
	}
	if execution == nil {
		return nil, NotFoundError("execution not found")
	}
	return execution, nil
}

func (s *executionService) ListTaskExecutions(ctx context.Context, taskID int64, page int, pageSize int) ([]domain.TaskExecution, int64, error) {
	return s.executionRepo.ListByTaskID(ctx, taskID, page, pageSize)
}

func (s *executionService) ListExecutions(ctx context.Context, filter repository.ExecutionListFilter) ([]domain.TaskExecution, int64, error) {
	return s.executionRepo.List(ctx, filter)
}

func (s *executionService) GetExecutionLogs(ctx context.Context, executionNo string) (*string, error) {
	execution, err := s.GetExecution(ctx, executionNo)
	if err != nil {
		return nil, err
	}
	return execution.OutputLog, nil
}

func (s *executionService) RetryExecution(ctx context.Context, executionNo string, triggerBy string) (string, error) {
	current, err := s.executionRepo.GetByExecutionNo(ctx, executionNo)
	if err != nil {
		return "", err
	}
	if current == nil {
		return "", NotFoundError("execution not found")
	}
	if !canRetryExecution(current.Status) {
		return "", ConflictError("execution can only retry failed, timeout, or cancelled states")
	}
	task, err := s.taskRepo.GetByID(ctx, current.TaskID)
	if err != nil {
		return "", err
	}
	if task == nil {
		return "", NotFoundError("task not found")
	}
	if task.Status != domain.TaskStatusActive {
		return "", ConflictError("task is not active")
	}
	newExecutionNo := generateExecutionNo(task.ID)
	now := time.Now().UTC()
	retryCount := current.RetryCount + 1
	retryExecution := &domain.TaskExecution{
		TaskID:      task.ID,
		ExecutionNo: newExecutionNo,
		TriggerType: domain.TriggerTypeRetry,
		Status:      domain.ExecutionStatusPending,
		RetryCount:  retryCount,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.executionRepo.Create(ctx, retryExecution); err != nil {
		return "", err
	}
	payload, err := json.Marshal(queue.TaskMessage{
		TaskID:      task.ID,
		ExecutionNo: newExecutionNo,
		TriggerType: string(domain.TriggerTypeRetry),
		TriggerBy:   defaultActor(triggerBy),
		RetryCount:  retryCount,
	})
	if err != nil {
		return "", err
	}
	if err := s.queueRepo.EnqueueTask(ctx, payload); err != nil {
		if cleanupErr := s.executionRepo.DeleteByExecutionNo(ctx, newExecutionNo); cleanupErr != nil {
			return "", cleanupErr
		}
		return "", err
	}
	return newExecutionNo, nil
}

func (s *executionService) CancelExecution(ctx context.Context, executionNo string, _ string) error {
	execution, err := s.executionRepo.GetByExecutionNo(ctx, executionNo)
	if err != nil {
		return err
	}
	if execution == nil {
		return NotFoundError("execution not found")
	}
	if execution.Status != domain.ExecutionStatusPending {
		return ConflictError("only pending execution can be cancelled")
	}
	return s.executionRepo.UpdateStatus(ctx, executionNo, domain.ExecutionStatusCancelled, "")
}

func canRetryExecution(status domain.ExecutionStatus) bool {
	return status == domain.ExecutionStatusFailed ||
		status == domain.ExecutionStatusTimeout ||
		status == domain.ExecutionStatusCancelled
}
