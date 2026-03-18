package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

// Quantity represents a unit-based quantity that may be provided as either
// a plain integer or a namespaced string such as "10kb", "5minutes", or "1unit".
//
// It is intentionally generic so it can be reused for fields such as
// unitOfMeasure, minimumUnits, and roundingIncrement.
type Quantity int64

// UnitOfMeasure is kept as a compatibility alias.
type UnitOfMeasure = Quantity

// RoundingIncrement can reuse the same parsing rules.
type RoundingIncrement = Quantity

func (u *Quantity) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*u = 1
		return nil
	}

	var number int64
	if err := json.Unmarshal(data, &number); err == nil {
		if number == 0 {
			number = 1
		}
		*u = Quantity(number)
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("unitOfMeasure must be a number or string: %w", err)
	}

	value, err := parseUnitOfMeasure(text)
	if err != nil {
		return err
	}

	*u = Quantity(value)
	return nil
}

func (u Quantity) AsUnits() decimal.Decimal {
	return decimal.NewFromInt(int64(u))
}

func parseUnitOfMeasure(value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 1, nil
	}

	lower := strings.ToLower(trimmed)
	multipliers := map[string]int64{
		"kb":      1024,
		"mb":      1024 * 1024,
		"gb":      1024 * 1024 * 1024,
		"minutes": 60,
		"min":     60,
		"seconds": 1,
		"sec":     1,
		"s":       1,
		"b":       1,
		"units":   1,
		"unit":    1,
	}

	for suffix, multiplier := range multipliers {
		if strings.HasSuffix(lower, suffix) {
			numberPart := strings.TrimSpace(trimmed[:len(trimmed)-len(suffix)])
			if numberPart == "" {
				return 0, fmt.Errorf("invalid unitOfMeasure %q", value)
			}

			number, err := strconv.ParseInt(numberPart, 10, 64)
			if err != nil {
				continue
			}

			if number == 0 {
				return 1, nil
			}

			return number * multiplier, nil
		}
	}

	number, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid unitOfMeasure %q", value)
	}

	if number == 0 {
		return 1, nil
	}

	return number, nil
}
