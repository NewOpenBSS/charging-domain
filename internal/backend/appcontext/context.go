package appcontext

import (
	"go-ocs/internal/auth/keycloak"
	"go-ocs/internal/store"
)

// AppContext is the dependency injection container for the charging-backend application.
type AppContext struct {
	Config  *BackendConfig
	Metrics *AppMetrics
	Store   *store.Store
	Auth    *keycloak.Client // nil when auth.enabled = false
}
