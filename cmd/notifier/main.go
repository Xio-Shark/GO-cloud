package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-cloud/internal/bootstrap"
	"go-cloud/internal/healthcheck"
	"go-cloud/internal/metrics"
	"go-cloud/internal/notifier"
	redisrepo "go-cloud/internal/repository/redis"
	"go-cloud/internal/service"
)

func main() {
	cfg := bootstrap.LoadConfig("notifier")
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(bootstrap.HealthcheckExitCode(cfg.AdminAddr))
	}
	bootstrap.SetupLogger(cfg)
	metrics.MustRegisterAll()

	rdb := bootstrap.NewRedis(cfg)
	queueRepo := redisrepo.NewQueueRepository(rdb)
	notifierSvc := service.NewNotifierService(queueRepo, notifier.NewWebhookNotifier(cfg.NotifierRequestTimeout), cfg.QueuePopTimeout)
	adminServer := bootstrap.NewAdminServer(cfg.AdminAddr, func(ctx context.Context) error {
		return healthcheck.CheckDependencies(ctx, nil, rdb)
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 2)

	go serveAdmin(adminServer, errCh)
	go func() {
		if err := notifierSvc.ConsumeLoop(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		slog.Default().Error("notifier exited unexpectedly", "error", err)
		stop()
	}
	shutdownAdmin(adminServer, cfg.GracefulShutdownTimeout)
}

func serveAdmin(server *http.Server, errCh chan<- error) {
	slog.Default().Info("admin server listening", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errCh <- err
	}
}

func shutdownAdmin(server *http.Server, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Default().Error("admin server shutdown failed", "error", err)
	}
}
