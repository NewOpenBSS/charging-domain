package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/ocserrors"
	"go-ocs/internal/store/sqlc"
	"time"
)

func CreateTrace(dc *engine.ChargingContext) error {

	executionTime := time.Since(dc.StartTime).Milliseconds()

	req, err := json.Marshal(dc.Request)
	if err != nil {
		return ocserrors.CreateGeneralError(fmt.Sprintf("Failed to marshal ChargingData: %s", err))
	}

	resp, err := json.Marshal(dc.Response)
	if err != nil {
		return ocserrors.CreateGeneralError(fmt.Sprintf("Failed to marshal ChargingData: %s", err))
	}

	rec := sqlc.CreateChargingTraceParams{
		ChargingID:    *dc.Request.ChargingId,
		SequenceNr:    int32(*dc.Request.InvocationSequenceNumber),
		Msisdn:        dc.ChargingData.Subscriber.Msisdn,
		Request:       req,
		Response:      resp,
		ExecutionTime: executionTime,
	}

	dc.Response.Runtime = &executionTime
	_, err = dc.AppContext.Store.Q.CreateChargingTrace(context.Background(), rec)
	if err != nil {
		return ocserrors.CreateGeneralError(fmt.Sprintf("Failed to save ChargingTrace: %s", err))
	}

	return nil
}
