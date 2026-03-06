package executor

import (
	"context"

	"go-cloud/internal/domain"
)

type Result struct {
	ExitCode  *int
	OutputLog string
	ErrMsg    *string
}

type Executor interface {
	Execute(ctx context.Context, task domain.Task) Result
	Supports(taskType domain.TaskType) bool
}
