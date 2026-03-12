package steps

import (
	"context"
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/logging"
)

func HandleError(dc *engine.ChargingContext, err error) error {
	//If we are not completed, we need to clean up resources
	for _, unit := range dc.Request.MultipleUnitUsage {

		// Check if we have a grant for this rating group
		grants, ok := dc.ChargingData.Grants[*unit.RatingGroup]
		if ok {
			for _, g := range grants {
				if g.InvocationSequenceNumber == *dc.Request.InvocationSequenceNumber {
					if dc.ChargingData.Subscriber != nil {
						err := dc.AppContext.QuotaManager.Release(context.Background(), dc.ChargingData.Subscriber.SubscriberId, g.GrantId)
						if err != nil {
							logging.Error("Failed to release quota", "err", err)
						}
					}
				}
				break
			}
		}
	}

	return err
}
