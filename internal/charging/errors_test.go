package charging

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChargingError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ChargingError
		expected string
	}{
		{
			name:     "invalid call direction",
			err:      NewInvalidCallDirection("INVALID"),
			expected: "INVALID_CALL_DIRECTION: invalid CallDirection: INVALID",
		},
		{
			name:     "invalid rate key",
			err:      NewInvalidRateKey("invalid rate key \"x\": expected 4 or 5 components"),
			expected: "INVALID_RATE_KEY: invalid rate key \"x\": expected 4 or 5 components",
		},
		{
			name:     "invalid unit type",
			err:      NewInvalidUnitType("BANANAS"),
			expected: "INVALID_UNIT_TYPE: invalid UnitType: BANANAS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestChargingError_ErrorsAs(t *testing.T) {
	t.Run("ParseCallDirection returns ChargingError", func(t *testing.T) {
		_, err := ParseCallDirection("UNKNOWN")
		assert.Error(t, err)
		var ce *ChargingError
		assert.True(t, errors.As(err, &ce))
		assert.Equal(t, CodeInvalidCallDirection, ce.Code)
	})

	t.Run("ParseRateKey returns ChargingError for bad component count", func(t *testing.T) {
		_, err := ParseRateKey("only.three.parts")
		assert.Error(t, err)
		var ce *ChargingError
		assert.True(t, errors.As(err, &ce))
		assert.Equal(t, CodeInvalidRateKey, ce.Code)
	})

	t.Run("ParseRateKey returns ChargingError for invalid direction", func(t *testing.T) {
		_, err := ParseRateKey("Voice.Home.INVALID.Local")
		assert.Error(t, err)
		var ce *ChargingError
		assert.True(t, errors.As(err, &ce))
		assert.Equal(t, CodeInvalidRateKey, ce.Code)
	})

	t.Run("ParseUnitType returns ChargingError", func(t *testing.T) {
		_, err := ParseUnitType("BANANAS")
		assert.Error(t, err)
		var ce *ChargingError
		assert.True(t, errors.As(err, &ce))
		assert.Equal(t, CodeInvalidUnitType, ce.Code)
	})
}
