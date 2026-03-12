package steps

import (
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/engine/business"
	"go-ocs/internal/logging"
)

func Classify(dc *engine.ChargingContext) error {

	logging.Debug("ClassifyStep - Classifying the request")

	classify, err := business.ClassifyService(dc)
	if err != nil {
		return err
	}

	dc.ChargingData.Classifications = classify

	if logging.IsDebug() {
		for rg, c := range classify {
			logging.Debug("ClassifyStep - Classified service", "rg", rg, "rateKey", c.Ratekey.String())
		}
	}

	return nil
}
