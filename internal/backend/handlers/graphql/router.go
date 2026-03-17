package graphql

import (
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"go-ocs/internal/auth/keycloak"
	"go-ocs/internal/backend/appcontext"
	"go-ocs/internal/backend/graphql/generated"
	"go-ocs/internal/backend/resolvers"
	"go-ocs/internal/logging"
)

// NewRouter builds the Chi router for the GraphQL endpoint.
// It mounts the gqlgen-generated server at graphqlPath and the GraphQL Playground
// at graphqlPath+"/playground".
func NewRouter(appCtx *appcontext.AppContext) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(logging.Middleware)
	r.Use(keycloak.Middleware(appCtx.Auth))

	graphqlPath := appCtx.Config.Server.GraphqlPath

	resolver := &resolvers.Resolver{
		CarrierSvc: appCtx.CarrierSvc,
	}

	srv := handler.NewDefaultServer(
		generated.NewExecutableSchema(generated.Config{Resolvers: resolver}),
	)

	r.Handle(graphqlPath, srv)
	r.Get(graphqlPath+"/playground", playground.Handler("Charging Admin", graphqlPath))

	logging.Info("GraphQL router configured", "path", graphqlPath)
	return r
}
