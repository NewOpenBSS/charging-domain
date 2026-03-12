package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/model"
	"go-ocs/internal/chargeengine/ocserrors"
	"go-ocs/internal/logging"
	"go-ocs/internal/nchf"
)

func handleRetransmission(dc *engine.ChargingContext) error {
	if dc.Request.RetransmissionIndicator != nil && *dc.Request.RetransmissionIndicator {
		trace, traceErr := dc.AppContext.Store.Q.FindChargingTraceByIdSeqNr(
			context.Background(),
			*dc.Request.ChargingId,
			int32(*dc.Request.InvocationSequenceNumber),
		)

		if traceErr != nil {
			return fmt.Errorf("retransmit requested but trace lookup failed for chargingId=%s seqNr=%d: %w",
				*dc.Request.ChargingId,
				*dc.Request.InvocationSequenceNumber,
				traceErr,
			)
		}

		resp := nchf.ChargingDataResponse{}
		err := json.Unmarshal(trace.Response, &resp)
		if err != nil {
			return fmt.Errorf("Failed to unmarshal ChargingData: %s", err)
		}

		logging.Debug("Retransmitting ChargingData for", "chargingId=", *dc.Request.ChargingId, "seq", *dc.Request.InvocationSequenceNumber)

		return ocserrors.CreateRetransmit(&resp)
	}

	return nil
}

func CreateChargeDataStep(dc *engine.ChargingContext) error {

	// Check for a retransmission request
	if retransmission := handleRetransmission(dc); retransmission != nil {
		return retransmission
	}

	if _, err := dc.AppContext.Store.Q.GetChargingDataByChargeId(context.Background(), *dc.Request.ChargingId); err == nil {
		return fmt.Errorf("ChargingData already exists for ChargingID: %s", *dc.Request.ChargingId)
	}

	dc.ChargingData = model.NewChargingData()
	logging.Debug("Creating new ChargingData for", "chargingId=", *dc.Request.ChargingId, "seq", *dc.Request.InvocationSequenceNumber)

	return nil
}

func LoadChargeDataStep(dc *engine.ChargingContext) error {

	// Check for a retransmission request
	if retransmission := handleRetransmission(dc); retransmission != nil {
		return retransmission
	}

	rec, err := dc.AppContext.Store.Q.GetChargingDataByChargeId(context.Background(), *dc.Request.ChargingId)
	if err != nil {
		return fmt.Errorf("Failed to load ChargingData: %s", err)
	}

	if rec.SequenceNumber >= *dc.Request.InvocationSequenceNumber {
		return ocserrors.CreateInvalidReferenced(fmt.Sprintf("Duplicate invocation without retransmission flag. Expected new sequence number, received: %d", *dc.Request.InvocationSequenceNumber))
	}

	dc.ChargingData = model.NewChargingData()
	err = json.Unmarshal(rec.ChargeData, &dc.ChargingData)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal ChargingData: %s", err)
	}

	dc.ChargingData.NewRecord = false
	logging.Debug("Loaded ChargingData for", "chargingId=", *dc.Request.ChargingId, "seq", *dc.Request.InvocationSequenceNumber)

	return nil
}

func SaveChargeDataStep(dc *engine.ChargingContext) error {

	// Marshal ChargingData
	chargeData, err := json.Marshal(dc.ChargingData)
	if err != nil {
		return ocserrors.CreateGeneralError(fmt.Sprintf("Failed to marshal ChargingData: %s", err))
	}

	// Save ChargingData
	if dc.ChargingData.NewRecord {
		err = dc.AppContext.Store.Q.CreateChargeData(context.Background(), *dc.Request.ChargingId, int64(*dc.Request.InvocationSequenceNumber), chargeData)
	} else {
		err = dc.AppContext.Store.Q.UpdateChargeData(context.Background(), *dc.Request.ChargingId, int64(*dc.Request.InvocationSequenceNumber), chargeData)
	}
	if err != nil {
		return ocserrors.CreateGeneralError(fmt.Sprintf("Failed to saving ChargingData: %s", err))
	}

	logging.Debug("Saving ChargingData for", "chargingId=", *dc.Request.ChargingId, "seq", *dc.Request.InvocationSequenceNumber)

	return nil
}

func ReleaseChargeDataStep(dc *engine.ChargingContext) error {

	for grant := range dc.ChargingData.Grants {
		logging.Debug("Releasing grant", "grantId=", grant)
	}

	// Save ChargingData
	err := dc.AppContext.Store.Q.DeleteChargeDate(context.Background(), *dc.Request.ChargingId)
	if err != nil {
		return ocserrors.CreateGeneralError(fmt.Sprintf("Failed to deleting ChargingData: %s", err))
	}

	logging.Debug("Deleting ChargingData for", "chargingId=", *dc.Request.ChargingId)

	return nil
}
