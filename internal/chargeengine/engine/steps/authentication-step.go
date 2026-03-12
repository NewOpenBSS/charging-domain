package steps

import (
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/ocserrors"
	"go-ocs/internal/logging"
	"strings"
)

func Authenticate(dc *engine.ChargingContext) error {

	if dc.Request.SubscriberIdentifier == nil {
		return ocserrors.CreateUnknownSubscriber("Subscriber identifier is nil")
	}

	// Nationalise the MSISDN
	msisdn := *dc.Request.SubscriberIdentifier
	if len(msisdn) > 0 && msisdn[0] != '0' {
		nationalDialCode := dc.AppContext.Config.Engine.NationalDialCode

		if strings.HasPrefix(msisdn, nationalDialCode) {
			msisdn = "0" + msisdn[len(nationalDialCode):]
		} else {
			msisdn = "0" + msisdn
		}
	}

	logging.Debug("Authentication Step:: Authenticating", "msisdn", msisdn)
	subscriber, err := dc.Infra.FindSubscriber(msisdn)
	if err != nil {
		return ocserrors.CreateUnknownSubscriber(err.Error())
	}

	dc.ChargingData.Subscriber = subscriber

	logging.Debug("Authentication Step:: Subscriber known and active", "msisdn", msisdn)
	return nil
}
