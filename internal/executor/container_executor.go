package executor

import (
	"context"
	"encoding/json"
	"fmt"

	"go-cloud/internal/domain"
)

type JobRequest struct {
	TaskID         int64
	ExecutionNo    string
	Namespace      string
	Image          string
	Command        []string
	Env            map[string]string
	TimeoutSeconds int
}

type JobResult struct {
	ExitCode int
	Logs     string
}

type JobRunner interface {
	RunJob(ctx context.Context, request JobRequest) (JobResult, error)
}

type KubernetesJobExecutorConfig struct {
	Namespace string
}

type KubernetesJobExecutor struct {
	runner JobRunner
	config KubernetesJobExecutorConfig
}

type containerPayload struct {
	Image   string            `json:"image"`
	Command []string          `json:"command"`
	Env     map[string]string `json:"env"`
}

func NewKubernetesJobExecutor(runner JobRunner, config KubernetesJobExecutorConfig) *KubernetesJobExecutor {
	return &KubernetesJobExecutor{
		runner: runner,
		config: config,
	}
}

func (e *KubernetesJobExecutor) Supports(taskType domain.TaskType) bool {
	return taskType == domain.TaskTypeContainer
}

func (e *KubernetesJobExecutor) Execute(ctx context.Context, task domain.Task) Result {
	payload := containerPayload{}
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return failedResult(-1, err.Error(), "")
	}
	if payload.Image == "" {
		return failedResult(-1, "container image is required", "")
	}
	if len(payload.Command) == 0 {
		return failedResult(-1, "container command is required", "")
	}
	if e.runner == nil {
		return failedResult(-1, "kubernetes job runner is not configured", "")
	}

	jobResult, err := e.runner.RunJob(ctx, JobRequest{
		TaskID:         task.ID,
		Namespace:      e.config.Namespace,
		Image:          payload.Image,
		Command:        payload.Command,
		Env:            payload.Env,
		TimeoutSeconds: task.TimeoutSeconds,
	})
	if err != nil {
		return failedResult(1, err.Error(), jobResult.Logs)
	}

	exitCode := jobResult.ExitCode
	if exitCode != 0 {
		return failedResult(exitCode, fmt.Sprintf("kubernetes job failed with exit code %d", exitCode), jobResult.Logs)
	}
	return Result{
		ExitCode:  &exitCode,
		OutputLog: jobResult.Logs,
	}
}
