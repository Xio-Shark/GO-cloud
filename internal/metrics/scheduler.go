package metrics

import "github.com/prometheus/client_golang/prometheus"

var SchedulerScansTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "go_cloud_scheduler_scans_total",
		Help: "Total number of scheduler scan rounds",
	},
)

var SchedulerDueTasks = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "go_cloud_scheduler_due_tasks",
		Help: "Current due task count in one scheduler scan",
	},
)

var SchedulerDispatchedTasksTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "go_cloud_scheduler_dispatched_tasks_total",
		Help: "Total number of dispatched tasks",
	},
)

var SchedulerDuplicateBlockedTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "go_cloud_scheduler_duplicate_blocked_total",
		Help: "Total number of scheduler dispatches blocked by distributed lock",
	},
)
