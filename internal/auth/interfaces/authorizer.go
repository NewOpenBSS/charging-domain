package interfaces

import (
	"context"
	"go-ocs/internal/auth/keycloak"
)

// Authorizer performs role-based access checks against extracted claims.
type Authorizer interface {
	RequireRealmRole(ctx context.Context, claims *keycloak.KeycloakClaims, role string) error
	RequireClientRole(ctx context.Context, claims *keycloak.KeycloakClaims, clientID, role string) error
}
