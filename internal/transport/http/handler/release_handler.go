package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"go-cloud/internal/domain"
	"go-cloud/internal/dto"
	"go-cloud/internal/repository"
	"go-cloud/internal/service"
	"go-cloud/internal/transport/http/response"
)

type ReleaseHandler struct {
	releaseSvc service.ReleaseService
}

func NewReleaseHandler(releaseSvc service.ReleaseService) *ReleaseHandler {
	return &ReleaseHandler{releaseSvc: releaseSvc}
}

func (h *ReleaseHandler) CreateRelease(ctx *gin.Context) {
	request := dto.CreateReleaseRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.BadRequest(ctx, "invalid request body")
		return
	}
	release, err := h.releaseSvc.CreateRelease(ctx.Request.Context(), request)
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	response.Success(ctx, releaseToResponse(*release))
}

func (h *ReleaseHandler) GetRelease(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(ctx, "invalid release id")
		return
	}
	release, err := h.releaseSvc.GetRelease(ctx.Request.Context(), id)
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	response.Success(ctx, releaseToResponse(*release))
}

func (h *ReleaseHandler) ListReleases(ctx *gin.Context) {
	page := parseIntOrDefault(ctx.Query("page"), 1)
	pageSize := parseIntOrDefault(ctx.Query("page_size"), 20)
	filter := repository.ReleaseListFilter{
		Page:     page,
		PageSize: pageSize,
	}
	if value := ctx.Query("environment"); value != "" {
		filter.Environment = &value
	}
	if value := ctx.Query("status"); value != "" {
		filter.Status = &value
	}
	items, total, err := h.releaseSvc.ListReleases(ctx.Request.Context(), filter)
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, releaseToResponse(item))
	}
	response.Success(ctx, gin.H{
		"list":      list,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

func (h *ReleaseHandler) RollbackRelease(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(ctx, "invalid release id")
		return
	}
	request := dto.RollbackReleaseRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.BadRequest(ctx, "invalid request body")
		return
	}
	release, err := h.releaseSvc.RollbackRelease(ctx.Request.Context(), id, request)
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	response.Success(ctx, releaseToResponse(*release))
}

func releaseToResponse(release domain.ReleaseRecord) gin.H {
	return gin.H{
		"id":          release.ID,
		"app_name":    release.AppName,
		"version":     release.Version,
		"environment": release.Environment,
		"status":      release.Status,
		"operator":    release.Operator,
		"change_log":  release.ChangeLog,
		"created_at":  release.CreatedAt,
		"updated_at":  release.UpdatedAt,
	}
}
