package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"go-cloud/internal/domain"
	"go-cloud/internal/repository"
	"go-cloud/internal/service"
	"go-cloud/internal/transport/http/response"
)

type ExecutionHandler struct {
	executionSvc service.ExecutionService
}

func NewExecutionHandler(executionSvc service.ExecutionService) *ExecutionHandler {
	return &ExecutionHandler{executionSvc: executionSvc}
}

func (h *ExecutionHandler) GetExecution(ctx *gin.Context) {
	execution, err := h.executionSvc.GetExecution(ctx.Request.Context(), ctx.Param("execution_no"))
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	response.Success(ctx, executionToResponse(*execution))
}

func (h *ExecutionHandler) GetExecutionLogs(ctx *gin.Context) {
	logs, err := h.executionSvc.GetExecutionLogs(ctx.Request.Context(), ctx.Param("execution_no"))
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	response.Success(ctx, gin.H{
		"execution_no": ctx.Param("execution_no"),
		"output_log":   logs,
	})
}

func (h *ExecutionHandler) ListTaskExecutions(ctx *gin.Context) {
	taskID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(ctx, "invalid task id")
		return
	}
	page := parseIntOrDefault(ctx.Query("page"), 1)
	pageSize := parseIntOrDefault(ctx.Query("page_size"), 20)
	items, total, err := h.executionSvc.ListTaskExecutions(ctx.Request.Context(), taskID, page, pageSize)
	if err != nil {
		response.InternalError(ctx, err.Error())
		return
	}
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, executionToResponse(item))
	}
	response.Success(ctx, gin.H{
		"list":      list,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

func (h *ExecutionHandler) ListExecutions(ctx *gin.Context) {
	page := parseIntOrDefault(ctx.Query("page"), 1)
	pageSize := parseIntOrDefault(ctx.Query("page_size"), 20)
	filter := repository.ExecutionListFilter{
		Page:     page,
		PageSize: pageSize,
	}
	if value := ctx.Query("status"); value != "" {
		filter.Status = &value
	}
	if value := ctx.Query("task_id"); value != "" {
		taskID, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			response.BadRequest(ctx, "invalid task_id")
			return
		}
		filter.TaskID = &taskID
	}
	items, total, err := h.executionSvc.ListExecutions(ctx.Request.Context(), filter)
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, executionToResponse(item))
	}
	response.Success(ctx, gin.H{
		"list":      list,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

func (h *ExecutionHandler) RetryExecution(ctx *gin.Context) {
	executionNo, err := h.executionSvc.RetryExecution(ctx.Request.Context(), ctx.Param("execution_no"), ctx.Query("trigger_by"))
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	response.Success(ctx, gin.H{"execution_no": executionNo})
}

func (h *ExecutionHandler) CancelExecution(ctx *gin.Context) {
	if err := h.executionSvc.CancelExecution(ctx.Request.Context(), ctx.Param("execution_no"), ctx.Query("cancelled_by")); err != nil {
		writeServiceError(ctx, err)
		return
	}
	response.Success(ctx, gin.H{"execution_no": ctx.Param("execution_no"), "status": domain.ExecutionStatusCancelled})
}

func executionToResponse(execution domain.TaskExecution) gin.H {
	return gin.H{
		"id":            execution.ID,
		"task_id":       execution.TaskID,
		"execution_no":  execution.ExecutionNo,
		"trigger_type":  execution.TriggerType,
		"worker_id":     execution.WorkerID,
		"status":        execution.Status,
		"retry_count":   execution.RetryCount,
		"start_time":    execution.StartTime,
		"end_time":      execution.EndTime,
		"duration_ms":   execution.DurationMs,
		"exit_code":     execution.ExitCode,
		"error_message": execution.ErrorMessage,
		"output_log":    execution.OutputLog,
		"created_at":    execution.CreatedAt,
		"updated_at":    execution.UpdatedAt,
	}
}
