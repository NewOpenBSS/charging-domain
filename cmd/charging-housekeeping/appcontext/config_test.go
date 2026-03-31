package appcontext

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_DefaultThresholds(t *testing.T) {
	// Verify the three default threshold constants match the feature spec.
	assert.Equal(t, 24*time.Hour, defaultStaleSessions)
	assert.Equal(t, 36*time.Hour, defaultTracePurge)
	assert.Equal(t, 30*24*time.Hour, defaultRatePlanCleanup)
}

func TestConfig_EnvOverride(t *testing.T) {
	// Set env vars and verify they are parsed correctly.
	t.Setenv("STALE_SESSIONS_THRESHOLD", "48h")
	t.Setenv("TRACE_PURGE_THRESHOLD", "72h")
	t.Setenv("RATEPLAN_CLEANUP_THRESHOLD", "1440h")

	cfg := NewConfig("../housekeeping-config.yaml")

	assert.Equal(t, 48*time.Hour, cfg.StaleSessions)
	assert.Equal(t, 72*time.Hour, cfg.TracePurge)
	assert.Equal(t, 1440*time.Hour, cfg.RatePlanCleanup)
}

func TestConfig_InvalidEnvFallsBackToDefault(t *testing.T) {
	t.Setenv("STALE_SESSIONS_THRESHOLD", "not-a-duration")
	t.Setenv("TRACE_PURGE_THRESHOLD", "")
	t.Setenv("RATEPLAN_CLEANUP_THRESHOLD", "bad")

	cfg := NewConfig("../housekeeping-config.yaml")

	assert.Equal(t, defaultStaleSessions, cfg.StaleSessions)
	assert.Equal(t, defaultTracePurge, cfg.TracePurge)
	assert.Equal(t, defaultRatePlanCleanup, cfg.RatePlanCleanup)
}
