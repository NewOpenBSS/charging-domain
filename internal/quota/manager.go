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
	// ReserveQuota reserves quota for a subscriber. now is the reference time for reservation expiry.
	ReserveQuota(ctx context.Context, now time.Time, reservationId uuid.UUID, subscriberId uuid.UUID, reason ReasonCode, rateKey charging.RateKey, unitType charging.UnitType, requestedUnits int64, unitPrice decimal.Decimal, multiplier decimal.Decimal, validityTime time.Duration, allowOOBCharging bool) (int64, error)
	// Debit applies used units against the subscriber's reservation. now is the reference time for journal timestamps.
	Debit(ctx context.Context, now time.Time, subscriberID uuid.UUID, requestId string, reservationId uuid.UUID, usedUnits int64, unitType charging.UnitType, reclaimUnusedUnits bool) (*DebitResponse, error)
	// Release releases an active quota reservation for a subscriber.
	Release(ctx context.Context, subscriberId uuid.UUID, reservationId uuid.UUID) error
	// GetBalance returns the balances for all non-expired counters matching query for the
	// given subscriber. now is the reference time for expiry comparisons. Returns an empty
	// slice (not an error) if the subscriber has no quota or no counters match the query.
	GetBalance(ctx context.Context, now time.Time, subscriberID uuid.UUID, query BalanceQuery) ([]*CounterBalance, error)
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
	now time.Time,
	subscriberID uuid.UUID,
	op func(q *Quota) error,
) error {

	for tries := 0; tries < m.retryLimit; tries++ {

		loaded, err := m.repo.Load(ctx, subscriberID)
		if err != nil {
			logging.Error("Failed to load quota", "err", err)
			return err
		}

		var expiredBefore []ExpiredCounterEntry
		if loaded == nil {
			loaded, err = m.repo.Create(ctx, subscriberID)
			if err != nil {
				logging.Error("Failed to create quota", "err", err)
				return err
			}
		} else {
			// Remove expired entries
			expiredBefore = loaded.RemoveExpiredEntries(now)
		}

		if err := op(loaded.Quota); err != nil {
			logging.Error("Failed to execute operation on quota", "err", err)
			return err
		}

		// Remove expired entries (again)
		expiredAfter := loaded.RemoveExpiredEntries(now)

		// Check for usage notifications
		// this might result in a message being sent more than once (if the save fails)
		// but that's okay, as it will not happen that frequently
		loaded.CheckForUsageNotifications(m, subscriberID)

		err = m.repo.Save(ctx, loaded, now)
		if err == nil {
			publishExpiryJournals(m, subscriberID, expiredBefore, now)
			publishExpiryJournals(m, subscriberID, expiredAfter, now)
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

// ProcessExpiredQuota processes expired counters for a single subscriber by loading
// their quota, triggering expiry logic, and saving back. Journal events are published
// for all expired counters. This is equivalent to the JIT expiry that runs during
// active charging, but invoked explicitly for dormant subscribers by the housekeeping job.
//
// A nil quota row is silently skipped — if the subscriber has no quota record there is
// nothing to expire. This is not an error.
func (m *QuotaManager) ProcessExpiredQuota(ctx context.Context, now time.Time, subscriberID uuid.UUID) error {
	return m.executeWithQuota(ctx, now, subscriberID, func(_ *Quota) error {
		return nil
	})
}

// GetBalance returns the balances for all non-expired counters matching query for the
// given subscriber. now is the reference time for expiry comparisons. Returns an empty
// slice (not an error) if the subscriber has no quota record or no counters match the query.
func (m *QuotaManager) GetBalance(ctx context.Context, now time.Time, subscriberID uuid.UUID, query BalanceQuery) ([]*CounterBalance, error) {
	loaded, err := m.repo.Load(ctx, subscriberID)
	if err != nil {
		return nil, err
	}
	if loaded == nil {
		return []*CounterBalance{}, nil
	}

	result := make([]*CounterBalance, 0, len(loaded.Quota.Counters))
	for i := range loaded.Quota.Counters {
		c := &loaded.Quota.Counters[i]

		// Exclude expired or zero-balance counters — mirrors RemoveExpiredEntries logic.
		if c.Expiry != nil && !c.Expiry.After(now) {
			continue
		}
		if c.Balance != nil && c.Balance.IsZero() {
			continue
		}

		if !query.matches(c) {
			continue
		}

		result = append(result, counterToBalance(c))
	}

	return result, nil
}

// publishExpiryJournals publishes a QUOTA_EXPIRY QuotaJournalEvent for each expired counter
// entry. Journal events are only published after a successful Save, so this must only be called
// on the successful attempt inside executeWithQuota.
func publishExpiryJournals(m *QuotaManager, subscriberID uuid.UUID, entries []ExpiredCounterEntry, now time.Time) {
	for _, entry := range entries {
		taxRate := decimal.NewFromInt(1)
		if entry.Counter.TaxRate != nil {
			taxRate = *entry.Counter.TaxRate
		}
		taxCalc := CalculateTax(entry.BalanceAtExpiry, taxRate)

		PublishJournalEvent(
			m,
			entry.QuotaID,
			"", // no natural transactionId for expiry events
			&entry.Counter,
			ReasonQuotaExpiry,
			entry.BalanceAtExpiry,
			entry.Counter.UnitType,
			taxCalc,
			subscriberID,
			nil, // no CounterMetaData for expiry events
			now,
		)
	}
}
