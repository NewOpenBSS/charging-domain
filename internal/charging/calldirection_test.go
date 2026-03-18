package charging

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallDirection_String(t *testing.T) {
	tests := []struct {
		direction CallDirection
		expected  string
	}{
		{MO, "MO"},
		{MF, "MF"},
		{MT, "MT"},
		{ANY, "*"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.direction.String())
		})
	}
}

func TestParseCallDirection_ValidValues(t *testing.T) {
	tests := []struct {
		input    string
		expected CallDirection
	}{
		{"MO", MO},
		{"MF", MF},
		{"MT", MT},
		{"*", ANY},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseCallDirection(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestParseCallDirection_InvalidValue(t *testing.T) {
	_, err := ParseCallDirection("INVALID")
	require.Error(t, err)

	var ce *ChargingError
	require.True(t, errors.As(err, &ce), "expected *ChargingError")
	assert.Equal(t, CodeInvalidCallDirection, ce.Code)
	assert.Contains(t, ce.Message, "INVALID")
}

func TestParseCallDirection_EmptyString(t *testing.T) {
	_, err := ParseCallDirection("")
	require.Error(t, err)

	var ce *ChargingError
	require.True(t, errors.As(err, &ce))
	assert.Equal(t, CodeInvalidCallDirection, ce.Code)
}
