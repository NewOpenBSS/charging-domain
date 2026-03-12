package appcontext

import (
	"github.com/prometheus/client_golang/prometheus"
)

type AppMetrics struct {
	// Timers (measured as duration seconds)
	Runtime *prometheus.HistogramVec

	// Meter (rate) -> counter; Prometheus derives rates via rate()/irate()
	Rate      *prometheus.CounterVec
	ErrorRate *prometheus.CounterVec
}

func NewMetrics() *AppMetrics {

	m := &AppMetrics{
		Runtime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ocs_runtime_seconds",
				Help:    "OCS runtime duration.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		Rate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ocs_rate_total",
				Help: "Total OCS events (use rate() in Prometheus to get per-second).",
			},
			[]string{"method", "path"},
		),

		ErrorRate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ocs_error_rate_total",
				Help: "Total OCS failed events (use rate() in Prometheus to get per-second).",
			},
			[]string{"method", "path"},
		),
	}

	// Register everything with Prometheus.
	prometheus.DefaultRegisterer.MustRegister(
		m.Runtime,
		m.Rate,
		m.ErrorRate,
	)

	return m
}
