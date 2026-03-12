package charging

import (
	"encoding/json"
	"fmt"
	"strings"
)

type RateKey struct {
	ServiceType      string
	SourceType       string
	ServiceDirection CallDirection
	ServiceCategory  string
	ServiceWindow    string
}

func (k *RateKey) String() string {
	sd := k.ServiceType + "." + k.SourceType + "." + string(k.ServiceDirection) + "." + k.ServiceCategory
	if k.ServiceWindow == "" {
		return sd
	}

	return sd + "." + k.ServiceWindow
}

// MarshalJSON serialises the rate key as a single dot-separated string.
func (k RateKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

// UnmarshalJSON deserialises the rate key from a single dot-separated string.
func (k *RateKey) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	r, err := ParseRateKey(s)
	if err != nil {
		return err
	}

	*k = *r
	return nil
}

// Matches
// Determine if the current RateKey matches the provided other RateKey based on a set
// of predefined matching rules.
//
// A match occurs if all fields of the current  RateKey conform to the criteria defined by
// the corresponding fields of the provided other RateKey.
// Specific rules include:
// - Fields with wildcard or ANY values are treated as matching any value.
// - empty `serviceWindow` is considered a match if the other `serviceWindow` is empty or a wildcard.
//
// The specificity score is calculated based on the number of matching actual fields and the
// presence of a wildcard in the corresponding fields.

func (k *RateKey) Matches(other RateKey) (bool, int) {

	mf := func(value, pattern string) (bool, bool) {
		wildmatch := "*" == pattern || "*" == value
		return wildmatch || pattern == value, wildmatch
	}

	score := 0
	m, wm := mf(k.ServiceType, other.ServiceType)
	if !m {
		return false, score
	}
	if !wm {
		score++
	}

	m, wm = mf(k.SourceType, other.SourceType)
	if !m {
		return false, score
	}
	if !wm {
		score++
	}

	if other.ServiceDirection != ANY && k.ServiceDirection != ANY && k.ServiceDirection != other.ServiceDirection {
		return false, score
	}
	if other.ServiceDirection != ANY && k.ServiceDirection != ANY {
		score++
	}

	m, wm = mf(k.ServiceCategory, other.ServiceCategory)
	if !m {
		return false, score
	}
	if !wm {
		score++
	}

	if k.ServiceWindow == "" && other.ServiceWindow == "" {
		score++
		return true, score
	}

	w, wm := mf(k.ServiceWindow, other.ServiceWindow)
	if w && !wm {
		score++
	}

	return w, score
}

func ParseRateKey(s string) (*RateKey, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 4 && len(parts) != 5 {
		return nil, fmt.Errorf("invalid rate key %q: expected 4 or 5 components", s)
	}

	cd, err := ParseCallDirection(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid service direction %q in rate key %q: %w", parts[2], s, err)
	}

	rk := &RateKey{
		ServiceType:      parts[0],
		SourceType:       parts[1],
		ServiceDirection: cd,
		ServiceCategory:  parts[3],
	}
	if len(parts) == 5 {
		rk.ServiceWindow = parts[4]
	}

	return rk, nil
}

// FromString is kept as a compatibility wrapper around ParseRateKey.
func FromString(s string) *RateKey {
	rk, err := ParseRateKey(s)
	if err != nil {
		return nil
	}
	return rk
}
