package quota

import (
	"go-ocs/internal/charging"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestCounter_ReserveServiceUnits(t *testing.T) {
	counterID := uuid.New()
	balance := decimal.NewFromInt(100)
	c := &Counter{
		CounterID:    counterID,
		UnitType:     charging.UNITS,
		Balance:      &balance,
		Reservations: make(map[uuid.UUID]Reservation),
	}

	ref := uuid.New()
	unitPrice := decimal.NewFromFloat(1.5)
	multiplier := decimal.NewFromInt(1)
	taxRate := decimal.NewFromFloat(0.1)
	validity := 1 * time.Hour

	now := time.Now()

	// Test successful reservation
	reserved := c.ReserveServiceUnits(ref, 10, unitPrice, multiplier, taxRate, ReasonServiceUsage, validity, now)
	assert.Equal(t, int64(10), reserved)
	assert.Contains(t, c.Reservations, ref)
	assert.Equal(t, int64(10), *c.Reservations[ref].Units)

	// Test insufficient balance
	ref2 := uuid.New()
	reserved2 := c.ReserveServiceUnits(ref2, 100, unitPrice, multiplier, taxRate, ReasonServiceUsage, validity, now)
	assert.Equal(t, int64(90), reserved2) // 100 - 10 already reserved
	assert.Contains(t, c.Reservations, ref2)

	// Test monetary counter (should not reserve service units)
	cMonetary := &Counter{UnitType: charging.MONETARY}
	reserved3 := cMonetary.ReserveServiceUnits(uuid.New(), 10, unitPrice, multiplier, taxRate, ReasonServiceUsage, validity, now)
	assert.Equal(t, int64(0), reserved3)
}

func TestCounter_IsMatching(t *testing.T) {
	ck := charging.RateKey{ServiceType: "VOICE", SourceType: "*", ServiceDirection: charging.ANY, ServiceCategory: "*"}
	rk1 := charging.RateKey{ServiceType: "VOICE", SourceType: "PREPAID", ServiceDirection: charging.MO, ServiceCategory: "DOMESTIC"}
	rk2 := charging.RateKey{ServiceType: "DATA", SourceType: "PREPAID", ServiceDirection: charging.MO, ServiceCategory: "DOMESTIC"}

	c := &Counter{
		UnitType: charging.UNITS,
		CounterSelectionKeys: []charging.RateKey{
			ck,
		},
	}

	// Request RateKey (k):       VOICE.PREPAID.MO.DOMESTIC
	// Counter Selection Key (other): VOICE.*.ANY.*

	// k.Matches(other)
	// ServiceType: VOICE == VOICE, score++ (1)
	// SourceType: PREPAID == *, wildmatch, score remains 1
	// ServiceDirection: MO == ANY (other), other.ServiceDirection is ANY, so it matches.
	//                   But score only increments if other.ServiceDirection != ANY.
	// ServiceCategory: DOMESTIC == *, wildmatch, score remains 1
	// ServiceWindow: "" == "", score++ (2)

	matched, score := rk1.Matches(ck)
	assert.True(t, matched)
	assert.Equal(t, 2, score)

	matched, score = c.IsMatching(rk1, charging.UNITS)
	// c.IsMatching calls ck.Matches(rk1).
	// Let's check ck.Matches(rk1)
	// ServiceType: VOICE == VOICE, score++ (1)
	// SourceType: * == PREPAID, wildmatch, score remains 1
	// ServiceDirection: ANY == MO, k.ServiceDirection is ANY (*), so it matches.
	//                   Score is only incremented if k.ServiceDirection != ANY.
	// ServiceCategory: * == DOMESTIC, wildmatch, score remains 1
	// ServiceWindow: "" == "", score++ (2)
	assert.True(t, matched)
	assert.Equal(t, 2, score)

	matched, score = c.IsMatching(rk2, charging.UNITS)
	assert.False(t, matched)
	assert.Equal(t, 0, score)

	matched, score = c.IsMatching(rk1, charging.MONETARY)
	assert.False(t, matched)
}

func TestCounter_AvailableValue(t *testing.T) {
	balance := decimal.NewFromInt(100)
	c := &Counter{
		UnitType: charging.MONETARY,
		Balance:  &balance,
		Reservations: map[uuid.UUID]Reservation{
			uuid.New(): {Value: decimalPtr(decimal.NewFromInt(20))},
			uuid.New(): {Value: decimalPtr(decimal.NewFromInt(30))},
		},
	}

	available := c.AvailableValue()
	assert.True(t, decimal.NewFromInt(50).Equal(available))

	cNonMonetary := &Counter{UnitType: charging.UNITS}
	assert.True(t, decimal.Zero.Equal(cNonMonetary.AvailableValue()))
}

func TestCounter_AvailableServiceUnits(t *testing.T) {
	balance := decimal.NewFromInt(100)
	c := &Counter{
		UnitType: charging.UNITS,
		Balance:  &balance,
		Reservations: map[uuid.UUID]Reservation{
			uuid.New(): {Units: int64Ptr(20)},
			uuid.New(): {Units: int64Ptr(30)},
		},
	}

	available := c.AvailableServiceUnits()
	assert.Equal(t, int64(50), available)

	cMonetary := &Counter{UnitType: charging.MONETARY}
	assert.Equal(t, int64(0), cMonetary.AvailableServiceUnits())
}

func TestCounter_ReserveValue(t *testing.T) {
	balance := decimal.NewFromInt(100)
	c := &Counter{
		UnitType:     charging.MONETARY,
		Balance:      &balance,
		Reservations: make(map[uuid.UUID]Reservation),
	}

	now := time.Now()
	ref := uuid.New()
	amount := decimal.NewFromInt(40)
	reserved := c.ReserveValue(ref, amount, decimal.Zero, decimal.NewFromInt(1), decimal.Zero, ReasonServiceUsage, 1*time.Hour, now)
	// balance is 100. amount is 40. multiplier is 1.
	// amount = 1 * 40 = 40.
	// avail = 100 - 0 = 100.
	// amount <= avail, so amount remains 40.
	// c.Reservations[ref] = 40.
	// return amount / multiplier = 40 / 1 = 40.
	assert.True(t, decimal.NewFromInt(40).Equal(reserved))
	assert.Contains(t, c.Reservations, ref)

	// Test exceeding balance
	ref2 := uuid.New()
	amount2 := decimal.NewFromInt(100)
	reserved2 := c.ReserveValue(ref2, amount2, decimal.Zero, decimal.NewFromInt(1), decimal.Zero, ReasonServiceUsage, 1*time.Hour, now)
	// balance is 100. reservations sum to 40.
	// avail = 100 - 40 = 60.
	// amount2 = 100.
	// amount2 > avail, so amount2 = 60.
	// c.Reservations[ref2] = 60.
	// return 60.
	assert.True(t, decimal.NewFromInt(60).Equal(reserved2))
}

func TestCounter_DebitBalance(t *testing.T) {
	balance := decimal.NewFromInt(100)
	c := &Counter{Balance: &balance}

	c.DebitBalance(decimal.NewFromInt(40))
	assert.True(t, decimal.NewFromInt(60).Equal(*c.Balance))

	c.DebitBalance(decimal.NewFromInt(100))
	assert.True(t, decimal.Zero.Equal(*c.Balance))
}

func TestCounter_ReleaseReservation(t *testing.T) {
	ref := uuid.New()
	c := &Counter{
		Reservations: map[uuid.UUID]Reservation{
			ref: {},
		},
	}
	assert.Len(t, c.Reservations, 1)
	c.ReleaseReservation(ref)
	assert.Len(t, c.Reservations, 0)
}

func TestQuota_FindCountersByReservationAndType(t *testing.T) {
	resID := uuid.New()
	q := &Quota{
		Counters: []Counter{
			{
				CounterID: uuid.New(),
				UnitType:  charging.UNITS,
				Priority:  10,
				Reservations: map[uuid.UUID]Reservation{
					resID: {},
				},
			},
			{
				CounterID: uuid.New(),
				UnitType:  charging.UNITS,
				Priority:  20,
				Reservations: map[uuid.UUID]Reservation{
					resID: {},
				},
			},
			{
				CounterID: uuid.New(),
				UnitType:  charging.MONETARY,
				Reservations: map[uuid.UUID]Reservation{
					resID: {},
				},
			},
		},
	}

	counters := q.FindCountersByReservationAndType(resID, charging.UNITS)
	assert.Len(t, counters, 2)
	assert.Equal(t, 20, counters[0].Priority)
	assert.Equal(t, 10, counters[1].Priority)
}

func TestQuota_FindCounters(t *testing.T) {
	rk := charging.RateKey{ServiceType: "VOICE", SourceType: "*", ServiceDirection: "ANY", ServiceCategory: "*"}
	q := &Quota{
		Counters: []Counter{
			{
				CounterID: uuid.New(),
				UnitType:  charging.UNITS,
				Priority:  10,
				CounterSelectionKeys: []charging.RateKey{
					{ServiceType: "VOICE", SourceType: "*", ServiceDirection: "ANY", ServiceCategory: "*"},
				},
			},
			{
				CounterID: uuid.New(),
				UnitType:  charging.UNITS,
				Priority:  20,
				CounterSelectionKeys: []charging.RateKey{
					{ServiceType: "VOICE", SourceType: "*", ServiceDirection: "ANY", ServiceCategory: "*"},
				},
			},
		},
	}

	counters := q.FindCounters(rk, charging.UNITS, ReasonServiceUsage)
	assert.Len(t, counters, 2)
	assert.Equal(t, 20, counters[0].Priority)
}

func TestReservation_CalculateTotalAmount(t *testing.T) {
	units := int64(10)
	unitPrice := decimal.NewFromFloat(2.0)
	taxRate := decimal.NewFromFloat(0.1)

	r := &Reservation{
		Units:     &units,
		UnitPrice: &unitPrice,
		TaxRate:   &taxRate,
	}

	total := r.CalculateTotalAmount()
	// 10 * 2.0 * (1 + 0.1) = 20 * 1.1 = 22.0
	assert.True(t, decimal.NewFromFloat(22.0).Equal(*total))
}

func TestLoan_Clawback(t *testing.T) {
	loan := &Loan{
		TransactFee:        decimal.NewFromInt(2),
		LoanBalance:        decimal.NewFromInt(10),
		MinRepayment:       decimal.NewFromInt(5),
		ClawbackPercentage: decimal.NewFromFloat(0.5),
	}

	// Provisioning 20. Clawback 50% = 10.
	// Loan balance is 10, which is more than MinRepayment (5).
	// So clawbackAmount = 10.
	// clawbackAmount >= TransactFee (10 >= 2).
	// feePaid = TransactFee = 2.
	// loanPaid = 10 - 2 = 8.
	// return loanPaid (8), feePaid (2).
	loanPaid, feePaid := loan.Clawback(decimal.NewFromInt(20))
	assert.True(t, decimal.NewFromInt(8).Equal(loanPaid))
	assert.True(t, decimal.NewFromInt(2).Equal(feePaid))
}

func TestQuota_FindCounterByID(t *testing.T) {
	counterID := uuid.New()
	q := &Quota{
		Counters: []Counter{
			{CounterID: counterID},
			{CounterID: uuid.New()},
		},
	}

	found := q.FindCounterByID(counterID)
	assert.NotNil(t, found)
	assert.Equal(t, counterID, found.CounterID)

	notFound := q.FindCounterByID(uuid.New())
	assert.Nil(t, notFound)
}

func TestReservation_CalcRemainingValue(t *testing.T) {
	unitPrice := decimal.NewFromFloat(1.5)
	multiplier := decimal.NewFromInt(2)
	taxRate := decimal.NewFromFloat(0.1)

	r := &Reservation{
		UnitPrice:  &unitPrice,
		Multiplier: &multiplier,
		TaxRate:    &taxRate,
	}

	// 1.5 * 2 * 10 * (1 + 0.1) = 3 * 10 * 1.1 = 33
	remaining := r.CalcRemainingValue(10)
	assert.True(t, decimal.NewFromFloat(33.0).Equal(remaining))
}

func TestNewEmptyQuota(t *testing.T) {
	q := NewEmptyQuota()
	assert.NotNil(t, q.QuotaID)
	assert.Empty(t, q.Counters)
}

func TestQuota_AddCounter(t *testing.T) {
	q := NewEmptyQuota()
	c := Counter{CounterID: uuid.New()}
	q.AddCounter(c)
	assert.Len(t, q.Counters, 1)
	assert.Equal(t, c.CounterID, q.Counters[0].CounterID)
}

func decimalPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}

func int64Ptr(i int64) *int64 {
	return &i
}
