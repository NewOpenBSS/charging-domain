package quota

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestLoadedQuota_RemoveExpiredEntries(t *testing.T) {
	now := time.Now()
	future := now.Add(1 * time.Hour)
	past := now.Add(-1 * time.Hour)

	subscriberID := uuid.New()
	_ = subscriberID

	t.Run("should keep non-expired counters and non-expired reservations", func(t *testing.T) {
		resID1 := uuid.New()
		resID2 := uuid.New()
		balance := decimal.NewFromInt(100)

		c1 := Counter{
			Expiry:  &future,
			Balance: &balance,
			Reservations: map[uuid.UUID]Reservation{
				resID1: {Expiry: future},
				resID2: {Expiry: past},
			},
		}

		l := &LoadedQuota{
			Quota: &Quota{
				Counters: []Counter{c1},
			},
		}

		l.RemoveExpiredEntries(now)

		assert.Len(t, l.Quota.Counters, 1)
		assert.Len(t, l.Quota.Counters[0].Reservations, 1)
		assert.Contains(t, l.Quota.Counters[0].Reservations, resID1)
		assert.NotContains(t, l.Quota.Counters[0].Reservations, resID2)
	})

	t.Run("should remove expired counters", func(t *testing.T) {
		balance := decimal.NewFromInt(100)
		c1 := Counter{
			Expiry:  &past,
			Balance: &balance,
		}
		c2 := Counter{
			Expiry:  &future,
			Balance: &balance,
		}

		l := &LoadedQuota{
			Quota: &Quota{
				Counters: []Counter{c1, c2},
			},
		}

		l.RemoveExpiredEntries(now)

		assert.Len(t, l.Quota.Counters, 1)
		assert.Equal(t, &future, l.Quota.Counters[0].Expiry)
	})

	t.Run("should remove counters with zero balance", func(t *testing.T) {
		balance := decimal.Zero
		c1 := Counter{
			Expiry:  &future,
			Balance: &balance,
		}

		l := &LoadedQuota{
			Quota: &Quota{
				Counters: []Counter{c1},
			},
		}

		l.RemoveExpiredEntries(now)

		assert.Len(t, l.Quota.Counters, 0)
	})
}

func TestLoadedQuota_CheckForUsageNotifications(t *testing.T) {
	subscriberID := uuid.New()
	initialBalance := decimal.NewFromInt(100)

	t.Run("should notify when threshold is reached", func(t *testing.T) {
		thresholds := []int{50, 80, 90}
		balance := decimal.NewFromInt(40) // 60% used
		c := Counter{
			InitialBalance: &initialBalance,
			Balance:        &balance,
			Notifications: &Notifications{
				Thresholds: thresholds,
			},
		}

		l := &LoadedQuota{
			Quota: &Quota{
				Counters: []Counter{c},
			},
		}

		l.CheckForUsageNotifications(nil, subscriberID)

		assert.NotNil(t, l.Quota.Counters[0].Notifications.LastThresholdNotified)
		assert.Equal(t, 50, *l.Quota.Counters[0].Notifications.LastThresholdNotified)
	})

	t.Run("should notify highest reached threshold", func(t *testing.T) {
		thresholds := []int{50, 80, 90}
		balance := decimal.NewFromInt(15) // 85% used
		c := Counter{
			InitialBalance: &initialBalance,
			Balance:        &balance,
			Notifications: &Notifications{
				Thresholds: thresholds,
			},
		}

		l := &LoadedQuota{
			Quota: &Quota{
				Counters: []Counter{c},
			},
		}

		l.CheckForUsageNotifications(nil, subscriberID)

		assert.NotNil(t, l.Quota.Counters[0].Notifications.LastThresholdNotified)
		assert.Equal(t, 80, *l.Quota.Counters[0].Notifications.LastThresholdNotified)
	})

	t.Run("should not notify if already notified", func(t *testing.T) {
		thresholds := []int{50, 80, 90}
		lastNotified := 80
		balance := decimal.NewFromInt(15) // 85% used
		c := Counter{
			InitialBalance: &initialBalance,
			Balance:        &balance,
			Notifications: &Notifications{
				Thresholds:            thresholds,
				LastThresholdNotified: &lastNotified,
			},
		}

		l := &LoadedQuota{
			Quota: &Quota{
				Counters: []Counter{c},
			},
		}

		l.CheckForUsageNotifications(nil, subscriberID)

		assert.Equal(t, 80, *l.Quota.Counters[0].Notifications.LastThresholdNotified)
	})

	t.Run("should notify next threshold", func(t *testing.T) {
		thresholds := []int{50, 80, 90}
		lastNotified := 50
		balance := decimal.NewFromInt(15) // 85% used
		c := Counter{
			InitialBalance: &initialBalance,
			Balance:        &balance,
			Notifications: &Notifications{
				Thresholds:            thresholds,
				LastThresholdNotified: &lastNotified,
			},
		}

		l := &LoadedQuota{
			Quota: &Quota{
				Counters: []Counter{c},
			},
		}

		l.CheckForUsageNotifications(nil, subscriberID)

		assert.Equal(t, 80, *l.Quota.Counters[0].Notifications.LastThresholdNotified)
	})

	t.Run("should handle zero initial balance", func(t *testing.T) {
		zero := decimal.Zero
		balance := decimal.NewFromInt(15)
		c := Counter{
			InitialBalance: &zero,
			Balance:        &balance,
			Notifications: &Notifications{
				Thresholds: []int{50},
			},
		}

		l := &LoadedQuota{
			Quota: &Quota{
				Counters: []Counter{c},
			},
		}

		// Should not panic and should skip
		l.CheckForUsageNotifications(nil, subscriberID)

		assert.Nil(t, l.Quota.Counters[0].Notifications.LastThresholdNotified)
	})

	t.Run("should handle balance greater than initial balance", func(t *testing.T) {
		balance := decimal.NewFromInt(120) // -20% used
		c := Counter{
			InitialBalance: &initialBalance,
			Balance:        &balance,
			Notifications: &Notifications{
				Thresholds: []int{50},
			},
		}

		l := &LoadedQuota{
			Quota: &Quota{
				Counters: []Counter{c},
			},
		}

		l.CheckForUsageNotifications(nil, subscriberID)

		assert.Nil(t, l.Quota.Counters[0].Notifications.LastThresholdNotified)
	})
}
