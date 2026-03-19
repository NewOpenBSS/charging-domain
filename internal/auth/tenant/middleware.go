package tenant

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
)

type contextKey string

// WholesaleIDContextKey is the key used to store the resolved wholesaler UUID
// in the request context.
const WholesaleIDContextKey contextKey = "tenant_wholesale_id"

// Middleware returns a Chi-compatible HTTP middleware that resolves the request's
// Host header to a wholesaler UUID and stores it in the context.
// If the Host cannot be resolved (unknown tenant or Resolver is nil) the request
// proceeds without a wholesale ID — downstream handlers decide whether to error.
func Middleware(r *Resolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if r != nil {
				if uid, ok := r.ResolveHost(req.Host); ok {
					req = req.WithContext(
						context.WithValue(req.Context(), WholesaleIDContextKey, uid),
					)
				}
			}
			next.ServeHTTP(w, req)
		})
	}
}

// WholesaleIDFromContext retrieves the wholesaler UUID stored by Middleware.
// Returns false if no wholesale ID is present (unknown host or auth disabled).
func WholesaleIDFromContext(ctx context.Context) (pgtype.UUID, bool) {
	uid, ok := ctx.Value(WholesaleIDContextKey).(pgtype.UUID)
	return uid, ok && uid.Valid
}
