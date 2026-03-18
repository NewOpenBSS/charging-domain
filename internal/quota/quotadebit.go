package quota

import (
	"context"
	"fmt"
	"go-ocs/internal/charging"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type DebitResponse struct {

	/**
	 * Represents the number of units that have been debited.
	 * This field indicates the quantity of units deducted from the
	 * available balance as part of the debit operation.
	 */
	UnitsDebited int64

	/**
	 * Represents the monetary value associated with the debited units.
	 * This field indicates the financial worth of the units deducted from
	 * the available balance during the debit operation. It is expressed
	 * as a decimal value to allow precise representations of currency.
	 */
	UnitsValue decimal.Decimal

	/**
	 * Represents the number of units that have been accounted for in the debit operation.
	 */
	ValueUnits int64

	/**
	 * Represents the number of units that could not be accounted for in the debit operation.
	 * This field indicates the quantity of units that remain untracked or unclassified
	 * following the processing of the debit request. It is used to capture scenarios where
	 * discrepancies between requested and processed units arise.
	 */
	UnaccountedUnits int64
}

func NewDebitResponse(unitsDebited int64, unitsValue decimal.Decimal, valueUnits int64, unaccountedUnits int64) *DebitResponse {
	return &DebitResponse{
		UnitsDebited:     unitsDebited,
		UnitsValue:       unitsValue,
		ValueUnits:       valueUnits,
		UnaccountedUnits: unaccountedUnits,
	}
}

// Debit applies used units against the subscriber's quota reservation. now is the reference
// time for journal event timestamps, ensuring deterministic audit records.
func (m *QuotaManager) Debit(ctx context.Context,
	now time.Time,
	subscriberId uuid.UUID,
	requestId string,
	reservationId uuid.UUID,
	usedUnits int64,
	unitType charging.UnitType,
	reclaimUnusedUnits bool) (*DebitResponse, error) {

	resp := NewDebitResponse(0, decimal.Zero, 0, 0)
	err := m.executeWithQuota(ctx, now, subscriberId, func(q *Quota) error {
		var unitsDebited int64 = 0
		unitsRemaining := usedUnits

		if unitType != charging.MONETARY {
			serviceUnits := debitServiceReservations(m, q, requestId, reservationId, unitType, reclaimUnusedUnits, usedUnits, subscriberId, now)
			unitsDebited += serviceUnits
			unitsRemaining = usedUnits - serviceUnits
		}

		if unitsRemaining > 0 {
			resp = debitMonetaryReservations(m, q, requestId, reservationId, unitType, reclaimUnusedUnits, unitsRemaining, subscriberId, now)
			resp.UnitsDebited += unitsDebited
		} else {
			resp.UnitsDebited = unitsDebited
			resp.UnaccountedUnits = unitsRemaining
		}

		return nil
	})

	return resp, err
}

func debitServiceReservations(m *QuotaManager, q *Quota, chargingId string, reservationId uuid.UUID, unitType charging.UnitType, reclaimUnusedUnits bool, usedUnits int64, subscriberId uuid.UUID, now time.Time) int64 {

	unitsRemaining := usedUnits
	counters := q.FindCountersByReservationAndType(reservationId, unitType)
	for _, c := range counters {
		r := c.Reservations[reservationId]

		//Apply the multiplier
		unitsRemaining = r.Multiplier.Mul(decimal.NewFromInt(unitsRemaining)).IntPart()

		var accountUnits int64
		if *r.Units <= unitsRemaining {
			accountUnits = *r.Units
			unitsRemaining -= accountUnits
			*r.Units = 0
		} else {
			accountUnits = unitsRemaining
			*r.Units = *r.Units - accountUnits
			unitsRemaining = 0
		}

		// Update the counter balance
		c.DebitBalance(decimal.NewFromInt(accountUnits))

		fmt.Printf("Posting the quota journal")
		PublishJournalEvent(m,
			q.QuotaID,
			chargingId,
			c,
			r.Reason,
			decimal.NewFromInt(accountUnits),
			unitType,
			CalculateTax(r.UnitPrice.Mul(decimal.NewFromInt(accountUnits)), *r.TaxRate),
			subscriberId,
			nil,
			now)

		if reclaimUnusedUnits || *r.Units == 0 {
			c.ReleaseReservation(reservationId)
		}

		unitsRemaining = decimal.NewFromInt(unitsRemaining).Div(*r.Multiplier).IntPart()
	}

	return usedUnits - unitsRemaining
}

func debitMonetaryReservations(m *QuotaManager, q *Quota, chargingId string, reservationId uuid.UUID, unitType charging.UnitType, reclaimUnusedUnits bool, usedUnits int64, subscriberId uuid.UUID, now time.Time) *DebitResponse {

	counters := q.FindCountersByReservationAndType(reservationId, charging.MONETARY)
	if len(counters) == 0 {
		return NewDebitResponse(0, decimal.Zero, 0, usedUnits)
	}

	firstReservation, ok := counters[0].Reservations[reservationId]
	if !ok {
		return NewDebitResponse(0, decimal.Zero, 0, usedUnits)
	}

	valueRemaining := firstReservation.CalcRemainingValue(usedUnits)
	valueDebited := decimal.Zero

	for _, c := range counters {
		if valueRemaining.LessThanOrEqual(decimal.Zero) {
			c.ReleaseReservation(reservationId)
			continue
		}

		r, ok := c.Reservations[reservationId]
		if ok {
			var accountValue decimal.Decimal
			if valueRemaining.GreaterThan(*r.Value) {
				accountValue = *r.Value
				valueRemaining = valueRemaining.Sub(*r.Value)
				*r.Value = decimal.Zero
			} else {
				accountValue = valueRemaining
				*r.Value = r.Value.Sub(valueRemaining)
				valueRemaining = decimal.Zero
			}

			c.DebitBalance(accountValue)
			valueDebited = valueDebited.Add(accountValue)

			PublishJournalEvent(m,
				q.QuotaID,
				chargingId,
				c,
				r.Reason,
				accountValue,
				unitType,
				CalculateTax(r.UnitPrice.Mul(accountValue), *r.TaxRate),
				subscriberId,
				nil,
				now)

			if reclaimUnusedUnits || r.Value.IsZero() {
				c.ReleaseReservation(reservationId)
			}
		}
	}

	var unallocatedUnits int64 = 0
	if valueRemaining.GreaterThan(decimal.Zero) {
		unallocatedUnits = valueRemaining.Div(*firstReservation.Multiplier).Div(*firstReservation.UnitPrice).IntPart()
	}

	valueUnitsDebited := usedUnits - unallocatedUnits
	return NewDebitResponse(valueUnitsDebited, valueDebited, valueUnitsDebited, unallocatedUnits)
}
