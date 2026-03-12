package common

import (
	"encoding/json"
	"fmt"
	"time"
)

const timeLayout = "15:04" // Go equivalent of "HH:mm"

type LocalTime struct {
	time.Time
}

func (lt *LocalTime) UnmarshalJSON(b []byte) error {
	// Expect a JSON string
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	t, err := time.Parse(timeLayout, s)
	if err != nil {
		return fmt.Errorf("invalid time format, expected HH:mm: %w", err)
	}

	lt.Time = t
	return nil
}

func (lt *LocalTime) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	t, err := time.Parse(timeLayout, s)
	if err != nil {
		return fmt.Errorf("invalid time format, expected HH:mm: %w", err)
	}

	lt.Time = t
	return nil
}

func (lt LocalTime) Duration(other LocalTime) time.Duration {
	if lt.Time.After(other.Time) {
		return other.Time.Sub(lt.Time)
	}
	return lt.Time.Sub(other.Time)
}

func (lt LocalTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(lt.Format(timeLayout))
}
