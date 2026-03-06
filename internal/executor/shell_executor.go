package executor

import (
	"context"
	"encoding/json"
	"os/exec"
	"runtime"
	"strings"

	"go-cloud/internal/domain"
)

type ShellExecutor struct{}

type shellPayload struct {
	Command string            `json:"command"`
	Workdir string            `json:"workdir"`
	Env     map[string]string `json:"env"`
}

func NewShellExecutor() *ShellExecutor {
	return &ShellExecutor{}
}

func (e *ShellExecutor) Supports(taskType domain.TaskType) bool {
	return taskType == domain.TaskTypeShell
}

func (e *ShellExecutor) Execute(ctx context.Context, task domain.Task) Result {
	payload := shellPayload{}
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return failedResult(-1, err.Error(), "")
	}
	if payload.Command == "" {
		return failedResult(-1, "shell command is empty", "")
	}
	command, args := shellCommand(payload.Command)
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = payload.Workdir
	cmd.Env = append(cmd.Environ(), flattenEnv(payload.Env)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return failedResult(1, err.Error(), string(output))
	}
	code := 0
	return Result{ExitCode: &code, OutputLog: string(output)}
}

func shellCommand(raw string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", raw}
	}
	return "sh", []string{"-c", raw}
}

func flattenEnv(source map[string]string) []string {
	items := make([]string, 0, len(source))
	for key, value := range source {
		items = append(items, key+"="+value)
	}
	return items
}

func failedResult(code int, message string, output string) Result {
	result := Result{OutputLog: strings.TrimSpace(output)}
	result.ExitCode = &code
	result.ErrMsg = &message
	return result
}
