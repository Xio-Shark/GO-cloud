package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"go-cloud/internal/domain"
)

type HTTPExecutor struct {
	client *http.Client
}

type httpPayload struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func NewHTTPExecutor(timeout time.Duration) *HTTPExecutor {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &HTTPExecutor{client: &http.Client{Timeout: timeout}}
}

func (e *HTTPExecutor) Supports(taskType domain.TaskType) bool {
	return taskType == domain.TaskTypeHTTP
}

func (e *HTTPExecutor) Execute(ctx context.Context, task domain.Task) Result {
	payload := httpPayload{}
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return failedResult(-1, err.Error(), "")
	}
	if payload.Method == "" {
		payload.Method = http.MethodGet
	}
	request, err := http.NewRequestWithContext(ctx, payload.Method, payload.URL, bytes.NewBufferString(payload.Body))
	if err != nil {
		return failedResult(-1, err.Error(), "")
	}
	for key, value := range payload.Headers {
		request.Header.Set(key, value)
	}
	response, err := e.client.Do(request)
	if err != nil {
		return failedResult(1, err.Error(), "")
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode >= http.StatusBadRequest {
		return failedResult(response.StatusCode, response.Status, string(body))
	}
	code := response.StatusCode
	return Result{ExitCode: &code, OutputLog: string(body)}
}
