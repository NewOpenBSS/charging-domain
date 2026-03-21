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

// RemoveExpiredEntries removes counters that have passed their expiry or have a zero balance,
// and prunes expired reservations from surviving counters. now is the reference time for all
// expiry comparisons, enabling deterministic behaviour in tests.
func (l *LoadedQuota) RemoveExpiredEntries(now time.Time) {

	counters := make([]Counter, 0)
	for _, c := range l.Quota.Counters {
		// A nil Expiry means the counter never expires. Only retain counters whose
		// expiry has not passed and whose balance is non-zero.
		if (c.Expiry == nil || c.Expiry.After(now)) && !c.Balance.IsZero() {
			reservations := make(map[uuid.UUID]Reservation)
			for k, r := range c.Reservations {
				if r.Expiry.After(now) {
					reservations[k] = r
				}
			}
			c.Reservations = reservations
			counters = append(counters, c)
		}
	}
	l.Quota.Counters = counters
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
