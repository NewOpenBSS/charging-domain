package business

import (
	"context"
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/ocserrors"
	"go-ocs/internal/charging"
	"go-ocs/internal/quota"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ReserveQuota attempts to reserve quota for a subscriber based on the
// provided rating parameters.
//
// The reservation is identified by reservationId and linked to the subscriberId.
// The reservation reason, rateKey, and unitType define the context of the usage
// being reserved.
//
// requestedUnits specifies the amount of service units requested. The reservation
// cost is calculated using unitPrice and multiplier.
//
// validityTime defines the lifetime of the reservation in seconds. If
// allowOOBCharging is true, the reservation may proceed even if the subscriber
// does not have sufficient balance.
//
// Returns:
//
//	grantedUnits    - the number of units actually granted
//	err             - an error if the reservation could not be completed.
func ReserveQuota(
	dc *engine.ChargingContext,
	reservationId uuid.UUID,
	subscriberId uuid.UUID,
	reason quota.ReasonCode,
	rateKey charging.RateKey,
	unitType charging.UnitType,
	requestedUnits int64,
	unitPrice decimal.Decimal,
	multiplier decimal.Decimal,
	validityTime time.Duration,
	allowOOBCharging bool) (int64, error) {

	grantedUnits, err := dc.AppContext.QuotaManager.ReserveQuota(context.Background(), reservationId, subscriberId, reason, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, allowOOBCharging)
	if err != nil {
		return 0, ocserrors.CreateGeneralError("Error reserving quota:" + err.Error())
	}
	return grantedUnits, nil
}

type DebitResult struct {
	DebitedUnits     int64
	MonetaryValue    decimal.Decimal
	MonetaryUnits    int64
	UnaccountedUnits int64
}

// DebitQuota debits quota for a subscriber for the given reservation and usage.
//
// It applies `usedUnits` against the appropriate counter based on `unitType` (e.g. service units
// or monetary units) and returns a DebitResult describing what was accounted and debited.
//
// If `reclaimUnusedUnits` is true, the implementation may attempt to reconcile the debit against
// any previously reserved units associated with `reservationId` and reclaim any unused portion
// of that reservation, adjusting accounted/unaccounted units accordingly.
//
// Parameters:
//   - subscriberId: The subscriber whose quota will be debited.
//   - requestId: A caller-provided identifier used for idempotency/tracing/auditing.
//   - reservationId: The reservation that this debit is associated with.
//   - usedUnits: The number of units consumed to debit.
//   - unitType: The unit type being debited (service vs monetary).
//   - reclaimUnusedUnits: Whether to reclaim any unused reserved units.
//
// Returns:
//   - *DebitResult: Details of the debit (debited, accounted, unaccounted, monetary value).
//   - error: Non-nil if the debit could not be completed.
func DebitQuota(dc *engine.ChargingContext, subscriberId uuid.UUID, requestId string, reservationId uuid.UUID, usedUnits int64, unitType charging.UnitType, reclaimUnusedUnits bool) (*DebitResult, error) {

	resp, err := dc.AppContext.QuotaManager.Debit(context.Background(), subscriberId, requestId, reservationId, usedUnits, unitType, reclaimUnusedUnits)
	if err != nil {
		return nil, err
	}

	return &DebitResult{
		DebitedUnits:     resp.UnitsDebited,
		MonetaryValue:    resp.UnitsValue,
		MonetaryUnits:    resp.UnitsDebited,
		UnaccountedUnits: resp.UnaccountedUnits,
	}, nil
}
