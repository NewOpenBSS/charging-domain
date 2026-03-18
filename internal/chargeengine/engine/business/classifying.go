package business

import (
	"fmt"
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/model"
	"go-ocs/internal/charging"
	"go-ocs/internal/ruleevaluator"
	"sort"
	"strings"
	"time"
)

func GetServiceWindow(
	serviceWindows map[string]model.ServiceWindow,
	validWindows map[string]struct{},
	defaultServiceWindow string,
	now time.Time, // pass in (or inject a clock) to make it testable
) string {

	// Collect candidates: only keys in validWindows
	type kv struct {
		key string
		win model.ServiceWindow
	}
	candidates := make([]kv, 0, len(serviceWindows))

	for k, w := range serviceWindows {
		if _, ok := validWindows[k]; !ok {
			continue
		}
		candidates = append(candidates, kv{key: k, win: w})
	}

	// Sort by shortest duration (like your comparingByValue + getWindowDuration)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].win.Duration() < candidates[j].win.Duration()
	})

	// Find first that matches "now"
	for _, c := range candidates {
		if c.win.IsWithin(now) {
			return c.key
		}
	}

	return defaultServiceWindow
}

type EvaluatorData struct {
	Req  any
	Info any
	Unit any
}

func ClassifyService(dc *engine.ChargingContext) (map[int64]model.Classification, error) {

	plan, err := dc.Infra.FetchClassificationPlan()
	if err != nil {
		return nil, err
	}

	cit, info := dc.Request.GetChargeInformation()
	data := EvaluatorData{
		Req:  dc.Request,
		Info: &info,
		Unit: nil,
	}

	evaluator := ruleevaluator.NewRuleEvaluator(&data)
	evaluator.RegisterFunction("serviceDirection", serviceDirection())
	evaluator.RegisterFunction("sourceByMccMnc", findCarrierBySource(dc, plan.DefaultSourceType))
	evaluator.RegisterFunction("startsWith", startsWith())

	chargingInformation := model.ChargingInformation(cit)

	switch chargingInformation {
	case model.PDU:
		evaluator.RegisterFunction("serviceCategory", dataServiceCategory(dc))
	case model.IMS:
		evaluator.RegisterFunction("serviceCategory", voiceServiceCategory(dc))
	case model.NEF:
		evaluator.RegisterFunction("serviceCategory", ussdServiceCategory(dc))
	case model.SMS:
		evaluator.RegisterFunction("serviceCategory", smsServiceCategory(dc))
	default:
		return nil, fmt.Errorf("Unsupported charging information type: %v", chargingInformation)
	}

	classifications := make(map[int64]model.Classification, 3)
	for _, serviceType := range plan.ServiceTypes {
		if serviceType.ChargingInformation != chargingInformation {
			continue
		}

		// Check if the service type has a service type rule and evaluate it
		if serviceType.ServiceTypeRule != "" {
			ruleMatch, err := evaluator.Evaluate(serviceType.ServiceTypeRule)
			if err != nil {
				return nil, fmt.Errorf("Error evaluating service type rule: %v", err)
			}
			if ruleMatch == false {
				continue
			}
		}

		//From this point, we have a matching service type and charging information

		//Calculate the source type
		s, err := evaluator.Evaluate(serviceType.SourceType)
		if err != nil {
			return nil, fmt.Errorf("Error evaluating source type: %v", err)
		}
		sourceType, ok := s.(string)
		if !ok {
			sourceType = plan.DefaultSourceType
		}

		//Calculate the service direction
		cd, err := evaluator.Evaluate(serviceType.ServiceDirection)
		if err != nil {
			return nil, fmt.Errorf("Error evaluating service direction: %v", err)
		}
		sd, ok := cd.(charging.CallDirection)
		if !ok {
			sd = charging.MO
		}

		//Calculate the service window
		serviceWindow := ""
		if plan.UseServiceWindows {
			serviceWindow = GetServiceWindow(plan.ServiceWindows, serviceType.ServiceWindowMap, plan.DefaultServiceWindow, dc.StartTime)
		} //if

		for _, unit := range dc.Request.MultipleUnitUsage {
			data.Unit = &unit

			//Calculate the service category
			serviceCategory := ""
			if serviceType.ServiceCategory != "" {
				sc, err := evaluator.Evaluate(serviceType.ServiceCategory)
				if err != nil {
					return nil, fmt.Errorf("Error evaluating service category: %v", err)
				}
				if sc != nil {
					scStr, ok := sc.(string)
					if !ok {
						return nil, fmt.Errorf("Service category must be a string")
					}
					serviceCategory = scStr
				}
			}

			if serviceCategory == "" {
				serviceCategory = serviceType.DefaultServiceCategory
			}

			classifications[*unit.RatingGroup] = model.Classification{
				Ratekey: charging.RateKey{
					ServiceType:      serviceType.ServiceType,
					SourceType:       sourceType,
					ServiceDirection: sd,
					ServiceCategory:  serviceCategory,
					ServiceWindow:    serviceWindow,
				},
				UnitType: serviceType.UnitType,
			}
		}
	}

	return classifications, nil
}

func findCarrierBySource(dc *engine.ChargingContext, defaultSourceType string) ruleevaluator.EvaluatorFunc {
	return func(args []any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("sourceByMccMnc expects 2 args")
		}

		mcc, ok := args[0].(string)
		if !ok {
			return defaultSourceType, nil
		}

		mnc, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("sourceByMccMnc expects mnc to be string")
		}

		return dc.Infra.FindCarrierBySource(mcc, mnc), nil
	}
}

func voiceServiceCategory(dc *engine.ChargingContext) ruleevaluator.EvaluatorFunc {
	return func(args []any) (any, error) {
		number, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("voiceServiceCategory expects number to be string")
		}

		return dc.Infra.FindCarrierByDestination(number), nil
	}
}

func smsServiceCategory(dc *engine.ChargingContext) ruleevaluator.EvaluatorFunc {
	return func(args []any) (any, error) {
		number, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("smsServiceCategory expects number to be string")
		}

		return dc.Infra.FindCarrierByDestination(number), nil
	}
}

func ussdServiceCategory(dc *engine.ChargingContext) ruleevaluator.EvaluatorFunc {
	return func(args []any) (any, error) {
		return "BANKING", nil
	}
}

func dataServiceCategory(dc *engine.ChargingContext) ruleevaluator.EvaluatorFunc {
	return func(args []any) (any, error) {
		return "INTERNET", nil
	}
}

func startsWith() ruleevaluator.EvaluatorFunc {
	return func(args []any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("startsWith expects 2 args")
		}

		s, ok1 := args[0].(string)
		prefix, ok2 := args[1].(string)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("startsWith args must be strings")
		}

		//startsWith(*100*1*4#, '*100#')"
		if len(s) > len(prefix) && s[len(prefix)-1] == '*' {
			s = s[0:len(prefix)-1] + "#"
		}

		return strings.HasPrefix(s, prefix), nil
	}
}

func serviceDirection() ruleevaluator.EvaluatorFunc {
	return func(args []any) (any, error) {
		if len(args) == 1 {
			direction, ok := args[0].(string)
			if ok {
				return direction, nil
			}
		}
		return charging.MO, nil
	}
}
