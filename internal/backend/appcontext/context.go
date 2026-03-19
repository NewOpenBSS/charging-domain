package appcontext

import (
	"go-ocs/internal/auth/keycloak"
	"go-ocs/internal/auth/tenant"
	"go-ocs/internal/backend/services"
	"go-ocs/internal/events"
	"go-ocs/internal/quota"
	"go-ocs/internal/store"
)

// AppContext is the dependency injection container for the charging-backend application.
type AppContext struct {
	Config            *BackendConfig
	Metrics           *AppMetrics
	Store             *store.Store
	Auth              *keycloak.Client // nil when auth.enabled = false
	KafkaManager      *events.KafkaManager
	TenantResolver    *tenant.Resolver
	CarrierSvc        *services.CarrierService
	ClassificationSvc *services.ClassificationService
	NumberPlanSvc     *services.NumberPlanService
	RatePlanSvc       *services.RatePlanService
	QuotaSvc          *services.QuotaService
}

// NewAppContext constructs a fully wired AppContext from the supplied config, store,
// Kafka manager, and optional Keycloak client (nil disables authentication).
func NewAppContext(cfg *BackendConfig, s *store.Store, kafka *events.KafkaManager, auth *keycloak.Client) *AppContext {
	quotaManager := quota.NewQuotaManager(*s, 3, kafka)
	return &AppContext{
		Config:            cfg,
		Metrics:           NewMetrics(),
		Store:             s,
		Auth:              auth,
		KafkaManager:      kafka,
		TenantResolver:    tenant.NewResolver(s, cfg.Server.TenantRefreshInterval),
		CarrierSvc:        services.NewCarrierService(s),
		ClassificationSvc: services.NewClassificationService(s),
		NumberPlanSvc:     services.NewNumberPlanService(s),
		RatePlanSvc:       services.NewRatePlanService(s),
		QuotaSvc:          services.NewQuotaService(quotaManager),
	}
}
