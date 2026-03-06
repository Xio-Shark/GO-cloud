package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	Registry = prometheus.NewRegistry()
	once     sync.Once
)

func MustRegisterAll() {
	once.Do(func() {
		Registry.MustRegister(
			HTTPRequestsTotal,
			HTTPRequestDurationSeconds,
			SchedulerScansTotal,
			SchedulerDueTasks,
			SchedulerDispatchedTasksTotal,
			SchedulerDuplicateBlockedTotal,
			WorkerConsumedTasksTotal,
			WorkerSucceededTasksTotal,
			WorkerFailedTasksTotal,
			WorkerRetriedTasksTotal,
			WorkerRunningTasksGauge,
			WorkerExecutionDurationMs,
			NotifierSentTotal,
			NotifierFailedTotal,
			NotifierRequestDurationMs,
		)
	})
}
