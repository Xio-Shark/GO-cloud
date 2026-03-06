package metrics

import "github.com/prometheus/client_golang/prometheus"

var NotifierSentTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "go_cloud_notifier_sent_total",
		Help: "Total number of successful notifications",
	},
)

var NotifierFailedTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "go_cloud_notifier_failed_total",
		Help: "Total number of failed notifications",
	},
)

var NotifierRequestDurationMs = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "go_cloud_notifier_request_duration_ms",
		Help:    "Notifier webhook request duration in milliseconds",
		Buckets: prometheus.LinearBuckets(10, 50, 20),
	},
)
