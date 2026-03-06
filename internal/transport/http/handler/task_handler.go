package handler

import (
	"encoding/json"
	"errors"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"

	"go-cloud/internal/domain"
	"go-cloud/internal/dto"
	"go-cloud/internal/repository"
	"go-cloud/internal/service"
	"go-cloud/internal/transport/http/response"
)

type TaskHandler struct {
	taskSvc service.TaskService
}

func NewTaskHandler(taskSvc service.TaskService) *TaskHandler {
	return &TaskHandler{taskSvc: taskSvc}
}

func (h *TaskHandler) CreateTask(ctx *gin.Context) {
	request := dto.CreateTaskRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.BadRequest(ctx, "invalid request body")
		return
	}
	task, err := h.taskSvc.CreateTask(ctx.Request.Context(), request)
	if err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}
	response.Success(ctx, taskToResponse(*task))
}

func (h *TaskHandler) GetTask(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(ctx, "invalid task id")
		return
	}
	task, err := h.taskSvc.GetTask(ctx.Request.Context(), id)
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	response.Success(ctx, taskToResponse(*task))
}

func (h *TaskHandler) ListTasks(ctx *gin.Context) {
	page := parseIntOrDefault(ctx.Query("page"), 1)
	pageSize := parseIntOrDefault(ctx.Query("page_size"), 20)
	filter := repository.TaskListFilter{
		Page:     page,
		PageSize: pageSize,
	}
	if value := ctx.Query("status"); value != "" {
		filter.Status = &value
	}
	if value := ctx.Query("task_type"); value != "" {
		filter.TaskType = &value
	}
	tasks, total, err := h.taskSvc.ListTasks(ctx.Request.Context(), filter)
	if err != nil {
		response.InternalError(ctx, err.Error())
		return
	}
	items := make([]gin.H, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, taskToResponse(task))
	}
	response.Success(ctx, gin.H{
		"list":      items,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

func (h *TaskHandler) UpdateTask(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(ctx, "invalid task id")
		return
	}
	request := dto.UpdateTaskRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.BadRequest(ctx, "invalid request body")
		return
	}
	if err := h.taskSvc.UpdateTask(ctx.Request.Context(), id, request); err != nil {
		writeServiceError(ctx, err)
		return
	}
	response.Success(ctx, gin.H{"updated": true, "task_id": id})
}

func (h *TaskHandler) PauseTask(ctx *gin.Context) {
	h.updateTaskStatus(ctx, domain.TaskStatusPaused)
}

func (h *TaskHandler) ResumeTask(ctx *gin.Context) {
	h.updateTaskStatus(ctx, domain.TaskStatusActive)
}

func (h *TaskHandler) TriggerTask(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(ctx, "invalid task id")
		return
	}
	request := dto.TriggerTaskRequest{}
	if err := ctx.ShouldBindJSON(&request); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(ctx, "invalid request body")
		return
	}
	executionNo, err := h.taskSvc.TriggerTask(ctx.Request.Context(), id, request.TriggerBy)
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	response.Success(ctx, gin.H{"task_id": id, "execution_no": executionNo})
}

func (h *TaskHandler) DeleteTask(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(ctx, "invalid task id")
		return
	}
	if err := h.taskSvc.DeleteTask(ctx.Request.Context(), id, ctx.Query("deleted_by")); err != nil {
		writeServiceError(ctx, err)
		return
	}
	response.Success(ctx, gin.H{"task_id": id, "status": domain.TaskStatusDeleted})
}

func (h *TaskHandler) updateTaskStatus(ctx *gin.Context, status domain.TaskStatus) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(ctx, "invalid task id")
		return
	}
	actor := ctx.Query("updated_by")
	if status == domain.TaskStatusPaused {
		err = h.taskSvc.PauseTask(ctx.Request.Context(), id, actor)
	} else {
		err = h.taskSvc.ResumeTask(ctx.Request.Context(), id, actor)
	}
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	response.Success(ctx, gin.H{"task_id": id, "status": status})
}

func taskToResponse(task domain.Task) gin.H {
	return gin.H{
		"id":              task.ID,
		"name":            task.Name,
		"description":     task.Description,
		"task_type":       task.TaskType,
		"schedule_type":   task.ScheduleType,
		"cron_expr":       task.CronExpr,
		"payload":         json.RawMessage(task.Payload),
		"timeout_seconds": task.TimeoutSeconds,
		"retry_times":     task.RetryTimes,
		"callback_url":    task.CallbackURL,
		"status":          task.Status,
		"created_by":      task.CreatedBy,
		"updated_by":      task.UpdatedBy,
		"last_run_time":   task.LastRunTime,
		"next_run_time":   task.NextRunTime,
		"created_at":      task.CreatedAt,
		"updated_at":      task.UpdatedAt,
	}
}

func writeServiceError(ctx *gin.Context, err error) {
	if service.HasErrorCode(err, service.ErrorCodeValidation) {
		response.BadRequest(ctx, err.Error())
		return
	}
	if service.HasErrorCode(err, service.ErrorCodeNotFound) {
		response.NotFound(ctx, err.Error())
		return
	}
	if service.HasErrorCode(err, service.ErrorCodeConflict) {
		response.Conflict(ctx, err.Error())
		return
	}
	response.InternalError(ctx, err.Error())
}

func parseIntOrDefault(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
