package quota

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type LoadedQuota struct {
	Quota   *Quota
	Version time.Time
}

// ExpiredCounterEntry records a counter whose balance was written off due to expiry.
// It contains a copy of the counter (with Balance zeroed to reflect the post-expiry state)
// and the balance that was present at the moment of expiry, for use as AdjustedUnits in the
// QUOTA_EXPIRY journal event.
type ExpiredCounterEntry struct {
	Counter         Counter         // copy of the counter; Balance is zero (post-expiry state)
	BalanceAtExpiry decimal.Decimal // the balance written off; used as AdjustedUnits in journal
	QuotaID         uuid.UUID       // quota this counter belongs to; needed for journal event
}

// RemoveExpiredEntries removes or cleans up counters that have passed their expiry or have a
// zero balance, and prunes expired reservations from surviving counters. now is the reference
// time for all expiry comparisons, enabling deterministic behaviour in tests.
//
// Returns a slice of ExpiredCounterEntry for every counter whose balance was written off. The
// slice is always non-nil; it is empty when nothing expired. Callers should publish a
// QUOTA_EXPIRY journal event for each entry.
func (l *LoadedQuota) RemoveExpiredEntries(now time.Time) []ExpiredCounterEntry {
	var expired []ExpiredCounterEntry
	counters := make([]Counter, 0, len(l.Quota.Counters))

	for i := range l.Quota.Counters {
		c := &l.Quota.Counters[i]

		// Step 1 — Prune expired reservations for all counters (existing behaviour).
		unexpiredReservations := make(map[uuid.UUID]Reservation)
		for k, r := range c.Reservations {
			if r.Expiry.After(now) {
				unexpiredReservations[k] = r
			}
		}
		c.Reservations = unexpiredReservations

		isExpired := c.Expiry != nil && !c.Expiry.After(now)

		if !isExpired {
			// Non-expired counter: keep unless zero balance with no outstanding loan.
			if c.Balance != nil && c.Balance.IsZero() {
				if c.Loan == nil || c.Loan.LoanBalance.IsZero() {
					// Zero balance, no loan — remove silently.
					continue
				}
			}
			counters = append(counters, *c)
			continue
		}

		// Expired counter — determine which case applies.
		hasUnexpiredReservations := len(c.Reservations) > 0

		// Case A: expired but still has unexpired reservations — keep unchanged.
		if hasUnexpiredReservations {
			counters = append(counters, *c)
			continue
		}

		hasOutstandingLoan := c.Loan != nil && c.Loan.LoanBalance.GreaterThan(decimal.Zero)

		balanceAtExpiry := decimal.Zero
		if c.Balance != nil {
			balanceAtExpiry = *c.Balance
		}

		if hasOutstandingLoan {
			// Case B: expired, loan outstanding — zero balance, keep counter, return entry.
			if !balanceAtExpiry.IsZero() {
				zero := decimal.Zero
				counterCopy := *c
				counterCopy.Balance = &zero
				expired = append(expired, ExpiredCounterEntry{
					Counter:         counterCopy,
					BalanceAtExpiry: balanceAtExpiry,
					QuotaID:         l.Quota.QuotaID,
				})
				c.Balance = &zero
			}
			counters = append(counters, *c)
			continue
		}

		if !balanceAtExpiry.IsZero() {
			// Case C: expired, no loan, balance > 0 — return entry, remove counter.
			zero := decimal.Zero
			counterCopy := *c
			counterCopy.Balance = &zero
			expired = append(expired, ExpiredCounterEntry{
				Counter:         counterCopy,
				BalanceAtExpiry: balanceAtExpiry,
				QuotaID:         l.Quota.QuotaID,
			})
			continue
		}

		// Case D: expired, no loan, zero balance — remove silently.
	}

	l.Quota.Counters = counters
	return expired
}

func (l *LoadedQuota) CheckForUsageNotifications(m *QuotaManager, subscriberID uuid.UUID) {
	for i := range l.Quota.Counters {
		c := &l.Quota.Counters[i]

		if c.Notifications == nil || c.InitialBalance == nil || c.Balance == nil || c.InitialBalance.IsZero() {
			continue
		}

		lastVal := 0
		if c.Notifications.LastThresholdNotified != nil {
			lastVal = *c.Notifications.LastThresholdNotified
		}

		currentUsed := int(
			decimal.NewFromInt(100).
				Sub(
					c.Balance.Div(*c.InitialBalance).Mul(decimal.NewFromInt(100)),
				).
				IntPart(),
		)

		if currentUsed < 0 {
			currentUsed = 0
		}
		if currentUsed > 100 {
			currentUsed = 100
		}

		var maxThreshold *int

		for _, t := range c.Notifications.Thresholds {
			if t > lastVal && t <= currentUsed {
				if maxThreshold == nil || t > *maxThreshold {
					v := t
					maxThreshold = &v
				}
			}
		}

		if maxThreshold != nil {
			msg := fmt.Sprintf("Quota threshold reached: %d%%\n", *maxThreshold)
			PublishNotificationEvent(m, subscriberID, msg)
			c.Notifications.LastThresholdNotified = maxThreshold
		}
	}
}
