package domain

import "time"

type TriggerType string

const (
	TriggerTypeManual   TriggerType = "manual"
	TriggerTypeSchedule TriggerType = "schedule"
	TriggerTypeRetry    TriggerType = "retry"
)

type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusSuccess   ExecutionStatus = "success"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusTimeout   ExecutionStatus = "timeout"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

type TaskExecution struct {
	ID           int64
	TaskID       int64
	ExecutionNo  string
	TriggerType  TriggerType
	WorkerID     string
	Status       ExecutionStatus
	RetryCount   int
	StartTime    *time.Time
	EndTime      *time.Time
	DurationMs   int64
	ExitCode     *int
	ErrorMessage *string
	OutputLog    *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
