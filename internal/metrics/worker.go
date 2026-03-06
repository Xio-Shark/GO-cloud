package metrics

import "github.com/prometheus/client_golang/prometheus"

var WorkerConsumedTasksTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "go_cloud_worker_consumed_tasks_total",
		Help: "Total number of consumed task messages",
	},
)

var WorkerSucceededTasksTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "go_cloud_worker_succeeded_tasks_total",
		Help: "Total number of succeeded task executions",
	},
)

var WorkerFailedTasksTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "go_cloud_worker_failed_tasks_total",
		Help: "Total number of failed task executions",
	},
)

var WorkerRetriedTasksTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "go_cloud_worker_retried_tasks_total",
		Help: "Total number of retried task executions",
	},
)

var WorkerRunningTasksGauge = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "go_cloud_worker_running_tasks",
		Help: "Current number of running tasks",
	},
)

var WorkerExecutionDurationMs = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "go_cloud_worker_execution_duration_ms",
		Help:    "Worker execution duration in milliseconds",
		Buckets: prometheus.LinearBuckets(10, 50, 20),
	},
)
