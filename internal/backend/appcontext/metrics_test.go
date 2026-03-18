package appcontext

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetrics(t *testing.T) {
	// Use a fresh registry to avoid conflicts with the default registry.
	reg := prometheus.NewRegistry()

	m := &AppMetrics{
		Runtime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "test_backend_runtime_seconds",
				Help:    "Test request duration.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		Rate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "test_backend_rate_total",
				Help: "Test request count.",
			},
			[]string{"method", "path"},
		),
		ErrorRate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "test_backend_error_rate_total",
				Help: "Test error count.",
			},
			[]string{"method", "path"},
		),
	}

	require.NoError(t, reg.Register(m.Runtime))
	require.NoError(t, reg.Register(m.Rate))
	require.NoError(t, reg.Register(m.ErrorRate))

	assert.NotNil(t, m.Runtime)
	assert.NotNil(t, m.Rate)
	assert.NotNil(t, m.ErrorRate)

	// Verify metrics can be observed.
	m.Rate.WithLabelValues("GET", "/health").Inc()
	m.ErrorRate.WithLabelValues("GET", "/health").Inc()
	m.Runtime.WithLabelValues("GET", "/health").Observe(0.1)
}
