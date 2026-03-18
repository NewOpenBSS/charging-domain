package steps

import (
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/engine/business"
	"go-ocs/internal/model"
	"go-ocs/internal/chargeengine/ocserrors"
	"go-ocs/internal/charging"
	"go-ocs/internal/logging"
	"go-ocs/internal/nchf"
	"go-ocs/internal/quota"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func calcTariff(rateLine *model.RateLine, dependantTariff *model.Tariff, decimalDigits int32) (*model.Tariff, error) {
	if rateLine.Barred {
		return nil, ocserrors.CreateServiceBarred("Service access denied due to rating policy.")
	}

	var unitPrice decimal.Decimal

	switch rateLine.TariffType {
	case model.ACTUAL:
		unitOfMeasure := rateLine.UnitOfMeasure.AsUnits()
		unitPrice = rateLine.BaseTariff.DivRound(unitOfMeasure, decimalDigits)

	case model.PERCENTAGE:
		if dependantTariff == nil {
			return nil, ocserrors.CreateGeneralError("No dependant tariff")
		}

		unitPrice = dependantTariff.UnitPrice.
			Mul(decimal.NewFromInt(1).Add(rateLine.BaseTariff)).
			Round(decimalDigits)

	case model.MARKUP:
		if dependantTariff == nil {
			return nil, ocserrors.CreateGeneralError("No dependant tariff")
		}
		unitPrice = dependantTariff.UnitPrice.Add(rateLine.BaseTariff)
	}

	return &model.Tariff{
		UnitPrice:    unitPrice,
		Multiplier:   rateLine.Multiplier,
		QosProfileId: rateLine.QosProfile,
		RateLine:     rateLine,
	}, nil
}

func rateUnit(dc *engine.ChargingContext, u nchf.MultipleUnitUsage) (*model.Grants, error) {
	if u.RatingGroup == nil {
		return nil, ocserrors.CreateClassificationError("No Rating Group")
	}

	classification, ok := dc.ChargingData.Classifications[*u.RatingGroup]
	if !ok {
		return nil, ocserrors.CreateClassificationError("Not Classified")
	}

	// Get the rate lines for the classification
	settlement, wholesale, retail, err := business.RateService(dc, classification)
	if err != nil {
		return nil, err
	}
	if settlement == nil || wholesale == nil || retail == nil {
		return nil, ocserrors.CreateGeneralError("No rate lines found for classification:")
	}

	decimalDigits := dc.AppContext.Config.Engine.DecimalDigits
	// Calculate the tariffs
	settlementTariff, err := calcTariff(settlement, nil, decimalDigits)
	if err != nil {
		return nil, err
	}
	wholesaleTariff, err := calcTariff(wholesale, settlementTariff, decimalDigits)
	if err != nil {
		return nil, err
	}
	retailTariff, err := calcTariff(retail, wholesaleTariff, decimalDigits)
	if err != nil {
		return nil, err
	}

	// Retrieve the requested units
	var requestedUnits int64 = 0
	if u.RequestedUnit != nil {
		switch classification.UnitType {
		case charging.UNITS, charging.MONETARY:
			if u.RequestedUnit.ServiceSpecificUnits != nil {
				requestedUnits = *u.RequestedUnit.ServiceSpecificUnits
			}

		case charging.SECONDS:
			if u.RequestedUnit.Time != nil {
				requestedUnits = int64(*u.RequestedUnit.Time)
			}

		case charging.OCTETS:
			if u.RequestedUnit.TotalVolume != nil {
				requestedUnits = *u.RequestedUnit.TotalVolume
			}
		}
	}

	validityWindow := dc.AppContext.Config.Engine.DefaultValidityWindow

	//Skip calculating scaling factor for the first request
	interval := time.Since(dc.ChargingData.EventTime)
	scalingWindowIn := dc.AppContext.Config.Engine.ScalingValidityWindow
	if *dc.Request.InvocationSequenceNumber > 0 && interval > 0 && interval < scalingWindowIn {
		scalingFactor := scalingWindowIn.Seconds() / interval.Seconds()
		requestedUnits = requestedUnits * int64(math.Ceil(scalingFactor))
		validityWindow = scalingWindowIn
		logging.Debug("RateStep - Applied scaling factor", "scalingFactor", scalingFactor, "scaledUnits", requestedUnits)
	}

	//Normalise the units
	normalisedUnits := retail.NormaliseUnits(requestedUnits)
	logging.Debug("RateStep - Normalised units", "normalisedUnits", normalisedUnits, "requestedUnits", requestedUnits)

	grantId, err := uuid.NewRandom()
	if err != nil {
		return nil, ocserrors.CreateGeneralError("Failed to generate GrantId")
	}

	grantedUnits := normalisedUnits
	isFinalUnitIndication := false

	if !retail.Multiplier.IsZero() {
		grantedUnits, err = business.ReserveQuota(
			dc,
			grantId,
			dc.ChargingData.Subscriber.SubscriberId,
			quota.ReasonServiceUsage,
			classification.Ratekey,
			classification.UnitType,
			normalisedUnits,
			retailTariff.UnitPrice,
			retailTariff.Multiplier,
			validityWindow,
			dc.ChargingData.Subscriber.AllowOOBCharging)
		if err != nil {
			return nil, err
		}
		isFinalUnitIndication = normalisedUnits > grantedUnits
	}

	grant := model.Grants{
		GrantId:                  grantId,
		InvocationSequenceNumber: *dc.Request.InvocationSequenceNumber,
		FinalUnitIndication:      isFinalUnitIndication,
		ValidityTime:             int32(validityWindow.Seconds()),
		GrantedTime:              dc.StartTime,
		UnitsGranted:             grantedUnits,
		RatingGroup:              *u.RatingGroup,
		UnitType:                 classification.UnitType,
		RateKey:                  classification.Ratekey,
		SettlementTariff:         *settlementTariff,
		WholesaleTariff:          *wholesaleTariff,
		RetailTariff:             *retailTariff,
	}

	logging.Debug("RateStep - Units Granted", "units", grantedUnits, "ratingGroup", *u.RatingGroup)

	return &grant, nil
}

func Rate(dc *engine.ChargingContext) error {

	logging.Debug("RateStep - Rating the charging data request")

	grants := make(map[int64][]model.Grants, len(dc.Request.MultipleUnitUsage))
	for _, unitUsage := range dc.Request.MultipleUnitUsage {
		grant, err := rateUnit(dc, unitUsage)
		if err != nil {
			return err
		}

		grants[grant.RatingGroup] = append(grants[grant.RatingGroup], *grant)
	}
	dc.ChargingData.Grants = grants

	return nil
}
