package keycloak

import (
	"context"
	"fmt"
	"go-ocs/internal/auth/config"
	"go-ocs/internal/logging"

	"github.com/Nerzal/gocloak/v13"
)

// Client wraps gocloak and provides token validation and user attribute lookup.
type Client struct {
	gocloak *gocloak.GoCloak
	config  config.KeycloakConfig
	realm   string
}

// NewClient initialises the gocloak client using the provided KeycloakConfig.
// Returns nil without error when auth is disabled.
func NewClient(cfg config.KeycloakConfig) (*Client, error) {
	if !cfg.Enabled {
		logging.Warn("Keycloak auth is DISABLED - all requests will be unauthenticated")
		return nil, nil
	}

	gc := gocloak.NewClient(cfg.IssuerURL)
	realm := extractRealm(cfg.IssuerURL)

	logging.Info("Keycloak client initialised", "issuer", cfg.IssuerURL, "realm", realm, "clientId", cfg.ClientID)

	return &Client{
		gocloak: gc,
		config:  cfg,
		realm:   realm,
	}, nil
}

// ValidateToken introspects the token via Keycloak and returns extracted claims.
func (c *Client) ValidateToken(ctx context.Context, rawToken string) (*KeycloakClaims, error) {
	result, err := c.gocloak.RetrospectToken(ctx, rawToken, c.config.ClientID, c.config.ClientSecret, c.realm)
	if err != nil {
		return nil, fmt.Errorf("token introspection failed: %w", err)
	}

	if result.Active == nil || !*result.Active {
		return nil, fmt.Errorf("token is not active")
	}

	claims, err := decodeKeycloakClaims(rawToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}

	return claims, nil
}

// extractRealm parses the realm name from a Keycloak issuer URL.
// e.g. https://keycloak.example.com/realms/charging-realm -> "charging-realm"
func extractRealm(issuerURL string) string {
	for i := len(issuerURL) - 1; i >= 0; i-- {
		if issuerURL[i] == '/' {
			return issuerURL[i+1:]
		}
	}
	return issuerURL
}
