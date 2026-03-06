package executor

import (
	"context"
	"encoding/json"
	"testing"

	"go-cloud/internal/domain"
)

func TestKubernetesJobExecutorSupportsContainerTask(t *testing.T) {
	exec := NewKubernetesJobExecutor(&jobRuntimeStub{}, KubernetesJobExecutorConfig{
		Namespace: "go-cloud",
	})

	if !exec.Supports(domain.TaskTypeContainer) {
		t.Fatal("expected container task support")
	}
}

func TestKubernetesJobExecutorExecuteReturnsFailureForInvalidPayload(t *testing.T) {
	exec := NewKubernetesJobExecutor(&jobRuntimeStub{}, KubernetesJobExecutorConfig{
		Namespace: "go-cloud",
	})

	result := exec.Execute(context.Background(), domain.Task{
		TaskType: domain.TaskTypeContainer,
		Payload:  []byte(`{"image":123}`),
	})
	if result.ErrMsg == nil {
		t.Fatal("expected error message")
	}
}

func TestKubernetesJobExecutorExecuteRunsKubernetesJob(t *testing.T) {
	runtime := &jobRuntimeStub{
		result: JobResult{
			ExitCode: 0,
			Logs:     "job ok\n",
		},
	}
	exec := NewKubernetesJobExecutor(runtime, KubernetesJobExecutorConfig{
		Namespace: "go-cloud",
	})

	payload, _ := json.Marshal(map[string]any{
		"image":   "busybox:1.36",
		"command": []string{"sh", "-c", "echo ok"},
		"env": map[string]string{
			"APP_ENV": "test",
		},
	})
	result := exec.Execute(context.Background(), domain.Task{
		ID:       1,
		TaskType: domain.TaskTypeContainer,
		Payload:  payload,
	})

	if result.ErrMsg != nil {
		t.Fatalf("expected nil error, got %v", *result.ErrMsg)
	}
	if runtime.request.Image != "busybox:1.36" {
		t.Fatalf("expected image busybox:1.36, got %s", runtime.request.Image)
	}
	if result.OutputLog != "job ok\n" {
		t.Fatalf("expected output log, got %s", result.OutputLog)
	}
}

type jobRuntimeStub struct {
	request JobRequest
	result  JobResult
	err     error
}

func (s *jobRuntimeStub) RunJob(_ context.Context, request JobRequest) (JobResult, error) {
	s.request = request
	return s.result, s.err
}
