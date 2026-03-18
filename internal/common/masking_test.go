package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskMSISDN(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard 12-digit MSISDN",
			input:    "027123456789",
			expected: "027****789",
		},
		{
			name:     "9-digit local MSISDN",
			input:    "021123456",
			expected: "021****456",
		},
		{
			name:     "exactly 6 characters — masked entirely",
			input:    "123456",
			expected: "***",
		},
		{
			name:     "fewer than 6 characters — masked entirely",
			input:    "12345",
			expected: "***",
		},
		{
			name:     "empty string — masked entirely",
			input:    "",
			expected: "***",
		},
		{
			name:     "7 characters — minimum for partial masking",
			input:    "1234567",
			expected: "123****567",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, MaskMSISDN(tt.input))
		})
	}
}
