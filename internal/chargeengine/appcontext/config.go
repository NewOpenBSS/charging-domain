// Package main provides configuration loading and validation for the go-ocs service.
// It loads environment-based settings and merges them with a Diameter YAML configuration file.
package appcontext

import (
	_ "embed"
	"go-ocs/internal/baseconfig"
	"go-ocs/internal/events"
	"go-ocs/internal/logging"
	"time"

	"github.com/google/uuid"
)

// Config represents the top-level application configuration loaded from
// environment variables and supplemented by Diameter YAML configuration.
type Config struct {
	Base        baseconfig.BaseConfig `yaml:"base"`
	Engine      EngineConfig          `yaml:"engine"`
	Kafkaconfig *events.KafkaConfig   `yaml:"kafka"`
}

type EngineConfig struct {
	NationalDialCode      string        `yaml:"nationalDialCode"`
	Addr                  string        `yaml:"addr"`
	Path                  string        `yaml:"path"`
	SettlementPlanId      uuid.UUID     `yaml:"settlementPlanId"`
	DecimalDigits         int32         `yaml:"decimalDigits"`
	DefaultValidityWindow time.Duration `yaml:"defaultValidityWindow"`
	ScalingValidityWindow time.Duration `yaml:"scalingValidityWindow"`
}

func NewConfig(configFilename string) *Config {
	cfg := &Config{
		Engine: EngineConfig{
			Addr:                  ":8080",
			Path:                  "/api",
			NationalDialCode:      "64",
			SettlementPlanId:      uuid.New(),
			DecimalDigits:         22,
			DefaultValidityWindow: 3 * time.Hour,
			ScalingValidityWindow: 3 * time.Minute,
		},
		Kafkaconfig: events.NewKafkaConfig(),
	}

	if err := baseconfig.LoadConfig(configFilename, cfg); err != nil {
		logging.Fatal("Failed to load config", "err", err)
	}

	/*
		if cfg.Engine.Addr == "" {
			cfg.Engine.Addr = ":8080"
		}

		if cfg.Engine.Path == "" {
			cfg.Engine.Path = "/api"
		}

		if cfg.Engine.NationalDialCode == "" {
			cfg.Engine.NationalDialCode = "64"
		}
	*/

	return cfg
}
