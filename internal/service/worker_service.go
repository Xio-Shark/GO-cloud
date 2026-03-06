package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"go-cloud/internal/domain"
	"go-cloud/internal/executor"
	"go-cloud/internal/metrics"
	"go-cloud/internal/queue"
	"go-cloud/internal/repository"
	"go-cloud/pkg/traceutil"
)

type WorkerService interface {
	ConsumeLoop(ctx context.Context) error
	HandleOneMessage(ctx context.Context, raw []byte, workerID string) error
}

type workerService struct {
	taskRepo      repository.TaskRepository
	executionRepo repository.ExecutionRepository
	queueRepo     repository.QueueRepository
	executors     []executor.Executor
	workerID      string
	pollTimeout   time.Duration
}

func NewWorkerService(taskRepo repository.TaskRepository, executionRepo repository.ExecutionRepository, queueRepo repository.QueueRepository, executors []executor.Executor, workerID string) WorkerService {
	if workerID == "" {
		workerID = "worker-default"
	}
	return &workerService{
		taskRepo:      taskRepo,
		executionRepo: executionRepo,
		queueRepo:     queueRepo,
		executors:     executors,
		workerID:      workerID,
		pollTimeout:   5 * time.Second,
	}
}

func (s *workerService) ConsumeLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		raw, err := s.queueRepo.DequeueTask(ctx, s.pollTimeout)
		if err != nil {
			return err
		}
		if len(raw) == 0 {
			continue
		}
		messageCtx := traceutil.WithTraceID(ctx, traceutil.NewTraceID())
		metrics.WorkerConsumedTasksTotal.Inc()
		if err := s.HandleOneMessage(messageCtx, raw, s.workerID); err != nil {
			return err
		}
	}
}

func (s *workerService) HandleOneMessage(ctx context.Context, raw []byte, workerID string) error {
	var message queue.TaskMessage
	if err := json.Unmarshal(raw, &message); err != nil {
		return err
	}
	execution, err := s.executionRepo.GetByExecutionNo(ctx, message.ExecutionNo)
	if err != nil {
		return err
	}
	if execution == nil {
		return errors.New("execution not found")
	}
	if execution.Status == domain.ExecutionStatusCancelled {
		return nil
	}
	task, err := s.taskRepo.GetByID(ctx, message.TaskID)
	if err != nil {
		return err
	}
	if task == nil {
		return errors.New("task not found")
	}
	if task.Status != domain.TaskStatusActive {
		return errors.New("task is not active")
	}
	if err := s.executionRepo.UpdateStatus(ctx, message.ExecutionNo, domain.ExecutionStatusRunning, workerID); err != nil {
		return err
	}
	metrics.WorkerRunningTasksGauge.Inc()
	defer metrics.WorkerRunningTasksGauge.Dec()

	exec := s.pickExecutor(task.TaskType)
	if exec == nil {
		errMsg := "unsupported task type"
		exitCode := -1
		if err := s.executionRepo.Finish(ctx, message.ExecutionNo, domain.ExecutionStatusFailed, 0, &exitCode, &errMsg, nil); err != nil {
			return err
		}
		return errors.New(errMsg)
	}

	timeout := time.Duration(defaultTimeout(task.TimeoutSeconds)) * time.Second
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now().UTC()
	result := exec.Execute(execCtx, *task)
	durationMs := time.Since(start).Milliseconds()
	status := domain.ExecutionStatusSuccess
	if execCtx.Err() == context.DeadlineExceeded {
		status = domain.ExecutionStatusTimeout
		timeoutMessage := "execution timeout"
		result.ErrMsg = &timeoutMessage
	}
	if result.ErrMsg != nil && status != domain.ExecutionStatusTimeout {
		status = domain.ExecutionStatusFailed
	}
	output := result.OutputLog
	if err := s.executionRepo.Finish(ctx, message.ExecutionNo, status, durationMs, result.ExitCode, result.ErrMsg, &output); err != nil {
		return err
	}
	metrics.WorkerExecutionDurationMs.Observe(float64(durationMs))
	if status == domain.ExecutionStatusFailed || status == domain.ExecutionStatusTimeout {
		if message.RetryCount < task.RetryTimes {
			if err := s.enqueueRetry(ctx, *task, message); err != nil {
				return err
			}
			metrics.WorkerRetriedTasksTotal.Inc()
			slog.Default().WarnContext(ctx, "worker scheduled retry", "trace_id", traceutil.FromContext(ctx), "task_id", task.ID, "execution_no", message.ExecutionNo, "retry_count", message.RetryCount+1)
			return nil
		}
		metrics.WorkerFailedTasksTotal.Inc()
	}
	if status == domain.ExecutionStatusSuccess {
		metrics.WorkerSucceededTasksTotal.Inc()
	}
	if task.CallbackURL != "" {
		if err := s.enqueueNotification(ctx, *task, message, status, result, durationMs, workerID); err != nil {
			return err
		}
	}
	slog.Default().InfoContext(ctx, "worker finished execution", "trace_id", traceutil.FromContext(ctx), "task_id", task.ID, "execution_no", message.ExecutionNo, "status", status)
	return nil
}

func (s *workerService) enqueueRetry(ctx context.Context, task domain.Task, message queue.TaskMessage) error {
	retryCount := message.RetryCount + 1
	now := time.Now().UTC()
	retryExecutionNo := generateExecutionNo(task.ID)
	execution := &domain.TaskExecution{
		TaskID:      task.ID,
		ExecutionNo: retryExecutionNo,
		TriggerType: domain.TriggerTypeRetry,
		Status:      domain.ExecutionStatusPending,
		RetryCount:  retryCount,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.executionRepo.Create(ctx, execution); err != nil {
		return err
	}
	payload, err := json.Marshal(queue.TaskMessage{
		TaskID:      task.ID,
		ExecutionNo: retryExecutionNo,
		TriggerType: string(domain.TriggerTypeRetry),
		TriggerBy:   "worker-retry",
		RetryCount:  retryCount,
	})
	if err != nil {
		return err
	}
	return s.queueRepo.EnqueueTask(ctx, payload)
}

func (s *workerService) enqueueNotification(ctx context.Context, task domain.Task, message queue.TaskMessage, status domain.ExecutionStatus, result executor.Result, durationMs int64, workerID string) error {
	payload, err := json.Marshal(queue.NotificationMessage{
		TaskID:        task.ID,
		TaskName:      task.Name,
		ExecutionNo:   message.ExecutionNo,
		CallbackURL:   task.CallbackURL,
		Status:        string(status),
		OutputLog:     stringPtr(result.OutputLog),
		ErrorMessage:  result.ErrMsg,
		TraceID:       traceutil.FromContext(ctx),
		TriggeredBy:   message.TriggerBy,
		RetryCount:    message.RetryCount,
		WorkerID:      workerID,
		ExecutionTime: durationMs,
	})
	if err != nil {
		return err
	}
	return s.queueRepo.EnqueueNotification(ctx, payload)
}

func (s *workerService) pickExecutor(taskType domain.TaskType) executor.Executor {
	for _, exec := range s.executors {
		if exec.Supports(taskType) {
			return exec
		}
	}
	return nil
}

func stringPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
