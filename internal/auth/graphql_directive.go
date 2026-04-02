package auth

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"go-ocs/internal/auth/keycloak"
	"go-ocs/internal/backend/graphql/generated"
)

// NewGraphQLDirectiveConfig returns a DirectiveRoot with the @auth directive handler
// wired in. When authEnabled is false the handler is a no-op pass-through.
func NewGraphQLDirectiveConfig(authEnabled bool) generated.DirectiveRoot {
	return generated.DirectiveRoot{
		Auth: authDirectiveHandler(authEnabled),
	}
}

// authDirectiveHandler returns the gqlgen directive handler for @auth.
// The handler verifies that the caller is authenticated (has valid claims in
// context). It does not check specific permissions — that will be added later.
func authDirectiveHandler(authEnabled bool) func(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	return func(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
		if !authEnabled {
			return next(ctx)
		}

		claims, ok := keycloak.ClaimsFromContext(ctx)
		if !ok || claims == nil {
			return nil, &gqlerror.Error{
				Message: "unauthenticated",
				Extensions: map[string]any{
					"code": "UNAUTHENTICATED",
				},
			}
		}

		return next(ctx)
	}
}
