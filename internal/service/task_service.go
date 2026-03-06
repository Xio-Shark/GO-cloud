package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"go-cloud/internal/domain"
	"go-cloud/internal/dto"
	"go-cloud/internal/queue"
	"go-cloud/internal/repository"
	"go-cloud/pkg/cronutil"
	"go-cloud/pkg/traceutil"
)

type TaskService interface {
	CreateTask(ctx context.Context, req dto.CreateTaskRequest) (*domain.Task, error)
	GetTask(ctx context.Context, id int64) (*domain.Task, error)
	ListTasks(ctx context.Context, filter repository.TaskListFilter) ([]domain.Task, int64, error)
	UpdateTask(ctx context.Context, id int64, req dto.UpdateTaskRequest) error
	PauseTask(ctx context.Context, id int64, updatedBy string) error
	ResumeTask(ctx context.Context, id int64, updatedBy string) error
	TriggerTask(ctx context.Context, id int64, triggerBy string) (string, error)
	DeleteTask(ctx context.Context, id int64, deletedBy string) error
}

type taskService struct {
	taskRepo      repository.TaskRepository
	executionRepo repository.ExecutionRepository
	queueRepo     repository.QueueRepository
}

func NewTaskService(taskRepo repository.TaskRepository, executionRepo repository.ExecutionRepository, queueRepo repository.QueueRepository) TaskService {
	return &taskService{
		taskRepo:      taskRepo,
		executionRepo: executionRepo,
		queueRepo:     queueRepo,
	}
}

func (s *taskService) CreateTask(ctx context.Context, req dto.CreateTaskRequest) (*domain.Task, error) {
	if err := validateCreateTask(req); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	nextRunTime, err := cronutil.NextRunTime(domain.ScheduleType(req.ScheduleType), req.CronExpr, req.RunAt, now)
	if err != nil {
		return nil, ValidationError(err.Error())
	}

	task := &domain.Task{
		Name:           req.Name,
		Description:    req.Description,
		TaskType:       domain.TaskType(req.TaskType),
		ScheduleType:   domain.ScheduleType(req.ScheduleType),
		CronExpr:       req.CronExpr,
		Payload:        payload,
		TimeoutSeconds: defaultTimeout(req.TimeoutSeconds),
		RetryTimes:     defaultRetry(req.RetryTimes),
		CallbackURL:    req.CallbackURL,
		Status:         domain.TaskStatusActive,
		CreatedBy:      defaultActor(req.CreatedBy),
		UpdatedBy:      defaultActor(req.CreatedBy),
		NextRunTime:    nextRunTime,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, err
	}
	slog.Default().InfoContext(ctx, "task created", "trace_id", traceutil.FromContext(ctx), "task_id", task.ID, "task_name", task.Name)
	return task, nil
}

func (s *taskService) GetTask(ctx context.Context, id int64) (*domain.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, NotFoundError("task not found")
	}
	return task, nil
}

func (s *taskService) ListTasks(ctx context.Context, filter repository.TaskListFilter) ([]domain.Task, int64, error) {
	return s.taskRepo.List(ctx, filter)
}

func (s *taskService) UpdateTask(ctx context.Context, id int64, req dto.UpdateTaskRequest) error {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if task == nil {
		return NotFoundError("task not found")
	}
	if req.Name != nil {
		task.Name = *req.Name
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.CronExpr != nil {
		task.CronExpr = *req.CronExpr
	}
	if req.TimeoutSeconds != nil {
		task.TimeoutSeconds = defaultTimeout(*req.TimeoutSeconds)
	}
	if req.RetryTimes != nil {
		task.RetryTimes = defaultRetry(*req.RetryTimes)
	}
	if req.CallbackURL != nil {
		task.CallbackURL = *req.CallbackURL
	}
	if req.Payload != nil {
		payload, err := json.Marshal(req.Payload)
		if err != nil {
			return err
		}
		task.Payload = payload
	}
	if req.CronExpr != nil && task.ScheduleType == domain.ScheduleTypeCron {
		nextRunTime, err := cronutil.NextRunTime(task.ScheduleType, task.CronExpr, nil, time.Now().UTC())
		if err != nil {
			return ValidationError(err.Error())
		}
		task.NextRunTime = nextRunTime
	}
	task.UpdatedBy = defaultActor(req.UpdatedBy)
	task.UpdatedAt = time.Now().UTC()
	return s.taskRepo.Update(ctx, task)
}

func (s *taskService) PauseTask(ctx context.Context, id int64, updatedBy string) error {
	return s.updateTaskStatus(ctx, id, domain.TaskStatusPaused, updatedBy)
}

func (s *taskService) ResumeTask(ctx context.Context, id int64, updatedBy string) error {
	return s.updateTaskStatus(ctx, id, domain.TaskStatusActive, updatedBy)
}

func (s *taskService) TriggerTask(ctx context.Context, id int64, triggerBy string) (string, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if task == nil {
		return "", NotFoundError("task not found")
	}
	if task.Status != domain.TaskStatusActive {
		return "", ConflictError("task is not active")
	}
	executionNo := generateExecutionNo(task.ID)
	now := time.Now().UTC()
	execution := &domain.TaskExecution{
		TaskID:      task.ID,
		ExecutionNo: executionNo,
		TriggerType: domain.TriggerTypeManual,
		Status:      domain.ExecutionStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.executionRepo.Create(ctx, execution); err != nil {
		return "", err
	}
	message, err := json.Marshal(queue.TaskMessage{
		TaskID:      task.ID,
		ExecutionNo: executionNo,
		TriggerType: string(domain.TriggerTypeManual),
		TriggerBy:   defaultActor(triggerBy),
	})
	if err != nil {
		return "", err
	}
	if err := s.queueRepo.EnqueueTask(ctx, message); err != nil {
		if cleanupErr := s.executionRepo.DeleteByExecutionNo(ctx, executionNo); cleanupErr != nil {
			return "", cleanupErr
		}
		return "", err
	}
	slog.Default().InfoContext(ctx, "task triggered", "trace_id", traceutil.FromContext(ctx), "task_id", task.ID, "execution_no", executionNo)
	return executionNo, nil
}

func (s *taskService) DeleteTask(ctx context.Context, id int64, deletedBy string) error {
	if _, err := s.getTaskOrNotFound(ctx, id); err != nil {
		return err
	}
	if err := s.ensureNoActiveExecutions(ctx, id); err != nil {
		return err
	}
	return s.taskRepo.UpdateStatus(ctx, id, domain.TaskStatusDeleted, defaultActor(deletedBy))
}

func generateExecutionNo(taskID int64) string {
	return fmt.Sprintf("exec_%d_%d", taskID, time.Now().UTC().UnixNano())
}

func defaultTimeout(v int) int {
	if v <= 0 {
		return 60
	}
	return v
}

func defaultRetry(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func defaultActor(actor string) string {
	if actor == "" {
		return "system"
	}
	return actor
}

func validateCreateTask(req dto.CreateTaskRequest) error {
	if req.Name == "" {
		return ValidationError("name is required")
	}
	if !isSupportedTaskType(req.TaskType) {
		return ValidationError("task_type is required and must be shell, http, or container")
	}
	if !isSupportedScheduleType(req.ScheduleType) {
		return ValidationError("schedule_type is required and must be manual, once, or cron")
	}
	if req.Payload == nil {
		return ValidationError("payload is required")
	}
	if req.ScheduleType == string(domain.ScheduleTypeOnce) && req.RunAt == nil {
		return ValidationError("run_at is required for once schedule")
	}
	if req.ScheduleType == string(domain.ScheduleTypeCron) && req.CronExpr == "" {
		return ValidationError("cron_expr is required for cron schedule")
	}
	return nil
}

func isSupportedTaskType(taskType string) bool {
	return taskType == string(domain.TaskTypeShell) ||
		taskType == string(domain.TaskTypeHTTP) ||
		taskType == string(domain.TaskTypeContainer)
}

func isSupportedScheduleType(scheduleType string) bool {
	return scheduleType == string(domain.ScheduleTypeManual) ||
		scheduleType == string(domain.ScheduleTypeOnce) ||
		scheduleType == string(domain.ScheduleTypeCron)
}

func (s *taskService) ensureNoActiveExecutions(ctx context.Context, taskID int64) error {
	for _, status := range activeExecutionStatuses() {
		exists, err := s.executionExists(ctx, taskID, status)
		if err != nil {
			return err
		}
		if exists {
			return ConflictError("task has active executions")
		}
	}
	return nil
}

func (s *taskService) updateTaskStatus(ctx context.Context, id int64, status domain.TaskStatus, updatedBy string) error {
	if _, err := s.getTaskOrNotFound(ctx, id); err != nil {
		return err
	}
	return s.taskRepo.UpdateStatus(ctx, id, status, defaultActor(updatedBy))
}

func (s *taskService) getTaskOrNotFound(ctx context.Context, id int64) (*domain.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, NotFoundError("task not found")
	}
	return task, nil
}

func (s *taskService) executionExists(ctx context.Context, taskID int64, status string) (bool, error) {
	filter := repository.ExecutionListFilter{
		TaskID:   &taskID,
		Status:   &status,
		Page:     1,
		PageSize: 1,
	}
	items, total, err := s.executionRepo.List(ctx, filter)
	if err != nil {
		return false, err
	}
	return total > 0 || len(items) > 0, nil
}

func activeExecutionStatuses() []string {
	return []string{
		string(domain.ExecutionStatusPending),
		string(domain.ExecutionStatusRunning),
	}
}
