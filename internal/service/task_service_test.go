package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go-cloud/internal/domain"
	"go-cloud/internal/dto"
)

func TestTaskServiceCreateTaskInitializesNextRunTimeForCronSchedule(t *testing.T) {
	taskRepo := &taskRepoStub{}
	svc := NewTaskService(taskRepo, &executionRepoStub{}, &queueRepoStub{})

	task, err := svc.CreateTask(context.Background(), dto.CreateTaskRequest{
		Name:         "nightly-report",
		Description:  "run every five minutes",
		TaskType:     string(domain.TaskTypeShell),
		ScheduleType: string(domain.ScheduleTypeCron),
		CronExpr:     "*/5 * * * *",
		Payload: map[string]any{
			"command": "echo ok",
		},
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if task == nil {
		t.Fatal("CreateTask returned nil task")
	}
	if task.NextRunTime == nil {
		t.Fatal("expected next_run_time to be initialized")
	}
	if task.Status != domain.TaskStatusActive {
		t.Fatalf("expected status active, got %s", task.Status)
	}
	if !task.NextRunTime.After(task.CreatedAt) && !task.NextRunTime.Equal(task.CreatedAt) {
		t.Fatalf("expected next_run_time >= created_at, got next=%v created=%v", task.NextRunTime, task.CreatedAt)
	}
}

func TestTaskServiceCreateTaskRejectsMissingTaskType(t *testing.T) {
	svc := NewTaskService(&taskRepoStub{}, &executionRepoStub{}, &queueRepoStub{})

	_, err := svc.CreateTask(context.Background(), dto.CreateTaskRequest{
		Name:         "invalid-task-type",
		ScheduleType: string(domain.ScheduleTypeManual),
		Payload: map[string]any{
			"command": "echo invalid",
		},
	})
	if err == nil {
		t.Fatal("expected error for missing task_type")
	}
	if !strings.Contains(err.Error(), "task_type is required") {
		t.Fatalf("expected task_type validation error, got %v", err)
	}
}

func TestTaskServiceCreateTaskRejectsOnceScheduleWithoutRunAt(t *testing.T) {
	svc := NewTaskService(&taskRepoStub{}, &executionRepoStub{}, &queueRepoStub{})

	_, err := svc.CreateTask(context.Background(), dto.CreateTaskRequest{
		Name:         "once-without-run-at",
		TaskType:     string(domain.TaskTypeShell),
		ScheduleType: string(domain.ScheduleTypeOnce),
		Payload: map[string]any{
			"command": "echo once",
		},
	})
	if err == nil {
		t.Fatal("expected error for missing run_at")
	}
	if !strings.Contains(err.Error(), "run_at is required") {
		t.Fatalf("expected run_at validation error, got %v", err)
	}
}

func TestTaskServicePauseTaskReturnsNotFoundWhenTaskMissing(t *testing.T) {
	svc := NewTaskService(&taskRepoStub{allowMissingStatusUpdate: true}, &executionRepoStub{}, &queueRepoStub{})

	err := svc.PauseTask(context.Background(), 404, "tester")
	if err == nil {
		t.Fatal("expected error for missing task")
	}
	if !strings.Contains(err.Error(), "task not found") {
		t.Fatalf("expected task not found error, got %v", err)
	}
}

func TestTaskServiceTriggerTaskDeletesExecutionWhenEnqueueFails(t *testing.T) {
	taskRepo := &taskRepoStub{
		tasks: map[int64]*domain.Task{
			9: {
				ID:           9,
				Name:         "manual-task",
				TaskType:     domain.TaskTypeShell,
				ScheduleType: domain.ScheduleTypeManual,
				Status:       domain.TaskStatusActive,
			},
		},
	}
	executionRepo := &executionRepoStub{}
	queueRepo := &queueRepoStub{enqueueTaskErr: errors.New("queue unavailable")}
	svc := NewTaskService(taskRepo, executionRepo, queueRepo)

	_, err := svc.TriggerTask(context.Background(), 9, "tester")
	if err == nil {
		t.Fatal("expected enqueue error")
	}
	if len(executionRepo.deleted) != 1 {
		t.Fatalf("expected created execution to be deleted, got %d deletions", len(executionRepo.deleted))
	}
}

func TestTaskServiceCreateTaskAllowsContainerType(t *testing.T) {
	svc := NewTaskService(&taskRepoStub{}, &executionRepoStub{}, &queueRepoStub{})

	task, err := svc.CreateTask(context.Background(), dto.CreateTaskRequest{
		Name:         "container-task",
		TaskType:     string(domain.TaskTypeContainer),
		ScheduleType: string(domain.ScheduleTypeManual),
		Payload: map[string]any{
			"image":   "busybox:1.36",
			"command": []string{"sh", "-c", "echo ok"},
		},
	})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if task.TaskType != domain.TaskTypeContainer {
		t.Fatalf("expected container task type, got %s", task.TaskType)
	}
}

func TestTaskServiceDeleteTaskRejectsActiveExecutions(t *testing.T) {
	taskRepo := &taskRepoStub{
		tasks: map[int64]*domain.Task{
			7: {
				ID:     7,
				Name:   "cleanup-me",
				Status: domain.TaskStatusActive,
			},
		},
	}
	executionRepo := &executionRepoStub{
		byExecutionNo: map[string]*domain.TaskExecution{
			"exec-pending": {
				TaskID:      7,
				ExecutionNo: "exec-pending",
				Status:      domain.ExecutionStatusPending,
			},
		},
	}
	svc := NewTaskService(taskRepo, executionRepo, &queueRepoStub{})

	err := svc.DeleteTask(context.Background(), 7, "tester")
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "active executions") {
		t.Fatalf("expected active executions conflict, got %v", err)
	}
}

func TestTaskServiceDeleteTaskMarksTaskDeleted(t *testing.T) {
	taskRepo := &taskRepoStub{
		tasks: map[int64]*domain.Task{
			8: {
				ID:     8,
				Name:   "cleanup-me",
				Status: domain.TaskStatusActive,
			},
		},
	}
	svc := NewTaskService(taskRepo, &executionRepoStub{}, &queueRepoStub{})

	if err := svc.DeleteTask(context.Background(), 8, "tester"); err != nil {
		t.Fatalf("DeleteTask returned error: %v", err)
	}
	if taskRepo.tasks[8].Status != domain.TaskStatusDeleted {
		t.Fatalf("expected deleted status, got %s", taskRepo.tasks[8].Status)
	}
}
