package keycloak

import (
	"context"
	"fmt"
	"time"

	"go-ocs/internal/auth/config"
	"go-ocs/internal/logging"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
)

// Client validates JWTs locally using the Keycloak JWKS endpoint.
// No client secret is required — works with public and confidential clients alike.
type Client struct {
	jwks   *keyfunc.JWKS
	config config.KeycloakConfig
}

// NewClient initialises the JWKS-backed JWT validator.
// Returns nil without error when auth is disabled.
func NewClient(cfg config.KeycloakConfig) (*Client, error) {
	if !cfg.Enabled {
		logging.Warn("Keycloak auth is DISABLED - all requests will be unauthenticated")
		return nil, nil
	}

	jwksURL := cfg.IssuerURL + "/protocol/openid-connect/certs"

	options := keyfunc.Options{
		RefreshInterval:   time.Hour,
		RefreshRateLimit:  5 * time.Minute,
		RefreshTimeout:    10 * time.Second,
		RefreshUnknownKID: true,
	}

	jwks, err := keyfunc.Get(jwksURL, options)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS from %s: %w", jwksURL, err)
	}

	logging.Info("Keycloak client initialised", "issuer", cfg.IssuerURL, "jwks", jwksURL)

	return &Client{
		jwks:   jwks,
		config: cfg,
	}, nil
}

// ValidateToken verifies the JWT signature using the JWKS public keys and returns extracted claims.
func (c *Client) ValidateToken(_ context.Context, rawToken string) (*KeycloakClaims, error) {
	claims := &KeycloakClaims{}

	token, err := jwt.ParseWithClaims(rawToken, claims, c.jwks.Keyfunc)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	return claims, nil
}

// Stop releases the background JWKS refresh goroutine.
// Call this during application shutdown.
func (c *Client) Stop() {
	if c.jwks != nil {
		c.jwks.EndBackground()
	}
}
