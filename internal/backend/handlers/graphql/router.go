package graphql

import (
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vektah/gqlparser/v2/ast"

	"go-ocs/internal/auth/keycloak"
	"go-ocs/internal/auth/tenant"
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
	r.Use(tenant.Middleware(appCtx.TenantResolver))

	graphqlPath := appCtx.Config.Server.GraphqlPath

	resolver := &resolvers.Resolver{
		CarrierSvc:        appCtx.CarrierSvc,
		ClassificationSvc: appCtx.ClassificationSvc,
		NumberPlanSvc:     appCtx.NumberPlanSvc,
		RatePlanSvc:       appCtx.RatePlanSvc,
	}

	srv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.Use(extension.Introspection{})

	r.Handle("/", srv)
	r.Get("/playground", playground.Handler("Charging Admin", graphqlPath))

	logging.Info("GraphQL router configured", "path", graphqlPath)
	return r
}
