package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go-ocs/internal/common"
)

// makeLocalTime constructs a common.LocalTime at the given hour and minute,
// anchored to Go's zero date (0000-01-01) as produced by time.Parse("15:04").
func makeLocalTime(hour, minute int) common.LocalTime {
	return common.LocalTime{Time: time.Date(0, 1, 1, hour, minute, 0, 0, time.UTC)}
}

func TestServiceWindow_Duration(t *testing.T) {
	tests := []struct {
		name     string
		start    common.LocalTime
		end      common.LocalTime
		expected time.Duration
	}{
		{
			name:     "two hour window",
			start:    makeLocalTime(8, 0),
			end:      makeLocalTime(10, 0),
			expected: 2 * time.Hour,
		},
		{
			name:     "thirty minute window",
			start:    makeLocalTime(9, 0),
			end:      makeLocalTime(9, 30),
			expected: 30 * time.Minute,
		},
		{
			name:     "reversed start and end returns absolute duration",
			start:    makeLocalTime(10, 0),
			end:      makeLocalTime(8, 0),
			expected: 2 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sw := ServiceWindow{StartTime: tt.start, EndTime: tt.end}
			assert.Equal(t, tt.expected, sw.Duration())
		})
	}
}

func TestServiceWindow_IsWithin(t *testing.T) {
	// LocalTime is parsed from "HH:mm" and anchored to the zero date (0000-01-01).
	// The test timestamp must use the same date for meaningful comparisons.
	zeroDate := func(hour, minute int) time.Time {
		return time.Date(0, 1, 1, hour, minute, 0, 0, time.UTC)
	}

	sw := ServiceWindow{
		StartTime: makeLocalTime(8, 0),
		EndTime:   makeLocalTime(18, 0),
	}

	tests := []struct {
		name     string
		t        time.Time
		expected bool
	}{
		{"within window at noon", zeroDate(12, 0), true},
		{"at start boundary (not within)", zeroDate(8, 0), false},
		{"just after start", zeroDate(8, 1), true},
		{"just before end", zeroDate(17, 59), true},
		{"at end boundary (not within)", zeroDate(18, 0), false},
		{"before window", zeroDate(7, 0), false},
		{"after window", zeroDate(19, 0), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, sw.IsWithin(tt.t))
		})
	}
}
