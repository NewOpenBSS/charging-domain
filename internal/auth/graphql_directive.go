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
// The handler checks that the caller has at least one of the declared permissions.
func authDirectiveHandler(authEnabled bool) func(ctx context.Context, obj any, next graphql.Resolver, permissions []string) (any, error) {
	return func(ctx context.Context, obj any, next graphql.Resolver, permissions []string) (any, error) {
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

		for _, p := range permissions {
			if HasPermission(claims, Permission(p)) {
				return next(ctx)
			}
		}

		return nil, &gqlerror.Error{
			Message: "forbidden: insufficient permissions",
			Extensions: map[string]any{
				"code": "FORBIDDEN",
			},
		}
	}
}
