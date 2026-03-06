package queue

type TaskMessage struct {
	TaskID      int64  `json:"task_id"`
	ExecutionNo string `json:"execution_no"`
	TriggerType string `json:"trigger_type"`
	TriggerBy   string `json:"trigger_by"`
	RetryCount  int    `json:"retry_count"`
}

type NotificationMessage struct {
	TaskID        int64   `json:"task_id"`
	TaskName      string  `json:"task_name"`
	ExecutionNo   string  `json:"execution_no"`
	CallbackURL   string  `json:"callback_url"`
	Status        string  `json:"status"`
	OutputLog     *string `json:"output_log,omitempty"`
	ErrorMessage  *string `json:"error_message,omitempty"`
	TraceID       string  `json:"trace_id"`
	TriggeredBy   string  `json:"triggered_by"`
	RetryCount    int     `json:"retry_count"`
	WorkerID      string  `json:"worker_id"`
	ExecutionTime int64   `json:"execution_time_ms"`
}
