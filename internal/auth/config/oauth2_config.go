package config

import "time"

// KeycloakConfig holds all settings required to connect to and authenticate with a Keycloak server.
// JWKS-based validation is used — no client secret is required.
type KeycloakConfig struct {
	Enabled       bool           `yaml:"enabled"`
	IssuerURL     string         `yaml:"issuerUrl"`
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
