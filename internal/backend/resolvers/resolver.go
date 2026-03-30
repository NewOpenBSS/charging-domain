package resolvers

// This file will not be regenerated automatically.
//
// It serves as the dependency injection container for all resolvers.
// Add service dependencies here as new resources are implemented.

import "go-ocs/internal/backend/services"

// Resolver is the root dependency container injected into all generated resolver types.
type Resolver struct {
	CarrierSvc           *services.CarrierService
	ClassificationSvc    *services.ClassificationService
	NumberPlanSvc        *services.NumberPlanService
	RatePlanSvc          *services.RatePlanService
	QuotaSvc             *services.QuotaService
	ChargingTraceSvc     *services.ChargingTraceService
	DestinationGroupSvc  *services.DestinationGroupService
	SourceGroupSvc       *services.SourceGroupService
}
