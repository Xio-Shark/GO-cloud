package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"go-cloud/internal/domain"
	"go-cloud/internal/dto"
	"go-cloud/internal/repository"
	"go-cloud/internal/service"
)

func TestReleaseHandlerCreateReleaseReturnsBadRequestForValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/releases", strings.NewReader(`{"version":"v1.0.0"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler := NewReleaseHandler(releaseServiceHandlerStub{
		createErr: service.ValidationError("app_name is required"),
	})
	handler.CreateRelease(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestReleaseHandlerListReleasesReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/releases?environment=dev", nil)

	handler := NewReleaseHandler(releaseServiceHandlerStub{
		listItems: []domain.ReleaseRecord{
			{
				ID:          1,
				AppName:     "api-server",
				Version:     "v1.0.0",
				Environment: "dev",
				Status:      "pending",
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			},
		},
		listTotal: 1,
	})
	handler.ListReleases(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "api-server") {
		t.Fatalf("expected response to contain api-server, got %s", recorder.Body.String())
	}
}

func TestReleaseHandlerGetReleaseReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "99"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/releases/99", nil)

	handler := NewReleaseHandler(releaseServiceHandlerStub{
		getErr: service.NotFoundError("release not found"),
	})
	handler.GetRelease(ctx)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestReleaseHandlerRollbackReleaseReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "1"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/releases/1/rollback", strings.NewReader(`{"operator":"qa"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler := NewReleaseHandler(releaseServiceHandlerStub{
		rollbackValue: &domain.ReleaseRecord{
			ID:          2,
			AppName:     "api-server",
			Version:     "v1.0.0",
			Environment: "dev",
			Status:      domain.ReleaseStatusRolledBack,
		},
	})
	handler.RollbackRelease(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "rolled_back") {
		t.Fatalf("expected rolled_back response, got %s", recorder.Body.String())
	}
}

type releaseServiceHandlerStub struct {
	createValue   *domain.ReleaseRecord
	createErr     error
	getValue      *domain.ReleaseRecord
	getErr        error
	listItems     []domain.ReleaseRecord
	listTotal     int64
	listErr       error
	rollbackValue *domain.ReleaseRecord
	rollbackErr   error
}

func (s releaseServiceHandlerStub) CreateRelease(_ context.Context, _ dto.CreateReleaseRequest) (*domain.ReleaseRecord, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	if s.createValue != nil {
		return s.createValue, nil
	}
	return &domain.ReleaseRecord{ID: 1, AppName: "api-server", Version: "v1.0.0", Environment: "dev", Status: "pending"}, nil
}

func (s releaseServiceHandlerStub) GetRelease(_ context.Context, _ int64) (*domain.ReleaseRecord, error) {
	return s.getValue, s.getErr
}

func (s releaseServiceHandlerStub) ListReleases(_ context.Context, _ repository.ReleaseListFilter) ([]domain.ReleaseRecord, int64, error) {
	return s.listItems, s.listTotal, s.listErr
}

func (s releaseServiceHandlerStub) RollbackRelease(_ context.Context, _ int64, _ dto.RollbackReleaseRequest) (*domain.ReleaseRecord, error) {
	return s.rollbackValue, s.rollbackErr
}
