package model

import (
	"go-ocs/internal/charging"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Subscriber struct {
	Msisdn               string    `json:"msisdn"`
	ContractId           uuid.UUID `json:"contractId"`
	SubscriberId         uuid.UUID `json:"subscriberId"`
	RatePlanId           uuid.UUID `json:"ratePlanId"`
	WholesaleId          uuid.UUID `json:"wholesaleId"`
	WholesalerRatePlanId uuid.UUID `json:"wholesalerRatePlanId"`
	AllowOOBCharging     bool      `json:"allowOobCharging"`
}

type ChargingData struct {
	NewRecord       bool                     //Control record used to indicate if this is a new record or an update
	Subscriber      *Subscriber              `json:"subscriber"`
	EventTime       time.Time                `json:"eventTime"`
	Classifications map[int64]Classification `json:"classifications,omitempty"`
	Grants          map[int64][]Grants       `json:"grants,omitempty"`
}

func NewChargingData() *ChargingData {
	return &ChargingData{
		NewRecord: true,
	}
}

type Tariff struct {
	UnitPrice    decimal.Decimal `json:"unitPrice"`
	Multiplier   decimal.Decimal `json:"multiplier"`
	QosProfileId string          `json:"qosProfileId,omitempty"`
	RateLine     *RateLine       `json:"rateLine,omitempty"`
}

type Grants struct {
	GrantId                  uuid.UUID         `json:"grantId"`
	InvocationSequenceNumber int64             `json:"invocationSequenceNumber"`
	FinalUnitIndication      bool              `json:"finalUnitIndication"`
	ValidityTime             int32             `json:"validityTime"`
	GrantedTime              time.Time         `json:"grantedTime"`
	UnitsGranted             int64             `json:"unitsGranted"`
	RatingGroup              int64             `json:"ratingGroup"`
	UnitType                 charging.UnitType `json:"unitType"`
	RateKey                  charging.RateKey  `json:"rateKey"`
	ServiceIdentifier        string            `json:"serviceIdentifier,omitempty"`
	SettlementTariff         Tariff            `json:"settlementTariff"`
	WholesaleTariff          Tariff            `json:"wholesaleTariff"`
	RetailTariff             Tariff            `json:"retailTariff"`
}
