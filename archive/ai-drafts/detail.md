可以。下面这版我按**“借成熟项目思路，但不照搬过度设计”**来给你。

我建议你这套项目的骨架主要缝合这几类成熟思路：

* `golang-standards/project-layout` 的 `cmd/`、`internal/`、`pkg/` 分层，用来区分入口、私有业务代码和可复用组件。([GitHub][1])
* `go-clean-template` 的 handler / usecase(service) / repo 分层，核心业务不要直接依赖 Gin、MySQL 细节。([GitHub][2])
* `go-zero` 那种把服务入口、内部模型、任务/消费者拆开的工程化思路，但不直接引入整套框架。([go-zero][3])
* Uber Fx 的“消灭全局变量、用依赖注入管理生命周期”的思想，但你这个项目初版更适合**手写依赖装配**，先不用把 Fx 真加进来。([Go Packages][4])

## 1. 最终建议的代码目录

这版是“够工程化、但还不难维护”的平衡方案。

```text
go-job-platform/
├── cmd/
│   ├── api-server/
│   │   └── main.go
│   ├── scheduler/
│   │   └── main.go
│   ├── worker/
│   │   └── main.go
│   └── notifier/
│       └── main.go
│
├── internal/
│   ├── bootstrap/              # 依赖装配、配置加载、服务启动
│   │   ├── app.go
│   │   ├── config.go
│   │   ├── logger.go
│   │   ├── mysql.go
│   │   ├── redis.go
│   │   └── http.go
│   │
│   ├── domain/                 # 领域模型
│   │   ├── task.go
│   │   ├── execution.go
│   │   ├── release.go
│   │   └── errors.go
│   │
│   ├── dto/                    # 请求/响应 DTO
│   │   ├── task_dto.go
│   │   ├── execution_dto.go
│   │   └── release_dto.go
│   │
│   ├── repository/             # 仓储接口 + 实现
│   │   ├── interfaces.go
│   │   ├── mysql/
│   │   │   ├── task_repo.go
│   │   │   ├── execution_repo.go
│   │   │   └── release_repo.go
│   │   └── redis/
│   │       ├── queue_repo.go
│   │       ├── lock_repo.go
│   │       └── cache_repo.go
│   │
│   ├── service/                # 业务编排层
│   │   ├── task_service.go
│   │   ├── execution_service.go
│   │   ├── scheduler_service.go
│   │   ├── worker_service.go
│   │   ├── release_service.go
│   │   └── notifier_service.go
│   │
│   ├── executor/               # 任务执行器
│   │   ├── executor.go
│   │   ├── shell_executor.go
│   │   ├── http_executor.go
│   │   └── container_executor.go
│   │
│   ├── queue/                  # 队列消息结构
│   │   └── task_message.go
│   │
│   ├── transport/              # 对外接口层
│   │   └── http/
│   │       ├── handler/
│   │       │   ├── task_handler.go
│   │       │   ├── execution_handler.go
│   │       │   ├── release_handler.go
│   │       │   └── health_handler.go
│   │       ├── middleware/
│   │       │   ├── recover.go
│   │       │   ├── requestid.go
│   │       │   └── logger.go
│   │       └── router/
│   │           └── router.go
│   │
│   ├── metrics/
│   │   ├── http_metrics.go
│   │   ├── scheduler_metrics.go
│   │   └── worker_metrics.go
│   │
│   └── pkgx/                   # 项目内部通用小工具，别滥放
│       ├── response/
│       │   └── response.go
│       ├── pagination/
│       │   └── pagination.go
│       └── timeutil/
│           └── timeutil.go
│
├── pkg/                        # 真正准备复用到其他仓库的包，没有就先留空
│
├── deployments/
├── docs/
├── scripts/
├── go.mod
└── Makefile
```

### 为什么这么分

这版最核心的原则是：

* `cmd/` 只放**进程入口**。
* `internal/domain` 只放**业务对象和规则**。
* `service` 只编排业务，不直接写 SQL。
* `repository` 负责数据访问。
* `transport/http/handler` 只做参数绑定和调用 service。
* `executor` 把任务执行方式抽象出来，后面扩 shell/http/container 比较自然。

这套思路是从 `project-layout` 的目录约束，加上 clean architecture 常见的“interface adapter + usecase + repository”裁剪出来的。([GitHub][1])

---

## 2. 依赖关系怎么控制

你项目里最好严格遵守这个依赖方向：

```text
handler -> service -> repository interface -> repository implementation
                         |
                         -> executor interface / queue interface
```

也就是：

* handler 不能直接调 MySQL
* service 不要依赖 Gin 的 `Context`
* domain 不要 import GORM
* repository/mysql 才能碰数据库细节

这和 clean template 把业务逻辑从框架与存储细节中分离的思路一致。([GitHub][2])

---

## 3. 每个模块的接口设计

下面是能直接开工的接口版本。

# 3.1 domain

## `internal/domain/task.go`

```go
package domain

import "time"

type TaskType string
type ScheduleType string
type TaskStatus string

const (
	TaskTypeShell     TaskType = "shell"
	TaskTypeHTTP      TaskType = "http"
	TaskTypeContainer TaskType = "container"

	ScheduleTypeManual ScheduleType = "manual"
	ScheduleTypeOnce   ScheduleType = "once"
	ScheduleTypeCron   ScheduleType = "cron"

	TaskStatusActive  TaskStatus = "active"
	TaskStatusPaused  TaskStatus = "paused"
	TaskStatusDeleted TaskStatus = "deleted"
)

type Task struct {
	ID             int64
	Name           string
	Description    string
	TaskType       TaskType
	ScheduleType   ScheduleType
	CronExpr       *string
	Payload        []byte
	TimeoutSeconds int
	RetryTimes     int
	Status         TaskStatus
	CallbackURL    *string
	CreatedBy      string
	UpdatedBy      string
	NextRunTime    *time.Time
	LastRunTime    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
```

## `internal/domain/execution.go`

```go
package domain

import "time"

type ExecutionStatus string
type TriggerType string

const (
	TriggerTypeManual    TriggerType = "manual"
	TriggerTypeScheduler TriggerType = "scheduler"
	TriggerTypeRetry     TriggerType = "retry"

	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusSuccess   ExecutionStatus = "success"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusTimeout   ExecutionStatus = "timeout"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

type TaskExecution struct {
	ID           int64
	TaskID       int64
	ExecutionNo  string
	TriggerType  TriggerType
	WorkerID     string
	Status       ExecutionStatus
	StartTime    *time.Time
	EndTime      *time.Time
	DurationMs   int64
	RetryCount   int
	ExitCode     *int
	ErrorMessage *string
	OutputLog    *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
```

---

# 3.2 DTO

## `internal/dto/task_dto.go`

```go
package dto

type CreateTaskRequest struct {
	Name           string         `json:"name" binding:"required"`
	Description    string         `json:"description"`
	TaskType       string         `json:"task_type" binding:"required,oneof=shell http container"`
	ScheduleType   string         `json:"schedule_type" binding:"required,oneof=manual once cron"`
	CronExpr       *string        `json:"cron_expr"`
	Payload        map[string]any `json:"payload" binding:"required"`
	TimeoutSeconds int            `json:"timeout_seconds"`
	RetryTimes     int            `json:"retry_times"`
	CallbackURL    *string        `json:"callback_url"`
	CreatedBy      string         `json:"created_by"`
}

type UpdateTaskRequest struct {
	Description    *string        `json:"description"`
	CronExpr       *string        `json:"cron_expr"`
	Payload        map[string]any `json:"payload"`
	TimeoutSeconds *int           `json:"timeout_seconds"`
	RetryTimes     *int           `json:"retry_times"`
	UpdatedBy      string         `json:"updated_by"`
}

type TriggerTaskRequest struct {
	TriggerBy string `json:"trigger_by"`
}
```

---

# 3.3 repository interfaces

## `internal/repository/interfaces.go`

```go
package repository

import (
	"context"
	"time"

	"go-job-platform/internal/domain"
)

type TaskFilter struct {
	Status   *domain.TaskStatus
	TaskType *domain.TaskType
	Page     int
	PageSize int
}

type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, id int64) (*domain.Task, error)
	List(ctx context.Context, filter TaskFilter) ([]domain.Task, int64, error)
	Update(ctx context.Context, task *domain.Task) error
	UpdateStatus(ctx context.Context, id int64, status domain.TaskStatus, updatedBy string) error
	ListDueTasks(ctx context.Context, now time.Time, limit int) ([]domain.Task, error)
	UpdateNextRunTime(ctx context.Context, id int64, lastRun, nextRun *time.Time) error
}

type ExecutionRepository interface {
	Create(ctx context.Context, execution *domain.TaskExecution) error
	GetByExecutionNo(ctx context.Context, executionNo string) (*domain.TaskExecution, error)
	ListByTaskID(ctx context.Context, taskID int64, page, pageSize int) ([]domain.TaskExecution, int64, error)
	UpdateStatus(ctx context.Context, executionNo string, status domain.ExecutionStatus, workerID string) error
	Finish(ctx context.Context, executionNo string, status domain.ExecutionStatus, durationMs int64, exitCode *int, errMsg *string, outputLog *string) error
}

type ReleaseRepository interface {
	Create(ctx context.Context, record *domain.ReleaseRecord) error
	List(ctx context.Context, env string, app string, page, pageSize int) ([]domain.ReleaseRecord, int64, error)
}

type QueueRepository interface {
	EnqueueTask(ctx context.Context, msg []byte) error
	DequeueTask(ctx context.Context, timeout time.Duration) ([]byte, error)
	EnqueueRetryTask(ctx context.Context, msg []byte) error
}

type LockRepository interface {
	Acquire(ctx context.Context, key string, value string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, key string, value string) error
}

type CacheRepository interface {
	SetExecutionStatus(ctx context.Context, executionNo string, status string, ttl time.Duration) error
	GetExecutionStatus(ctx context.Context, executionNo string) (string, error)
}
```

---

# 3.4 service interfaces

我建议 service 也显式定义接口，方便测试和替换。

## `internal/service/task_service.go`

```go
package service

import (
	"context"

	"go-job-platform/internal/domain"
	"go-job-platform/internal/dto"
)

type TaskService interface {
	CreateTask(ctx context.Context, req dto.CreateTaskRequest) (*domain.Task, error)
	GetTask(ctx context.Context, id int64) (*domain.Task, error)
	ListTasks(ctx context.Context, req ListTasksRequest) ([]domain.Task, int64, error)
	UpdateTask(ctx context.Context, id int64, req dto.UpdateTaskRequest) error
	PauseTask(ctx context.Context, id int64, updatedBy string) error
	ResumeTask(ctx context.Context, id int64, updatedBy string) error
	TriggerTask(ctx context.Context, id int64, triggerBy string) (string, error)
}

type ListTasksRequest struct {
	Status   *string
	TaskType *string
	Page     int
	PageSize int
}
```

## `internal/service/execution_service.go`

```go
package service

import (
	"context"

	"go-job-platform/internal/domain"
)

type ExecutionService interface {
	GetExecution(ctx context.Context, executionNo string) (*domain.TaskExecution, error)
	ListExecutionsByTask(ctx context.Context, taskID int64, page, pageSize int) ([]domain.TaskExecution, int64, error)
	GetExecutionLogs(ctx context.Context, executionNo string) (string, error)
	RetryExecution(ctx context.Context, executionNo string, operator string) (string, error)
}
```

## `internal/service/scheduler_service.go`

```go
package service

import "context"

type SchedulerService interface {
	DispatchDueTasks(ctx context.Context, limit int) (int, error)
}
```

## `internal/service/worker_service.go`

```go
package service

import "context"

type WorkerService interface {
	ConsumeLoop(ctx context.Context) error
	HandleOneMessage(ctx context.Context, raw []byte, workerID string) error
}
```

---

# 3.5 executor interface

## `internal/executor/executor.go`

```go
package executor

import (
	"context"

	"go-job-platform/internal/domain"
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
```

## `internal/executor/shell_executor.go`

```go
package executor

import (
	"context"
	"encoding/json"
	"os/exec"

	"go-job-platform/internal/domain"
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
	var p shellPayload
	if err := json.Unmarshal(task.Payload, &p); err != nil {
		msg := err.Error()
		code := -1
		return Result{ExitCode: &code, ErrMsg: &msg}
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", p.Command)
	if p.Workdir != "" {
		cmd.Dir = p.Workdir
	}

	if len(p.Env) > 0 {
		env := cmd.Environ()
		for k, v := range p.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}

	out, err := cmd.CombinedOutput()
	log := string(out)

	if err != nil {
		msg := err.Error()
		code := 1
		return Result{ExitCode: &code, OutputLog: log, ErrMsg: &msg}
	}

	code := 0
	return Result{ExitCode: &code, OutputLog: log}
}
```

---

## 4. handler / service / repository 代码骨架

下面给你最关键的一条链路：
**创建任务 + 手动触发任务**。

# 4.1 handler

## `internal/transport/http/handler/task_handler.go`

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go-job-platform/internal/dto"
	"go-job-platform/internal/service"
	"go-job-platform/internal/pkgx/response"
)

type TaskHandler struct {
	taskSvc service.TaskService
}

func NewTaskHandler(taskSvc service.TaskService) *TaskHandler {
	return &TaskHandler{taskSvc: taskSvc}
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	task, err := h.taskSvc.CreateTask(c.Request.Context(), req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"id":            task.ID,
		"name":          task.Name,
		"status":        task.Status,
		"next_run_time": task.NextRunTime,
		"created_at":    task.CreatedAt,
	})
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid task id")
		return
	}

	task, err := h.taskSvc.GetTask(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, task)
}

func (h *TaskHandler) TriggerTask(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid task id")
		return
	}

	var req dto.TriggerTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	executionNo, err := h.taskSvc.TriggerTask(c.Request.Context(), id, req.TriggerBy)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"task_id":      id,
			"execution_no": executionNo,
			"status":       "pending",
		},
	})
}
```

---

# 4.2 service 实现

## `internal/service/task_service_impl.go`

```go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"go-job-platform/internal/domain"
	"go-job-platform/internal/dto"
	"go-job-platform/internal/queue"
	"go-job-platform/internal/repository"
)

type taskService struct {
	taskRepo      repository.TaskRepository
	executionRepo repository.ExecutionRepository
	queueRepo     repository.QueueRepository
}

func NewTaskService(
	taskRepo repository.TaskRepository,
	executionRepo repository.ExecutionRepository,
	queueRepo repository.QueueRepository,
) TaskService {
	return &taskService{
		taskRepo:      taskRepo,
		executionRepo: executionRepo,
		queueRepo:     queueRepo,
	}
}

func (s *taskService) CreateTask(ctx context.Context, req dto.CreateTaskRequest) (*domain.Task, error) {
	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, err
	}

	task := &domain.Task{
		Name:           req.Name,
		Description:    req.Description,
		TaskType:       domain.TaskType(req.TaskType),
		ScheduleType:   domain.ScheduleType(req.ScheduleType),
		CronExpr:       req.CronExpr,
		Payload:        payloadBytes,
		TimeoutSeconds: max(req.TimeoutSeconds, 60),
		RetryTimes:     req.RetryTimes,
		Status:         domain.TaskStatusActive,
		CreatedBy:      req.CreatedBy,
		UpdatedBy:      req.CreatedBy,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if req.CallbackURL != nil {
		task.CallbackURL = req.CallbackURL
	}

	// 这里后续可以加 cron 计算 next_run_time 的逻辑
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *taskService) GetTask(ctx context.Context, id int64) (*domain.Task, error) {
	return s.taskRepo.GetByID(ctx, id)
}

func (s *taskService) ListTasks(ctx context.Context, req ListTasksRequest) ([]domain.Task, int64, error) {
	filter := repository.TaskFilter{
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	if req.Status != nil {
		status := domain.TaskStatus(*req.Status)
		filter.Status = &status
	}
	if req.TaskType != nil {
		tt := domain.TaskType(*req.TaskType)
		filter.TaskType = &tt
	}

	return s.taskRepo.List(ctx, filter)
}

func (s *taskService) UpdateTask(ctx context.Context, id int64, req dto.UpdateTaskRequest) error {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if task == nil {
		return errors.New("task not found")
	}

	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.CronExpr != nil {
		task.CronExpr = req.CronExpr
	}
	if req.TimeoutSeconds != nil {
		task.TimeoutSeconds = *req.TimeoutSeconds
	}
	if req.RetryTimes != nil {
		task.RetryTimes = *req.RetryTimes
	}
	if req.UpdatedBy != "" {
		task.UpdatedBy = req.UpdatedBy
	}
	if req.Payload != nil {
		payloadBytes, err := json.Marshal(req.Payload)
		if err != nil {
			return err
		}
		task.Payload = payloadBytes
	}

	task.UpdatedAt = time.Now()
	return s.taskRepo.Update(ctx, task)
}

func (s *taskService) PauseTask(ctx context.Context, id int64, updatedBy string) error {
	return s.taskRepo.UpdateStatus(ctx, id, domain.TaskStatusPaused, updatedBy)
}

func (s *taskService) ResumeTask(ctx context.Context, id int64, updatedBy string) error {
	return s.taskRepo.UpdateStatus(ctx, id, domain.TaskStatusActive, updatedBy)
}

func (s *taskService) TriggerTask(ctx context.Context, id int64, triggerBy string) (string, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if task == nil {
		return "", errors.New("task not found")
	}
	if task.Status != domain.TaskStatusActive {
		return "", errors.New("task is not active")
	}

	executionNo := fmt.Sprintf("exec_%d_%s", time.Now().Unix(), uuid.NewString()[:8])

	execution := &domain.TaskExecution{
		TaskID:      task.ID,
		ExecutionNo: executionNo,
		TriggerType: domain.TriggerTypeManual,
		Status:      domain.ExecutionStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.executionRepo.Create(ctx, execution); err != nil {
		return "", err
	}

	msg := queue.TaskMessage{
		TaskID:      task.ID,
		ExecutionNo: executionNo,
		TriggerType: string(domain.TriggerTypeManual),
		TriggerBy:   triggerBy,
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	if err := s.queueRepo.EnqueueTask(ctx, raw); err != nil {
		return "", err
	}

	return executionNo, nil
}

func max(v, defaultValue int) int {
	if v <= 0 {
		return defaultValue
	}
	return v
}
```

---

# 4.3 repository/mysql 骨架

你问的是“最好缝成熟模块”。这里我的建议是：

* ORM 层可以用 **GORM**
* 但 **repository 输出 domain 对象**
* 不要让 service 直接依赖 GORM model

这是比较常见、也比较稳的折中方案。

## `internal/repository/mysql/task_repo.go`

```go
package mysql

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"go-job-platform/internal/domain"
	"go-job-platform/internal/repository"
)

type taskModel struct {
	ID             int64      `gorm:"column:id;primaryKey;autoIncrement"`
	Name           string     `gorm:"column:name"`
	Description    string     `gorm:"column:description"`
	TaskType       string     `gorm:"column:task_type"`
	ScheduleType   string     `gorm:"column:schedule_type"`
	CronExpr       *string    `gorm:"column:cron_expr"`
	Payload        []byte     `gorm:"column:payload"`
	TimeoutSeconds int        `gorm:"column:timeout_seconds"`
	RetryTimes     int        `gorm:"column:retry_times"`
	Status         string     `gorm:"column:status"`
	CallbackURL    *string    `gorm:"column:callback_url"`
	CreatedBy      string     `gorm:"column:created_by"`
	UpdatedBy      string     `gorm:"column:updated_by"`
	NextRunTime    *time.Time `gorm:"column:next_run_time"`
	LastRunTime    *time.Time `gorm:"column:last_run_time"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

func (taskModel) TableName() string { return "tasks" }

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) repository.TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, task *domain.Task) error {
	m := toTaskModel(task)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return err
	}
	task.ID = m.ID
	return nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	var m taskModel
	err := r.db.WithContext(ctx).First(&m, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d := toTaskDomain(m)
	return &d, nil
}

func (r *TaskRepository) List(ctx context.Context, filter repository.TaskFilter) ([]domain.Task, int64, error) {
	var (
		models []taskModel
		total  int64
	)

	db := r.db.WithContext(ctx).Model(&taskModel{})

	if filter.Status != nil {
		db = db.Where("status = ?", string(*filter.Status))
	}
	if filter.TaskType != nil {
		db = db.Where("task_type = ?", string(*filter.TaskType))
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 10
	}

	offset := (filter.Page - 1) * filter.PageSize
	if err := db.Order("id desc").Offset(offset).Limit(filter.PageSize).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	out := make([]domain.Task, 0, len(models))
	for _, m := range models {
		out = append(out, toTaskDomain(m))
	}
	return out, total, nil
}

func (r *TaskRepository) Update(ctx context.Context, task *domain.Task) error {
	return r.db.WithContext(ctx).
		Model(&taskModel{}).
		Where("id = ?", task.ID).
		Updates(toTaskModel(task)).Error
}

func (r *TaskRepository) UpdateStatus(ctx context.Context, id int64, status domain.TaskStatus, updatedBy string) error {
	return r.db.WithContext(ctx).
		Model(&taskModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     string(status),
			"updated_by": updatedBy,
			"updated_at": time.Now(),
		}).Error
}

func (r *TaskRepository) ListDueTasks(ctx context.Context, now time.Time, limit int) ([]domain.Task, error) {
	var models []taskModel
	err := r.db.WithContext(ctx).
		Where("status = ? AND next_run_time IS NOT NULL AND next_run_time <= ?", "active", now).
		Order("next_run_time asc").
		Limit(limit).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	out := make([]domain.Task, 0, len(models))
	for _, m := range models {
		out = append(out, toTaskDomain(m))
	}
	return out, nil
}

func (r *TaskRepository) UpdateNextRunTime(ctx context.Context, id int64, lastRun, nextRun *time.Time) error {
	return r.db.WithContext(ctx).
		Model(&taskModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"last_run_time": lastRun,
			"next_run_time": nextRun,
			"updated_at":    time.Now(),
		}).Error
}

func toTaskModel(task *domain.Task) taskModel {
	return taskModel{
		ID:             task.ID,
		Name:           task.Name,
		Description:    task.Description,
		TaskType:       string(task.TaskType),
		ScheduleType:   string(task.ScheduleType),
		CronExpr:       task.CronExpr,
		Payload:        task.Payload,
		TimeoutSeconds: task.TimeoutSeconds,
		RetryTimes:     task.RetryTimes,
		Status:         string(task.Status),
		CallbackURL:    task.CallbackURL,
		CreatedBy:      task.CreatedBy,
		UpdatedBy:      task.UpdatedBy,
		NextRunTime:    task.NextRunTime,
		LastRunTime:    task.LastRunTime,
		CreatedAt:      task.CreatedAt,
		UpdatedAt:      task.UpdatedAt,
	}
}

func toTaskDomain(m taskModel) domain.Task {
	return domain.Task{
		ID:             m.ID,
		Name:           m.Name,
		Description:    m.Description,
		TaskType:       domain.TaskType(m.TaskType),
		ScheduleType:   domain.ScheduleType(m.ScheduleType),
		CronExpr:       m.CronExpr,
		Payload:        m.Payload,
		TimeoutSeconds: m.TimeoutSeconds,
		RetryTimes:     m.RetryTimes,
		Status:         domain.TaskStatus(m.Status),
		CallbackURL:    m.CallbackURL,
		CreatedBy:      m.CreatedBy,
		UpdatedBy:      m.UpdatedBy,
		NextRunTime:    m.NextRunTime,
		LastRunTime:    m.LastRunTime,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}
```

---

# 4.4 repository/redis 骨架

## `internal/repository/redis/queue_repo.go`

```go
package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"go-job-platform/internal/repository"
)

const (
	taskQueueKey  = "queue:task:pending"
	retryQueueKey = "queue:task:retry"
)

type QueueRepository struct {
	rdb *goredis.Client
}

func NewQueueRepository(rdb *goredis.Client) repository.QueueRepository {
	return &QueueRepository{rdb: rdb}
}

func (r *QueueRepository) EnqueueTask(ctx context.Context, msg []byte) error {
	return r.rdb.LPush(ctx, taskQueueKey, msg).Err()
}

func (r *QueueRepository) DequeueTask(ctx context.Context, timeout time.Duration) ([]byte, error) {
	res, err := r.rdb.BRPop(ctx, timeout, taskQueueKey).Result()
	if err != nil {
		return nil, err
	}
	if len(res) != 2 {
		return nil, nil
	}
	return []byte(res[1]), nil
}

func (r *QueueRepository) EnqueueRetryTask(ctx context.Context, msg []byte) error {
	return r.rdb.LPush(ctx, retryQueueKey, msg).Err()
}
```

---

# 4.5 queue message

## `internal/queue/task_message.go`

```go
package queue

type TaskMessage struct {
	TaskID      int64  `json:"task_id"`
	ExecutionNo string `json:"execution_no"`
	TriggerType string `json:"trigger_type"`
	TriggerBy   string `json:"trigger_by"`
}
```

---

# 4.6 router 骨架

## `internal/transport/http/router/router.go`

```go
package router

import (
	"github.com/gin-gonic/gin"

	"go-job-platform/internal/transport/http/handler"
)

type Handlers struct {
	TaskHandler      *handler.TaskHandler
	ExecutionHandler *handler.ExecutionHandler
	ReleaseHandler   *handler.ReleaseHandler
	HealthHandler    *handler.HealthHandler
}

func New(h Handlers) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/healthz", h.HealthHandler.Healthz)
	r.GET("/readyz", h.HealthHandler.Readyz)
	r.GET("/metrics", h.HealthHandler.Metrics)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/tasks", h.TaskHandler.CreateTask)
		v1.GET("/tasks/:id", h.TaskHandler.GetTask)
		v1.POST("/tasks/:id/trigger", h.TaskHandler.TriggerTask)
	}

	return r
}
```

---

# 4.7 bootstrap 装配

你说想缝成熟设计，我建议这里学 Fx 的思想，但先手写装配。
也就是**显式构造函数**，不要全局变量，不要 `init()` 偷偷干活。这个方向和 Fx 的理念一致。([Go Packages][4])

## `internal/bootstrap/app.go`

```go
package bootstrap

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	goredis "github.com/redis/go-redis/v9"

	mysqlrepo "go-job-platform/internal/repository/mysql"
	redisrepo "go-job-platform/internal/repository/redis"
	"go-job-platform/internal/service"
	"go-job-platform/internal/transport/http/handler"
	"go-job-platform/internal/transport/http/router"
)

func BuildHTTPServer(db *gorm.DB, rdb *goredis.Client) *gin.Engine {
	taskRepo := mysqlrepo.NewTaskRepository(db)
	executionRepo := mysqlrepo.NewExecutionRepository(db)
	queueRepo := redisrepo.NewQueueRepository(rdb)

	taskSvc := service.NewTaskService(taskRepo, executionRepo, queueRepo)

	taskHandler := handler.NewTaskHandler(taskSvc)
	executionHandler := handler.NewExecutionHandler(nil)
	releaseHandler := handler.NewReleaseHandler(nil)
	healthHandler := handler.NewHealthHandler()

	return router.New(router.Handlers{
		TaskHandler:      taskHandler,
		ExecutionHandler: executionHandler,
		ReleaseHandler:   releaseHandler,
		HealthHandler:    healthHandler,
	})
}
```

---

## 5. 哪些“成熟项目模块”值得缝，哪些不值得

### 值得缝的

* `cmd + internal + pkg` 的布局
* service / repository 分层
* 构造函数注入依赖
* request DTO / domain model 分离
* router / middleware / handler 分离
* metrics / healthz / readyz 标准端点
* queue / lock 单独抽接口

这些做了，项目会很像成熟服务。([GitHub][1])

### 暂时别缝的

* 一上来就全仓库 DDD
* 复杂 CQRS / event sourcing
* 过多的 interface 套 interface
* 十几层 package
* 强行引入 Fx、Wire、go-zero 全家桶

你这个项目现在最需要的是**可讲清楚、能快速迭代**，不是“框架拼装感”。Fx/go-zero 都是成熟方案，但你现在更应该借它们的思路，而不是把整套体系搬进来。([GitHub][5])

---

## 6. 最后给你一版“最小可开工文件集”

先把这几个文件落出来：

```text
cmd/api-server/main.go
internal/bootstrap/app.go
internal/domain/task.go
internal/domain/execution.go
internal/dto/task_dto.go
internal/repository/interfaces.go
internal/repository/mysql/task_repo.go
internal/repository/redis/queue_repo.go
internal/service/task_service.go
internal/service/task_service_impl.go
internal/transport/http/handler/task_handler.go
internal/transport/http/router/router.go
internal/queue/task_message.go
internal/executor/executor.go
internal/executor/shell_executor.go
internal/pkgx/response/response.go
```

先把这条链打通：

**创建任务 -> 存 MySQL -> 手动触发 -> 写 execution -> 推 Redis 队列**

这一步跑通后，再补：

* worker 消费
* execution finish
* scheduler 定时派发
* metrics
* K8s readiness/liveness

下一步最合适的是我继续给你：
**`main.go + bootstrap + response + health handler + execution repository/service/handler` 的完整代码骨架**。

[1]: https://github.com/golang-standards/project-layout?utm_source=chatgpt.com "golang-standards/project-layout: Standard Go Project Layout"
[2]: https://github.com/evrone/go-clean-template/blob/master/README.md?utm_source=chatgpt.com "go-clean-template/README.md at master"
[3]: https://go-zero.dev/en/docs/concepts/layout?utm_source=chatgpt.com "Project Structure | go-zero Documentation"
[4]: https://pkg.go.dev/go.uber.org/fx?utm_source=chatgpt.com "fx package - go.uber.org/fx - Go Packages"
[5]: https://github.com/zeromicro/go-zero?utm_source=chatgpt.com "zeromicro/go-zero: A cloud-native Go microservices ..."
