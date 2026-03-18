package model

import (
	"fmt"
	"strings"
)

type RatePlanType string

const (
	RatePlanSettlement RatePlanType = "SETTLEMENT"
	RatePlanWholesale  RatePlanType = "WHOLESALE"
	RatePlanRetail     RatePlanType = "RETAIL"
)

// String returns the string value of the RatePlanType
func (r RatePlanType) String() string {
	return string(r)
}

// ParseRatePlanType converts a string into a RatePlanType (case insensitive)
func ParseRatePlanType(value string) (RatePlanType, error) {
	switch strings.ToUpper(value) {
	case "SETTLEMENT":
		return RatePlanSettlement, nil
	case "WHOLESALE":
		return RatePlanWholesale, nil
	case "RETAIL":
		return RatePlanRetail, nil
	default:
		return "", fmt.Errorf("unknown RatePlanType: %s", value)
	}
}
