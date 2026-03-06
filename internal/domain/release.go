package domain

import "time"

const (
	ReleaseStatusPending    = "pending"
	ReleaseStatusDeployed   = "deployed"
	ReleaseStatusFailed     = "failed"
	ReleaseStatusRolledBack = "rolled_back"
)

type ReleaseRecord struct {
	ID          int64
	AppName     string
	Version     string
	Environment string
	Status      string
	Operator    string
	ChangeLog   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
