package events

import "github.com/google/uuid"

// SubscriberEventType identifies the kind of change that occurred on a subscriber
// in the Retail CRM domain.
type SubscriberEventType string

const (
	// SubscriberEventCreated signals that a new subscriber was provisioned.
	SubscriberEventCreated SubscriberEventType = "CREATED"

	// SubscriberEventUpdated signals that subscriber data was updated.
	SubscriberEventUpdated SubscriberEventType = "UPDATED"

	// SubscriberEventMsisdnSwap signals that the subscriber's MSISDN was changed.
	SubscriberEventMsisdnSwap SubscriberEventType = "MSISDN_SWAP"

	// SubscriberEventSimSwap signals that the subscriber's SIM card was swapped.
	SubscriberEventSimSwap SubscriberEventType = "SIM_SWAP"

	// SubscriberEventDeleted signals that the subscriber was removed.
	SubscriberEventDeleted SubscriberEventType = "DELETED"
)

// SubscriberEvent is the Kafka message payload published by the Retail CRM domain
// on the public.subscriber-event topic. Each event carries a full snapshot of the
// subscriber record alongside the EventType that indicates what changed.
type SubscriberEvent struct {
	// EventType describes the nature of the change.
	EventType SubscriberEventType `json:"eventType"`

	// SubscriberID is the unique identifier assigned by the OSS service.
	SubscriberID uuid.UUID `json:"subscriberId"`

	// RatePlanID is the rate plan linked to this subscriber.
	RatePlanID uuid.UUID `json:"ratePlanId"`

	// CustomerID is the customer account linked to this subscriber.
	CustomerID uuid.UUID `json:"customerId"`

	// WholesaleID is the wholesaler on whose network this subscriber is provisioned.
	WholesaleID uuid.UUID `json:"wholesaleId"`

	// Msisdn is the Mobile Station International Subscriber Directory Number.
	Msisdn string `json:"msisdn"`

	// Iccid is the Integrated Circuit Card Identifier for the subscriber's SIM.
	Iccid string `json:"iccid"`

	// ContractID is the contract linked to the MSISDN.
	ContractID uuid.UUID `json:"contractId"`

	// Status is the subscriber lifecycle status: ACTIVE, INACTIVE, or BLOCKED.
	Status string `json:"status"`

	// AllowOobCharging indicates whether out-of-bundle charging is permitted.
	AllowOobCharging bool `json:"allowOobCharging"`
}
