package httptransport

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go-cloud/internal/metrics"
	"go-cloud/internal/transport/http/handler"
	"go-cloud/internal/transport/http/middleware"
)

type Handlers struct {
	HealthHandler    *handler.HealthHandler
	TaskHandler      *handler.TaskHandler
	ExecutionHandler *handler.ExecutionHandler
	ReleaseHandler   *handler.ReleaseHandler
}

func NewRouter(handlers Handlers) *gin.Engine {
	engine := gin.New()
	engine.Use(middleware.TraceID(), middleware.Metrics(), middleware.AccessLog(), middleware.Recovery())

	engine.GET("/healthz", handlers.HealthHandler.Healthz)
	engine.GET("/readyz", handlers.HealthHandler.Readyz)
	engine.GET("/metrics", gin.WrapH(promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{})))

	v1 := engine.Group("/api/v1")
	v1.POST("/tasks", handlers.TaskHandler.CreateTask)
	v1.GET("/tasks", handlers.TaskHandler.ListTasks)
	v1.GET("/tasks/:id", handlers.TaskHandler.GetTask)
	v1.PUT("/tasks/:id", handlers.TaskHandler.UpdateTask)
	v1.DELETE("/tasks/:id", handlers.TaskHandler.DeleteTask)
	v1.POST("/tasks/:id/pause", handlers.TaskHandler.PauseTask)
	v1.POST("/tasks/:id/resume", handlers.TaskHandler.ResumeTask)
	v1.POST("/tasks/:id/trigger", handlers.TaskHandler.TriggerTask)
	v1.GET("/tasks/:id/executions", handlers.ExecutionHandler.ListTaskExecutions)
	v1.GET("/executions", handlers.ExecutionHandler.ListExecutions)
	v1.GET("/executions/:execution_no", handlers.ExecutionHandler.GetExecution)
	v1.GET("/executions/:execution_no/logs", handlers.ExecutionHandler.GetExecutionLogs)
	v1.POST("/executions/:execution_no/retry", handlers.ExecutionHandler.RetryExecution)
	v1.POST("/executions/:execution_no/cancel", handlers.ExecutionHandler.CancelExecution)
	v1.POST("/releases", handlers.ReleaseHandler.CreateRelease)
	v1.GET("/releases", handlers.ReleaseHandler.ListReleases)
	v1.GET("/releases/:id", handlers.ReleaseHandler.GetRelease)
	v1.POST("/releases/:id/rollback", handlers.ReleaseHandler.RollbackRelease)

	return engine
}
