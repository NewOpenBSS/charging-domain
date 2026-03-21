package appcontext

import (
	"go-ocs/internal/auth/keycloak"
	"go-ocs/internal/auth/tenant"
	"go-ocs/internal/backend/consumer"
	"go-ocs/internal/backend/services"
	"go-ocs/internal/events"
	"go-ocs/internal/quota"
	"go-ocs/internal/store"
)

// AppContext is the dependency injection container for the charging-backend application.
type AppContext struct {
	Config                   *BackendConfig
	Metrics                  *AppMetrics
	Store                    *store.Store
	Auth                     *keycloak.Client // nil when auth.enabled = false
	KafkaManager             *events.KafkaManager
	TenantResolver           *tenant.Resolver
	SubscriberConsumer       *consumer.SubscriberEventConsumer
	WholesaleConsumer        *consumer.WholesaleContractConsumer
	QuotaProvisioningConsumer *consumer.QuotaProvisioningConsumer
	CarrierSvc               *services.CarrierService
	ClassificationSvc        *services.ClassificationService
	NumberPlanSvc            *services.NumberPlanService
	RatePlanSvc              *services.RatePlanService
	QuotaSvc                 *services.QuotaService
	ChargingTraceSvc         *services.ChargingTraceService
}

// NewAppContext constructs a fully wired AppContext from the supplied config, store,
// Kafka manager, and optional Keycloak client (nil disables authentication).
func NewAppContext(cfg *BackendConfig, s *store.Store, kafka *events.KafkaManager, auth *keycloak.Client) *AppContext {
	quotaManager := quota.NewQuotaManager(*s, 3, kafka)
	subscriberStorer := consumer.NewStoreSubscriberAdapter(s)
	wholesaleStorer := consumer.NewStoreWholesaleAdapter(s)
	return &AppContext{
		Config:                    cfg,
		Metrics:                   NewMetrics(),
		Store:                     s,
		Auth:                      auth,
		KafkaManager:              kafka,
		TenantResolver:            tenant.NewResolver(s, cfg.Server.TenantRefreshInterval),
		SubscriberConsumer:        consumer.NewSubscriberEventConsumer(cfg.Kafkaconfig, subscriberStorer, subscriberEventTopic(cfg.Kafkaconfig)),
		WholesaleConsumer:         consumer.NewWholesaleContractConsumer(cfg.Kafkaconfig, wholesaleStorer, wholesaleContractEventTopic(cfg.Kafkaconfig)),
		QuotaProvisioningConsumer: consumer.NewQuotaProvisioningConsumer(cfg.Kafkaconfig, quotaManager, quotaProvisioningTopic(cfg.Kafkaconfig)),
		CarrierSvc:                services.NewCarrierService(s),
		ClassificationSvc:         services.NewClassificationService(s),
		NumberPlanSvc:             services.NewNumberPlanService(s),
		RatePlanSvc:               services.NewRatePlanService(s),
		QuotaSvc:                  services.NewQuotaService(quotaManager),
		ChargingTraceSvc:          services.NewChargingTraceService(s),
	}
}

// subscriberEventTopic resolves the subscriber-event topic name from the Kafka
// topics map, falling back to the canonical topic name if not configured.
func subscriberEventTopic(cfg *events.KafkaConfig) string {
	const defaultTopic = "public.subscriber-event"
	if cfg == nil || cfg.Topics == nil {
		return defaultTopic
	}
	if t, ok := cfg.Topics["subscriber-event"]; ok {
		return t
	}
	return defaultTopic
}

// wholesaleContractEventTopic resolves the wholesale-contract-event topic name from
// the Kafka topics map, falling back to the canonical topic name if not configured.
func wholesaleContractEventTopic(cfg *events.KafkaConfig) string {
	const defaultTopic = "public.wholesale-contract-event"
	if cfg == nil || cfg.Topics == nil {
		return defaultTopic
	}
	if t, ok := cfg.Topics["wholesale-contract-event"]; ok {
		return t
	}
	return defaultTopic
}

// quotaProvisioningTopic resolves the quota-provisioning topic name from the Kafka
// topics map, falling back to the canonical topic name if not configured.
func quotaProvisioningTopic(cfg *events.KafkaConfig) string {
	const defaultTopic = "public.quota-provisioning"
	if cfg == nil || cfg.Topics == nil {
		return defaultTopic
	}
	if t, ok := cfg.Topics["quota-provisioning"]; ok {
		return t
	}
	return defaultTopic
}
