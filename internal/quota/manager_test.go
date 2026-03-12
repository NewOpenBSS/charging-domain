package quota

import (
	"context"
	"errors"
	"go-ocs/internal/charging"
	"go-ocs/internal/events"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/twmb/franz-go/pkg/kgo"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Load(ctx context.Context, subscriberID uuid.UUID) (*LoadedQuota, error) {
	args := m.Called(ctx, subscriberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LoadedQuota), args.Error(1)
}

func (m *MockRepository) Create(ctx context.Context, subscriberID uuid.UUID) (*LoadedQuota, error) {
	args := m.Called(ctx, subscriberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LoadedQuota), args.Error(1)
}

func (m *MockRepository) Save(ctx context.Context, loaded *LoadedQuota) error {
	args := m.Called(ctx, loaded)
	return args.Error(0)
}

func TestQuotaManager_ExecuteWithQuota(t *testing.T) {
	subscriberID := uuid.New()
	ctx := context.Background()

	t.Run("successful execution", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:       mockRepo,
			retryLimit: 3,
		}

		loadedQuota := &LoadedQuota{
			Quota: &Quota{QuotaID: uuid.New()},
		}

		mockRepo.On("Load", ctx, subscriberID).Return(loadedQuota, nil)
		mockRepo.On("Save", ctx, loadedQuota).Return(nil)

		err := manager.executeWithQuota(ctx, subscriberID, func(q *Quota) error {
			assert.Equal(t, loadedQuota.Quota, q)
			return nil
		})

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("create quota if Load returns nil", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:       mockRepo,
			retryLimit: 3,
		}

		newQuota := &LoadedQuota{
			Quota: &Quota{QuotaID: uuid.New()},
		}

		mockRepo.On("Load", ctx, subscriberID).Return(nil, nil)
		mockRepo.On("Create", ctx, subscriberID).Return(newQuota, nil)
		mockRepo.On("Save", ctx, newQuota).Return(nil)

		err := manager.executeWithQuota(ctx, subscriberID, func(q *Quota) error {
			assert.Equal(t, newQuota.Quota, q)
			return nil
		})

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("retry on conflict and eventually succeed", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:       mockRepo,
			retryLimit: 3,
		}

		loadedQuota1 := &LoadedQuota{Quota: &Quota{QuotaID: uuid.New()}}
		loadedQuota2 := &LoadedQuota{Quota: &Quota{QuotaID: uuid.New()}}

		// First try: conflict
		mockRepo.On("Load", ctx, subscriberID).Return(loadedQuota1, nil).Once()
		mockRepo.On("Save", ctx, loadedQuota1).Return(ErrConflict).Once()

		// Second try: success
		mockRepo.On("Load", ctx, subscriberID).Return(loadedQuota2, nil).Once()
		mockRepo.On("Save", ctx, loadedQuota2).Return(nil).Once()

		err := manager.executeWithQuota(ctx, subscriberID, func(q *Quota) error {
			return nil
		})

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("retry limit exceeded", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:       mockRepo,
			retryLimit: 2,
		}

		loadedQuota := &LoadedQuota{Quota: &Quota{QuotaID: uuid.New()}}

		mockRepo.On("Load", ctx, subscriberID).Return(loadedQuota, nil).Twice()
		mockRepo.On("Save", ctx, loadedQuota).Return(ErrConflict).Twice()

		err := manager.executeWithQuota(ctx, subscriberID, func(q *Quota) error {
			return nil
		})

		assert.ErrorIs(t, err, ErrRetryLimitExceeded)
		mockRepo.AssertExpectations(t)
	})

	t.Run("fail on Load error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:       mockRepo,
			retryLimit: 3,
		}

		expectedErr := errors.New("load failed")
		mockRepo.On("Load", ctx, subscriberID).Return(nil, expectedErr)

		err := manager.executeWithQuota(ctx, subscriberID, func(q *Quota) error {
			return nil
		})

		assert.ErrorIs(t, err, expectedErr)
		mockRepo.AssertExpectations(t)
	})

	t.Run("fail on operation error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:       mockRepo,
			retryLimit: 3,
		}

		loadedQuota := &LoadedQuota{Quota: &Quota{QuotaID: uuid.New()}}
		mockRepo.On("Load", ctx, subscriberID).Return(loadedQuota, nil)

		expectedErr := errors.New("op failed")
		err := manager.executeWithQuota(ctx, subscriberID, func(q *Quota) error {
			return expectedErr
		})

		assert.ErrorIs(t, err, expectedErr)
		mockRepo.AssertExpectations(t)
	})
}

func TestQuotaManager_ReserveQuota(t *testing.T) {
	subscriberID := uuid.New()
	reservationID := uuid.New()
	ctx := context.Background()

	t.Run("reserve service units successfully", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:       mockRepo,
			retryLimit: 1,
			taxRate:    decimal.NewFromFloat(0.15),
		}

		rateKey := charging.RateKey{ServiceType: "data"}
		unitType := charging.UNITS
		requestedUnits := int64(100)
		unitPrice := decimal.NewFromFloat(1.0)
		multiplier := decimal.NewFromFloat(1.0)
		validityTime := time.Hour

		counterID := uuid.New()
		unitPriceCounter := decimal.NewFromFloat(1.0)
		taxRateCounter := decimal.NewFromFloat(0.15)
		balance := decimal.NewFromInt(1000)
		expiry := time.Now().Add(time.Hour)

		quota := &Quota{
			QuotaID: uuid.New(),
			Counters: []Counter{
				{
					CounterID:            counterID,
					CounterSelectionKeys: []charging.RateKey{rateKey},
					UnitType:             unitType,
					Balance:              &balance,
					UnitPrice:            &unitPriceCounter,
					TaxRate:              &taxRateCounter,
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
		assert.Equal(t, int64(100), *res.Units)
		mockRepo.AssertExpectations(t)
	})
}

func TestQuotaManager_Debit(t *testing.T) {
	subscriberID := uuid.New()
	reservationID := uuid.New()
	ctx := context.Background()

	t.Run("debit service units successfully", func(t *testing.T) {
		// Mock Kafka client to avoid panic in PublishJournalEvent
		// We use a listener that does nothing or we could try to mock kgo.Client if possible,
		// but kgo.Client is a struct. However, we can use a client with no brokers.
		cl, _ := kgo.NewClient(kgo.SeedBrokers("localhost:9092"))
		defer cl.Close()

		mockRepo := new(MockRepository)
		manager := &QuotaManager{
			repo:         mockRepo,
			retryLimit:   1,
			kafkaManager: &events.KafkaManager{KafkaClient: cl},
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

		resp, err := manager.Debit(ctx, subscriberID, "req-1", reservationID, 50, unitType, false)

		assert.NoError(t, err)
		assert.Equal(t, int64(50), resp.UnitsDebited)
		assert.Equal(t, int64(950), quota.Counters[0].Balance.IntPart())
		assert.Equal(t, int64(50), *quota.Counters[0].Reservations[reservationID].Units)
		mockRepo.AssertExpectations(t)
	})
}
