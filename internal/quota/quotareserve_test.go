package quota

import (
	"context"
	"errors"
	"go-ocs/internal/charging"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestQuotaManager_ReserveQuota_Comprehensive(t *testing.T) {
	subscriberID := uuid.New()
	reservationID := uuid.New()
	ctx := context.Background()

	t.Run("reserve monetary units successfully", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:       mockRepo,
			retryLimit: 1,
			taxRate:    decimal.NewFromFloat(0.15),
		}

		rateKey := charging.RateKey{ServiceType: "data"}
		unitType := charging.MONETARY
		requestedUnits := int64(100)
		unitPrice := decimal.NewFromFloat(1.0)
		multiplier := decimal.NewFromFloat(1.0)
		validityTime := time.Hour

		balance := decimal.NewFromInt(200) // Should be enough for 100 * 1.0 * (1 + 0.15) = 115
		expiry := time.Now().Add(time.Hour)

		quota := &Quota{
			QuotaID: uuid.New(),
			Counters: []Counter{
				{
					CounterID:            uuid.New(),
					CounterSelectionKeys: []charging.RateKey{rateKey},
					UnitType:             charging.MONETARY,
					Balance:              &balance,
					Priority:             10,
					Expiry:               &expiry,
					Reservations:         make(map[uuid.UUID]Reservation),
				},
			},
		}
		loadedQuota := &LoadedQuota{Quota: quota}

		mockRepo.On("Load", ctx, subscriberID).Return(loadedQuota, nil)
		mockRepo.On("Save", ctx, loadedQuota).Return(nil)

		granted, err := manager.ReserveQuota(ctx, reservationID, subscriberID, ReasonServiceUsage, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, false)

		assert.NoError(t, err)
		assert.Equal(t, int64(100), granted)
		assert.Len(t, quota.Counters[0].Reservations, 1)
		res := quota.Counters[0].Reservations[reservationID]
		assert.Equal(t, decimal.NewFromFloat(115.0).String(), res.Value.String())
	})

	t.Run("reserve with OOB charging", func(t *testing.T) {
		// This test expects service units to be exhausted and fallback to monetary
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:       mockRepo,
			retryLimit: 1,
			taxRate:    decimal.NewFromFloat(0.15),
		}

		rateKey := charging.RateKey{ServiceType: "data"}
		unitType := charging.UNITS
		requestedUnits := int64(150)
		unitPrice := decimal.NewFromFloat(1.0)
		multiplier := decimal.NewFromFloat(1.0)
		validityTime := time.Hour

		serviceBalance := decimal.NewFromInt(100)
		monetaryBalance := decimal.NewFromInt(100) // Need 50 * 1.0 * 1.15 = 57.5
		expiry := time.Now().Add(time.Hour)

		quota := &Quota{
			QuotaID: uuid.New(),
			Counters: []Counter{
				{
					CounterID:            uuid.New(),
					CounterSelectionKeys: []charging.RateKey{rateKey},
					UnitType:             charging.UNITS,
					Balance:              &serviceBalance,
					Priority:             20,
					Expiry:               &expiry,
					Reservations:         make(map[uuid.UUID]Reservation),
				},
				{
					CounterID:            uuid.New(),
					CounterSelectionKeys: []charging.RateKey{rateKey},
					UnitType:             charging.MONETARY,
					Balance:              &monetaryBalance,
					Priority:             10,
					Expiry:               &expiry,
					Reservations:         make(map[uuid.UUID]Reservation),
				},
			},
		}
		loadedQuota := &LoadedQuota{Quota: quota}

		mockRepo.On("Load", ctx, subscriberID).Return(loadedQuota, nil)
		mockRepo.On("Save", ctx, loadedQuota).Return(nil)

		granted, err := manager.ReserveQuota(ctx, reservationID, subscriberID, ReasonServiceUsage, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, true)

		assert.NoError(t, err)
		assert.Equal(t, int64(150), granted)

		// Check service counter
		assert.Len(t, quota.Counters[0].Reservations, 1)
		assert.Equal(t, int64(100), *quota.Counters[0].Reservations[reservationID].Units)

		// Check monetary counter
		assert.Len(t, quota.Counters[1].Reservations, 1)
		assert.Equal(t, decimal.NewFromFloat(57.5).String(), quota.Counters[1].Reservations[reservationID].Value.String())
	})

	t.Run("insufficient balance", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:       mockRepo,
			retryLimit: 1,
			taxRate:    decimal.NewFromFloat(0.15),
		}

		rateKey := charging.RateKey{ServiceType: "data"}
		unitType := charging.UNITS
		requestedUnits := int64(200)
		unitPrice := decimal.NewFromFloat(1.0)
		multiplier := decimal.NewFromFloat(1.0)
		validityTime := time.Hour

		serviceBalance := decimal.NewFromInt(50)
		monetaryBalance := decimal.NewFromFloat(57.5) // 50 * 1.0 * 1.15 = 57.5
		expiry := time.Now().Add(time.Hour)

		quota := &Quota{
			QuotaID: uuid.New(),
			Counters: []Counter{
				{
					CounterID:            uuid.New(),
					CounterSelectionKeys: []charging.RateKey{rateKey},
					UnitType:             charging.UNITS,
					Balance:              &serviceBalance,
					Priority:             20,
					Expiry:               &expiry,
					Reservations:         make(map[uuid.UUID]Reservation),
				},
				{
					CounterID:            uuid.New(),
					CounterSelectionKeys: []charging.RateKey{rateKey},
					UnitType:             charging.MONETARY,
					Balance:              &monetaryBalance,
					Priority:             10,
					Expiry:               &expiry,
					Reservations:         make(map[uuid.UUID]Reservation),
				},
			},
		}
		loadedQuota := &LoadedQuota{Quota: quota}

		mockRepo.On("Load", ctx, subscriberID).Return(loadedQuota, nil)
		mockRepo.On("Save", ctx, loadedQuota).Return(nil)

		granted, err := manager.ReserveQuota(ctx, reservationID, subscriberID, ReasonServiceUsage, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, true)

		assert.NoError(t, err)
		assert.Equal(t, int64(100), granted) // 50 (service) + 50 (monetary)
	})

	t.Run("repository failures", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:       mockRepo,
			retryLimit: 1,
		}

		rateKey := charging.RateKey{ServiceType: "data"}
		unitType := charging.UNITS
		requestedUnits := int64(100)
		unitPrice := decimal.NewFromFloat(1.0)
		multiplier := decimal.NewFromFloat(1.0)
		validityTime := time.Hour

		t.Run("load failure", func(t *testing.T) {
			loadErr := errors.New("load failed")
			mockRepo.On("Load", ctx, subscriberID).Return(nil, loadErr).Once()

			granted, err := manager.ReserveQuota(ctx, reservationID, subscriberID, ReasonServiceUsage, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, false)

			assert.ErrorIs(t, err, loadErr)
			assert.Equal(t, int64(0), granted)
		})

		t.Run("save failure", func(t *testing.T) {
			saveErr := errors.New("save failed")
			quota := &Quota{QuotaID: uuid.New()}
			loadedQuota := &LoadedQuota{Quota: quota}

			mockRepo.On("Load", ctx, subscriberID).Return(loadedQuota, nil).Once()
			mockRepo.On("Save", ctx, loadedQuota).Return(saveErr).Once()

			granted, err := manager.ReserveQuota(ctx, reservationID, subscriberID, ReasonServiceUsage, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, false)

			assert.ErrorIs(t, err, saveErr)
			assert.Equal(t, int64(0), granted)
		})
	})
}
