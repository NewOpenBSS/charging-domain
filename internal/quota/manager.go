package quota

import (
	"context"
	"errors"
	"go-ocs/internal/charging"
	"go-ocs/internal/events"
	"go-ocs/internal/logging"
	"go-ocs/internal/store"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrConflict           = errors.New("quota conflict")
	ErrRetryLimitExceeded = errors.New("quota retry limit exceeded")
)

type QuotaManagerInterface interface {
	ReserveQuota(ctx context.Context, reservationId uuid.UUID, subscriberId uuid.UUID, reason ReasonCode, rateKey charging.RateKey, unitType charging.UnitType, requestedUnits int64, unitPrice decimal.Decimal, multiplier decimal.Decimal, validityTime time.Duration, allowOOBCharging bool) (int64, error)
	Debit(ctx context.Context, subscriberID uuid.UUID, requestId string, reservationId uuid.UUID, usedUnits int64, unitType charging.UnitType, reclaimUnusedUnits bool) (*DebitResponse, error)
	Release(ctx context.Context, subscriberId uuid.UUID, reservationId uuid.UUID) error
}

type QuotaManager struct {
	repo         Repository
	store        store.Store
	kafkaManager *events.KafkaManager
	retryLimit   int
	taxRate      decimal.Decimal
}

type OperationResult[T any] struct {
	Value T
}

type Operation[T any] func(ctx context.Context, quota *Quota, subscriberID uuid.UUID) (OperationResult[T], error)

func NewQuotaManager(store store.Store, retryLimit int, kafkaManager *events.KafkaManager) *QuotaManager {
	if retryLimit <= 0 {
		retryLimit = 3
	}

	quotaRepo := NewQuotaRepository(store)

	return &QuotaManager{
		repo:         quotaRepo,
		store:        store,
		retryLimit:   retryLimit,
		kafkaManager: kafkaManager,
		taxRate:      decimal.NewFromFloat(0.15),
	}
}

func (m *QuotaManager) executeWithQuota(
	ctx context.Context,
	subscriberID uuid.UUID,
	op func(q *Quota) error,
) error {

	for tries := 0; tries < m.retryLimit; tries++ {

		loaded, err := m.repo.Load(ctx, subscriberID)
		if err != nil {
			logging.Error("Failed to load quota", "err", err)
			return err
		}

		if loaded == nil {
			loaded, err = m.repo.Create(ctx, subscriberID)
			if err != nil {
				logging.Error("Failed to create quota", "err", err)
				return err
			}
		} else {
			// Remove expired entries
			loaded.RemoveExpiredEntries()
		}

		if err := op(loaded.Quota); err != nil {
			logging.Error("Failed to execute operation on quota", "err", err)
			return err
		}

		// Remove expired entries (again)
		loaded.RemoveExpiredEntries()

		// Check for usage notifications
		// this might result in a message being sent more than once (if the save fails)
		// but that's okay, as it will not happen that frequently
		loaded.CheckForUsageNotifications(m, subscriberID)

		err = m.repo.Save(ctx, loaded)
		if err == nil {
			return nil
		}

		if !errors.Is(err, ErrConflict) {
			logging.Error("Quota saved successfully", "subscriberID", subscriberID)
			return err
		}
	}

	logging.Error("Failed to save quota. Retry limited exceeded", "retries", m.retryLimit, "subscriberID", subscriberID)
	return ErrRetryLimitExceeded
}
