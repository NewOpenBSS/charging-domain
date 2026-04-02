package auth

import "go-ocs/internal/auth/keycloak"

// Permission represents a named permission that can be required by an endpoint.
// Permission constants are defined by domain-specific packages, not here.
type Permission string

// HasPermission checks whether the given claims contain the specified permission.
// Returns false if claims is nil or the permissions slice is empty.
func HasPermission(claims *keycloak.KeycloakClaims, permission Permission) bool {
	if claims == nil {
		return false
	}
	target := string(permission)
	for _, p := range claims.Permissions {
		if p == target {
			return true
		}
	}
	return false
}
