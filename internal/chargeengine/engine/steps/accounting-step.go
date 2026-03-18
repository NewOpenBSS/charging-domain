package steps

import (
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/engine/business"
	"go-ocs/internal/model"
	"go-ocs/internal/chargeengine/ocserrors"
	"go-ocs/internal/charging"
	"go-ocs/internal/events"
	"go-ocs/internal/logging"
	"go-ocs/internal/nchf"
	"sort"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

func mapChargeEvent(request *nchf.ChargingDataRequest, ratedAt time.Time) *events.ChargeEvent {
	return events.NewChargeEvent(*request.ChargingId, request.InvocationTimeStamp.Time(), ratedAt)
}

func mapChargeService(grant *model.Grants, unitsUsed int64) *events.ChargeService {
	return events.NewChargeService(grant.RateKey, grant.ServiceIdentifier, unitsUsed, grant.UnitType)
}

func mapChargeSubscriber(subscriber *model.Subscriber) *events.ChargeSubscriber {
	return events.NewChargeSubscriber(
		subscriber.WholesaleId,
		subscriber.ContractId,
		subscriber.SubscriberId,
		subscriber.Msisdn,
		subscriber.RatePlanId,
		subscriber.WholesalerRatePlanId,
		subscriber.AllowOOBCharging,
	)
}

func mapChargeInfo(tariff *model.Tariff, unitsValue decimal.Decimal, unitsDebited int64, unaccountedValue decimal.Decimal, unaccountedUnits int64) *events.ChargeInfo {

	return events.NewChargeInfo(
		unitsDebited,
		tariff.Multiplier,
		tariff.RateLine.MinimumUnits.AsUnits().IntPart(),
		tariff.RateLine.RoundingIncrement.AsUnits().IntPart(),
		tariff.UnitPrice,
		unitsValue,
		tariff.Multiplier.IsZero(),
		tariff.QosProfileId,
		unaccountedUnits,
		unaccountedValue,
		tariff.RateLine.GroupKey,
	)
}

func buildAndSendChargeRecord(dc *engine.ChargingContext, chargeRecordId string, grant *model.Grants, debitResult business.DebitResult, unitsUsed int64) error {

	if dc.AppContext != nil && dc.AppContext.KafkaManager != nil {

		decimalZero := decimal.NewFromInt(0)
		normaliseUnits := grant.SettlementTariff.RateLine.NormaliseUnits(unitsUsed)
		normaliseUnitsDec := decimal.NewFromInt(normaliseUnits)

		chargeRecord := events.NewChargeRecord(
			chargeRecordId,
			mapChargeEvent(dc.Request, grant.GrantedTime),
			mapChargeSubscriber(dc.ChargingData.Subscriber),
			mapChargeService(grant, unitsUsed),

			mapChargeInfo(&grant.SettlementTariff,
				grant.SettlementTariff.UnitPrice.Mul(normaliseUnitsDec),
				normaliseUnits,
				decimalZero,
				0),
			mapChargeInfo(&grant.WholesaleTariff,
				grant.WholesaleTariff.UnitPrice.Mul(normaliseUnitsDec),
				normaliseUnits, decimalZero,
				0),
			mapChargeInfo(&grant.RetailTariff,
				debitResult.MonetaryValue,
				debitResult.MonetaryUnits,
				grant.RetailTariff.UnitPrice.Mul(decimal.NewFromInt(debitResult.UnaccountedUnits)),
				debitResult.UnaccountedUnits),
		)

		dc.AppContext.KafkaManager.PublishEvent("charge-record", chargeRecordId, chargeRecord)
	}

	return nil
}

func processUnitsUsed(dc *engine.ChargingContext, grants []model.Grants, unit *nchf.MultipleUnitUsage) ([]model.Grants, error) {

	var usedUnits int64
	currentGrants := grants
	for _, used := range unit.UsedUnitContainer {

		//retrieve the used units

		cls, ok := dc.ChargingData.Classifications[*unit.RatingGroup]
		if !ok {
			return currentGrants, ocserrors.CreateGeneralError("unknown ratingGroup classification")
		}

		switch cls.UnitType {
		case charging.UNITS, charging.MONETARY:
			if used.ServiceSpecificUnits == nil {
				return currentGrants, ocserrors.CreateGeneralError("ServiceSpecificUnits missing for UNITS/MONETARY")
			}
			usedUnits = *used.ServiceSpecificUnits

		case charging.SECONDS:
			if used.Time == nil {
				return currentGrants, ocserrors.CreateGeneralError("Time missing for SECONDS")
			}
			usedUnits = int64(*used.Time)

		case charging.OCTETS:
			if used.TotalVolume == nil {
				return currentGrants, ocserrors.CreateGeneralError("TotalVolume missing for OCTETS")
			}
			usedUnits = *used.TotalVolume

		default:
			return currentGrants, ocserrors.CreateGeneralError("unsupported unit type")
		}

		if usedUnits > 0 {
			nextGrants := []model.Grants{}
			for _, grant := range currentGrants {
				retailUsedUnits := grant.RetailTariff.RateLine.NormaliseUnits(usedUnits)

				if grant.RetailTariff.Multiplier.IsZero() {
					retailUsedUnits = 0
				}

				remaining := grant.UnitsGranted - retailUsedUnits
				if remaining < 0 {
					logging.Warn("AccountingStep - Invalid grant state: more units debited than granted.")
					return currentGrants, ocserrors.CreateUsedMoreThanGranted("Invalid grant state: more units debited than granted.")
				}

				reclaimUnits := (used.Reclaimable != nil && *used.Reclaimable)
				chargeRecordId := *dc.Request.ChargingId + ";" +
					strconv.FormatInt(*dc.Request.InvocationSequenceNumber, 10) + ";" +
					grant.GrantId.String()

				var resp *business.DebitResult = nil

				if grant.RetailTariff.Multiplier.IsZero() {
					resp = &business.DebitResult{
						DebitedUnits:     0,
						UnaccountedUnits: 0,
						MonetaryValue:    decimal.NewFromInt(0),
						MonetaryUnits:    0,
					}
				} else {
					var err error

					resp, err = business.DebitQuota(dc,
						dc.ChargingData.Subscriber.SubscriberId,
						chargeRecordId,
						grant.GrantId,
						retailUsedUnits,
						grant.UnitType,
						reclaimUnits)
					if err != nil {
						return currentGrants, err
					}
				}

				grant.UnitsGranted = remaining

				if reclaimUnits || remaining == 0 {
					logging.Debug("AccountingStep - Reclaiming.", "units", grant.UnitsGranted, "grantId", grant.GrantId)
				} else {
					nextGrants = append(nextGrants, grant)
				}

				err := buildAndSendChargeRecord(dc, chargeRecordId, &grant, *resp, usedUnits)
				if err != nil {
					return currentGrants, err
				}
			}
			currentGrants = nextGrants
		}
	}

	return currentGrants, nil
}

func Accounting(dc *engine.ChargingContext) error {

	logging.Debug("AccountingStep - Accounting step started")

	for _, unit := range dc.Request.MultipleUnitUsage {
		logging.Debug("AccountingStep - Processing unit: {}", unit)

		grants := dc.ChargingData.Grants[*unit.RatingGroup]

		// Filter out expired grants
		validGrants := []model.Grants{}
		for _, grant := range grants {
			// Check if the grant is still valid
			if grant.GrantedTime.Add(time.Duration(grant.ValidityTime) * time.Second).After(dc.StartTime) {
				validGrants = append(validGrants, grant)
			}
		}
		if len(validGrants) > 1 {
			sort.SliceStable(validGrants, func(i, j int) bool {
				return validGrants[i].InvocationSequenceNumber < validGrants[j].InvocationSequenceNumber
			})
		}
		dc.ChargingData.Grants[*unit.RatingGroup] = validGrants

		validGrants, err := processUnitsUsed(dc, validGrants, &unit)
		if err != nil {
			return err
		}
		dc.ChargingData.Grants[*unit.RatingGroup] = validGrants
	}

	return nil

}
