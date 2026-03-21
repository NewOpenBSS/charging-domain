package events

import "github.com/google/uuid"

// WholesaleContractEventType identifies the kind of lifecycle change published by
// the Wholesale CRM onto the wholesale-contract-event Kafka topic.
type WholesaleContractEventType string

const (
	// WholesaleContractProvisioned signals that a new wholesale contract was provisioned
	// or an existing one was re-provisioned with updated details.
	WholesaleContractProvisioned WholesaleContractEventType = "WholesaleContractProvisionedEvent"

	// WholesaleContractDeregistering signals that a wholesale contract is being
	// deregistered. The wholesaler should be deactivated and removed if no subscribers
	// remain on it.
	WholesaleContractDeregistering WholesaleContractEventType = "WholesaleContractDeregisteringEvent"

	// WholesaleContractSuspend signals that a wholesale contract has been suspended
	// or unsuspended. Suspend=true disables the wholesaler; Suspend=false re-enables it.
	WholesaleContractSuspend WholesaleContractEventType = "WholesaleContractSuspendEvent"
)

// WholesaleContractProvisionedEvent is the Kafka message payload published by the
// Wholesale CRM when a contract is provisioned or re-provisioned. It carries a full
// snapshot of the wholesaler record. Fields not stored in the wholesaler table
// (e.g. registrationNumber, contactInfo) are intentionally excluded.
type WholesaleContractProvisionedEvent struct {
	// EventType identifies this as a provisioned event.
	EventType WholesaleContractEventType `json:"eventType"`

	// WholesalerID is the unique identifier for the wholesaler.
	WholesalerID uuid.UUID `json:"wholesalerId"`

	// ContractID is the contract associated with this wholesaler.
	ContractID uuid.UUID `json:"contractId"`

	// RatePlanID is the rate plan linked to this wholesaler.
	RatePlanID uuid.UUID `json:"ratePlanId"`

	// LegalName is the registered legal name of the wholesaler.
	LegalName string `json:"legalName"`

	// DisplayName is the human-readable display name for the wholesaler.
	DisplayName string `json:"displayName"`

	// Realm is the Keycloak realm associated with the wholesaler.
	Realm string `json:"realm"`

	// Hosts is the list of valid hostnames for this wholesaler.
	Hosts []string `json:"hosts"`

	// NchfUrl is the NCHF endpoint URL for the wholesaler.
	NchfUrl string `json:"nchfUrl"`

	// RateLimit is the maximum NCHF request rate for this wholesaler.
	// Stored as numeric in the DB; float64 is sufficient for rate limit values.
	RateLimit float64 `json:"rateLimit"`

	// Active indicates whether the wholesaler is currently enabled for trading.
	Active bool `json:"active"`
}

// WholesaleContractDeregisteringEvent is the Kafka message payload published by
// the Wholesale CRM when a wholesaler contract is being deregistered. The OCS
// should deactivate the wholesaler and remove it once all subscribers are gone.
type WholesaleContractDeregisteringEvent struct {
	// EventType identifies this as a deregistering event.
	EventType WholesaleContractEventType `json:"eventType"`

	// WholesalerID is the unique identifier for the wholesaler being deregistered.
	WholesalerID uuid.UUID `json:"wholesalerId"`
}

// WholesaleContractSuspendEvent is the Kafka message payload published by the
// Wholesale CRM when a wholesaler contract is suspended or unsuspended.
type WholesaleContractSuspendEvent struct {
	// EventType identifies this as a suspend event.
	EventType WholesaleContractEventType `json:"eventType"`

	// WholesalerID is the unique identifier for the wholesaler being suspended.
	WholesalerID uuid.UUID `json:"wholesalerId"`

	// Suspend indicates the new suspension state. True means the wholesaler is
	// suspended (active=false); false means it is unsuspended (active=true).
	Suspend bool `json:"suspend"`
}
