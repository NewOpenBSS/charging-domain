package steps

import (
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/charging"
	"go-ocs/internal/logging"
	"go-ocs/internal/nchf"
)

func BuildResponse(dc *engine.ChargingContext) error {

	logging.Debug("BuildResponseStep - Building Charging Data response")

	sessionFailover := "FAILOVER_NOT_SUPPORTED"
	response := nchf.ChargingDataResponse{}
	response.InvocationSequenceNumber = dc.Request.InvocationSequenceNumber
	response.InvocationTimeStamp = dc.Request.InvocationTimeStamp
	response.SessionFailover = &sessionFailover

	infolist := make([]nchf.MultipleUnitInformation, 0)
	for _, unit := range dc.Request.MultipleUnitUsage {

		// Check if we have a grant for this rating group
		haveGrant := false
		grants, ok := dc.ChargingData.Grants[*unit.RatingGroup]
		if ok {
			for _, grant := range grants {
				if grant.InvocationSequenceNumber != *dc.Request.InvocationSequenceNumber {
					continue
				}
				haveGrant = true
				info := &nchf.MultipleUnitInformation{}
				info.RatingGroup = unit.RatingGroup

				unitsGranted := grant.UnitsGranted
				grantedUnit := nchf.GrantedUnit{}
				switch grant.UnitType {
				case charging.OCTETS:
					grantedUnit.TotalVolume = &unitsGranted
				case charging.SECONDS:
					grantedUnit.Time = &unitsGranted
				default:
					grantedUnit.ServiceSpecificUnits = &unitsGranted
				}
				info.GrantedUnit = &grantedUnit

				if unitsGranted == 0 {
					info.ResultCode = nchf.ResultCodePtr(nchf.ResultCodeQuotaLimitReached)
				} else {
					info.ResultCode = nchf.ResultCodePtr(nchf.ResultCodeSuccess)
					info.ValidityTime = &grant.ValidityTime
					info.QosProfile = &grant.RetailTariff.QosProfileId

					if grant.FinalUnitIndication {
						action := nchf.FinalUnitActionTerminate
						info.FinalUnitIndication = &nchf.FinalUnitIndication{
							FinalUnitAction: &action,
						}
					}
				}
				infolist = append(infolist, *info)

				logging.Debug("BuildResponseStep - Grant: {}", grant)
				break
			}
		}

		if !haveGrant {
			info := &nchf.MultipleUnitInformation{}
			info.RatingGroup = unit.RatingGroup
			info.ResultCode = nchf.ResultCodePtr(nchf.ResultCodeSuccess)
			infolist = append(infolist, *info)
		}
	}
	if len(infolist) > 0 {
		response.MultipleUnitInformation = infolist
	}

	dc.Response = &response

	return nil
}
