package keycloak

import (
	"testing"

	authconfig "go-ocs/internal/auth/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractRealm(t *testing.T) {
	tests := []struct {
		name      string
		issuerURL string
		expected  string
	}{
		{
			name:      "standard keycloak issuer URL",
			issuerURL: "https://keycloak.example.com/realms/charging-realm",
			expected:  "charging-realm",
		},
		{
			name:      "simple realm name",
			issuerURL: "https://keycloak.example.com/realms/myrealm",
			expected:  "myrealm",
		},
		{
			name:      "no slashes",
			issuerURL: "myrealm",
			expected:  "myrealm",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractRealm(tc.issuerURL)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNewClient_AuthDisabled(t *testing.T) {
	cfg := authconfig.KeycloakConfig{
		Enabled: false,
	}

	client, err := NewClient(cfg)
	require.NoError(t, err)
	assert.Nil(t, client)
}

func TestNewClient_AuthEnabled(t *testing.T) {
	cfg := authconfig.KeycloakConfig{
		Enabled:      true,
		IssuerURL:    "https://keycloak.example.com/realms/test-realm",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	}

	client, err := NewClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "test-realm", client.realm)
	assert.Equal(t, cfg, client.config)
}
