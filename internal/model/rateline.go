package model

import (
	"fmt"
	"go-ocs/internal/charging"

	"github.com/shopspring/decimal"
)

type TariffType string

const (
	ACTUAL     TariffType = "ACTUAL"
	PERCENTAGE TariffType = "PERCENTAGE"
	MARKUP     TariffType = "MARKUP"
)

var tariffTypeDescMap = map[TariffType]string{
	ACTUAL:     "Actual",
	PERCENTAGE: "Wholesale plus Percentage",
	MARKUP:     "Wholesale plus Markup",
}

func ParseTariffType(s string) (TariffType, error) {
	u := TariffType(s)
	if _, ok := tariffTypeDescMap[u]; !ok {
		return "", fmt.Errorf("invalid tariff type: %s", s)
	}
	return u, nil
}

type RateLine struct {

	// ClassificationKey uniquely identifies the rate configuration
	// based on service type, source type, direction, category and window.
	ClassificationKey charging.RateKey `json:"classificationKey"`

	// GroupKey is used for grouping rated events in invoicing or reporting.
	GroupKey string `json:"groupKey,omitempty"`

	// Description provides additional context for the rate line.
	Description string `json:"description,omitempty"`

	// TariffType determines how the tariff is calculated.
	// Default: ACTUAL
	TariffType TariffType `json:"tariffType"`

	// UnitType defines the measurement unit used for rating.
	UnitType charging.UnitType `json:"unitType"`

	// BaseTariff is the base rate applied to usage.
	BaseTariff decimal.Decimal `json:"baseTariff"`

	// UnitOfMeasure represents the quantity step used for rating.
	// Default: 1
	UnitOfMeasure Quantity `json:"unitOfMeasure"`

	// Multiplier scales the base tariff.
	// Default: 1
	Multiplier decimal.Decimal `json:"multiplier"`

	// QoSProfile defines the quality-of-service profile for the rate.
	QosProfile string `json:"qosProfile,omitempty"`

	// MinimumUnits defines the minimum number of units charged.
	// Default: 1
	MinimumUnits Quantity `json:"minimumUnits"`

	// RoundingIncrement defines rounding granularity.
	// Default: 1
	RoundingIncrement Quantity `json:"roundingIncrement"`

	// Barred indicates the rate line cannot be used.
	Barred bool `json:"barred"`

	// MonetaryOnly indicates the rate line applies only to monetary charges.
	MonetaryOnly bool `json:"monetaryOnly"`
}

func (r *RateLine) NormaliseUnits(units int64) int64 {
	// Clamp to minimum.
	if units < r.MinimumUnits.AsUnits().IntPart() {
		units = r.MinimumUnits.AsUnits().IntPart()
	}

	inc := r.RoundingIncrement.AsUnits().IntPart()
	if inc <= 1 { // 0, 1, or misconfigured negative: no rounding needed
		return units
	}

	rem := units % inc
	if rem == 0 {
		return units
	}

	return units + (inc - rem) // round up to next multiple
}
