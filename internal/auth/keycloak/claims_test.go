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

func TestKeycloakClaims_Decode(t *testing.T) {
	// Build an unsigned token and verify KeycloakClaims can be populated via ParseWithClaims.
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

	decoded := &KeycloakClaims{}
	_, err = jwt.ParseWithClaims(tokenString, decoded, func(_ *jwt.Token) (interface{}, error) {
		return jwt.UnsafeAllowNoneSignatureType, nil
	})
	require.NoError(t, err)

	assert.Equal(t, "testuser", decoded.PreferredUsername)
	assert.Equal(t, "test@example.com", decoded.Email)
	assert.True(t, decoded.HasRealmRole("admin"))
	assert.True(t, decoded.HasClientRole("test-client", "viewer"))
}
