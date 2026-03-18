package ocserrors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOcsError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *OcsError
		expected string
	}{
		{"general error", CreateGeneralError("something went wrong"), "GENERAL_ERROR: something went wrong"},
		{"out of funds", CreateOutOfFunds("no credit"), "OUT_OF_FUNDS: no credit"},
		{"unknown subscriber", CreateUnknownSubscriber("not found"), "UNKNOWN_SUBSCRIBER: not found"},
		{"unknown carrier", CreateUnknownCarrier("carrier X"), "UNKNOWN_CARRIER: carrier X"},
		{"classification error", CreateClassificationError("no match"), "UNABLE_TO_CLASSIFY: no match"},
		{"no rating entry", CreateNoRatingEntry("rate key missing"), "NO_RATING_ENTRY: rate key missing"},
		{"service barred", CreateServiceBarred("barred"), "SERVICE_BARRED: barred"},
		{"used more than granted", CreateUsedMoreThanGranted("overuse"), "USED_MORE_THAN_GRANTED: overuse"},
		{"invalid referenced", CreateInvalidReferenced("bad ref"), "INVALID_REFERENCE: bad ref"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestOcsError_ErrorsAs(t *testing.T) {
	err := CreateGeneralError("test")
	var oe *OcsError
	require.True(t, errors.As(err, &oe))
	assert.Equal(t, CodeGeneralError, oe.Code)
	assert.Equal(t, "test", oe.Message)
}

func TestOcsError_Codes(t *testing.T) {
	tests := []struct {
		name string
		err  *OcsError
		code Code
	}{
		{"CreateGeneralError", CreateGeneralError("m"), CodeGeneralError},
		{"CreateOutOfFunds", CreateOutOfFunds("m"), CodeOutOfFunds},
		{"CreateUnknownSubscriber", CreateUnknownSubscriber("m"), CodeUnknownSubscriber},
		{"CreateUnknownCarrier", CreateUnknownCarrier("m"), CodeUnknownCarrier},
		{"CreateClassificationError", CreateClassificationError("m"), CodeUnabledToClassification},
		{"CreateNoRatingEntry", CreateNoRatingEntry("m"), CodeNoRatingEntry},
		{"CreateServiceBarred", CreateServiceBarred("m"), CodeServiceBarred},
		{"CreateUsedMoreThanGranted", CreateUsedMoreThanGranted("m"), CodeUsedMoreThanGranted},
		{"CreateInvalidReferenced", CreateInvalidReferenced("m"), CodeInvalidReference},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.Code)
		})
	}
}

func TestRetransmitError_Error(t *testing.T) {
	rt := CreateRetransmit(nil)
	assert.Equal(t, "RETRANSMIT: Retransmitting session", rt.Error())
}

func TestRetransmitError_GetResponse(t *testing.T) {
	t.Run("nil response", func(t *testing.T) {
		rt := CreateRetransmit(nil)
		assert.Nil(t, rt.GetResponse())
	})
}

func TestRetransmitError_Code(t *testing.T) {
	rt := CreateRetransmit(nil)
	assert.Equal(t, CodeRetransmit, rt.code)
}
