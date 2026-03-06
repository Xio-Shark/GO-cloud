package model

import "time"

type ReleaseRecord struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	AppName     string `gorm:"size:64;not null"`
	Version     string `gorm:"size:128;not null"`
	Environment string `gorm:"size:32;not null"`
	Status      string `gorm:"size:32;not null"`
	Operator    string `gorm:"size:64;not null;default:''"`
	ChangeLog   string `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (ReleaseRecord) TableName() string {
	return "release_records"
}
