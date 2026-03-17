package appcontext

import "github.com/prometheus/client_golang/prometheus"

// AppMetrics holds the Prometheus metrics registered for the charging-backend.
type AppMetrics struct {
	Runtime   *prometheus.HistogramVec
	Rate      *prometheus.CounterVec
	ErrorRate *prometheus.CounterVec
}

// NewMetrics creates and registers the Prometheus metrics for the charging-backend.
func NewMetrics() *AppMetrics {
	m := &AppMetrics{
		Runtime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "charging_backend_runtime_seconds",
				Help:    "Charging backend request duration.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		Rate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "charging_backend_rate_total",
				Help: "Total charging backend requests.",
			},
			[]string{"method", "path"},
		),
		ErrorRate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "charging_backend_error_rate_total",
				Help: "Total charging backend errors.",
			},
			[]string{"method", "path"},
		),
	}

	prometheus.DefaultRegisterer.MustRegister(
		m.Runtime,
		m.Rate,
		m.ErrorRate,
	)

	return m
}
