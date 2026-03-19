package quota

import (
	"go-ocs/internal/charging"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BalanceQuery defines the filter criteria for a balance inquiry.
// Nil pointer fields mean "no filter on this dimension" — all counters
// matching the non-nil criteria are included.
type BalanceQuery struct {
	// UnitType restricts results to counters of this unit type.
	// nil returns counters of all unit types.
	UnitType *charging.UnitType

	// Transferable restricts results to counters where CanTransfer matches.
	// nil returns counters regardless of their CanTransfer flag.
	Transferable *bool

	// Convertible restricts results to counters where CanConvert matches.
	// nil returns counters regardless of their CanConvert flag.
	Convertible *bool
}

// CounterBalance is the balance result for a single matching counter.
type CounterBalance struct {
	CounterID        uuid.UUID
	ProductID        uuid.UUID
	ProductName      string
	UnitType         charging.UnitType
	TotalBalance     decimal.Decimal
	AvailableBalance decimal.Decimal
	Expiry           *time.Time
	CanTransfer      bool
	CanConvert       bool
}

// matches reports whether the counter satisfies all non-nil criteria in q.
func (q BalanceQuery) matches(c *Counter) bool {
	if q.UnitType != nil && c.UnitType != *q.UnitType {
		return false
	}
	if q.Transferable != nil && c.CanTransfer != *q.Transferable {
		return false
	}
	if q.Convertible != nil && c.CanConvert != *q.Convertible {
		return false
	}
	return true
}

// counterToBalance converts a matching Counter into a CounterBalance result.
func counterToBalance(c *Counter) *CounterBalance {
	cb := &CounterBalance{
		CounterID:   c.CounterID,
		ProductID:   c.ProductID,
		ProductName: c.ProductName,
		UnitType:    c.UnitType,
		CanTransfer: c.CanTransfer,
		CanConvert:  c.CanConvert,
	}

	if c.Balance != nil {
		cb.TotalBalance = *c.Balance
	}

	if c.UnitType == charging.MONETARY {
		cb.AvailableBalance = c.AvailableValue()
	} else {
		cb.AvailableBalance = decimal.NewFromInt(c.AvailableServiceUnits())
	}

	if c.Expiry != nil {
		t := *c.Expiry
		cb.Expiry = &t
	}

	return cb
}
