package bootstrap

import (
	"context"
	"net/http"
	"time"

	"go-cloud/internal/metrics"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HealthChecker func(ctx context.Context) error

func NewAdminServer(addr string, checker HealthChecker) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler(checker))
	mux.HandleFunc("/readyz", healthHandler(checker))
	mux.Handle("/metrics", promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{}))
	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func healthHandler(checker HealthChecker) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if checker != nil {
			if err := checker(request.Context()); err != nil {
				http.Error(writer, err.Error(), http.StatusServiceUnavailable)
				return
			}
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status":"ok"}`))
	}
}
