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

func (l *LoadedQuota) RemoveExpiredEntries() {

	counters := make([]Counter, 0)
	for _, c := range l.Quota.Counters {
		// Check if counter is expired and has zero balance
		if c.Expiry.After(time.Now()) && !c.Balance.IsZero() {
			reservations := make(map[uuid.UUID]Reservation)
			for k, r := range c.Reservations {
				if r.Expiry.After(time.Now()) {
					reservations[k] = r
				}
			}
			c.Reservations = reservations
			counters = append(counters, c)
		}
	}
	l.Quota.Counters = counters
}

func (l *LoadedQuota) CheckForUsageNotifications(subscriberID uuid.UUID) {
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
			// TODO: replace with proper logging / event publishing
			fmt.Printf("%s: Quota threshold reached: %d%%\n", subscriberID, *maxThreshold)
			c.Notifications.LastThresholdNotified = maxThreshold
		}
	}
}
