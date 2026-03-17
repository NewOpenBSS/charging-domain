package graphql

import (
	"net/http"
	"time"

	"go-ocs/internal/auth/keycloak"
	"go-ocs/internal/backend/appcontext"
	"go-ocs/internal/logging"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter builds the Chi router for the GraphQL endpoint.
// Phase 1: mounts a placeholder handler and the GraphQL Playground.
// Phase 2 will wire in the generated gqlgen schema and resolvers.
func NewRouter(appCtx *appcontext.AppContext) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(logging.Middleware)
	r.Use(keycloak.Middleware(appCtx.Auth))

	graphqlPath := appCtx.Config.Server.GraphqlPath

	// Placeholder until gqlgen schema and resolvers are generated in Phase 2.
	r.Get(graphqlPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"GraphQL endpoint - schema not yet defined"}`))
	})

	r.Get(graphqlPath+"/playground", playground.Handler("Charging Admin", graphqlPath))

	logging.Info("GraphQL router configured", "path", graphqlPath)
	return r
}
