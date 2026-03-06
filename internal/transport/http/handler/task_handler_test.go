package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"go-cloud/internal/domain"
	"go-cloud/internal/dto"
	"go-cloud/internal/repository"
	"go-cloud/internal/service"
)

func TestTaskHandlerCreateTaskReturnsBadRequestForValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(`{"name":"bad"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler := NewTaskHandler(taskServiceHandlerStub{
		createTaskErr: service.ValidationError("task_type is required"),
	})
	handler.CreateTask(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestTaskHandlerTriggerTaskReturnsConflictForInactiveTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "3"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/3/trigger", strings.NewReader(`{"trigger_by":"qa"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler := NewTaskHandler(taskServiceHandlerStub{
		triggerTaskErr: service.ConflictError("task is not active"),
	})
	handler.TriggerTask(ctx)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", recorder.Code)
	}
}

func TestTaskHandlerPauseTaskReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "404"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/404/pause", nil)

	handler := NewTaskHandler(taskServiceHandlerStub{
		pauseTaskErr: service.NotFoundError("task not found"),
	})
	handler.PauseTask(ctx)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestTaskHandlerDeleteTaskReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "8"}}
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/8?deleted_by=qa", nil)

	handler := NewTaskHandler(taskServiceHandlerStub{})
	handler.DeleteTask(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "deleted") {
		t.Fatalf("expected deleted response, got %s", recorder.Body.String())
	}
}

func TestTaskHandlerDeleteTaskReturnsConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "8"}}
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/8", nil)

	handler := NewTaskHandler(taskServiceHandlerStub{
		deleteTaskErr: service.ConflictError("task has active executions"),
	})
	handler.DeleteTask(ctx)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", recorder.Code)
	}
}

type taskServiceHandlerStub struct {
	createTaskErr  error
	getTaskErr     error
	listTasks      []domain.Task
	listTasksTotal int64
	listTasksErr   error
	updateTaskErr  error
	pauseTaskErr   error
	resumeTaskErr  error
	triggerTaskErr error
	deleteTaskErr  error
}

func (s taskServiceHandlerStub) CreateTask(_ context.Context, _ dto.CreateTaskRequest) (*domain.Task, error) {
	if s.createTaskErr != nil {
		return nil, s.createTaskErr
	}
	return &domain.Task{ID: 1, Name: "ok"}, nil
}

func (s taskServiceHandlerStub) GetTask(_ context.Context, _ int64) (*domain.Task, error) {
	if s.getTaskErr != nil {
		return nil, s.getTaskErr
	}
	return &domain.Task{ID: 1, Name: "ok"}, nil
}

func (s taskServiceHandlerStub) ListTasks(_ context.Context, _ repository.TaskListFilter) ([]domain.Task, int64, error) {
	return s.listTasks, s.listTasksTotal, s.listTasksErr
}

func (s taskServiceHandlerStub) UpdateTask(_ context.Context, _ int64, _ dto.UpdateTaskRequest) error {
	return s.updateTaskErr
}

func (s taskServiceHandlerStub) PauseTask(_ context.Context, _ int64, _ string) error {
	return s.pauseTaskErr
}

func (s taskServiceHandlerStub) ResumeTask(_ context.Context, _ int64, _ string) error {
	return s.resumeTaskErr
}

func (s taskServiceHandlerStub) TriggerTask(_ context.Context, _ int64, _ string) (string, error) {
	if s.triggerTaskErr != nil {
		return "", s.triggerTaskErr
	}
	return "exec-1", nil
}

func (s taskServiceHandlerStub) DeleteTask(_ context.Context, _ int64, _ string) error {
	return s.deleteTaskErr
}
