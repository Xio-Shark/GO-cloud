package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"go-cloud/internal/domain"
	"go-cloud/internal/repository"
	"go-cloud/internal/service"
)

func TestExecutionHandlerRetryExecutionReturnsConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "execution_no", Value: "exec-1"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/executions/exec-1/retry", nil)

	handler := NewExecutionHandler(executionServiceHandlerStub{
		retryErr: service.ConflictError("execution can only retry failed states"),
	})
	handler.RetryExecution(ctx)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", recorder.Code)
	}
}

func TestExecutionHandlerListExecutionsReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/executions?status=pending&task_id=1", nil)

	handler := NewExecutionHandler(executionServiceHandlerStub{
		listAllItems: []domain.TaskExecution{
			{
				TaskID:      1,
				ExecutionNo: "exec-1",
				Status:      domain.ExecutionStatusPending,
			},
		},
		listAllTotal: 1,
	})
	handler.ListExecutions(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "exec-1") {
		t.Fatalf("expected response to contain exec-1, got %s", recorder.Body.String())
	}
}

func TestExecutionHandlerCancelExecutionReturnsConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "execution_no", Value: "exec-1"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/executions/exec-1/cancel?cancelled_by=qa", nil)

	handler := NewExecutionHandler(executionServiceHandlerStub{
		cancelErr: service.ConflictError("only pending execution can be cancelled"),
	})
	handler.CancelExecution(ctx)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", recorder.Code)
	}
}

type executionServiceHandlerStub struct {
	getExecutionValue *domain.TaskExecution
	getExecutionErr   error
	listItems         []domain.TaskExecution
	listTotal         int64
	listErr           error
	listAllItems      []domain.TaskExecution
	listAllTotal      int64
	listAllErr        error
	logsValue         *string
	logsErr           error
	retryValue        string
	retryErr          error
	cancelErr         error
}

func (s executionServiceHandlerStub) GetExecution(_ context.Context, _ string) (*domain.TaskExecution, error) {
	return s.getExecutionValue, s.getExecutionErr
}

func (s executionServiceHandlerStub) ListTaskExecutions(_ context.Context, _ int64, _ int, _ int) ([]domain.TaskExecution, int64, error) {
	return s.listItems, s.listTotal, s.listErr
}

func (s executionServiceHandlerStub) ListExecutions(_ context.Context, _ repository.ExecutionListFilter) ([]domain.TaskExecution, int64, error) {
	return s.listAllItems, s.listAllTotal, s.listAllErr
}

func (s executionServiceHandlerStub) GetExecutionLogs(_ context.Context, _ string) (*string, error) {
	return s.logsValue, s.logsErr
}

func (s executionServiceHandlerStub) RetryExecution(_ context.Context, _ string, _ string) (string, error) {
	return s.retryValue, s.retryErr
}

func (s executionServiceHandlerStub) CancelExecution(_ context.Context, _ string, _ string) error {
	return s.cancelErr
}
