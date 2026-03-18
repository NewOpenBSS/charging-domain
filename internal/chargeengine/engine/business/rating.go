package business

import (
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/model"
	"go-ocs/internal/chargeengine/ocserrors"
	"go-ocs/internal/charging"
	"go-ocs/internal/logging"
)

func findBestRateLine(ratePlan *model.RatePlan, ratekey charging.RateKey) (*model.RateLine, error) {

	matchList := make(map[int]*model.RateLine)

	for _, l := range ratePlan.RateLines {
		match, score := ratekey.Matches(l.ClassificationKey)
		if match {
			matchList[score] = &l
		}
	}

	maxKey := -1
	var maxVal *model.RateLine

	for k, v := range matchList {
		if k > maxKey {
			maxKey = k
			maxVal = v
		}
	}

	if maxVal == nil {
		logging.Error("rate plan does not contain a matching rate line",
			"rateplan", ratePlan.RatePlanID,
			"ratekey", ratekey.String())
		return nil, ocserrors.CreateNoRatingEntry("No Rating Entry for Service")
	}

	return maxVal, nil
}

func RateService(dc *engine.ChargingContext, classification model.Classification) (*model.RateLine, *model.RateLine, *model.RateLine, error) {

	settlementPlan, err := dc.Infra.FindRatingPlan(dc.AppContext.Config.Engine.SettlementPlanId)
	if err != nil {
		return nil, nil, nil, err
	}
	settlementLine, err := findBestRateLine(settlementPlan, classification.Ratekey)
	if err != nil {
		return nil, nil, nil, err
	}

	wholesalePlan, err := dc.Infra.FindRatingPlan(dc.ChargingData.Subscriber.WholesalerRatePlanId)
	if err != nil {
		return nil, nil, nil, err
	}
	wholesaleLine, err := findBestRateLine(wholesalePlan, classification.Ratekey)
	if err != nil {
		return nil, nil, nil, err
	}

	retailRatePlan, err := dc.Infra.FindRatingPlan(dc.ChargingData.Subscriber.RatePlanId)
	if err != nil {
		return nil, nil, nil, err
	}
	retailLine, err := findBestRateLine(retailRatePlan, classification.Ratekey)
	if err != nil {
		return nil, nil, nil, err
	}

	return settlementLine, wholesaleLine, retailLine, nil
}
