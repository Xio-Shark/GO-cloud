package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"go-cloud/internal/domain"
	"go-cloud/internal/executor"
	"go-cloud/internal/queue"
	"go-cloud/internal/repository"
)

func TestWorkerServiceHandleOneMessageEnqueuesNotificationOnSuccess(t *testing.T) {
	taskRepo := &taskRepoStub{
		tasks: map[int64]*domain.Task{
			1: {
				ID:             1,
				Name:           "call-api",
				TaskType:       domain.TaskTypeHTTP,
				ScheduleType:   domain.ScheduleTypeManual,
				Status:         domain.TaskStatusActive,
				TimeoutSeconds: 10,
				CallbackURL:    "http://notifier.local/hook",
			},
		},
	}
	executionRepo := &executionRepoStub{
		byExecutionNo: map[string]*domain.TaskExecution{
			"exec-1": {
				TaskID:      1,
				ExecutionNo: "exec-1",
				Status:      domain.ExecutionStatusPending,
			},
		},
	}
	queueRepo := &queueRepoStub{}
	svc := NewWorkerService(taskRepo, executionRepo, queueRepo, []executor.Executor{
		executorStub{
			taskType: domain.TaskTypeHTTP,
			result: executor.Result{
				ExitCode:  intPtr(200),
				OutputLog: `{"ok":true}`,
			},
		},
	}, "worker-1")

	raw, _ := json.Marshal(queue.TaskMessage{
		TaskID:      1,
		ExecutionNo: "exec-1",
		TriggerType: string(domain.TriggerTypeManual),
	})
	if err := svc.HandleOneMessage(context.Background(), raw, "worker-1"); err != nil {
		t.Fatalf("HandleOneMessage returned error: %v", err)
	}
	if len(queueRepo.notificationPayloads) != 1 {
		t.Fatalf("expected 1 notification message, got %d", len(queueRepo.notificationPayloads))
	}
	if len(executionRepo.finished) != 1 {
		t.Fatalf("expected execution to be finished once, got %d", len(executionRepo.finished))
	}
	if executionRepo.finished[0].status != domain.ExecutionStatusSuccess {
		t.Fatalf("expected success status, got %s", executionRepo.finished[0].status)
	}
}

func TestWorkerServiceHandleOneMessageSchedulesRetryOnFailure(t *testing.T) {
	taskRepo := &taskRepoStub{
		tasks: map[int64]*domain.Task{
			2: {
				ID:             2,
				Name:           "retry-me",
				TaskType:       domain.TaskTypeShell,
				ScheduleType:   domain.ScheduleTypeManual,
				Status:         domain.TaskStatusActive,
				TimeoutSeconds: 10,
				RetryTimes:     1,
			},
		},
	}
	executionRepo := &executionRepoStub{
		byExecutionNo: map[string]*domain.TaskExecution{
			"exec-2": {
				TaskID:      2,
				ExecutionNo: "exec-2",
				Status:      domain.ExecutionStatusPending,
			},
		},
	}
	queueRepo := &queueRepoStub{}
	svc := NewWorkerService(taskRepo, executionRepo, queueRepo, []executor.Executor{
		executorStub{
			taskType: domain.TaskTypeShell,
			result: executor.Result{
				ExitCode: intPtr(1),
				ErrMsg:   strPtr("boom"),
			},
		},
	}, "worker-1")

	raw, _ := json.Marshal(queue.TaskMessage{
		TaskID:      2,
		ExecutionNo: "exec-2",
		TriggerType: string(domain.TriggerTypeManual),
	})
	if err := svc.HandleOneMessage(context.Background(), raw, "worker-1"); err != nil {
		t.Fatalf("HandleOneMessage returned error: %v", err)
	}
	if len(executionRepo.created) != 1 {
		t.Fatalf("expected retry execution to be created, got %d", len(executionRepo.created))
	}
	if executionRepo.created[0].RetryCount != 1 {
		t.Fatalf("expected retry_count 1, got %d", executionRepo.created[0].RetryCount)
	}
	if len(queueRepo.taskPayloads) != 1 {
		t.Fatalf("expected retry task to be queued, got %d", len(queueRepo.taskPayloads))
	}
	if len(queueRepo.notificationPayloads) != 0 {
		t.Fatalf("expected no notification before final retry exhaustion, got %d", len(queueRepo.notificationPayloads))
	}
}

func TestWorkerServiceHandleOneMessageSkipsCancelledExecution(t *testing.T) {
	taskRepo := &taskRepoStub{
		tasks: map[int64]*domain.Task{
			3: {
				ID:             3,
				Name:           "skip-cancelled",
				TaskType:       domain.TaskTypeShell,
				ScheduleType:   domain.ScheduleTypeManual,
				Status:         domain.TaskStatusActive,
				TimeoutSeconds: 10,
			},
		},
	}
	executionRepo := &executionRepoStub{
		byExecutionNo: map[string]*domain.TaskExecution{
			"exec-cancelled": {
				TaskID:      3,
				ExecutionNo: "exec-cancelled",
				Status:      domain.ExecutionStatusCancelled,
			},
		},
	}
	queueRepo := &queueRepoStub{}
	exec := &countingExecutorStub{
		taskType: domain.TaskTypeShell,
		result: executor.Result{
			ExitCode: intPtr(0),
		},
	}
	svc := NewWorkerService(taskRepo, executionRepo, queueRepo, []executor.Executor{exec}, "worker-1")

	raw, _ := json.Marshal(queue.TaskMessage{
		TaskID:      3,
		ExecutionNo: "exec-cancelled",
		TriggerType: string(domain.TriggerTypeManual),
	})
	if err := svc.HandleOneMessage(context.Background(), raw, "worker-1"); err != nil {
		t.Fatalf("HandleOneMessage returned error: %v", err)
	}
	if exec.called != 0 {
		t.Fatalf("expected cancelled execution to be skipped, got %d executor calls", exec.called)
	}
	if executionRepo.byExecutionNo["exec-cancelled"].Status != domain.ExecutionStatusCancelled {
		t.Fatalf("expected execution status cancelled, got %s", executionRepo.byExecutionNo["exec-cancelled"].Status)
	}
	if len(executionRepo.finished) != 0 {
		t.Fatalf("expected no finish update, got %d", len(executionRepo.finished))
	}
}

func TestWorkerServiceHandleOneMessageUsesContainerExecutor(t *testing.T) {
	taskRepo := &taskRepoStub{
		tasks: map[int64]*domain.Task{
			4: {
				ID:             4,
				Name:           "container-task",
				TaskType:       domain.TaskTypeContainer,
				ScheduleType:   domain.ScheduleTypeManual,
				Status:         domain.TaskStatusActive,
				TimeoutSeconds: 10,
			},
		},
	}
	executionRepo := &executionRepoStub{
		byExecutionNo: map[string]*domain.TaskExecution{
			"exec-container": {
				TaskID:      4,
				ExecutionNo: "exec-container",
				Status:      domain.ExecutionStatusPending,
			},
		},
	}
	queueRepo := &queueRepoStub{}
	exec := &countingExecutorStub{
		taskType: domain.TaskTypeContainer,
		result: executor.Result{
			ExitCode: intPtr(0),
		},
	}
	svc := NewWorkerService(taskRepo, executionRepo, queueRepo, []executor.Executor{exec}, "worker-1")

	raw, _ := json.Marshal(queue.TaskMessage{
		TaskID:      4,
		ExecutionNo: "exec-container",
		TriggerType: string(domain.TriggerTypeManual),
	})
	if err := svc.HandleOneMessage(context.Background(), raw, "worker-1"); err != nil {
		t.Fatalf("HandleOneMessage returned error: %v", err)
	}
	if exec.called != 1 {
		t.Fatalf("expected container executor to run once, got %d", exec.called)
	}
}

type taskRepoStub struct {
	tasks                    map[int64]*domain.Task
	dueTasks                 []domain.Task
	updatedTasks             []domain.Task
	allowMissingStatusUpdate bool
}

func (s *taskRepoStub) Create(_ context.Context, task *domain.Task) error {
	if s.tasks == nil {
		s.tasks = map[int64]*domain.Task{}
	}
	task.ID = int64(len(s.tasks) + 1)
	taskCopy := *task
	s.tasks[task.ID] = &taskCopy
	return nil
}

func (s *taskRepoStub) GetByID(_ context.Context, id int64) (*domain.Task, error) {
	if s.tasks == nil {
		return nil, nil
	}
	task, ok := s.tasks[id]
	if !ok {
		return nil, nil
	}
	taskCopy := *task
	return &taskCopy, nil
}

func (s *taskRepoStub) List(_ context.Context, _ repository.TaskListFilter) ([]domain.Task, int64, error) {
	var items []domain.Task
	for _, task := range s.tasks {
		items = append(items, *task)
	}
	return items, int64(len(items)), nil
}

func (s *taskRepoStub) Update(_ context.Context, task *domain.Task) error {
	taskCopy := *task
	s.updatedTasks = append(s.updatedTasks, taskCopy)
	if s.tasks == nil {
		s.tasks = map[int64]*domain.Task{}
	}
	s.tasks[task.ID] = &taskCopy
	return nil
}

func (s *taskRepoStub) UpdateStatus(_ context.Context, id int64, status domain.TaskStatus, _ string) error {
	if s.tasks == nil || s.tasks[id] == nil {
		if s.allowMissingStatusUpdate {
			return nil
		}
		return errors.New("task not found")
	}
	s.tasks[id].Status = status
	return nil
}

func (s *taskRepoStub) ListDueTasks(_ context.Context, _ time.Time, _ int) ([]domain.Task, error) {
	return append([]domain.Task(nil), s.dueTasks...), nil
}

type executionRepoStub struct {
	byExecutionNo map[string]*domain.TaskExecution
	created       []domain.TaskExecution
	deleted       []string
	finished      []finishCall
}

type finishCall struct {
	executionNo string
	status      domain.ExecutionStatus
}

func (s *executionRepoStub) Create(_ context.Context, execution *domain.TaskExecution) error {
	if s.byExecutionNo == nil {
		s.byExecutionNo = map[string]*domain.TaskExecution{}
	}
	executionCopy := *execution
	s.byExecutionNo[execution.ExecutionNo] = &executionCopy
	s.created = append(s.created, executionCopy)
	return nil
}

func (s *executionRepoStub) DeleteByExecutionNo(_ context.Context, executionNo string) error {
	delete(s.byExecutionNo, executionNo)
	s.deleted = append(s.deleted, executionNo)
	return nil
}

func (s *executionRepoStub) GetByExecutionNo(_ context.Context, executionNo string) (*domain.TaskExecution, error) {
	execution, ok := s.byExecutionNo[executionNo]
	if !ok {
		return nil, nil
	}
	executionCopy := *execution
	return &executionCopy, nil
}

func (s *executionRepoStub) ListByTaskID(_ context.Context, _ int64, _, _ int) ([]domain.TaskExecution, int64, error) {
	var items []domain.TaskExecution
	for _, execution := range s.byExecutionNo {
		items = append(items, *execution)
	}
	return items, int64(len(items)), nil
}

func (s *executionRepoStub) List(_ context.Context, filter repository.ExecutionListFilter) ([]domain.TaskExecution, int64, error) {
	items := make([]domain.TaskExecution, 0, len(s.byExecutionNo))
	for _, execution := range s.byExecutionNo {
		if filter.TaskID != nil && execution.TaskID != *filter.TaskID {
			continue
		}
		if filter.Status != nil && string(execution.Status) != *filter.Status {
			continue
		}
		items = append(items, *execution)
	}
	return items, int64(len(items)), nil
}

func (s *executionRepoStub) UpdateStatus(_ context.Context, executionNo string, status domain.ExecutionStatus, workerID string) error {
	execution, ok := s.byExecutionNo[executionNo]
	if !ok {
		return errors.New("execution not found")
	}
	execution.Status = status
	execution.WorkerID = workerID
	return nil
}

func (s *executionRepoStub) Finish(_ context.Context, executionNo string, status domain.ExecutionStatus, _ int64, exitCode *int, errorMessage *string, outputLog *string) error {
	execution, ok := s.byExecutionNo[executionNo]
	if !ok {
		return errors.New("execution not found")
	}
	execution.Status = status
	execution.ExitCode = exitCode
	execution.ErrorMessage = errorMessage
	execution.OutputLog = outputLog
	s.finished = append(s.finished, finishCall{executionNo: executionNo, status: status})
	return nil
}

type queueRepoStub struct {
	taskPayloads         [][]byte
	notificationPayloads [][]byte
	enqueueTaskErr       error
}

func (s *queueRepoStub) EnqueueTask(_ context.Context, payload []byte) error {
	if s.enqueueTaskErr != nil {
		return s.enqueueTaskErr
	}
	s.taskPayloads = append(s.taskPayloads, payload)
	return nil
}

func (s *queueRepoStub) DequeueTask(_ context.Context, _ time.Duration) ([]byte, error) {
	return nil, nil
}

func (s *queueRepoStub) EnqueueNotification(_ context.Context, payload []byte) error {
	s.notificationPayloads = append(s.notificationPayloads, payload)
	return nil
}

func (s *queueRepoStub) DequeueNotification(_ context.Context, _ time.Duration) ([]byte, error) {
	return nil, nil
}

type lockRepoStub struct {
	acquired bool
}

func (s *lockRepoStub) AcquireTaskDispatchLock(_ context.Context, _ int64, _ time.Duration) (bool, error) {
	return s.acquired, nil
}

type executorStub struct {
	taskType domain.TaskType
	result   executor.Result
}

func (s executorStub) Supports(taskType domain.TaskType) bool {
	return taskType == s.taskType
}

func (s executorStub) Execute(_ context.Context, _ domain.Task) executor.Result {
	return s.result
}

type countingExecutorStub struct {
	taskType domain.TaskType
	result   executor.Result
	called   int
}

func (s *countingExecutorStub) Supports(taskType domain.TaskType) bool {
	return taskType == s.taskType
}

func (s *countingExecutorStub) Execute(_ context.Context, _ domain.Task) executor.Result {
	s.called++
	return s.result
}

func intPtr(v int) *int {
	return &v
}

func strPtr(v string) *string {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}
