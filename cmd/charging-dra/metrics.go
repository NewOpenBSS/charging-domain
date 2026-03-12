package main

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type DRAMetrics struct {
	// Timers (measured as duration seconds)
	Runtime  *prometheus.HistogramVec
	Overhead *prometheus.HistogramVec

	// Meter (rate) -> counter; Prometheus derives rates via rate()/irate()
	Rate      prometheus.Counter
	ErrorRate prometheus.Counter
}

func NewDRAMetrics(reg prometheus.Registerer) *DRAMetrics {
	m := &DRAMetrics{
		Runtime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "dra_runtime_seconds",
				Help:    "DRA runtime duration.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"request_type"},
		),
		Overhead: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "dra_overhead_seconds",
				Help:    "DRA processing overhead duration.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"request_type"},
		),
		Rate: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "dra_rate_total",
			Help: "Total DRA events (use rate() in Prometheus to get per-second).",
		}),
		ErrorRate: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "dra_error_rate_total",
			Help: "Total DRA failed events (use rate() in Prometheus to get per-second).",
		}),
	}

	// Register everything with Prometheus.
	reg.MustRegister(
		m.Runtime,
		m.Overhead,
		m.Rate,
		m.ErrorRate,
	)

	return m
}

func (m *DRAMetrics) IncRate() {
	m.Rate.Inc()
}

func (m *DRAMetrics) IncErrorRate() {
	m.ErrorRate.Inc()
}

func (m *DRAMetrics) ObserveRuntimeOverhead(reqType string, d time.Duration, o time.Duration) {
	m.Runtime.
		WithLabelValues(reqType).
		Observe(d.Seconds())
	m.Overhead.
		WithLabelValues(reqType).
		Observe(o.Seconds())
}
