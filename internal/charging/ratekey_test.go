package charging

import (
	"testing"
)

func TestRateKey_String(t *testing.T) {
	tests := []struct {
		name     string
		rk       RateKey
		expected string
	}{
		{
			name: "With ServiceWindow",
			rk: RateKey{
				ServiceType:      "Voice",
				SourceType:       "Postpaid",
				ServiceDirection: MO,
				ServiceCategory:  "Local",
				ServiceWindow:    "Night",
			},
			expected: "Voice.Postpaid.MO.Local.Night",
		},
		{
			name: "Without ServiceWindow",
			rk: RateKey{
				ServiceType:      "Data",
				SourceType:       "Prepaid",
				ServiceDirection: ANY,
				ServiceCategory:  "Standard",
				ServiceWindow:    "",
			},
			expected: "Data.Prepaid.*.Standard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rk.String(); got != tt.expected {
				t.Errorf("RateKey.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *RateKey
	}{
		{
			name:  "Standard string",
			input: "Voice.Postpaid.MO.Local.Night",
			expected: &RateKey{
				ServiceType:      "Voice",
				SourceType:       "Postpaid",
				ServiceDirection: MO,
				ServiceCategory:  "Local",
				ServiceWindow:    "Night",
			},
		},
		{
			name:  "Wildcard direction",
			input: "Data.Prepaid.*.Standard.Peak",
			expected: &RateKey{
				ServiceType:      "Data",
				SourceType:       "Prepaid",
				ServiceDirection: ANY,
				ServiceCategory:  "Standard",
				ServiceWindow:    "Peak",
			},
		},
		{
			name:     "Invalid direction returns nil",
			input:    "SMS.Postpaid.INVALID.Global.W1",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromString(tt.input)
			if tt.expected == nil {
				if got != nil {
					t.Errorf("FromString(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("FromString(%q) returned nil, want %v", tt.input, tt.expected)
			}
			if got.ServiceType != tt.expected.ServiceType ||
				got.SourceType != tt.expected.SourceType ||
				got.ServiceDirection != tt.expected.ServiceDirection ||
				got.ServiceCategory != tt.expected.ServiceCategory ||
				got.ServiceWindow != tt.expected.ServiceWindow {
				t.Errorf("FromString() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRateKey_Matches(t *testing.T) {
	tests := []struct {
		name      string
		k         RateKey // The search key (actual event)
		other     RateKey // The pattern (from RateLine)
		wantMatch bool
		wantScore int
	}{
		{
			name: "Exact match no window",
			k: RateKey{
				ServiceType:      "Voice",
				SourceType:       "Postpaid",
				ServiceDirection: MO,
				ServiceCategory:  "Local",
				ServiceWindow:    "",
			},
			other: RateKey{
				ServiceType:      "Voice",
				SourceType:       "Postpaid",
				ServiceDirection: MO,
				ServiceCategory:  "Local",
				ServiceWindow:    "",
			},
			wantMatch: true,
			wantScore: 5,
		},
		{
			name: "Exact match with window",
			k: RateKey{
				ServiceType:      "Voice",
				SourceType:       "Postpaid",
				ServiceDirection: MO,
				ServiceCategory:  "Local",
				ServiceWindow:    "Peak",
			},
			other: RateKey{
				ServiceType:      "Voice",
				SourceType:       "Postpaid",
				ServiceDirection: MO,
				ServiceCategory:  "Local",
				ServiceWindow:    "Peak",
			},
			wantMatch: true,
			wantScore: 5,
		},
		{
			name: "Wildcard service type in other matches",
			k: RateKey{
				ServiceType:      "Voice",
				SourceType:       "Postpaid",
				ServiceDirection: MO,
				ServiceCategory:  "Local",
				ServiceWindow:    "Peak",
			},
			other: RateKey{
				ServiceType:      "*",
				SourceType:       "Postpaid",
				ServiceDirection: MO,
				ServiceCategory:  "Local",
				ServiceWindow:    "Peak",
			},
			wantMatch: true,
			wantScore: 4,
		},
		{
			name: "Wildcard service direction in other matches",
			k: RateKey{
				ServiceType:      "Voice",
				SourceType:       "Postpaid",
				ServiceDirection: MO,
				ServiceCategory:  "Local",
				ServiceWindow:    "Peak",
			},
			other: RateKey{
				ServiceType:      "Voice",
				SourceType:       "Postpaid",
				ServiceDirection: ANY,
				ServiceCategory:  "Local",
				ServiceWindow:    "Peak",
			},
			wantMatch: true,
			wantScore: 4,
		},
		{
			name: "Mismatch in ServiceType",
			k: RateKey{
				ServiceType:      "SMS",
				SourceType:       "Postpaid",
				ServiceDirection: MO,
				ServiceCategory:  "Local",
				ServiceWindow:    "Peak",
			},
			other: RateKey{
				ServiceType:      "Voice",
				SourceType:       "Postpaid",
				ServiceDirection: MO,
				ServiceCategory:  "Local",
				ServiceWindow:    "Peak",
			},
			wantMatch: false,
			wantScore: 0,
		},
		{
			name: "ServiceWindow wildcard match in other",
			k: RateKey{
				ServiceType:      "Voice",
				SourceType:       "Postpaid",
				ServiceDirection: MO,
				ServiceCategory:  "Local",
				ServiceWindow:    "Peak",
			},
			other: RateKey{
				ServiceType:      "Voice",
				SourceType:       "Postpaid",
				ServiceDirection: MO,
				ServiceCategory:  "Local",
				ServiceWindow:    "*",
			},
			wantMatch: true,
			wantScore: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMatch, gotScore := tt.k.Matches(tt.other)
			if gotMatch != tt.wantMatch {
				t.Errorf("RateKey.Matches() gotMatch = %v, want %v", gotMatch, tt.wantMatch)
			}
			if gotScore != tt.wantScore {
				t.Errorf("RateKey.Matches() gotScore = %v, want %v", gotScore, tt.wantScore)
			}
		})
	}
}
