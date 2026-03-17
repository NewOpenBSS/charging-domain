package engine

import (
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/engine/business/interfaces"
	"go-ocs/internal/chargeengine/model"
	"go-ocs/internal/nchf"
	"time"
)

// ChargingContext represents a context for managing charging processes including request, response, and associated data.
type ChargingContext struct {
	// sessionId represents the unique identifier of the charging session.
	SessionId string

	// AppContext provides application-level configuration, metrics, and data storage access.
	AppContext *appcontext.AppContext

	// serviceCtx provides access to the service-level configuration and data storage.
	Infra interfaces.Infrastructure

	// startTime represents the start time of the charging process.
	StartTime time.Time

	// request represents the charging data request associated with the charging context.
	Request *nchf.ChargingDataRequest

	// response represents the charging data response associated with the charging context.
	Response *nchf.ChargingDataResponse

	// chargingData represents the charging data associated with the charging context.
	ChargingData *model.ChargingData

	// ratingGroupsUnitsUsed tracks rating group IDs for which units have been consumed.
	RatingGroupsUnitsUsed []int
}

// NewChargingContext creates a new ChargingContext instance with the provided request.
// now is the UTC timestamp captured at the transport boundary and used as the single
// source of truth for all time-dependent operations in the pipeline.
func NewChargingContext(appContext *appcontext.AppContext, infra interfaces.Infrastructure, sessionId string, request *nchf.ChargingDataRequest, now time.Time) *ChargingContext {
	return &ChargingContext{
		StartTime:    now,
		SessionId:    sessionId,
		AppContext:   appContext,
		Infra:        infra,
		Request:      request,
		Response:     nchf.NewChargingDataResponse(),
		ChargingData: model.NewChargingData(),
	}
}
