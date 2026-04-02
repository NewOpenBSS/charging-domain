package auth

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-ocs/internal/auth/keycloak"
)

func TestHasPermission_Present(t *testing.T) {
	claims := &keycloak.KeycloakClaims{
		Permissions: []string{"read", "write", "admin"},
	}
	assert.True(t, HasPermission(claims, "read"))
	assert.True(t, HasPermission(claims, "write"))
	assert.True(t, HasPermission(claims, "admin"))
}

func TestHasPermission_Absent(t *testing.T) {
	claims := &keycloak.KeycloakClaims{
		Permissions: []string{"read", "write"},
	}
	assert.False(t, HasPermission(claims, "admin"))
	assert.False(t, HasPermission(claims, "delete"))
	assert.False(t, HasPermission(claims, ""))
}

func TestHasPermission_NilClaims(t *testing.T) {
	assert.False(t, HasPermission(nil, "read"))
}

func TestHasPermission_EmptyPermissions(t *testing.T) {
	claims := &keycloak.KeycloakClaims{
		Permissions: []string{},
	}
	assert.False(t, HasPermission(claims, "read"))
}

func TestHasPermission_NilPermissions(t *testing.T) {
	claims := &keycloak.KeycloakClaims{}
	assert.False(t, HasPermission(claims, "read"))
}

func TestKeycloakClaims_PermissionsField_Deserialise(t *testing.T) {
	// Build an unsigned token with permissions and verify the field deserialises correctly.
	sourceClaims := &keycloak.KeycloakClaims{
		Permissions:      []string{"charging:read", "charging:write"},
		PreferredUsername: "testuser",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodNone, sourceClaims)
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	decoded := &keycloak.KeycloakClaims{}
	_, err = jwt.ParseWithClaims(tokenString, decoded, func(_ *jwt.Token) (interface{}, error) {
		return jwt.UnsafeAllowNoneSignatureType, nil
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"charging:read", "charging:write"}, decoded.Permissions)
	assert.Equal(t, "testuser", decoded.PreferredUsername)
}

func TestKeycloakClaims_PermissionsField_AbsentInJWT(t *testing.T) {
	// When the JWT has no permissions claim, the field should be nil/empty.
	sourceClaims := &keycloak.KeycloakClaims{
		PreferredUsername: "testuser",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodNone, sourceClaims)
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	decoded := &keycloak.KeycloakClaims{}
	_, err = jwt.ParseWithClaims(tokenString, decoded, func(_ *jwt.Token) (interface{}, error) {
		return jwt.UnsafeAllowNoneSignatureType, nil
	})
	require.NoError(t, err)

	assert.Empty(t, decoded.Permissions)
	assert.False(t, HasPermission(decoded, "anything"))
}
