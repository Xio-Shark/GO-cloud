package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"go-cloud/internal/domain"
	"go-cloud/internal/metrics"
	"go-cloud/internal/queue"
	"go-cloud/internal/repository"
	"go-cloud/pkg/cronutil"
	"go-cloud/pkg/traceutil"
)

type SchedulerService interface {
	Run(ctx context.Context) error
	DispatchDueTasks(ctx context.Context, limit int) (int, error)
}

type schedulerService struct {
	taskRepo      repository.TaskRepository
	executionRepo repository.ExecutionRepository
	queueRepo     repository.QueueRepository
	lockRepo      repository.LockRepository
	scanInterval  time.Duration
	lockTTL       time.Duration
}

func NewSchedulerService(taskRepo repository.TaskRepository, executionRepo repository.ExecutionRepository, queueRepo repository.QueueRepository, lockRepo repository.LockRepository, scanInterval time.Duration) SchedulerService {
	if scanInterval <= 0 {
		scanInterval = 5 * time.Second
	}
	return &schedulerService{
		taskRepo:      taskRepo,
		executionRepo: executionRepo,
		queueRepo:     queueRepo,
		lockRepo:      lockRepo,
		scanInterval:  scanInterval,
		lockTTL:       30 * time.Second,
	}
}

func (s *schedulerService) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.scanInterval)
	defer ticker.Stop()

	for {
		runCtx := traceutil.WithTraceID(ctx, traceutil.NewTraceID())
		if _, err := s.DispatchDueTasks(runCtx, 100); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *schedulerService) DispatchDueTasks(ctx context.Context, limit int) (int, error) {
	now := time.Now().UTC()
	tasks, err := s.taskRepo.ListDueTasks(ctx, now, limit)
	if err != nil {
		return 0, err
	}
	metrics.SchedulerScansTotal.Inc()
	metrics.SchedulerDueTasks.Set(float64(len(tasks)))

	dispatched := 0
	for _, task := range tasks {
		acquired, err := s.lockRepo.AcquireTaskDispatchLock(ctx, task.ID, s.lockTTL)
		if err != nil || !acquired {
			if err != nil {
				return dispatched, err
			}
			metrics.SchedulerDuplicateBlockedTotal.Inc()
			continue
		}
		executionNo := generateExecutionNo(task.ID)
		execution := &domain.TaskExecution{
			TaskID:      task.ID,
			ExecutionNo: executionNo,
			TriggerType: domain.TriggerTypeSchedule,
			Status:      domain.ExecutionStatusPending,
			RetryCount:  0,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := s.executionRepo.Create(ctx, execution); err != nil {
			return dispatched, err
		}
		payload, err := json.Marshal(queue.TaskMessage{
			TaskID:      task.ID,
			ExecutionNo: executionNo,
			TriggerType: string(domain.TriggerTypeSchedule),
			TriggerBy:   "scheduler",
		})
		if err != nil {
			return dispatched, err
		}
		if err := s.queueRepo.EnqueueTask(ctx, payload); err != nil {
			if cleanupErr := s.executionRepo.DeleteByExecutionNo(ctx, executionNo); cleanupErr != nil {
				return dispatched, cleanupErr
			}
			return dispatched, err
		}
		task.LastRunTime = &now
		nextRunTime, err := cronutil.NextRunTime(task.ScheduleType, task.CronExpr, nil, now)
		if err != nil {
			return dispatched, err
		}
		task.NextRunTime = nextRunTime
		task.UpdatedAt = now
		if err := s.taskRepo.Update(ctx, &task); err != nil {
			return dispatched, err
		}
		slog.Default().InfoContext(ctx, "scheduler dispatched task", "trace_id", traceutil.FromContext(ctx), "task_id", task.ID, "execution_no", executionNo)
		metrics.SchedulerDispatchedTasksTotal.Inc()
		dispatched++
	}
	return dispatched, nil
}
