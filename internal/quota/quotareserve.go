package quota

import (
	"context"
	"go-ocs/internal/charging"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ReserveRequest struct {
	ReservationId  uuid.UUID
	SubscriberId   uuid.UUID
	Reason         ReasonCode
	RateKey        charging.RateKey
	UnitType       charging.UnitType
	RequestedUnits int64
	UnitPrice      decimal.Decimal
	Multiplier     decimal.Decimal
	ValidityTime   time.Duration
}

func NewReserveRequest(reservationId uuid.UUID, reason ReasonCode, rateKey charging.RateKey, unitType charging.UnitType, multiplier decimal.Decimal, requestedUnits int64, unitPrice decimal.Decimal, validityTime time.Duration) ReserveRequest {
	return ReserveRequest{
		ReservationId:  reservationId,
		Reason:         reason,
		RateKey:        rateKey,
		UnitType:       unitType,
		RequestedUnits: requestedUnits,
		UnitPrice:      unitPrice,
		Multiplier:     multiplier,
		ValidityTime:   validityTime,
	}
}

type ReserveResponse struct {

	/**
	 * Represents the unique identifier for a reservation.
	 * This identifier is used to distinguish and track a specific reservation
	 * within the reservation system.
	 */
	ReservationId uuid.UUID

	/**
	 * Represents the number of units granted in a reservation.
	 * This field indicates the quantity of the reserved units
	 * that have been successfully allocated or granted for use.
	 */
	UnitsGranted int64

	/**
	 * Represents the number of seconds of the reservation.
	 * This field specifies the exact point in time when the reservation
	 * is scheduled or has taken place. It is used to track and manage
	 * the temporal aspects of the reservation within the system.
	 */
	ValidityTime time.Duration
}

func NewReserveResponse(reservationId uuid.UUID, unitsGranted int64, validityTime time.Duration) *ReserveResponse {
	return &ReserveResponse{
		ReservationId: reservationId,
		UnitsGranted:  unitsGranted,
		ValidityTime:  validityTime,
	}
}

// ReserveQuota reserves quota units for a subscriber. now is the reference time used to
// set reservation expiry timestamps, ensuring deterministic behaviour across the call chain.
func (m *QuotaManager) ReserveQuota(
	ctx context.Context,
	now time.Time,
	reservationId uuid.UUID,
	subscriberId uuid.UUID,
	reason ReasonCode,
	rateKey charging.RateKey,
	unitType charging.UnitType,
	requestedUnits int64,
	unitPrice decimal.Decimal,
	multiplier decimal.Decimal,
	validityTime time.Duration,
	allowOOBCharging bool) (int64, error) {

	var grantedUnits int64
	err := m.executeWithQuota(ctx, now, subscriberId, func(q *Quota) error {
		if unitType == charging.MONETARY {
			grantedUnits = m.reserveMonetaryUnits(q, reservationId, subscriberId, reason, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, now)
			return nil
		}

		grantedUnits = m.reserveServiceUnits(q, reservationId, subscriberId, reason, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, now)
		if allowOOBCharging && grantedUnits < requestedUnits {
			grantedUnits += m.reserveMonetaryUnits(q, reservationId, subscriberId, reason, rateKey, unitType, requestedUnits-grantedUnits, unitPrice, multiplier, validityTime, now)
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return grantedUnits, nil
}

func (m *QuotaManager) reserveServiceUnits(
	q *Quota,
	reservationId uuid.UUID,
	subscriberId uuid.UUID,
	reason ReasonCode,
	rateKey charging.RateKey,
	unitType charging.UnitType,
	requestedUnits int64,
	unitPrice decimal.Decimal,
	multiplier decimal.Decimal,
	validityTime time.Duration,
	now time.Time) int64 {

	var grantedUnits int64
	for _, c := range q.FindCounters(rateKey, unitType, reason) {
		if grantedUnits >= requestedUnits {
			break
		}

		reserved := c.ReserveServiceUnits(
			reservationId,
			requestedUnits-grantedUnits,
			unitPrice,
			multiplier,
			m.taxRate,
			reason,
			validityTime,
			now,
		)
		grantedUnits += reserved
	}

	return grantedUnits
}

func (m *QuotaManager) reserveMonetaryUnits(
	q *Quota,
	reservationId uuid.UUID,
	subscriberId uuid.UUID,
	reason ReasonCode,
	rateKey charging.RateKey,
	unitType charging.UnitType,
	requestedUnits int64,
	unitPrice decimal.Decimal,
	multiplier decimal.Decimal,
	validityTime time.Duration,
	now time.Time) int64 {

	taxRateAddition := decimal.NewFromInt(1).Add(m.taxRate)

	//Find all the monetary counters
	counters := q.FindCounters(rateKey, charging.MONETARY, reason)

	//Calculate the available balance
	totalAvailable := decimal.NewFromInt(0)
	for _, c := range counters {
		totalAvailable = totalAvailable.Add(c.AvailableValue())
	}

	//Calculate how many units can be allocated based on the total available monetary balance and global tax rate
	unitPriceWithTax := unitPrice.Mul(taxRateAddition)
	grantedUnits := totalAvailable.Div(unitPriceWithTax).IntPart()
	if grantedUnits > requestedUnits {
		grantedUnits = requestedUnits
	}

	if grantedUnits > 0 {
		totalValueWithTax := unitPrice.Mul(decimal.NewFromInt(grantedUnits)).Mul(taxRateAddition)

		//Reserve the value
		valueReserved := decimal.Zero
		for _, c := range counters {
			if valueReserved.GreaterThanOrEqual(totalValueWithTax) {
				break
			}

			valueReserved = valueReserved.Add(c.ReserveValue(reservationId, totalValueWithTax.Sub(valueReserved), unitPrice, multiplier, m.taxRate, reason, validityTime, now))
		}
	}

	return grantedUnits
}
