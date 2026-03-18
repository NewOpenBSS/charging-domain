package interfaces

import (
	"go-ocs/internal/chargeengine/engine/providers/carriers"
	"go-ocs/internal/model"
	"go-ocs/internal/store/sqlc"

	"github.com/google/uuid"
)

type ClassificationInterface interface {
	FetchClassificationPlan() (*model.ClassificationPlan, error)
}

type CarrierInterface interface {
	FetchCarrierContainer() (*carriers.CarrierContainer, error)
	FindCarrierByMccMnc(mcc string, mnc string) (*sqlc.Carrier, error)
	FindCarrierBySource(mcc string, mnc string) string
}

type NumberPlanInterface interface {
	FindNumberPlan(number string) (*sqlc.AllNumbersRow, error)
	FindCarrierByDestination(number string) string
}

type RatingInterface interface {
	FindRatingPlan(uuid uuid.UUID) (*model.RatePlan, error)
}

type SubscriberInterface interface {
	FindSubscriber(msisdn string) (*model.Subscriber, error)
}
type Infrastructure interface {
	ClassificationInterface
	CarrierInterface
	NumberPlanInterface
	RatingInterface
	SubscriberInterface
}
