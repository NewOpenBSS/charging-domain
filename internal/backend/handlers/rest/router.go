package rest

import (
	"net/http"
	"time"

	"go-ocs/internal/auth"
	"go-ocs/internal/auth/keycloak"
	"go-ocs/internal/backend/appcontext"
	"go-ocs/internal/logging"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter builds the Chi router for the REST API mounted at the configured path.
// All /api/ routes use SecureRouter to enforce explicit permission declarations.
func NewRouter(appCtx *appcontext.AppContext) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))
	r.Use(logging.Middleware)
	r.Use(keycloak.Middleware(appCtx.Auth))

	authEnabled := appCtx.Auth != nil

	r.Route(appCtx.Config.Server.RestPath, func(r chi.Router) {
		sr := auth.NewSecureRouter(r, authEnabled)
		sr.Get("/health", auth.Public(), func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
	})

	return r
}
