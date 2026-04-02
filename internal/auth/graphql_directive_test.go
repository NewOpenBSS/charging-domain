package auth

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"go-ocs/internal/auth/keycloak"
)

// ctxWithClaims returns a context with the given KeycloakClaims injected.
func ctxWithClaims(claims *keycloak.KeycloakClaims) context.Context {
	return context.WithValue(context.Background(), keycloak.ClaimsContextKey, claims)
}

// successResolver returns the sentinel value "ok".
func successResolver(ctx context.Context) (any, error) {
	return "ok", nil
}

func TestAuthDirective_Authorised(t *testing.T) {
	directives := NewGraphQLDirectiveConfig(true)

	ctx := ctxWithClaims(&keycloak.KeycloakClaims{
		Permissions: []string{"read", "write"},
	})

	res, err := directives.Auth(ctx, nil, successResolver, []string{"read"})
	require.NoError(t, err)
	assert.Equal(t, "ok", res)
}

func TestAuthDirective_Unauthorised(t *testing.T) {
	directives := NewGraphQLDirectiveConfig(true)

	ctx := ctxWithClaims(&keycloak.KeycloakClaims{
		Permissions: []string{"read"},
	})

	res, err := directives.Auth(ctx, nil, successResolver, []string{"admin"})
	require.Error(t, err)
	assert.Nil(t, res)

	var gqlErr *gqlerror.Error
	require.ErrorAs(t, err, &gqlErr)
	assert.Equal(t, "FORBIDDEN", gqlErr.Extensions["code"])
}

func TestAuthDirective_Unauthenticated(t *testing.T) {
	directives := NewGraphQLDirectiveConfig(true)

	// No claims in context.
	res, err := directives.Auth(context.Background(), nil, successResolver, []string{"read"})
	require.Error(t, err)
	assert.Nil(t, res)

	var gqlErr *gqlerror.Error
	require.ErrorAs(t, err, &gqlErr)
	assert.Equal(t, "UNAUTHENTICATED", gqlErr.Extensions["code"])
}

func TestAuthDirective_AuthDisabled(t *testing.T) {
	directives := NewGraphQLDirectiveConfig(false)

	// No claims, but auth disabled — should pass through.
	res, err := directives.Auth(context.Background(), nil, successResolver, []string{"admin"})
	require.NoError(t, err)
	assert.Equal(t, "ok", res)
}

func TestAuthDirective_MultiplePermissions_AnyMatch(t *testing.T) {
	directives := NewGraphQLDirectiveConfig(true)

	ctx := ctxWithClaims(&keycloak.KeycloakClaims{
		Permissions: []string{"superadmin"},
	})

	res, err := directives.Auth(ctx, nil, successResolver, []string{"admin", "superadmin"})
	require.NoError(t, err)
	assert.Equal(t, "ok", res)
}

func TestNewGraphQLDirectiveConfig_ReturnsDirectiveRoot(t *testing.T) {
	config := NewGraphQLDirectiveConfig(true)
	assert.NotNil(t, config.Auth, "Auth directive handler should be set")
}

// Verify the handler function conforms to the expected gqlgen signature.
var _ func(ctx context.Context, obj any, next graphql.Resolver, permissions []string) (any, error) = NewGraphQLDirectiveConfig(true).Auth
