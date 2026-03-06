package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"go-cloud/internal/domain"
	"go-cloud/internal/queue"
)

func TestSchedulerServiceDispatchDueTasksCreatesExecutionAndAdvancesNextRunTime(t *testing.T) {
	now := time.Now().Add(-1 * time.Minute)
	taskRepo := &taskRepoStub{
		dueTasks: []domain.Task{
			{
				ID:           7,
				Name:         "dispatch-me",
				TaskType:     domain.TaskTypeShell,
				ScheduleType: domain.ScheduleTypeCron,
				CronExpr:     "*/5 * * * *",
				Status:       domain.TaskStatusActive,
				NextRunTime:  &now,
			},
		},
	}
	executionRepo := &executionRepoStub{}
	queueRepo := &queueRepoStub{}
	lockRepo := &lockRepoStub{acquired: true}
	svc := NewSchedulerService(taskRepo, executionRepo, queueRepo, lockRepo, time.Second)

	dispatched, err := svc.DispatchDueTasks(context.Background(), 10)
	if err != nil {
		t.Fatalf("DispatchDueTasks returned error: %v", err)
	}
	if dispatched != 1 {
		t.Fatalf("expected 1 dispatched task, got %d", dispatched)
	}
	if len(executionRepo.created) != 1 {
		t.Fatalf("expected 1 execution created, got %d", len(executionRepo.created))
	}
	if len(queueRepo.taskPayloads) != 1 {
		t.Fatalf("expected 1 queued task message, got %d", len(queueRepo.taskPayloads))
	}
	if len(taskRepo.updatedTasks) != 1 {
		t.Fatalf("expected task next_run_time to be updated once, got %d", len(taskRepo.updatedTasks))
	}

	var msg queue.TaskMessage
	if err := json.Unmarshal(queueRepo.taskPayloads[0], &msg); err != nil {
		t.Fatalf("unmarshal queued task message: %v", err)
	}
	if msg.TaskID != 7 {
		t.Fatalf("expected queued task_id 7, got %d", msg.TaskID)
	}
	if msg.TriggerType != string(domain.TriggerTypeSchedule) {
		t.Fatalf("expected trigger_type schedule, got %s", msg.TriggerType)
	}
	if taskRepo.updatedTasks[0].NextRunTime == nil || !taskRepo.updatedTasks[0].NextRunTime.After(now) {
		t.Fatalf("expected next_run_time to move forward, got %v", taskRepo.updatedTasks[0].NextRunTime)
	}
}

func TestSchedulerServiceDispatchDueTasksDeletesExecutionWhenEnqueueFails(t *testing.T) {
	now := time.Now().Add(-1 * time.Minute)
	taskRepo := &taskRepoStub{
		dueTasks: []domain.Task{
			{
				ID:           8,
				Name:         "dispatch-fail",
				TaskType:     domain.TaskTypeShell,
				ScheduleType: domain.ScheduleTypeCron,
				CronExpr:     "*/5 * * * *",
				Status:       domain.TaskStatusActive,
				NextRunTime:  &now,
			},
		},
	}
	executionRepo := &executionRepoStub{}
	queueRepo := &queueRepoStub{enqueueTaskErr: errors.New("queue unavailable")}
	lockRepo := &lockRepoStub{acquired: true}
	svc := NewSchedulerService(taskRepo, executionRepo, queueRepo, lockRepo, time.Second)

	_, err := svc.DispatchDueTasks(context.Background(), 10)
	if err == nil {
		t.Fatal("expected enqueue error")
	}
	if len(executionRepo.deleted) != 1 {
		t.Fatalf("expected created execution to be deleted, got %d deletions", len(executionRepo.deleted))
	}
}
