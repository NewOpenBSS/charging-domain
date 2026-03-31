package keycloak

import (
	"github.com/golang-jwt/jwt/v5"
)

// KeycloakClaims represents the full set of claims extracted from a Keycloak JWT.
// Standard JWT claims are embedded; Keycloak-specific claims are mapped explicitly.
type KeycloakClaims struct {
	jwt.RegisteredClaims

	// RealmAccess holds roles assigned at the Keycloak realm level.
	RealmAccess RealmAccess `json:"realm_access"`

	// ResourceAccess holds roles assigned per OAuth2 client/resource.
	ResourceAccess map[string]ResourceAccess `json:"resource_access"`

	// Standard OIDC user info fields.
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`

	// Groups contains custom user/group attributes populated via Keycloak token mappers.
	Groups []string `json:"groups"`
}

// RealmAccess holds the realm-level roles assigned to the token subject.
type RealmAccess struct {
	Roles []string `json:"roles"`
}

// ResourceAccess holds the client-level roles assigned to the token subject.
type ResourceAccess struct {
	Roles []string `json:"roles"`
}

// HasRealmRole checks whether the token contains the given realm-level role.
func (c *KeycloakClaims) HasRealmRole(role string) bool {
	for _, r := range c.RealmAccess.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasClientRole checks whether the token contains the given role for a specific client.
func (c *KeycloakClaims) HasClientRole(clientID, role string) bool {
	access, ok := c.ResourceAccess[clientID]
	if !ok {
		return false
	}
	for _, r := range access.Roles {
		if r == role {
			return true
		}
	}
	return false
}

