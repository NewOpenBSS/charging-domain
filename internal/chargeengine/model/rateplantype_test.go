package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRatePlanType_String(t *testing.T) {
	tests := []struct {
		rpt      RatePlanType
		expected string
	}{
		{RatePlanSettlement, "SETTLEMENT"},
		{RatePlanWholesale, "WHOLESALE"},
		{RatePlanRetail, "RETAIL"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.rpt.String())
		})
	}
}

func TestParseRatePlanType_ValidValues(t *testing.T) {
	tests := []struct {
		input    string
		expected RatePlanType
	}{
		{"SETTLEMENT", RatePlanSettlement},
		{"settlement", RatePlanSettlement},
		{"Settlement", RatePlanSettlement},
		{"WHOLESALE", RatePlanWholesale},
		{"wholesale", RatePlanWholesale},
		{"RETAIL", RatePlanRetail},
		{"retail", RatePlanRetail},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseRatePlanType(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestParseRatePlanType_InvalidValue(t *testing.T) {
	_, err := ParseRatePlanType("UNKNOWN")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "UNKNOWN")
}

func TestParseRatePlanType_EmptyString(t *testing.T) {
	_, err := ParseRatePlanType("")
	require.Error(t, err)
}
