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
	"go-cloud/internal/metrics"
)

func main() {
	cfg := bootstrap.LoadConfig("api-server")
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(bootstrap.HealthcheckExitCode(cfg.HTTPAddr))
	}
	bootstrap.SetupLogger(cfg)
	metrics.MustRegisterAll()

	db, err := bootstrap.NewMySQL(cfg)
	if err != nil {
		slog.Default().Error("init mysql failed", "error", err)
		return
	}
	rdb := bootstrap.NewRedis(cfg)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           bootstrap.BuildHTTPServer(cfg, db, rdb),
		ReadHeaderTimeout: 5 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Default().Info("api-server listening", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Default().Error("api-server exited unexpectedly", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.GracefulShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Default().Error("api-server shutdown failed", "error", err)
	}
}
