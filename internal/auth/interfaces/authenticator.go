package interfaces

import (
	"context"
	"go-ocs/internal/auth/keycloak"
)

// Authenticator validates a raw Bearer token and returns the extracted claims.
type Authenticator interface {
	ValidateToken(ctx context.Context, rawToken string) (*keycloak.KeycloakClaims, error)
}
