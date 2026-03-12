package events

import (
	"go-ocs/internal/charging"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ChargeRecord represents a consolidated record of charging details for an event,
// including information about the event, subscriber, associated service, and
// detailed charge breakdowns.
type ChargeRecord struct {
	// A unique identifier representing a specific charge record within the system.
	ChargeRecordID string `json:"chargeRecordId"`

	// Event metadata such as request ID and timestamps.
	Event ChargeEvent `json:"event"`

	// Subscriber involved in the chargeable transaction.
	Subscriber ChargeSubscriber `json:"subscriber"`

	// Service associated with the charge record.
	Service ChargeService `json:"service"`

	// Settlement details associated with the charge record.
	Settlement ChargeInfo `json:"settlement"`

	// Wholesale charging information.
	Wholesale ChargeInfo `json:"wholesale"`

	// Retail charging information.
	Retail ChargeInfo `json:"retail"`
}

func NewChargeRecord(chargeId string,
	chargeEvent *ChargeEvent,
	subscriber *ChargeSubscriber,
	service *ChargeService,
	settlement *ChargeInfo,
	wholesale *ChargeInfo,
	retail *ChargeInfo) *ChargeRecord {
	return &ChargeRecord{
		ChargeRecordID: chargeId,
		Event:          *chargeEvent,
		Subscriber:     *subscriber,
		Service:        *service,
		Settlement:     *settlement,
		Wholesale:      *wholesale,
		Retail:         *retail,
	}
}

// ChargeInfo represents detailed charging information including units,
// pricing, multipliers, rounding rules and unaccounted usage.
type ChargeInfo struct {
	// Contract associated with the charge information.
	ContractID uuid.UUID `json:"contractId"`

	// Normalised units of consumption.
	Units int64 `json:"units"`

	// Multiplier applied during rating calculations.
	Multiplier decimal.Decimal `json:"multiplier"`

	// Minimum billable units.
	MinimumUnits int64 `json:"minimumUnits"`

	// Increment used when rounding units.
	RoundingIncrement int64 `json:"roundingIncrement"`

	// Price per unit.
	UnitPrice decimal.Decimal `json:"unitPrice"`

	// Total calculated monetary value.
	Value decimal.Decimal `json:"value"`

	// Indicates if the charge is zero rated.
	ZeroRated bool `json:"zeroRated"`

	// QoS profile applied to the retail plan.
	QosProfile string `json:"qosProfile"`

	// Units that could not be accounted during rating.
	UnaccountedUnits int64 `json:"unaccountedUnits"`

	// Monetary value associated with unaccounted usage.
	UnaccountedValue decimal.Decimal `json:"unaccountedValue"`

	// Logical grouping key.
	GroupKey string `json:"groupKey"`
}

func NewChargeInfo(units int64, multiplier decimal.Decimal, minimumUnits int64, roundingIncrement int64, unitPrice decimal.Decimal, value decimal.Decimal, zeroRated bool, qosProfile string, unaccountedUnits int64, unaccountedValue decimal.Decimal, groupKey string) *ChargeInfo {
	return &ChargeInfo{
		ContractID:        uuid.Nil,
		Units:             units,
		Multiplier:        multiplier,
		MinimumUnits:      minimumUnits,
		RoundingIncrement: roundingIncrement,
		UnitPrice:         unitPrice,
		Value:             value,
		ZeroRated:         zeroRated,
	}
}

// ChargeSubscriber represents subscriber-related information associated
// with a charging event.
type ChargeSubscriber struct {
	// The wholesaler the subscriber is allocated to.
	WholesaleID uuid.UUID `json:"wholesaleId"`

	// Contract identifier for the retail customer associated with the event.
	ContractID uuid.UUID `json:"contractId"`

	// Unique identifier for the subscriber associated with the event.
	SubscriberID uuid.UUID `json:"subscriberId"`

	// Mobile Station International Subscriber Directory Number (MSISDN)
	// associated with the processed event.
	Msisdn string `json:"msisdn"`

	// Identifier for the wholesale rate plan associated with the event.
	RatePlanID uuid.UUID `json:"ratePlanId"`

	// Identifier for the MVNO-specific rate plan associated with the event.
	MvnoRatePlanID uuid.UUID `json:"mvnoRatePlanId"`

	// Indicates whether the subscriber allows out-of-bundle charging.
	AllowOOBCharging bool `json:"allowOobCharging"`
}

func NewChargeSubscriber(wholesaleID uuid.UUID, contractID uuid.UUID, subscriberID uuid.UUID, msisdn string, ratePlanID uuid.UUID, mvnoRatePlanID uuid.UUID, allowOOBCharging bool) *ChargeSubscriber {
	return &ChargeSubscriber{
		WholesaleID:      wholesaleID,
		ContractID:       contractID,
		SubscriberID:     subscriberID,
		Msisdn:           msisdn,
		RatePlanID:       ratePlanID,
		MvnoRatePlanID:   mvnoRatePlanID,
		AllowOOBCharging: allowOOBCharging,
	}
}

// ChargeEvent represents metadata associated with a charging request,
// including request identifiers and processing timestamps.
type ChargeEvent struct {
	// The request id
	RequestID string `json:"requestId"`

	// The request timestamp
	EventDateTime time.Time `json:"eventDateTime"`

	// Represents the timestamp at which the rating process was completed.
	// It is optional and will be nil if the rating process has not occurred.
	RatedAt *time.Time `json:"ratedAt,omitempty"`
}

func NewChargeEvent(requestId string, eventDateTime time.Time, ratedAt time.Time) *ChargeEvent {
	return &ChargeEvent{
		RequestID:     requestId,
		EventDateTime: eventDateTime,
		RatedAt:       &ratedAt,
	}
}

// ChargeService represents service-specific charging details for a record.
type ChargeService struct {
	// Represents a unique composite key used to identify a specific rate configuration.
	RateKey charging.RateKey `json:"rateKey"`

	// Identifier that specifies the context or unique usage of a service.
	// Examples:
	//   Data  → rating group (e.g. "100100100")
	//   Voice → called MSISDN
	//   SMS   → destination MSISDN
	//   USSD  → USSD string (e.g. "*160*111#")
	ServiceIdentifier string `json:"serviceIdentifier"`

	// Units of service consumed.
	// Examples:
	//   Data  → bytes
	//   Voice → seconds
	//   SMS   → message count
	UnitsUsed int64 `json:"unitsUsed"`

	// Type of units associated with the usage.
	UnitType charging.UnitType `json:"unitType"`
}

func NewChargeService(rateKey charging.RateKey, serviceIdentifier string, unitsUsed int64, unitType charging.UnitType) *ChargeService {
	return &ChargeService{
		RateKey:           rateKey,
		ServiceIdentifier: serviceIdentifier,
		UnitsUsed:         unitsUsed,
		UnitType:          unitType,
	}
}
