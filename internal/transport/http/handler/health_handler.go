package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go-cloud/internal/healthcheck"
)

type HealthHandler struct {
	checker healthcheck.Checker
}

func NewHealthHandler(checker healthcheck.Checker) *HealthHandler {
	return &HealthHandler{checker: checker}
}

func (h *HealthHandler) Healthz(ctx *gin.Context) {
	if h.checker != nil {
		if err := h.checker(ctx.Request.Context()); err != nil {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": err.Error()})
			return
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *HealthHandler) Readyz(ctx *gin.Context) {
	if h.checker != nil {
		if err := h.checker(ctx.Request.Context()); err != nil {
			ctx.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": err.Error()})
			return
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}
