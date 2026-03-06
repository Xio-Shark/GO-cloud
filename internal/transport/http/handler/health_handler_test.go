package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"go-cloud/internal/healthcheck"
)

func TestHealthHandlerHealthzReturnsOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/healthz", nil)

	handler := NewHealthHandler(healthcheck.Checker(func(context.Context) error { return nil }))
	handler.Healthz(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestHealthHandlerHealthzReturnsServiceUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/healthz", nil)

	handler := NewHealthHandler(healthcheck.Checker(func(context.Context) error {
		return errors.New("db down")
	}))
	handler.Healthz(ctx)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}
