package bootstrap

import (
	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"go-cloud/internal/gitops"
	"go-cloud/internal/healthcheck"
	"go-cloud/internal/repository/mysql"
	"go-cloud/internal/repository/redis"
	"go-cloud/internal/service"
	httptransport "go-cloud/internal/transport/http"
	"go-cloud/internal/transport/http/handler"
)

func BuildHTTPServer(cfg Config, db *gorm.DB, rdb *goredis.Client) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	taskRepo := mysqlrepo.NewTaskRepository(db)
	executionRepo := mysqlrepo.NewExecutionRepository(db)
	releaseRepo := mysqlrepo.NewReleaseRepository(db)
	queueRepo := redisrepo.NewQueueRepository(rdb)
	releaseUpdater := gitops.NewFileUpdater(cfg.GitOpsOverlaysRoot)

	taskSvc := service.NewTaskService(taskRepo, executionRepo, queueRepo)
	executionSvc := service.NewExecutionService(taskRepo, executionRepo, queueRepo)
	releaseSvc := service.NewReleaseService(releaseRepo, releaseUpdater)

	return httptransport.NewRouter(httptransport.Handlers{
		HealthHandler:    handler.NewHealthHandler(healthcheck.NewDependencyChecker(db, rdb)),
		TaskHandler:      handler.NewTaskHandler(taskSvc),
		ExecutionHandler: handler.NewExecutionHandler(executionSvc),
		ReleaseHandler:   handler.NewReleaseHandler(releaseSvc),
	})
}
