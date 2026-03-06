package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go-cloud/internal/domain"
	"go-cloud/internal/repository"
)

func TestExecutionServiceRetryExecutionRejectsSuccessfulExecution(t *testing.T) {
	taskRepo := &taskRepoStub{
		tasks: map[int64]*domain.Task{
			1: {
				ID:     1,
				Status: domain.TaskStatusActive,
			},
		},
	}
	executionRepo := &executionRepoStub{
		byExecutionNo: map[string]*domain.TaskExecution{
			"exec-success": {
				TaskID:      1,
				ExecutionNo: "exec-success",
				Status:      domain.ExecutionStatusSuccess,
				RetryCount:  0,
				TriggerType: domain.TriggerTypeManual,
			},
		},
	}
	svc := NewExecutionService(taskRepo, executionRepo, &queueRepoStub{})

	_, err := svc.RetryExecution(context.Background(), "exec-success", "tester")
	if err == nil {
		t.Fatal("expected retry rejection for success execution")
	}
	if !strings.Contains(err.Error(), "can only retry failed") {
		t.Fatalf("expected retry validation error, got %v", err)
	}
}

func TestExecutionServiceRetryExecutionRejectsPausedTask(t *testing.T) {
	taskRepo := &taskRepoStub{
		tasks: map[int64]*domain.Task{
			2: {
				ID:     2,
				Status: domain.TaskStatusPaused,
			},
		},
	}
	executionRepo := &executionRepoStub{
		byExecutionNo: map[string]*domain.TaskExecution{
			"exec-failed": {
				TaskID:      2,
				ExecutionNo: "exec-failed",
				Status:      domain.ExecutionStatusFailed,
				RetryCount:  0,
				TriggerType: domain.TriggerTypeManual,
			},
		},
	}
	svc := NewExecutionService(taskRepo, executionRepo, &queueRepoStub{})

	_, err := svc.RetryExecution(context.Background(), "exec-failed", "tester")
	if err == nil {
		t.Fatal("expected retry rejection for paused task")
	}
	if !strings.Contains(err.Error(), "task is not active") {
		t.Fatalf("expected task state error, got %v", err)
	}
}

func TestExecutionServiceRetryExecutionDeletesExecutionWhenEnqueueFails(t *testing.T) {
	taskRepo := &taskRepoStub{
		tasks: map[int64]*domain.Task{
			3: {
				ID:     3,
				Status: domain.TaskStatusActive,
			},
		},
	}
	executionRepo := &executionRepoStub{
		byExecutionNo: map[string]*domain.TaskExecution{
			"exec-failed": {
				TaskID:      3,
				ExecutionNo: "exec-failed",
				Status:      domain.ExecutionStatusFailed,
			},
		},
	}
	queueRepo := &queueRepoStub{enqueueTaskErr: errors.New("queue unavailable")}
	svc := NewExecutionService(taskRepo, executionRepo, queueRepo)

	_, err := svc.RetryExecution(context.Background(), "exec-failed", "tester")
	if err == nil {
		t.Fatal("expected enqueue error")
	}
	if len(executionRepo.deleted) != 1 {
		t.Fatalf("expected retry execution to be deleted, got %d deletions", len(executionRepo.deleted))
	}
}

func TestExecutionServiceListExecutionsSupportsTaskAndStatusFilter(t *testing.T) {
	executionRepo := &executionRepoStub{
		byExecutionNo: map[string]*domain.TaskExecution{
			"exec-1": {
				TaskID:      11,
				ExecutionNo: "exec-1",
				Status:      domain.ExecutionStatusPending,
			},
			"exec-2": {
				TaskID:      12,
				ExecutionNo: "exec-2",
				Status:      domain.ExecutionStatusSuccess,
			},
		},
	}
	svc := NewExecutionService(&taskRepoStub{}, executionRepo, &queueRepoStub{})
	status := string(domain.ExecutionStatusPending)

	items, total, err := svc.ListExecutions(context.Background(), repository.ExecutionListFilter{
		TaskID:   int64Ptr(11),
		Status:   &status,
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("ListExecutions returned error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(items) != 1 || items[0].ExecutionNo != "exec-1" {
		t.Fatalf("expected only exec-1, got %+v", items)
	}
}

func TestExecutionServiceCancelExecutionRejectsNonPendingExecution(t *testing.T) {
	executionRepo := &executionRepoStub{
		byExecutionNo: map[string]*domain.TaskExecution{
			"exec-running": {
				TaskID:      15,
				ExecutionNo: "exec-running",
				Status:      domain.ExecutionStatusRunning,
			},
		},
	}
	svc := NewExecutionService(&taskRepoStub{}, executionRepo, &queueRepoStub{})

	err := svc.CancelExecution(context.Background(), "exec-running", "tester")
	if err == nil {
		t.Fatal("expected conflict error for running execution")
	}
	if !strings.Contains(err.Error(), "only pending execution can be cancelled") {
		t.Fatalf("expected pending-only validation, got %v", err)
	}
}

func TestExecutionServiceCancelExecutionMarksPendingExecutionCancelled(t *testing.T) {
	executionRepo := &executionRepoStub{
		byExecutionNo: map[string]*domain.TaskExecution{
			"exec-pending": {
				TaskID:      16,
				ExecutionNo: "exec-pending",
				Status:      domain.ExecutionStatusPending,
			},
		},
	}
	svc := NewExecutionService(&taskRepoStub{}, executionRepo, &queueRepoStub{})

	err := svc.CancelExecution(context.Background(), "exec-pending", "tester")
	if err != nil {
		t.Fatalf("CancelExecution returned error: %v", err)
	}
	if executionRepo.byExecutionNo["exec-pending"].Status != domain.ExecutionStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", executionRepo.byExecutionNo["exec-pending"].Status)
	}
}
