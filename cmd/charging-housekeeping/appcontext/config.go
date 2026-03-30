package appcontext

import (
	"os"
	"time"

	"go-ocs/internal/baseconfig"
	"go-ocs/internal/events"
	"go-ocs/internal/logging"
)

const (
	defaultStaleSessions   = 24 * time.Hour
	defaultTracePurge      = 36 * time.Hour
	defaultRatePlanCleanup = 30 * 24 * time.Hour
)

// Config is the top-level configuration for the charging-housekeeping binary.
type Config struct {
	Base        baseconfig.BaseConfig `yaml:"base"`
	Kafkaconfig *events.KafkaConfig  `yaml:"kafka"`
	// Operational thresholds — read from env vars; YAML fields are not populated.
	StaleSessions   time.Duration // STALE_SESSIONS_THRESHOLD; default 24h
	TracePurge      time.Duration // TRACE_PURGE_THRESHOLD;    default 36h
	RatePlanCleanup time.Duration // RATEPLAN_CLEANUP_THRESHOLD; default 720h
}

// NewConfig loads the Config from the given YAML file and overrides the three
// operational thresholds from environment variables if set.
func NewConfig(configFilename string) *Config {
	cfg := &Config{
		Kafkaconfig:     events.NewKafkaConfig(),
		StaleSessions:   defaultStaleSessions,
		TracePurge:      defaultTracePurge,
		RatePlanCleanup: defaultRatePlanCleanup,
	}

	if err := baseconfig.LoadConfig(configFilename, cfg); err != nil {
		logging.Fatal("Failed to load config", "err", err)
	}

	// Override thresholds from environment variables if set.
	if v := os.Getenv("STALE_SESSIONS_THRESHOLD"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.StaleSessions = d
		} else {
			logging.Warn("Invalid STALE_SESSIONS_THRESHOLD; using default", "value", v, "default", defaultStaleSessions)
		}
	}
	if v := os.Getenv("TRACE_PURGE_THRESHOLD"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.TracePurge = d
		} else {
			logging.Warn("Invalid TRACE_PURGE_THRESHOLD; using default", "value", v, "default", defaultTracePurge)
		}
	}
	if v := os.Getenv("RATEPLAN_CLEANUP_THRESHOLD"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.RatePlanCleanup = d
		} else {
			logging.Warn("Invalid RATEPLAN_CLEANUP_THRESHOLD; using default", "value", v, "default", defaultRatePlanCleanup)
		}
	}

	return cfg
}
