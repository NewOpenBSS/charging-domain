package model

import "time"

// RatePlan represents a rate plan definition containing metadata
// and the set of rate lines that define the actual tariffs.
type RatePlan struct {

	// RatePlanID uniquely identifies the rate plan.
	RatePlanID string `json:"ratePlanId,omitempty"`

	// RatePlanName provides a human-readable name for the rate plan.
	RatePlanName string `json:"ratePlanName,omitempty"`

	// RatePlanType defines the type of the rate plan
	// (e.g. SETTLEMENT, WHOLESALE, RETAIL).
	RatePlanType RatePlanType `json:"ratePlanType,omitempty"`

	// EffectiveFrom indicates when the rate plan becomes active.
	EffectiveFrom time.Time `json:"effectiveFrom,omitempty"`

	// RateLines defines the set of rate rules associated with the plan.
	RateLines []RateLine `json:"rateLines,omitempty"`
}
