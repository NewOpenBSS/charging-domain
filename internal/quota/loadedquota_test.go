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

		entries := l.RemoveExpiredEntries(now)

		assert.Empty(t, entries)
		assert.Len(t, l.Quota.Counters, 1)
		assert.Len(t, l.Quota.Counters[0].Reservations, 1)
		assert.Contains(t, l.Quota.Counters[0].Reservations, resID1)
		assert.NotContains(t, l.Quota.Counters[0].Reservations, resID2)
	})

	t.Run("should remove expired counters with no reservations or loan (Case C)", func(t *testing.T) {
		balance := decimal.NewFromInt(100)
		c1 := Counter{
			Expiry:  &past,
			Balance: &balance,
		}
		balance2 := decimal.NewFromInt(50)
		c2 := Counter{
			Expiry:  &future,
			Balance: &balance2,
		}

		quotaID := uuid.New()
		l := &LoadedQuota{
			Quota: &Quota{
				QuotaID:  quotaID,
				Counters: []Counter{c1, c2},
			},
		}

		entries := l.RemoveExpiredEntries(now)

		// c1 removed; c2 kept
		assert.Len(t, l.Quota.Counters, 1)
		assert.Equal(t, &future, l.Quota.Counters[0].Expiry)

		// One entry returned for c1
		assert.Len(t, entries, 1)
		assert.Equal(t, decimal.NewFromInt(100), entries[0].BalanceAtExpiry)
		assert.True(t, entries[0].Counter.Balance.IsZero())
		assert.Equal(t, quotaID, entries[0].QuotaID)
	})

	t.Run("should remove counters with zero balance and no loan", func(t *testing.T) {
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

		entries := l.RemoveExpiredEntries(now)

		assert.Len(t, l.Quota.Counters, 0)
		assert.Empty(t, entries)
	})
}

func TestLoadedQuota_RemoveExpiredEntries_CaseA_UnexpiredReservationsBlockExpiry(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	balance := decimal.NewFromInt(100)
	resID := uuid.New()
	c := Counter{
		Expiry:  &past,
		Balance: &balance,
		Reservations: map[uuid.UUID]Reservation{
			resID: {Expiry: future}, // unexpired reservation — blocks removal
		},
	}

	l := &LoadedQuota{
		Quota: &Quota{
			Counters: []Counter{c},
		},
	}

	entries := l.RemoveExpiredEntries(now)

	// Counter kept, no entry returned
	assert.Len(t, l.Quota.Counters, 1)
	assert.Empty(t, entries)
}

func TestLoadedQuota_RemoveExpiredEntries_CaseB_OutstandingLoanZerosBalance(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)

	balance := decimal.NewFromInt(75)
	quotaID := uuid.New()
	c := Counter{
		CounterID: uuid.New(),
		Expiry:    &past,
		Balance:   &balance,
		Loan:      &Loan{LoanBalance: decimal.NewFromInt(50)},
	}

	l := &LoadedQuota{
		Quota: &Quota{
			QuotaID:  quotaID,
			Counters: []Counter{c},
		},
	}

	entries := l.RemoveExpiredEntries(now)

	// Counter kept (loan outstanding), balance zeroed
	assert.Len(t, l.Quota.Counters, 1)
	assert.True(t, l.Quota.Counters[0].Balance.IsZero(), "counter balance should be zeroed")

	// One entry returned
	assert.Len(t, entries, 1)
	assert.Equal(t, decimal.NewFromInt(75), entries[0].BalanceAtExpiry)
	assert.True(t, entries[0].Counter.Balance.IsZero())
	assert.Equal(t, quotaID, entries[0].QuotaID)
}

func TestLoadedQuota_RemoveExpiredEntries_CaseC_NoLoanBalancePositive(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)

	balance := decimal.NewFromInt(200)
	quotaID := uuid.New()
	counterID := uuid.New()
	c := Counter{
		CounterID: counterID,
		Expiry:    &past,
		Balance:   &balance,
	}

	l := &LoadedQuota{
		Quota: &Quota{
			QuotaID:  quotaID,
			Counters: []Counter{c},
		},
	}

	entries := l.RemoveExpiredEntries(now)

	// Counter removed
	assert.Len(t, l.Quota.Counters, 0)

	// Entry returned with correct values
	assert.Len(t, entries, 1)
	assert.Equal(t, decimal.NewFromInt(200), entries[0].BalanceAtExpiry)
	assert.True(t, entries[0].Counter.Balance.IsZero())
	assert.Equal(t, counterID, entries[0].Counter.CounterID)
	assert.Equal(t, quotaID, entries[0].QuotaID)
}

func TestLoadedQuota_RemoveExpiredEntries_CaseD_NoLoanZeroBalance(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)

	balance := decimal.Zero
	c := Counter{
		Expiry:  &past,
		Balance: &balance,
	}

	l := &LoadedQuota{
		Quota: &Quota{
			Counters: []Counter{c},
		},
	}

	entries := l.RemoveExpiredEntries(now)

	// Counter removed silently, no entry
	assert.Len(t, l.Quota.Counters, 0)
	assert.Empty(t, entries)
}

func TestLoadedQuota_RemoveExpiredEntries_ZeroBalanceNonExpiredWithLoanRetained(t *testing.T) {
	now := time.Now()
	future := now.Add(1 * time.Hour)

	balance := decimal.Zero
	c := Counter{
		Expiry:  &future,
		Balance: &balance,
		Loan:    &Loan{LoanBalance: decimal.NewFromInt(30)},
	}

	l := &LoadedQuota{
		Quota: &Quota{
			Counters: []Counter{c},
		},
	}

	entries := l.RemoveExpiredEntries(now)

	// Counter retained (outstanding loan), no entry
	assert.Len(t, l.Quota.Counters, 1)
	assert.Empty(t, entries)
}

func TestLoadedQuota_RemoveExpiredEntries_EntryCopyDoesNotSharePointer(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)

	balance := decimal.NewFromInt(50)
	c := Counter{
		Expiry:  &past,
		Balance: &balance,
	}

	l := &LoadedQuota{
		Quota: &Quota{
			QuotaID:  uuid.New(),
			Counters: []Counter{c},
		},
	}

	entries := l.RemoveExpiredEntries(now)

	assert.Len(t, entries, 1)
	// The copy's Balance pointer must not point to the same location as the original
	assert.True(t, entries[0].Counter.Balance.IsZero())
	// BalanceAtExpiry is the original value
	assert.Equal(t, decimal.NewFromInt(50), entries[0].BalanceAtExpiry)
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
