package appcontext

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "backend-config.yaml")

	yaml := `
base:
  appName: "test-backend"
  database:
    url: "postgres://localhost/test"
auth:
  enabled: false
`
	require.NoError(t, os.WriteFile(cfgFile, []byte(yaml), 0644))

	cfg := NewConfig(cfgFile)

	assert.Equal(t, ":8081", cfg.Server.Addr)
	assert.Equal(t, "/api/charging", cfg.Server.RestPath)
	assert.Equal(t, "/api/charging/graphql", cfg.Server.GraphqlPath)
	assert.Equal(t, 15*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 15*time.Second, cfg.Server.WriteTimeout)
	assert.False(t, cfg.Auth.Enabled)
}

func TestNewConfig_Override(t *testing.T) {
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "backend-config.yaml")

	yaml := `
base:
  appName: "custom-backend"
  database:
    url: "postgres://localhost/custom"
server:
  addr: ":9090"
  restPath: "/api/v2"
  graphqlPath: "/api/v2/graphql"
auth:
  enabled: false
`
	require.NoError(t, os.WriteFile(cfgFile, []byte(yaml), 0644))

	cfg := NewConfig(cfgFile)

	assert.Equal(t, ":9090", cfg.Server.Addr)
	assert.Equal(t, "/api/v2", cfg.Server.RestPath)
	assert.Equal(t, "/api/v2/graphql", cfg.Server.GraphqlPath)
	assert.Equal(t, "custom-backend", cfg.Base.AppName)
}
