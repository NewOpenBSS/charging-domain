package auth

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// buildFieldContext creates a minimal FieldContext for testing the deny-default middleware.
func buildFieldContext(objectName, fieldName string, directives ast.DirectiveList) *graphql.FieldContext {
	return &graphql.FieldContext{
		Object: objectName,
		Field: graphql.CollectedField{
			Field: &ast.Field{
				Name: fieldName,
				Definition: &ast.FieldDefinition{
					Name:       fieldName,
					Directives: directives,
				},
			},
		},
	}
}

func TestDenyByDefault_AnnotatedField_Passes(t *testing.T) {
	mw := DenyByDefaultFieldMiddleware(true)

	fc := buildFieldContext("Query", "listCarriers", ast.DirectiveList{
		{Name: "auth"},
	})
	ctx := graphql.WithFieldContext(context.Background(), fc)

	res, err := mw(ctx, successResolver)
	require.NoError(t, err)
	assert.Equal(t, "ok", res)
}

func TestDenyByDefault_UnannotatedField_Denied(t *testing.T) {
	mw := DenyByDefaultFieldMiddleware(true)

	fc := buildFieldContext("Query", "listCarriers", nil)
	ctx := graphql.WithFieldContext(context.Background(), fc)

	res, err := mw(ctx, successResolver)
	require.Error(t, err)
	assert.Nil(t, res)

	var gqlErr *gqlerror.Error
	require.ErrorAs(t, err, &gqlErr)
	assert.Equal(t, "FORBIDDEN", gqlErr.Extensions["code"])
}

func TestDenyByDefault_MutationField_Denied(t *testing.T) {
	mw := DenyByDefaultFieldMiddleware(true)

	fc := buildFieldContext("Mutation", "createCarrier", nil)
	ctx := graphql.WithFieldContext(context.Background(), fc)

	res, err := mw(ctx, successResolver)
	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDenyByDefault_IntrospectionField_Exempt(t *testing.T) {
	mw := DenyByDefaultFieldMiddleware(true)

	tests := []struct {
		name      string
		fieldName string
	}{
		{name: "__schema is exempt", fieldName: "__schema"},
		{name: "__type is exempt", fieldName: "__type"},
		{name: "_empty is exempt", fieldName: "_empty"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fc := buildFieldContext("Query", tc.fieldName, nil)
			ctx := graphql.WithFieldContext(context.Background(), fc)

			res, err := mw(ctx, successResolver)
			require.NoError(t, err)
			assert.Equal(t, "ok", res)
		})
	}
}

func TestDenyByDefault_NonRootObject_Passes(t *testing.T) {
	mw := DenyByDefaultFieldMiddleware(true)

	// A field on a nested type (not Query/Mutation) should always pass.
	fc := buildFieldContext("Carrier", "carrierName", nil)
	ctx := graphql.WithFieldContext(context.Background(), fc)

	res, err := mw(ctx, successResolver)
	require.NoError(t, err)
	assert.Equal(t, "ok", res)
}

func TestDenyByDefault_AuthDisabled_Bypass(t *testing.T) {
	mw := DenyByDefaultFieldMiddleware(false)

	// Unannotated field should pass when auth is disabled.
	fc := buildFieldContext("Query", "listCarriers", nil)
	ctx := graphql.WithFieldContext(context.Background(), fc)

	res, err := mw(ctx, successResolver)
	require.NoError(t, err)
	assert.Equal(t, "ok", res)
}

func TestDenyByDefault_NilFieldContext_Passes(t *testing.T) {
	mw := DenyByDefaultFieldMiddleware(true)

	// No field context — should pass (defensive).
	res, err := mw(context.Background(), successResolver)
	require.NoError(t, err)
	assert.Equal(t, "ok", res)
}
