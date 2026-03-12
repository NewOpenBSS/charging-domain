package chargeengine

import (
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/engine/business/interfaces"
	"go-ocs/internal/chargeengine/engine/steps"
	"go-ocs/internal/nchf"
)

func ProcessCharging(appCtx *appcontext.AppContext, infra interfaces.Infrastructure, sessionId string, request *nchf.ChargingDataRequest) (*nchf.ChargingDataResponse, error) {

	chargingCtx := engine.NewChargingContext(appCtx, infra, sessionId, request)

	// Create a new charge data record
	if err := steps.CreateChargeDataStep(chargingCtx); err != nil {
		return nil, err
	}

	// Authenticate the request
	if err := steps.Authenticate(chargingCtx); err != nil {
		return nil, err
	}

	// Classify the request
	if err := steps.Classify(chargingCtx); err != nil {
		return nil, err
	}

	// Rate the request
	if err := steps.Rate(chargingCtx); err != nil {
		return nil, err
	}

	// Build the response
	if err := steps.BuildResponse(chargingCtx); err != nil {
		return nil, err
	}

	//Save the charge data record
	if err := steps.SaveChargeDataStep(chargingCtx); err != nil {
		return nil, err
	}

	//Save the trace
	if err := steps.CreateTrace(chargingCtx); err != nil {
		return nil, err
	}

	// Return the response
	return chargingCtx.Response, nil
}

func ProcessOneTimeCharging(appCtx *appcontext.AppContext, infra interfaces.Infrastructure, sessionId string, request *nchf.ChargingDataRequest) (*nchf.ChargingDataResponse, error) {

	chargingCtx := engine.NewChargingContext(appCtx, infra, sessionId, request)

	// Create a new charge data record
	if err := steps.CreateChargeDataStep(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Authenticate the request
	if err := steps.Authenticate(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}
	
	// Classify the request
	if err := steps.Classify(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Rate the request
	if err := steps.Rate(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Account the request
	if err := steps.Accounting(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Build the response
	if err := steps.BuildResponse(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Delete the charge data record
	if err := steps.ReleaseChargeDataStep(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	//Save the trace
	if err := steps.CreateTrace(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Return the response
	return chargingCtx.Response, nil
}

func UpdateChargingData(appCtx *appcontext.AppContext, infra interfaces.Infrastructure, sessionId string, request *nchf.ChargingDataRequest) (*nchf.ChargingDataResponse, error) {
	chargingCtx := engine.NewChargingContext(appCtx, infra, sessionId, request)

	// Create a new charge data record
	if err := steps.LoadChargeDataStep(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Account the request
	if err := steps.Accounting(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Classify the request
	if err := steps.Classify(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Rate the request
	if err := steps.Rate(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Build the response
	if err := steps.BuildResponse(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Save the charge data record
	if err := steps.SaveChargeDataStep(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Create a trace of the request and result
	if err := steps.CreateTrace(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Return the response
	return chargingCtx.Response, nil
}

func ReleaseChargingData(appCtx *appcontext.AppContext, infra interfaces.Infrastructure, sessionId string, request *nchf.ChargingDataRequest) (*nchf.ChargingDataResponse, error) {
	chargingCtx := engine.NewChargingContext(appCtx, infra, sessionId, request)

	// Create a new charge data record
	if err := steps.LoadChargeDataStep(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Account the request
	if err := steps.Accounting(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Build the response
	if err := steps.BuildResponse(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Save the charge data record
	if err := steps.ReleaseChargeDataStep(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Create a trace of the request and result
	if err := steps.CreateTrace(chargingCtx); err != nil {
		return nil, steps.HandleError(chargingCtx, err)
	}

	// Return the response
	return chargingCtx.Response, nil
}
