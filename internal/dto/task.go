package dto

import "time"

type CreateTaskRequest struct {
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	TaskType       string         `json:"task_type"`
	ScheduleType   string         `json:"schedule_type"`
	CronExpr       string         `json:"cron_expr"`
	Payload        map[string]any `json:"payload"`
	TimeoutSeconds int            `json:"timeout_seconds"`
	RetryTimes     int            `json:"retry_times"`
	CallbackURL    string         `json:"callback_url"`
	CreatedBy      string         `json:"created_by"`
	RunAt          *time.Time     `json:"run_at"`
}

type UpdateTaskRequest struct {
	Name           *string        `json:"name"`
	Description    *string        `json:"description"`
	CronExpr       *string        `json:"cron_expr"`
	Payload        map[string]any `json:"payload"`
	TimeoutSeconds *int           `json:"timeout_seconds"`
	RetryTimes     *int           `json:"retry_times"`
	CallbackURL    *string        `json:"callback_url"`
	UpdatedBy      string         `json:"updated_by"`
}

type TriggerTaskRequest struct {
	TriggerBy string `json:"trigger_by"`
}
