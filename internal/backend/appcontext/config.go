package appcontext

import (
	"time"

	authconfig "go-ocs/internal/auth/config"
	"go-ocs/internal/baseconfig"
	"go-ocs/internal/logging"
)

// BackendConfig is the top-level configuration for the charging-backend application.
type BackendConfig struct {
	Base   baseconfig.BaseConfig     `yaml:"base"`
	Server ServerConfig              `yaml:"server"`
	Auth   authconfig.KeycloakConfig `yaml:"auth"`
}

// ServerConfig holds HTTP server settings for the charging-backend.
type ServerConfig struct {
	Addr         string        `yaml:"addr"`
	RestPath     string        `yaml:"restPath"`
	GraphqlPath  string        `yaml:"graphqlPath"`
	ReadTimeout  time.Duration `yaml:"readTimeout"`
	WriteTimeout time.Duration `yaml:"writeTimeout"`
}

// NewConfig loads the BackendConfig from the given YAML file, applying defaults first.
func NewConfig(configFilename string) *BackendConfig {
	cfg := &BackendConfig{
		Server: ServerConfig{
			Addr:         ":8081",
			RestPath:     "/api/charging",
			GraphqlPath:  "/api/charging/graphql",
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
		Auth: authconfig.NewKeycloakConfig(),
	}

	if err := baseconfig.LoadConfig(configFilename, cfg); err != nil {
		logging.Fatal("Failed to load backend config", "err", err)
	}

	if cfg.Auth.Enabled {
		if cfg.Auth.IssuerURL == "" {
			logging.Fatal("auth.issuerUrl is required when auth is enabled")
		}
		if cfg.Auth.ClientID == "" {
			logging.Fatal("auth.clientId is required when auth is enabled")
		}
	}

	return cfg
}
