package quota

import (
	"context"
	"go-ocs/internal/charging"
	"go-ocs/internal/events"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestQuotaManager_Debit_Comprehensive(t *testing.T) {
	subscriberID := uuid.New()
	ctx := context.Background()
	cl, _ := kgo.NewClient(kgo.SeedBrokers("localhost:9092"))
	defer cl.Close()

	// Helper to create a basic counter that won't panic in RemoveExpiredEntries
	newTestCounter := func(unitType charging.UnitType, balance decimal.Decimal) Counter {
		expiry := time.Now().Add(time.Hour)
		initialBalance := balance
		return Counter{
			CounterID:      uuid.New(),
			UnitType:       unitType,
			Balance:        &balance,
			InitialBalance: &initialBalance,
			Expiry:         &expiry,
			Reservations:   make(map[uuid.UUID]Reservation),
			UnitPrice:      new(decimal.Decimal), // default to 0
			TaxRate:        new(decimal.Decimal), // default to 0
		}
	}

	t.Run("debit service units partially and keep reservation", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:         mockRepo,
			retryLimit:   1,
			kafkaManager: &events.KafkaManager{KafkaClient: cl},
		}

		reservationID := uuid.New()
		balance := decimal.NewFromInt(1000)
		unitPrice := decimal.NewFromFloat(1.0)
		taxRate := decimal.NewFromFloat(0.15)
		multiplier := decimal.NewFromFloat(1.0)
		units := int64(100)
		expiry := time.Now().Add(time.Hour)

		counter := newTestCounter(charging.UNITS, balance)
		counter.Reservations[reservationID] = Reservation{
			Units:      &units,
			Multiplier: &multiplier,
			UnitPrice:  &unitPrice,
			TaxRate:    &taxRate,
			Reason:     ReasonServiceUsage,
			Expiry:     expiry,
		}

		quota := &Quota{
			QuotaID:  uuid.New(),
			Counters: []Counter{counter},
		}
		loadedQuota := &LoadedQuota{Quota: quota}

		mockRepo.On("Load", ctx, subscriberID).Return(loadedQuota, nil)
		mockRepo.On("Save", ctx, loadedQuota).Return(nil)

		resp, err := manager.Debit(ctx, time.Now(), subscriberID, "req-1", reservationID, 40, charging.UNITS, false)

		assert.NoError(t, err)
		assert.Equal(t, int64(40), resp.UnitsDebited)
		assert.Equal(t, int64(960), quota.Counters[0].Balance.IntPart())
		assert.Equal(t, int64(60), *quota.Counters[0].Reservations[reservationID].Units)
		assert.Contains(t, quota.Counters[0].Reservations, reservationID)
	})

	t.Run("debit service units and reclaim unused", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:         mockRepo,
			retryLimit:   1,
			kafkaManager: &events.KafkaManager{KafkaClient: cl},
		}

		reservationID := uuid.New()
		balance := decimal.NewFromInt(1000)
		unitPrice := decimal.NewFromFloat(1.0)
		taxRate := decimal.NewFromFloat(0.15)
		multiplier := decimal.NewFromFloat(1.0)
		units := int64(100)
		expiry := time.Now().Add(time.Hour)

		counter := newTestCounter(charging.UNITS, balance)
		counter.Reservations[reservationID] = Reservation{
			Units:      &units,
			Multiplier: &multiplier,
			UnitPrice:  &unitPrice,
			TaxRate:    &taxRate,
			Reason:     ReasonServiceUsage,
			Expiry:     expiry,
		}

		quota := &Quota{
			QuotaID:  uuid.New(),
			Counters: []Counter{counter},
		}
		loadedQuota := &LoadedQuota{Quota: quota}

		mockRepo.On("Load", ctx, subscriberID).Return(loadedQuota, nil)
		mockRepo.On("Save", ctx, loadedQuota).Return(nil)

		resp, err := manager.Debit(ctx, time.Now(), subscriberID, "req-2", reservationID, 40, charging.UNITS, true)

		assert.NoError(t, err)
		assert.Equal(t, int64(40), resp.UnitsDebited)
		assert.Equal(t, int64(960), quota.Counters[0].Balance.IntPart())
		assert.NotContains(t, quota.Counters[0].Reservations, reservationID)
	})

	t.Run("debit with multiplier", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:         mockRepo,
			retryLimit:   1,
			kafkaManager: &events.KafkaManager{KafkaClient: cl},
		}

		reservationID := uuid.New()
		balance := decimal.NewFromInt(1000)
		unitPrice := decimal.NewFromFloat(1.0)
		taxRate := decimal.NewFromFloat(0.15)
		multiplier := decimal.NewFromFloat(2.0) // 1 unit used = 2 units debited from balance
		units := int64(100)
		expiry := time.Now().Add(time.Hour)

		counter := newTestCounter(charging.UNITS, balance)
		counter.Reservations[reservationID] = Reservation{
			Units:      &units,
			Multiplier: &multiplier,
			UnitPrice:  &unitPrice,
			TaxRate:    &taxRate,
			Reason:     ReasonServiceUsage,
			Expiry:     expiry,
		}

		quota := &Quota{
			QuotaID:  uuid.New(),
			Counters: []Counter{counter},
		}
		loadedQuota := &LoadedQuota{Quota: quota}

		mockRepo.On("Load", ctx, subscriberID).Return(loadedQuota, nil)
		mockRepo.On("Save", ctx, loadedQuota).Return(nil)

		resp, err := manager.Debit(ctx, time.Now(), subscriberID, "req-3", reservationID, 10, charging.UNITS, false)

		assert.NoError(t, err)
		assert.Equal(t, int64(10), resp.UnitsDebited)
		assert.Equal(t, int64(980), quota.Counters[0].Balance.IntPart())
		assert.Equal(t, int64(80), *quota.Counters[0].Reservations[reservationID].Units)
	})

	t.Run("debit exceeding service reservation falling back to monetary", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:         mockRepo,
			retryLimit:   1,
			kafkaManager: &events.KafkaManager{KafkaClient: cl},
		}

		reservationID := uuid.New()
		unitPrice := decimal.NewFromFloat(2.0) // Each unit costs $2
		taxRate := decimal.NewFromFloat(0.0)
		multiplier := decimal.NewFromFloat(1.0)
		expiry := time.Now().Add(time.Hour)

		// Service counter
		serviceBalance := decimal.NewFromInt(1000)
		serviceUnits := int64(50)
		sc := newTestCounter(charging.UNITS, serviceBalance)
		sc.Reservations[reservationID] = Reservation{
			Units:      &serviceUnits,
			Multiplier: &multiplier,
			UnitPrice:  &unitPrice,
			TaxRate:    &taxRate,
			Reason:     ReasonServiceUsage,
			Expiry:     expiry,
		}

		// Monetary counter
		monetaryBalance := decimal.NewFromInt(1000)
		monetaryValue := decimal.NewFromInt(100) // $100 reserved
		mc := newTestCounter(charging.MONETARY, monetaryBalance)
		mc.Reservations[reservationID] = Reservation{
			Value:      &monetaryValue,
			Multiplier: &multiplier,
			UnitPrice:  &unitPrice,
			TaxRate:    &taxRate,
			Reason:     ReasonServiceUsage,
			Expiry:     expiry,
		}

		quota := &Quota{
			QuotaID:  uuid.New(),
			Counters: []Counter{sc, mc},
		}
		loadedQuota := &LoadedQuota{Quota: quota}

		mockRepo.On("Load", ctx, subscriberID).Return(loadedQuota, nil)
		mockRepo.On("Save", ctx, loadedQuota).Return(nil)

		resp, err := manager.Debit(ctx, time.Now(), subscriberID, "req-4", reservationID, 80, charging.UNITS, false)

		assert.NoError(t, err)
		assert.Equal(t, int64(80), resp.UnitsDebited)

		var svcCounter, monCounter *Counter
		for i := range quota.Counters {
			if quota.Counters[i].UnitType == charging.UNITS {
				svcCounter = &quota.Counters[i]
			} else if quota.Counters[i].UnitType == charging.MONETARY {
				monCounter = &quota.Counters[i]
			}
		}

		if assert.NotNil(t, svcCounter) && assert.NotNil(t, monCounter) {
			assert.Equal(t, int64(950), svcCounter.Balance.IntPart()) // 1000 - 50
			assert.Equal(t, int64(940), monCounter.Balance.IntPart()) // 1000 - 60

			_, okSvc := svcCounter.Reservations[reservationID]
			assert.False(t, okSvc, "service reservation should be released when zero units remain")

			resMon, okMon := monCounter.Reservations[reservationID]
			if assert.True(t, okMon, "monetary reservation should still exist") {
				assert.Equal(t, int64(40), resMon.Value.IntPart()) // 100 - 60
			}
		}
	})

	t.Run("monetary units only debit", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:         mockRepo,
			retryLimit:   1,
			kafkaManager: &events.KafkaManager{KafkaClient: cl},
		}

		reservationID := uuid.New()
		monetaryBalance := decimal.NewFromInt(1000)
		monetaryValue := decimal.NewFromInt(200)
		unitPrice := decimal.NewFromFloat(5.0)
		taxRate := decimal.NewFromFloat(0.0)
		multiplier := decimal.NewFromFloat(1.0)
		expiry := time.Now().Add(time.Hour)

		mc := newTestCounter(charging.MONETARY, monetaryBalance)
		mc.Reservations[reservationID] = Reservation{
			Value:      &monetaryValue,
			Multiplier: &multiplier,
			UnitPrice:  &unitPrice,
			TaxRate:    &taxRate,
			Reason:     ReasonServiceUsage,
			Expiry:     expiry,
		}

		quota := &Quota{
			QuotaID:  uuid.New(),
			Counters: []Counter{mc},
		}
		loadedQuota := &LoadedQuota{Quota: quota}

		mockRepo.On("Load", ctx, subscriberID).Return(loadedQuota, nil)
		mockRepo.On("Save", ctx, loadedQuota).Return(nil)

		// Debit 10 units. Each costs $5. Total $50.
		resp, err := manager.Debit(ctx, time.Now(), subscriberID, "req-5", reservationID, 10, charging.MONETARY, false)

		assert.NoError(t, err)
		assert.Equal(t, int64(10), resp.UnitsDebited)
		assert.Equal(t, int64(50), resp.UnitsValue.IntPart())
		assert.Equal(t, int64(950), quota.Counters[0].Balance.IntPart())
	})
}
