package keycloak

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

// ClaimsContextKey is the key used to store KeycloakClaims in the request context.
const ClaimsContextKey contextKey = "keycloak_claims"

// Middleware returns a Chi-compatible HTTP middleware that validates Bearer tokens.
// If auth is disabled (client is nil) the middleware is a no-op pass-through.
func Middleware(client *Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if client == nil {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing Authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				http.Error(w, "invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			rawToken := parts[1]

			claims, err := client.ValidateToken(r.Context(), rawToken)
			if err != nil {
				http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext retrieves KeycloakClaims previously injected by Middleware.
func ClaimsFromContext(ctx context.Context) (*KeycloakClaims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*KeycloakClaims)
	return claims, ok
}
