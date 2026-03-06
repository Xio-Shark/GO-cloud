package model

import "time"

type TaskExecution struct {
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	TaskID       int64  `gorm:"not null;index"`
	ExecutionNo  string `gorm:"size:64;not null;uniqueIndex"`
	TriggerType  string `gorm:"size:32;not null"`
	WorkerID     string `gorm:"size:64;not null;default:''"`
	Status       string `gorm:"size:32;not null;index:idx_status_created_at,priority:1"`
	StartTime    *time.Time
	EndTime      *time.Time
	DurationMs   int64 `gorm:"not null;default:0"`
	RetryCount   int   `gorm:"not null;default:0"`
	ExitCode     *int
	ErrorMessage *string   `gorm:"type:text"`
	OutputLog    *string   `gorm:"type:mediumtext"`
	CreatedAt    time.Time `gorm:"index:idx_status_created_at,priority:2"`
	UpdatedAt    time.Time
}

func (TaskExecution) TableName() string {
	return "task_executions"
}
