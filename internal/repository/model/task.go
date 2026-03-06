package model

import "time"

type Task struct {
	ID             int64  `gorm:"primaryKey;autoIncrement"`
	Name           string `gorm:"size:128;not null"`
	Description    string `gorm:"size:255;not null;default:''"`
	TaskType       string `gorm:"size:32;not null"`
	ScheduleType   string `gorm:"size:32;not null"`
	CronExpr       string `gorm:"size:64"`
	Payload        []byte `gorm:"type:json"`
	TimeoutSeconds int    `gorm:"not null;default:60"`
	RetryTimes     int    `gorm:"not null;default:0"`
	Status         string `gorm:"size:32;not null;index:idx_status_next_run,priority:1"`
	CallbackURL    string `gorm:"size:255"`
	CreatedBy      string `gorm:"size:64;not null;default:''"`
	UpdatedBy      string `gorm:"size:64;not null;default:''"`
	LastRunTime    *time.Time
	NextRunTime    *time.Time `gorm:"index:idx_status_next_run,priority:2"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (Task) TableName() string {
	return "tasks"
}
