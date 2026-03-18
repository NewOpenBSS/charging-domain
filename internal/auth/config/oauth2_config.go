package config

import "time"

// KeycloakConfig holds all settings required to connect to and authenticate with a Keycloak server.
type KeycloakConfig struct {
	Enabled       bool           `yaml:"enabled"`
	IssuerURL     string         `yaml:"issuerUrl"`
	ClientID      string         `yaml:"clientId"`
	ClientSecret  string         `yaml:"clientSecret"`
	Audience      string         `yaml:"audience"`
	SkipTLSVerify bool           `yaml:"skipTLSVerify"`
	JWKSExpiry    *time.Duration `yaml:"jwksExpiry"`
}

// NewKeycloakConfig returns a KeycloakConfig with default values.
func NewKeycloakConfig() KeycloakConfig {
	return KeycloakConfig{
		Enabled: true,
	}
}
