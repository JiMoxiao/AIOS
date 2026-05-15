package telemetry

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	runsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itw_runs_total",
			Help: "Total number of workflow runs.",
		},
		[]string{"status"},
	)

	nodeRunsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itw_node_runs_total",
			Help: "Total number of node runs.",
		},
		[]string{"node_type", "status", "model"},
	)

	fallbackTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "itw_fallback_total",
			Help: "Total number of fallback attempts.",
		},
		[]string{"model", "reason"},
	)

	runDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "itw_run_duration_seconds",
			Help:    "Workflow run duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
	)

	nodeLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "itw_node_latency_seconds",
			Help:    "Node latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"node_type", "model"},
	)

	registered = false
)

func MetricsHandler() http.Handler {
	ensureRegistered()
	return promhttp.Handler()
}

func RecordRun(status string, d time.Duration) {
	ensureRegistered()
	runsTotal.WithLabelValues(status).Inc()
	runDuration.Observe(d.Seconds())
}

func RecordNodeRun(nodeType, status, model string, latency time.Duration) {
	ensureRegistered()
	nodeRunsTotal.WithLabelValues(nodeType, status, model).Inc()
	if latency > 0 {
		nodeLatency.WithLabelValues(nodeType, model).Observe(latency.Seconds())
	}
}

func RecordFallback(model, reason string) {
	ensureRegistered()
	fallbackTotal.WithLabelValues(model, reason).Inc()
}

func ensureRegistered() {
	if registered {
		return
	}
	prometheus.MustRegister(runsTotal, nodeRunsTotal, fallbackTotal, runDuration, nodeLatency)
	registered = true
}

