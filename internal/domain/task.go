package domain

import "time"

type TaskType string

const (
	TaskTypeShell     TaskType = "shell"
	TaskTypeHTTP      TaskType = "http"
	TaskTypeContainer TaskType = "container"
)

type ScheduleType string

const (
	ScheduleTypeManual ScheduleType = "manual"
	ScheduleTypeOnce   ScheduleType = "once"
	ScheduleTypeCron   ScheduleType = "cron"
)

type TaskStatus string

const (
	TaskStatusActive  TaskStatus = "active"
	TaskStatusPaused  TaskStatus = "paused"
	TaskStatusDeleted TaskStatus = "deleted"
)

type Task struct {
	ID             int64
	Name           string
	Description    string
	TaskType       TaskType
	ScheduleType   ScheduleType
	CronExpr       string
	Payload        []byte
	TimeoutSeconds int
	RetryTimes     int
	CallbackURL    string
	Status         TaskStatus
	CreatedBy      string
	UpdatedBy      string
	LastRunTime    *time.Time
	NextRunTime    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
