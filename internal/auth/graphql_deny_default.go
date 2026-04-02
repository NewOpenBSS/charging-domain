package auth

import (
	"context"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"go-ocs/internal/logging"
)

// DenyByDefaultFieldMiddleware returns a gqlgen field middleware that rejects any
// query or mutation field that is NOT annotated with the @auth directive.
// Introspection fields (__schema, __type) and the _empty placeholder are exempt.
// When authEnabled is false the middleware is a no-op.
func DenyByDefaultFieldMiddleware(authEnabled bool) graphql.FieldMiddleware {
	return func(ctx context.Context, next graphql.Resolver) (any, error) {
		if !authEnabled {
			return next(ctx)
		}

		fc := graphql.GetFieldContext(ctx)
		if fc == nil {
			return next(ctx)
		}

		// Only enforce on top-level Query and Mutation fields.
		objectName := fc.Object
		if objectName != "Query" && objectName != "Mutation" {
			return next(ctx)
		}

		fieldName := fc.Field.Name

		// Exempt introspection and placeholder fields.
		if isExemptField(fieldName) {
			return next(ctx)
		}

		// Check if the field definition has the @auth directive.
		if fc.Field.Definition != nil {
			for _, d := range fc.Field.Definition.Directives {
				if d.Name == "auth" {
					return next(ctx)
				}
			}
		}

		logging.Warn("deny-by-default: rejecting unannotated field",
			"object", objectName,
			"field", fieldName,
		)

		return nil, &gqlerror.Error{
			Message: "forbidden: operation not annotated with @auth directive",
			Extensions: map[string]any{
				"code": "FORBIDDEN",
			},
		}
	}
}

// isExemptField returns true for fields that should not require @auth annotation.
func isExemptField(name string) bool {
	if strings.HasPrefix(name, "__") {
		return true
	}
	if name == "_empty" {
		return true
	}
	return false
}
