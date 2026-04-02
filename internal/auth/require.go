package auth

import (
	"net/http"

	"go-ocs/internal/auth/keycloak"
)

// Require returns a Chi-compatible HTTP middleware that enforces the caller
// holds at least one of the given permissions. If authEnabled is false the
// middleware is a no-op pass-through, mirroring the pattern used by keycloak.Middleware.
//
// Response codes:
//   - 401 Unauthorized — no claims in context (unauthenticated)
//   - 403 Forbidden    — claims present but none of the required permissions match
func Require(authEnabled bool, permissions ...Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authEnabled {
				next.ServeHTTP(w, r)
				return
			}

			claims, ok := keycloak.ClaimsFromContext(r.Context())
			if !ok || claims == nil {
				http.Error(w, "unauthenticated", http.StatusUnauthorized)
				return
			}

			for _, p := range permissions {
				if HasPermission(claims, p) {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, "forbidden: insufficient permissions", http.StatusForbidden)
		})
	}
}
