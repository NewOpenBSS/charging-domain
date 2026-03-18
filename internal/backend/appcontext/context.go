package appcontext

import (
	"go-ocs/internal/auth/keycloak"
	"go-ocs/internal/backend/services"
	"go-ocs/internal/store"
)

// AppContext is the dependency injection container for the charging-backend application.
type AppContext struct {
	Config     *BackendConfig
	Metrics    *AppMetrics
	Store      *store.Store
	Auth       *keycloak.Client // nil when auth.enabled = false
	CarrierSvc *services.CarrierService
}

// NewAppContext constructs a fully wired AppContext from the supplied config, store,
// and optional Keycloak client (nil disables authentication).
func NewAppContext(cfg *BackendConfig, s *store.Store, auth *keycloak.Client) *AppContext {
	return &AppContext{
		Config:     cfg,
		Metrics:    NewMetrics(),
		Store:      s,
		Auth:       auth,
		CarrierSvc: services.NewCarrierService(s),
	}
}
