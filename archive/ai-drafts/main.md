下面直接给你一套**能拼起来的完整代码骨架**。
我会覆盖这几个部分：

* `main.go`
* `bootstrap`
* `response`
* `health handler`
* `execution repository`
* `execution service`
* `execution handler`

默认前提：

* HTTP 框架：`gin`
* MySQL：`gorm`
* Redis：`go-redis`
* 日志先不强绑复杂框架，先用标准库 + `log`
* 代码目标是“先跑起来，再扩展”

---

# 一、建议文件结构

你这一轮至少需要这些文件：

```text
cmd/
└── api-server/
    └── main.go

internal/
├── bootstrap/
│   ├── app.go
│   ├── config.go
│   ├── http.go
│   ├── mysql.go
│   └── redis.go
├── domain/
│   ├── execution.go
│   └── release.go
├── repository/
│   ├── interfaces.go
│   └── mysql/
│       └── execution_repo.go
├── service/
│   ├── execution_service.go
│   └── execution_service_impl.go
├── transport/
│   └── http/
│       ├── handler/
│       │   ├── execution_handler.go
│       │   └── health_handler.go
│       └── router/
│           └── router.go
└── pkgx/
    └── response/
        └── response.go
```

---

# 二、`main.go`

## `cmd/api-server/main.go`

```go
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"go-job-platform/internal/bootstrap"
)

func main() {
	cfg := bootstrap.LoadConfig()

	db, err := bootstrap.NewMySQL(cfg)
	if err != nil {
		log.Fatalf("init mysql failed: %v", err)
	}

	rdb, err := bootstrap.NewRedis(cfg)
	if err != nil {
		log.Fatalf("init redis failed: %v", err)
	}

	engine := bootstrap.BuildHTTPServer(cfg, db, rdb)
	srv := bootstrap.NewHTTPServer(cfg, engine)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("api-server starting at :%s", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err.Error() != "http: Server closed" {
			log.Fatalf("http server start failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ServerShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http server shutdown failed: %v", err)
	}

	log.Println("api-server exited")
}
```

---

# 三、bootstrap

---

## `internal/bootstrap/config.go`

```go
package bootstrap

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppName string
	AppEnv  string

	HTTPPort string

	MySQLDSN string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	ServerReadTimeout      time.Duration
	ServerWriteTimeout     time.Duration
	ServerShutdownTimeout  time.Duration
}

func LoadConfig() Config {
	return Config{
		AppName: getEnv("APP_NAME", "go-job-platform"),
		AppEnv:  getEnv("APP_ENV", "dev"),

		HTTPPort: getEnv("HTTP_PORT", "8080"),

		MySQLDSN: getEnv(
			"MYSQL_DSN",
			"job_user:job_pass@tcp(127.0.0.1:3306)/job_platform?charset=utf8mb4&parseTime=True&loc=Local",
		),

		RedisAddr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		ServerReadTimeout:     getEnvDuration("SERVER_READ_TIMEOUT", 10*time.Second),
		ServerWriteTimeout:    getEnvDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
		ServerShutdownTimeout: getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 10*time.Second),
	}
}

func getEnv(key, defaultValue string) string {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	return v
}

func getEnvInt(key string, defaultValue int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	return n
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return defaultValue
	}
	return d
}
```

---

## `internal/bootstrap/mysql.go`

```go
package bootstrap

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewMySQL(cfg Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
```

---

## `internal/bootstrap/redis.go`

```go
package bootstrap

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

func NewRedis(cfg Config) (*goredis.Client, error) {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return rdb, nil
}
```

---

## `internal/bootstrap/http.go`

```go
package bootstrap

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewHTTPServer(cfg Config, engine *gin.Engine) *http.Server {
	return &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      engine,
		ReadTimeout:  cfg.ServerReadTimeout,
		WriteTimeout: cfg.ServerWriteTimeout,
	}
}
```

---

## `internal/bootstrap/app.go`

这一层负责把依赖装起来。

```go
package bootstrap

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	goredis "github.com/redis/go-redis/v9"

	mysqlrepo "go-job-platform/internal/repository/mysql"
	"go-job-platform/internal/service"
	"go-job-platform/internal/transport/http/handler"
	"go-job-platform/internal/transport/http/router"
)

func BuildHTTPServer(cfg Config, db *gorm.DB, rdb *goredis.Client) *gin.Engine {
	// repositories
	executionRepo := mysqlrepo.NewExecutionRepository(db)

	// services
	executionSvc := service.NewExecutionService(executionRepo)

	// handlers
	healthHandler := handler.NewHealthHandler(cfg, db, rdb)
	executionHandler := handler.NewExecutionHandler(executionSvc)

	return router.New(router.Handlers{
		HealthHandler:    healthHandler,
		ExecutionHandler: executionHandler,
	})
}
```

---

# 四、response

## `internal/pkgx/response/response.go`

这个统一一下返回格式，后面很好扩。

```go
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Body{
		Code:    40000,
		Message: message,
		Data:    nil,
	})
}

func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, Body{
		Code:    40400,
		Message: message,
		Data:    nil,
	})
}

func InternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, Body{
		Code:    50000,
		Message: message,
		Data:    nil,
	})
}
```

---

# 五、domain

---

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

## `internal/domain/release.go`

```go
package domain

import "time"

type ReleaseStatus string

const (
	ReleaseStatusPending    ReleaseStatus = "pending"
	ReleaseStatusDeployed   ReleaseStatus = "deployed"
	ReleaseStatusFailed     ReleaseStatus = "failed"
	ReleaseStatusRolledBack ReleaseStatus = "rolled_back"
)

type ReleaseRecord struct {
	ID        int64
	Env       string
	AppName   string
	ImageTag  string
	GitCommit string
	Operator  string
	Status    ReleaseStatus
	Message   string
	CreatedAt time.Time
}
```

---

# 六、repository interface

## `internal/repository/interfaces.go`

```go
package repository

import (
	"context"

	"go-job-platform/internal/domain"
)

type ExecutionRepository interface {
	Create(ctx context.Context, execution *domain.TaskExecution) error
	GetByExecutionNo(ctx context.Context, executionNo string) (*domain.TaskExecution, error)
	ListByTaskID(ctx context.Context, taskID int64, page, pageSize int) ([]domain.TaskExecution, int64, error)
	UpdateStatus(ctx context.Context, executionNo string, status domain.ExecutionStatus, workerID string) error
	Finish(
		ctx context.Context,
		executionNo string,
		status domain.ExecutionStatus,
		durationMs int64,
		exitCode *int,
		errMsg *string,
		outputLog *string,
	) error
}
```

---

# 七、execution repository

## `internal/repository/mysql/execution_repo.go`

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

type executionModel struct {
	ID           int64      `gorm:"column:id;primaryKey;autoIncrement"`
	TaskID       int64      `gorm:"column:task_id"`
	ExecutionNo  string     `gorm:"column:execution_no"`
	TriggerType  string     `gorm:"column:trigger_type"`
	WorkerID     string     `gorm:"column:worker_id"`
	Status       string     `gorm:"column:status"`
	StartTime    *time.Time `gorm:"column:start_time"`
	EndTime      *time.Time `gorm:"column:end_time"`
	DurationMs   int64      `gorm:"column:duration_ms"`
	RetryCount   int        `gorm:"column:retry_count"`
	ExitCode     *int       `gorm:"column:exit_code"`
	ErrorMessage *string    `gorm:"column:error_message"`
	OutputLog    *string    `gorm:"column:output_log"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
}

func (executionModel) TableName() string {
	return "task_executions"
}

type ExecutionRepository struct {
	db *gorm.DB
}

func NewExecutionRepository(db *gorm.DB) repository.ExecutionRepository {
	return &ExecutionRepository{db: db}
}

func (r *ExecutionRepository) Create(ctx context.Context, execution *domain.TaskExecution) error {
	m := toExecutionModel(execution)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return err
	}
	execution.ID = m.ID
	return nil
}

func (r *ExecutionRepository) GetByExecutionNo(ctx context.Context, executionNo string) (*domain.TaskExecution, error) {
	var m executionModel
	err := r.db.WithContext(ctx).
		Where("execution_no = ?", executionNo).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	d := toExecutionDomain(m)
	return &d, nil
}

func (r *ExecutionRepository) ListByTaskID(ctx context.Context, taskID int64, page, pageSize int) ([]domain.TaskExecution, int64, error) {
	var (
		models []executionModel
		total  int64
	)

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	db := r.db.WithContext(ctx).Model(&executionModel{}).Where("task_id = ?", taskID)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := db.Order("id desc").Offset(offset).Limit(pageSize).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	out := make([]domain.TaskExecution, 0, len(models))
	for _, m := range models {
		out = append(out, toExecutionDomain(m))
	}
	return out, total, nil
}

func (r *ExecutionRepository) UpdateStatus(ctx context.Context, executionNo string, status domain.ExecutionStatus, workerID string) error {
	now := time.Now()

	updates := map[string]interface{}{
		"status":     string(status),
		"worker_id":  workerID,
		"updated_at": now,
	}

	if status == domain.ExecutionStatusRunning {
		updates["start_time"] = now
	}

	return r.db.WithContext(ctx).
		Model(&executionModel{}).
		Where("execution_no = ?", executionNo).
		Updates(updates).Error
}

func (r *ExecutionRepository) Finish(
	ctx context.Context,
	executionNo string,
	status domain.ExecutionStatus,
	durationMs int64,
	exitCode *int,
	errMsg *string,
	outputLog *string,
) error {
	now := time.Now()

	updates := map[string]interface{}{
		"status":        string(status),
		"end_time":      now,
		"duration_ms":   durationMs,
		"exit_code":     exitCode,
		"error_message": errMsg,
		"output_log":    outputLog,
		"updated_at":    now,
	}

	return r.db.WithContext(ctx).
		Model(&executionModel{}).
		Where("execution_no = ?", executionNo).
		Updates(updates).Error
}

func toExecutionModel(e *domain.TaskExecution) executionModel {
	return executionModel{
		ID:           e.ID,
		TaskID:       e.TaskID,
		ExecutionNo:  e.ExecutionNo,
		TriggerType:  string(e.TriggerType),
		WorkerID:     e.WorkerID,
		Status:       string(e.Status),
		StartTime:    e.StartTime,
		EndTime:      e.EndTime,
		DurationMs:   e.DurationMs,
		RetryCount:   e.RetryCount,
		ExitCode:     e.ExitCode,
		ErrorMessage: e.ErrorMessage,
		OutputLog:    e.OutputLog,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}

func toExecutionDomain(m executionModel) domain.TaskExecution {
	return domain.TaskExecution{
		ID:           m.ID,
		TaskID:       m.TaskID,
		ExecutionNo:  m.ExecutionNo,
		TriggerType:  domain.TriggerType(m.TriggerType),
		WorkerID:     m.WorkerID,
		Status:       domain.ExecutionStatus(m.Status),
		StartTime:    m.StartTime,
		EndTime:      m.EndTime,
		DurationMs:   m.DurationMs,
		RetryCount:   m.RetryCount,
		ExitCode:     m.ExitCode,
		ErrorMessage: m.ErrorMessage,
		OutputLog:    m.OutputLog,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
```

---

# 八、execution service

---

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
}
```

---

## `internal/service/execution_service_impl.go`

```go
package service

import (
	"context"
	"errors"

	"go-job-platform/internal/domain"
	"go-job-platform/internal/repository"
)

type executionService struct {
	executionRepo repository.ExecutionRepository
}

func NewExecutionService(executionRepo repository.ExecutionRepository) ExecutionService {
	return &executionService{
		executionRepo: executionRepo,
	}
}

func (s *executionService) GetExecution(ctx context.Context, executionNo string) (*domain.TaskExecution, error) {
	execution, err := s.executionRepo.GetByExecutionNo(ctx, executionNo)
	if err != nil {
		return nil, err
	}
	if execution == nil {
		return nil, errors.New("execution not found")
	}
	return execution, nil
}

func (s *executionService) ListExecutionsByTask(ctx context.Context, taskID int64, page, pageSize int) ([]domain.TaskExecution, int64, error) {
	return s.executionRepo.ListByTaskID(ctx, taskID, page, pageSize)
}

func (s *executionService) GetExecutionLogs(ctx context.Context, executionNo string) (string, error) {
	execution, err := s.executionRepo.GetByExecutionNo(ctx, executionNo)
	if err != nil {
		return "", err
	}
	if execution == nil {
		return "", errors.New("execution not found")
	}
	if execution.OutputLog == nil {
		return "", nil
	}
	return *execution.OutputLog, nil
}
```

---

# 九、health handler

## `internal/transport/http/handler/health_handler.go`

这里做 3 件事：

* `healthz`：服务活着没
* `readyz`：依赖可用没
* `metrics`：先给占位，后续接 Prometheus

```go
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	goredis "github.com/redis/go-redis/v9"

	"go-job-platform/internal/bootstrap"
	"go-job-platform/internal/pkgx/response"
)

type HealthHandler struct {
	cfg bootstrap.Config
	db  *gorm.DB
	rdb *goredis.Client
}

func NewHealthHandler(cfg bootstrap.Config, db *gorm.DB, rdb *goredis.Client) *HealthHandler {
	return &HealthHandler{
		cfg: cfg,
		db:  db,
		rdb: rdb,
	}
}

func (h *HealthHandler) Healthz(c *gin.Context) {
	response.Success(c, gin.H{
		"app": h.cfg.AppName,
		"env": h.cfg.AppEnv,
		"status": "ok",
	})
}

func (h *HealthHandler) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := pingMySQL(ctx, h.db); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    50301,
			"message": "mysql not ready",
			"data":    err.Error(),
		})
		return
	}

	if err := h.rdb.Ping(ctx).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    50302,
			"message": "redis not ready",
			"data":    err.Error(),
		})
		return
	}

	response.Success(c, gin.H{
		"status": "ready",
	})
}

func (h *HealthHandler) Metrics(c *gin.Context) {
	c.String(http.StatusOK, "# metrics placeholder\n")
}

func pingMySQL(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
```

---

# 十、execution handler

## `internal/transport/http/handler/execution_handler.go`

```go
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"go-job-platform/internal/pkgx/response"
	"go-job-platform/internal/service"
)

type ExecutionHandler struct {
	executionSvc service.ExecutionService
}

func NewExecutionHandler(executionSvc service.ExecutionService) *ExecutionHandler {
	return &ExecutionHandler{
		executionSvc: executionSvc,
	}
}

func (h *ExecutionHandler) GetExecution(c *gin.Context) {
	executionNo := c.Param("execution_no")
	if executionNo == "" {
		response.BadRequest(c, "execution_no is required")
		return
	}

	execution, err := h.executionSvc.GetExecution(c.Request.Context(), executionNo)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, execution)
}

func (h *ExecutionHandler) ListTaskExecutions(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("task_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid task_id")
		return
	}

	page := parseIntOrDefault(c.Query("page"), 1)
	pageSize := parseIntOrDefault(c.Query("page_size"), 10)

	list, total, err := h.executionSvc.ListExecutionsByTask(c.Request.Context(), taskID, page, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":      list,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

func (h *ExecutionHandler) GetExecutionLogs(c *gin.Context) {
	executionNo := c.Param("execution_no")
	if executionNo == "" {
		response.BadRequest(c, "execution_no is required")
		return
	}

	logs, err := h.executionSvc.GetExecutionLogs(c.Request.Context(), executionNo)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"execution_no": executionNo,
		"logs":         logs,
	})
}

func parseIntOrDefault(v string, defaultValue int) int {
	if v == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultValue
	}
	return n
}
```

---

# 十一、router

## `internal/transport/http/router/router.go`

```go
package router

import (
	"github.com/gin-gonic/gin"

	"go-job-platform/internal/transport/http/handler"
)

type Handlers struct {
	HealthHandler    *handler.HealthHandler
	ExecutionHandler *handler.ExecutionHandler
}

func New(h Handlers) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/healthz", h.HealthHandler.Healthz)
	r.GET("/readyz", h.HealthHandler.Readyz)
	r.GET("/metrics", h.HealthHandler.Metrics)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/executions/:execution_no", h.ExecutionHandler.GetExecution)
		v1.GET("/executions/:execution_no/logs", h.ExecutionHandler.GetExecutionLogs)
		v1.GET("/tasks/:task_id/executions", h.ExecutionHandler.ListTaskExecutions)
	}

	return r
}
```

---

# 十二、这套骨架现在能支撑什么

跑起来之后，你至少有这些接口：

* `GET /healthz`
* `GET /readyz`
* `GET /metrics`
* `GET /api/v1/executions/:execution_no`
* `GET /api/v1/executions/:execution_no/logs`
* `GET /api/v1/tasks/:task_id/executions`

这能让你先把 **execution 查询链路**打通。

---

# 十三、下一步怎么接上你前面的任务系统

你现在最应该补的是这几块：

## 1. 补 task handler / task service

这样能创建任务、触发任务。

## 2. 补 worker

这样 execution 状态才能从 `pending -> running -> success/failed`

## 3. 补 scheduler

这样 cron 任务能自动派发。

## 4. 把 `metrics` 从 placeholder 换成 Prometheus

这一步会很加分。

---

# 十四、你可以直接继续创建的文件

建议你下一轮直接补这些：

```text
internal/transport/http/handler/task_handler.go
internal/service/task_service.go
internal/service/task_service_impl.go
internal/repository/mysql/task_repo.go
internal/repository/redis/queue_repo.go
internal/queue/task_message.go
cmd/worker/main.go
internal/service/worker_service.go
internal/service/worker_service_impl.go
internal/executor/shell_executor.go
```

最合适的下一步是：
我直接把 **task handler + task service + task repository + queue repository + worker main/service/executor** 这一整条链也给你补成同样风格的完整骨架。
