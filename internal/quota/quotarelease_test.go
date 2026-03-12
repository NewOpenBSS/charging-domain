package quota

import (
	"context"
	"go-ocs/internal/charging"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestQuotaManager_Release(t *testing.T) {
	subscriberID := uuid.New()
	reservationID := uuid.New()
	ctx := context.Background()

	t.Run("release reservations successfully", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:       mockRepo,
			retryLimit: 1,
		}

		rateKey := charging.RateKey{ServiceType: "data"}
		unitType := charging.UNITS
		balance := decimal.NewFromInt(1000)
		initialBalance := decimal.NewFromInt(1000)
		unitPrice := decimal.NewFromFloat(1.0)
		taxRate := decimal.NewFromFloat(0.15)
		multiplier := decimal.NewFromFloat(1.0)
		units := int64(100)
		expiry := time.Now().Add(time.Hour)

		quota := &Quota{
			QuotaID: uuid.New(),
			Counters: []Counter{
				{
					CounterID:            uuid.New(),
					CounterSelectionKeys: []charging.RateKey{rateKey},
					UnitType:             unitType,
					Balance:              &balance,
					InitialBalance:       &initialBalance,
					UnitPrice:            &unitPrice,
					TaxRate:              &taxRate,
					Expiry:               &expiry,
					Reservations: map[uuid.UUID]Reservation{
						reservationID: {
							Units:      &units,
							Multiplier: &multiplier,
							UnitPrice:  &unitPrice,
							TaxRate:    &taxRate,
							Reason:     ReasonServiceUsage,
							Expiry:     expiry,
						},
					},
				},
			},
		}
		loadedQuota := &LoadedQuota{Quota: quota}

		mockRepo.On("Load", ctx, subscriberID).Return(loadedQuota, nil)
		mockRepo.On("Save", ctx, loadedQuota).Return(nil)

		err := manager.Release(ctx, subscriberID, reservationID)

		assert.NoError(t, err)
		// Verify reservation is gone
		assert.Len(t, quota.Counters[0].Reservations, 0)
		mockRepo.AssertExpectations(t)
	})

	t.Run("release non-existent reservation", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:       mockRepo,
			retryLimit: 1,
		}

		expiry := time.Now().Add(time.Hour)
		balance := decimal.NewFromInt(1000)

		quota := &Quota{
			QuotaID: uuid.New(),
			Counters: []Counter{
				{
					CounterID:    uuid.New(),
					Expiry:       &expiry,
					Balance:      &balance,
					Reservations: make(map[uuid.UUID]Reservation),
				},
			},
		}
		loadedQuota := &LoadedQuota{Quota: quota}

		mockRepo.On("Load", ctx, subscriberID).Return(loadedQuota, nil)
		mockRepo.On("Save", ctx, loadedQuota).Return(nil)

		err := manager.Release(ctx, subscriberID, uuid.New())

		assert.NoError(t, err)
		assert.Len(t, quota.Counters[0].Reservations, 0)
		mockRepo.AssertExpectations(t)
	})
}
