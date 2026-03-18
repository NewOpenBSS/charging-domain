package model

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuantity_UnmarshalJSON_Number(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"plain integer", `10`, 10},
		{"large integer", `1048576`, 1048576},
		{"zero becomes one", `0`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q Quantity
			err := json.Unmarshal([]byte(tt.input), &q)
			require.NoError(t, err)
			assert.Equal(t, Quantity(tt.expected), q)
		})
	}
}

func TestQuantity_UnmarshalJSON_Null(t *testing.T) {
	var q Quantity
	err := json.Unmarshal([]byte(`null`), &q)
	require.NoError(t, err)
	assert.Equal(t, Quantity(1), q)
}

func TestQuantity_UnmarshalJSON_StringWithUnit(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"kilobytes", `"10kb"`, 10 * 1024},
		{"megabytes", `"2mb"`, 2 * 1024 * 1024},
		{"gigabytes", `"1gb"`, 1 * 1024 * 1024 * 1024},
		{"minutes", `"5minutes"`, 5 * 60},
		{"min", `"3min"`, 3 * 60},
		{"seconds", `"30seconds"`, 30},
		{"sec", `"10sec"`, 10},
		{"s suffix", `"60s"`, 60},
		{"bytes", `"512b"`, 512},
		{"units", `"100units"`, 100},
		{"unit", `"50unit"`, 50},
		{"zero units becomes one", `"0kb"`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q Quantity
			err := json.Unmarshal([]byte(tt.input), &q)
			require.NoError(t, err)
			assert.Equal(t, Quantity(tt.expected), q)
		})
	}
}

func TestQuantity_UnmarshalJSON_StringPlainNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"plain string number", `"42"`, 42},
		{"zero string becomes one", `"0"`, 1},
		{"empty string becomes one", `""`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q Quantity
			err := json.Unmarshal([]byte(tt.input), &q)
			require.NoError(t, err)
			assert.Equal(t, Quantity(tt.expected), q)
		})
	}
}

func TestQuantity_UnmarshalJSON_InvalidString(t *testing.T) {
	invalidInputs := []struct {
		name  string
		input string
	}{
		{"bare letters", `"abc"`},
		{"only unit suffix", `"kb"`},
	}

	for _, tt := range invalidInputs {
		t.Run(tt.name, func(t *testing.T) {
			var q Quantity
			err := json.Unmarshal([]byte(tt.input), &q)
			assert.Error(t, err)
		})
	}
}

func TestQuantity_AsUnits(t *testing.T) {
	tests := []struct {
		quantity Quantity
		expected decimal.Decimal
	}{
		{Quantity(1), decimal.NewFromInt(1)},
		{Quantity(100), decimal.NewFromInt(100)},
		{Quantity(1024), decimal.NewFromInt(1024)},
	}

	for _, tt := range tests {
		t.Run(tt.expected.String(), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.quantity.AsUnits())
		})
	}
}
