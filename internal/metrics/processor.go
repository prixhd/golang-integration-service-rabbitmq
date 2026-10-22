package metrics

import "github.com/prometheus/client_golang/prometheus"

var TasksTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "codeprocessor_tasks_total",
		Help: "Total number of processed code tasks",
	},
	[]string{"translator"},
)

var TaskDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "codeprocessor_task_duration_seconds",
		Help:    "Code task execution duration in seconds",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"translator"},
)

func RegisterProcessorMetrics() {

	prometheus.MustRegister(TasksTotal, TaskDuration)
}
