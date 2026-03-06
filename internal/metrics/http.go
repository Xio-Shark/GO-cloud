package metrics

import "github.com/prometheus/client_golang/prometheus"

var HTTPRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "go_cloud_http_requests_total",
		Help: "Total number of HTTP requests",
	},
	[]string{"method", "path", "status"},
)

var HTTPRequestDurationSeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "go_cloud_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "path"},
)
