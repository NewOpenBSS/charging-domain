package quota

import (
	"go-ocs/internal/charging"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Quota struct {
	QuotaID  uuid.UUID `json:"quotaId"`
	Counters []Counter `json:"counters"`
}

type Counter struct {
	CounterID uuid.UUID `json:"counterId"`

	ProductID uuid.UUID `json:"productId"`

	ProductName string `json:"productName"`

	Description string `json:"description"`

	UnitType charging.UnitType `json:"UnitType"`

	Priority int `json:"priority"`

	InitialBalance *decimal.Decimal `json:"initialBalance"`

	Balance *decimal.Decimal `json:"balance"`

	Expiry *time.Time `json:"expiry,omitempty"`

	Reservations map[uuid.UUID]Reservation `json:"reservations,omitempty"`

	CanTransfer bool `json:"canTransfer"`

	CanConvert bool `json:"canConvert"`

	UnitPrice *decimal.Decimal `json:"UnitPrice"`

	TaxRate *decimal.Decimal `json:"taxRate"`

	Notifications *Notifications `json:"notifications,omitempty"`

	Loan *Loan `json:"loan,omitempty"`

	CounterSelectionKeys []charging.RateKey `json:"counterSelectionKeys"`

	ExternalReference string `json:"externalReference,omitempty"`
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
func (c *Counter) ReserveServiceUnits(reference uuid.UUID, nrServiceUnits int64, unitPrice decimal.Decimal,
	multiplier decimal.Decimal, taxRate decimal.Decimal, reason ReasonCode, validityTime time.Duration, now time.Time) int64 {

	if c.UnitType == charging.MONETARY {
		return 0
	}

	nrMultiplierUnits := multiplier.Mul(decimal.NewFromInt(nrServiceUnits)).IntPart()
	reservedUnits := minInt64(c.AvailableServiceUnits(), nrMultiplierUnits)
	if reservedUnits > 0 {
		expiry := now.Add(validityTime)
		res := NewReservation(decimal.Zero, reservedUnits, unitPrice, multiplier, taxRate, reason, expiry)
		c.Reservations[reference] = *res
	}

	return reservedUnits
}

func (c *Counter) IsMatching(rateKey charging.RateKey, unitType charging.UnitType) (bool, int) {
	if c.UnitType != unitType {
		return false, 0
	}
	for _, ck := range c.CounterSelectionKeys {
		if matched, score := ck.Matches(rateKey); matched {
			return true, score
		}
	}
	return false, 0
}

func (c *Counter) AvailableValue() decimal.Decimal {
	if c.UnitType != charging.MONETARY {
		return decimal.Zero
	}

	totalReserved := decimal.Zero

	for _, r := range c.Reservations {
		totalReserved = totalReserved.Add(*r.Value)
	}

	return c.Balance.Sub(totalReserved)
}

func (c *Counter) AvailableServiceUnits() int64 {

	if c.UnitType == charging.MONETARY {
		return 0
	}

	var totalReserved int64 = 0

	for _, r := range c.Reservations {
		totalReserved += *r.Units
	}

	return c.Balance.IntPart() - totalReserved
}

func (c *Counter) ReserveValue(reference uuid.UUID, amount decimal.Decimal, unitPrice decimal.Decimal, multiplier decimal.Decimal, taxRate decimal.Decimal, reason ReasonCode, validityTime time.Duration, now time.Time) decimal.Decimal {

	if c.UnitType != charging.MONETARY {
		return decimal.Zero
	}

	//Apply the multiplier
	amount = multiplier.Mul(amount)

	avail := c.AvailableValue()
	if avail.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}

	if amount.GreaterThan(avail) {
		amount = avail
	}

	if amount.GreaterThan(decimal.Zero) {
		expiry := now.Add(validityTime)
		res := NewReservation(amount, 0, unitPrice, multiplier, taxRate, reason, expiry)
		c.Reservations[reference] = *res

		//Remove the multiplier
		amount = amount.Div(multiplier)
	}

	return amount
}

func (c *Counter) DebitBalance(units decimal.Decimal) {
	newBal := c.Balance.Sub(units)
	if newBal.LessThanOrEqual(decimal.Zero) {
		newBal = decimal.Zero
	}
	c.Balance = &newBal
}

func (c *Counter) ReleaseReservation(referenceId uuid.UUID) {
	delete(c.Reservations, referenceId)
}

func (q *Quota) FindCountersByReservationAndType(reservationId uuid.UUID, unitType charging.UnitType) (counters []*Counter) {

	list := make([]*Counter, 0, len(q.Counters))
	for i := range q.Counters {
		c := &q.Counters[i]

		_, ok := c.Reservations[reservationId]
		if ok && c.UnitType == unitType {
			list = append(list, c)
		}
	}

	// Sort counters by priority (highest priority first)
	sort.Slice(list, func(i, j int) bool {
		return list[i].Priority > list[j].Priority
	})

	return list
}

func (q *Quota) ReleaseReservations(reservationId uuid.UUID) {
	for i := range q.Counters {
		c := &q.Counters[i]
		c.ReleaseReservation(reservationId)
	}
}

func NewEmptyQuota() *Quota {
	return &Quota{
		QuotaID:  uuid.New(),
		Counters: []Counter{},
	}
}

func (q *Quota) AddCounter(counter Counter) {
	q.Counters = append(q.Counters, counter)
}

func (q *Quota) FindCounterByID(counterID uuid.UUID) *Counter {
	for i := range q.Counters {
		if q.Counters[i].CounterID == counterID {
			return &q.Counters[i]
		}
	}
	return nil
}

func (q *Quota) FindCounters(rateKey charging.RateKey, unitType charging.UnitType, reasonCode ReasonCode) []*Counter {

	list := make([]*Counter, 0, len(q.Counters))

	for i := range q.Counters {
		c := &q.Counters[i]
		if matched, _ := c.IsMatching(rateKey, unitType); matched {
			if reasonCode != ReasonConversion || c.CanConvert {
				list = append(list, c)
			}
		}
	}

	// Sort counters by priority (highest priority first)
	sort.Slice(list, func(i, j int) bool {
		return list[i].Priority > list[j].Priority
	})

	return list
}

type Reservation struct {
	Units *int64 `json:"units,omitempty"`

	Value *decimal.Decimal `json:"value,omitempty"`

	UnitPrice *decimal.Decimal `json:"UnitPrice,omitempty"`

	TaxRate *decimal.Decimal `json:"taxRate,omitempty"`

	Multiplier *decimal.Decimal `json:"Multiplier,omitempty"`

	Reason ReasonCode `json:"Reason"`

	Expiry time.Time `json:"expiry"`
}

func NewReservation(value decimal.Decimal,
	units int64,
	unitPrice decimal.Decimal,
	multiplier decimal.Decimal,
	taxRate decimal.Decimal,
	reason ReasonCode,
	expiry time.Time) *Reservation {
	return &Reservation{
		Units:      &units,
		Value:      &value,
		UnitPrice:  &unitPrice,
		Multiplier: &multiplier,
		TaxRate:    &taxRate,
		Reason:     reason,
		Expiry:     expiry,
	}
}

func (r *Reservation) CalculateTotalAmount() *decimal.Decimal {

	if r.UnitPrice == nil || r.Units == nil || r.TaxRate == nil {
		res := decimal.NewFromInt(0)
		return &res
	}

	units := decimal.NewFromInt(*r.Units)
	value := r.UnitPrice.Mul(units)

	tax := r.TaxRate.Add(decimal.NewFromInt(1))
	res := value.Mul(tax)
	return &res
}

func (r *Reservation) CalcRemainingValue(usedUnits int64) decimal.Decimal {
	value := r.UnitPrice.Mul(*r.Multiplier)
	value = value.Mul(decimal.NewFromInt(usedUnits))
	value = value.Mul(decimal.NewFromInt(1).Add(*r.TaxRate))

	return value
}

type Notifications struct {
	// Percentage thresholds (for example 50, 80, 100) at which notifications should be triggered.
	Thresholds []int `json:"thresholds"`

	// The most recent threshold value for which a notification was sent.
	LastThresholdNotified *int `json:"lastThresholdNotified,omitempty"`

	// Number of days before expiry when a notification should be sent. 0 implies none to be sent.
	DaysBeforeExpiry []int64 `json:"daysBeforeExpiry"`

	// The most recent day before expiry for which a notification was sent.
	LastDayBeforeExpiryNotified *int64 `json:"lastDayBeforeExpiryNotified,omitempty"`
}

type Loan struct {
	// The transaction fee associated with the loan and is included in the loanBalance.
	// This fee is to be paid first before the loan portion of the loan.
	TransactFee decimal.Decimal `json:"transactFee"`

	// The current outstanding loan balance for this quota counter.
	LoanBalance decimal.Decimal `json:"loanBalance"`

	// The minimum repayment amount for the loan.
	MinRepayment decimal.Decimal `json:"minRepayment"`

	// The percentage of incoming balance used to repay the loan (0–1).
	ClawbackPercentage decimal.Decimal `json:"clawbackPercentage"`
}

func (l *Loan) Clawback(provisioningValue decimal.Decimal) (decimal.Decimal, decimal.Decimal) {

	feeAmountPaid := decimal.Zero
	loanAmountPaid := decimal.Zero

	if l.LoanBalance.GreaterThan(decimal.Zero) {

		var clawbackAmount decimal.Decimal

		if l.LoanBalance.LessThanOrEqual(l.MinRepayment) {
			clawbackAmount = decimal.Min(l.LoanBalance, provisioningValue)

		} else {

			if l.ClawbackPercentage.Equal(decimal.Zero) {
				clawbackAmount = decimal.Min(l.MinRepayment, provisioningValue)
			} else {
				clawbackAmount = provisioningValue.Mul(l.ClawbackPercentage)
			}
		}

		if clawbackAmount.LessThan(l.TransactFee) {

			feeAmountPaid = clawbackAmount

		} else {

			feeAmountPaid = l.TransactFee
			loanAmountPaid = clawbackAmount.Sub(feeAmountPaid)

			if loanAmountPaid.GreaterThanOrEqual(l.LoanBalance) {
				loanAmountPaid = l.LoanBalance
			}
		}

		l.TransactFee = l.TransactFee.Sub(feeAmountPaid)
		l.LoanBalance = l.LoanBalance.Sub(loanAmountPaid)
	}

	return loanAmountPaid, feeAmountPaid
}
