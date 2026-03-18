package keycloak

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeycloakClaims_HasRealmRole(t *testing.T) {
	claims := &KeycloakClaims{
		RealmAccess: RealmAccess{Roles: []string{"admin", "user"}},
	}

	assert.True(t, claims.HasRealmRole("admin"))
	assert.True(t, claims.HasRealmRole("user"))
	assert.False(t, claims.HasRealmRole("superadmin"))
	assert.False(t, claims.HasRealmRole(""))
}

func TestKeycloakClaims_HasClientRole(t *testing.T) {
	claims := &KeycloakClaims{
		ResourceAccess: map[string]ResourceAccess{
			"my-client": {Roles: []string{"manager", "viewer"}},
		},
	}

	assert.True(t, claims.HasClientRole("my-client", "manager"))
	assert.True(t, claims.HasClientRole("my-client", "viewer"))
	assert.False(t, claims.HasClientRole("my-client", "admin"))
	assert.False(t, claims.HasClientRole("other-client", "manager"))
	assert.False(t, claims.HasClientRole("", "manager"))
}

func TestDecodeKeycloakClaims(t *testing.T) {
	// Build a test token using jwt/v5 with no signature (unsigned).
	sourceClaims := &KeycloakClaims{
		RealmAccess: RealmAccess{Roles: []string{"admin"}},
		ResourceAccess: map[string]ResourceAccess{
			"test-client": {Roles: []string{"viewer"}},
		},
		PreferredUsername: "testuser",
		Email:             "test@example.com",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodNone, sourceClaims)
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	decoded, err := decodeKeycloakClaims(tokenString)
	require.NoError(t, err)

	assert.Equal(t, "testuser", decoded.PreferredUsername)
	assert.Equal(t, "test@example.com", decoded.Email)
	assert.True(t, decoded.HasRealmRole("admin"))
	assert.True(t, decoded.HasClientRole("test-client", "viewer"))
}

func TestDecodeKeycloakClaims_InvalidToken(t *testing.T) {
	_, err := decodeKeycloakClaims("not.a.valid.jwt")
	assert.Error(t, err)
}
