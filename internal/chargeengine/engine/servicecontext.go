package engine

import (
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/engine/providers/carriers"
	"go-ocs/internal/chargeengine/engine/providers/classificationplan"
	"go-ocs/internal/chargeengine/engine/providers/numberplan"
	"go-ocs/internal/chargeengine/engine/providers/ratingplan"
	"go-ocs/internal/chargeengine/engine/providers/subscribers"
	"go-ocs/internal/chargeengine/model"
	"go-ocs/internal/store/sqlc"

	"github.com/google/uuid"
)

type ServiceContext struct {
	ClassificationService *classificationplan.ClassificationContainer
	CarriersService       *carriers.CarrierContainer
	NumberPlanService     *numberplan.NumberPlanContainer
	RatePlanContainer     *ratingplan.RatePlanContainer
	SubscriberContainer   *subscribers.SubscriberContainer
}

func NewServiceContext(ctx *appcontext.AppContext) (*ServiceContext, func()) {

	sc := &ServiceContext{
		ClassificationService: classificationplan.NewClassificationContainer(ctx),
		CarriersService:       carriers.NewCarrierContainer(ctx),
		NumberPlanService:     numberplan.NewNumberPlanContainer(ctx),
		RatePlanContainer:     ratingplan.NewRatePlanContainer(ctx),
		SubscriberContainer:   subscribers.NewSubscriberContainer(ctx),
	}
	return sc,
		func() {
			sc.ClassificationService.Shutdown()
			sc.CarriersService.Shutdown()
			sc.NumberPlanService.Shutdown()
			sc.RatePlanContainer.Shutdown()
			sc.SubscriberContainer.Shutdown()
		}
}

func (sc *ServiceContext) FetchClassificationPlan() (*model.Plan, error) {
	return sc.ClassificationService.FetchClassificationPlan()
}

func (sc *ServiceContext) FetchCarrierContainer() (*carriers.CarrierContainer, error) {
	return sc.CarriersService, nil
}

func (sc *ServiceContext) FindCarrierByMccMnc(mcc string, mnc string) (*sqlc.Carrier, error) {
	return sc.CarriersService.FindCarrierByMccMnc(mcc, mnc)
}

func (sc *ServiceContext) FindCarrierBySource(mcc string, mnc string) string {
	return sc.CarriersService.FindCarrierBySource(mcc, mnc)
}

func (sc *ServiceContext) FindNumberPlan(number string) (*sqlc.AllNumbersRow, error) {
	return sc.NumberPlanService.FindNumberPlan(number)
}

func (sc *ServiceContext) FindCarrierByDestination(number string) string {
	return sc.NumberPlanService.FindCarrierByDestination(number)
}

func (sc *ServiceContext) FindRatingPlan(uuid uuid.UUID) (*model.RatePlan, error) {
	return sc.RatePlanContainer.FindRatingPlan(uuid)
}

func (sc *ServiceContext) FindSubscriber(msisdn string) (*model.Subscriber, error) {
	return sc.SubscriberContainer.FindSubscriber(msisdn)
}
