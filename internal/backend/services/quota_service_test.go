package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go-ocs/internal/backend/graphql/model"
	"go-ocs/internal/charging"
	"go-ocs/internal/quota"
)

// ---------------------------------------------------------------------------
// Mock
// ---------------------------------------------------------------------------

type mockQuotaManager struct {
	mock.Mock
}

func (m *mockQuotaManager) ReserveQuota(ctx context.Context, now time.Time, reservationId uuid.UUID, subscriberId uuid.UUID, reason quota.ReasonCode, rateKey charging.RateKey, unitType charging.UnitType, requestedUnits int64, unitPrice decimal.Decimal, multiplier decimal.Decimal, validityTime time.Duration, allowOOBCharging bool) (int64, error) {
	args := m.Called(ctx, now, reservationId, subscriberId, reason, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, allowOOBCharging)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockQuotaManager) Debit(ctx context.Context, now time.Time, subscriberID uuid.UUID, requestId string, reservationId uuid.UUID, usedUnits int64, unitType charging.UnitType, reclaimUnusedUnits bool) (*quota.DebitResponse, error) {
	args := m.Called(ctx, now, subscriberID, requestId, reservationId, usedUnits, unitType, reclaimUnusedUnits)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*quota.DebitResponse), args.Error(1)
}

func (m *mockQuotaManager) Release(ctx context.Context, subscriberId uuid.UUID, reservationId uuid.UUID) error {
	args := m.Called(ctx, subscriberId, reservationId)
	return args.Error(0)
}

func (m *mockQuotaManager) GetBalance(ctx context.Context, now time.Time, subscriberID uuid.UUID, query quota.BalanceQuery) ([]*quota.CounterBalance, error) {
	args := m.Called(ctx, now, subscriberID, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*quota.CounterBalance), args.Error(1)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func dec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

func counterBalance(ut charging.UnitType, total, avail string) *quota.CounterBalance {
	return &quota.CounterBalance{
		CounterID:        uuid.New(),
		ProductID:        uuid.New(),
		UnitType:         ut,
		TotalBalance:     dec(total),
		AvailableBalance: dec(avail),
	}
}

// ---------------------------------------------------------------------------
// GetBalance tests
// ---------------------------------------------------------------------------

func TestQuotaService_GetBalance(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	subID := uuid.New()

	t.Run("returns nil when no matching counters", func(t *testing.T) {
		mgr := new(mockQuotaManager)
		svc := NewQuotaService(mgr)

		ut := model.UnitTypeMonetary
		req := model.QuotaBalanceRequestInput{
			SubscriberID: subID.String(),
			UnitType:     &ut,
			BalanceType:  model.BalanceTypeAvailableBalance,
		}

		mgr.On("GetBalance", ctx, mock.AnythingOfType("time.Time"), subID, quota.BalanceQuery{
			UnitType: func() *charging.UnitType { u := charging.MONETARY; return &u }(),
		}).Return([]*quota.CounterBalance{}, nil)

		result, err := svc.GetBalance(ctx, now, req)

		require.NoError(t, err)
		assert.Nil(t, result)
		mgr.AssertExpectations(t)
	})

	t.Run("aggregates multiple counters of the same unit type", func(t *testing.T) {
		mgr := new(mockQuotaManager)
		svc := NewQuotaService(mgr)

		ut := model.UnitTypeMonetary
		req := model.QuotaBalanceRequestInput{
			SubscriberID: subID.String(),
			UnitType:     &ut,
			BalanceType:  model.BalanceTypeAvailableBalance,
		}

		counters := []*quota.CounterBalance{
			counterBalance(charging.MONETARY, "100", "80"),
			counterBalance(charging.MONETARY, "50", "50"),
		}

		mgr.On("GetBalance", ctx, mock.AnythingOfType("time.Time"), subID, mock.AnythingOfType("quota.BalanceQuery")).
			Return(counters, nil)

		result, err := svc.GetBalance(ctx, now, req)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, model.UnitTypeMonetary, result.UnitType)
		assert.Equal(t, "150", result.TotalValue)
		assert.Equal(t, "130", result.AvailableBalance)
		mgr.AssertExpectations(t)
	})

	t.Run("returns error when unitType is missing", func(t *testing.T) {
		svc := NewQuotaService(new(mockQuotaManager))
		req := model.QuotaBalanceRequestInput{
			SubscriberID: subID.String(),
			BalanceType:  model.BalanceTypeAvailableBalance,
		}

		_, err := svc.GetBalance(ctx, now, req)
		assert.Error(t, err)
	})

	t.Run("returns error when subscriberId is invalid", func(t *testing.T) {
		svc := NewQuotaService(new(mockQuotaManager))
		ut := model.UnitTypeMonetary
		req := model.QuotaBalanceRequestInput{
			SubscriberID: "not-a-uuid",
			UnitType:     &ut,
			BalanceType:  model.BalanceTypeAvailableBalance,
		}

		_, err := svc.GetBalance(ctx, now, req)
		assert.Error(t, err)
	})

	t.Run("propagates manager error", func(t *testing.T) {
		mgr := new(mockQuotaManager)
		svc := NewQuotaService(mgr)

		ut := model.UnitTypeMonetary
		req := model.QuotaBalanceRequestInput{
			SubscriberID: subID.String(),
			UnitType:     &ut,
			BalanceType:  model.BalanceTypeAvailableBalance,
		}

		mgr.On("GetBalance", ctx, mock.AnythingOfType("time.Time"), subID, mock.AnythingOfType("quota.BalanceQuery")).
			Return(nil, errors.New("db error"))

		_, err := svc.GetBalance(ctx, now, req)
		assert.Error(t, err)
		mgr.AssertExpectations(t)
	})
}

// ---------------------------------------------------------------------------
// GetBalances tests
// ---------------------------------------------------------------------------

func TestQuotaService_GetBalances(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	subID := uuid.New()

	t.Run("groups and aggregates by unit type", func(t *testing.T) {
		mgr := new(mockQuotaManager)
		svc := NewQuotaService(mgr)

		req := model.QuotaBalanceRequestInput{
			SubscriberID: subID.String(),
			BalanceType:  model.BalanceTypeAvailableBalance,
		}

		counters := []*quota.CounterBalance{
			counterBalance(charging.MONETARY, "100", "80"),
			counterBalance(charging.UNITS, "500", "400"),
			counterBalance(charging.MONETARY, "50", "50"),
		}

		mgr.On("GetBalance", ctx, mock.AnythingOfType("time.Time"), subID, mock.AnythingOfType("quota.BalanceQuery")).
			Return(counters, nil)

		result, err := svc.GetBalances(ctx, now, req)

		require.NoError(t, err)
		require.Len(t, result, 2)
		// First group: MONETARY (order of first appearance)
		assert.Equal(t, model.UnitTypeMonetary, result[0].UnitType)
		assert.Equal(t, "150", result[0].TotalValue)
		assert.Equal(t, "130", result[0].AvailableBalance)
		// Second group: UNITS
		assert.Equal(t, model.UnitTypeUnits, result[1].UnitType)
		assert.Equal(t, "500", result[1].TotalValue)
		assert.Equal(t, "400", result[1].AvailableBalance)
		mgr.AssertExpectations(t)
	})

	t.Run("returns empty slice when no counters", func(t *testing.T) {
		mgr := new(mockQuotaManager)
		svc := NewQuotaService(mgr)

		req := model.QuotaBalanceRequestInput{
			SubscriberID: subID.String(),
			BalanceType:  model.BalanceTypeAvailableBalance,
		}

		mgr.On("GetBalance", ctx, mock.AnythingOfType("time.Time"), subID, mock.AnythingOfType("quota.BalanceQuery")).
			Return([]*quota.CounterBalance{}, nil)

		result, err := svc.GetBalances(ctx, now, req)

		require.NoError(t, err)
		assert.Empty(t, result)
		mgr.AssertExpectations(t)
	})

	t.Run("TRANSFERABLE_BALANCE sets Transferable filter", func(t *testing.T) {
		mgr := new(mockQuotaManager)
		svc := NewQuotaService(mgr)

		req := model.QuotaBalanceRequestInput{
			SubscriberID: subID.String(),
			BalanceType:  model.BalanceTypeTransferableBalance,
		}

		transferable := true
		expectedQuery := quota.BalanceQuery{Transferable: &transferable}
		mgr.On("GetBalance", ctx, mock.AnythingOfType("time.Time"), subID, expectedQuery).
			Return([]*quota.CounterBalance{}, nil)

		_, err := svc.GetBalances(ctx, now, req)

		require.NoError(t, err)
		mgr.AssertExpectations(t)
	})
}

// ---------------------------------------------------------------------------
// CancelReservations tests
// ---------------------------------------------------------------------------

func TestQuotaService_CancelReservations(t *testing.T) {
	ctx := context.Background()
	resID := uuid.New()
	subID := uuid.New()

	t.Run("delegates to Release and returns success", func(t *testing.T) {
		mgr := new(mockQuotaManager)
		svc := NewQuotaService(mgr)

		mgr.On("Release", ctx, subID, resID).Return(nil)

		result, err := svc.CancelReservations(ctx, resID.String(), subID.String())

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success)
		mgr.AssertExpectations(t)
	})

	t.Run("returns error on invalid reservationId", func(t *testing.T) {
		svc := NewQuotaService(new(mockQuotaManager))
		_, err := svc.CancelReservations(ctx, "bad", subID.String())
		assert.Error(t, err)
	})

	t.Run("propagates manager error", func(t *testing.T) {
		mgr := new(mockQuotaManager)
		svc := NewQuotaService(mgr)

		mgr.On("Release", ctx, subID, resID).Return(errors.New("quota locked"))

		_, err := svc.CancelReservations(ctx, resID.String(), subID.String())
		assert.Error(t, err)
		mgr.AssertExpectations(t)
	})
}

// ---------------------------------------------------------------------------
// ReserveQuota tests
// ---------------------------------------------------------------------------

func TestQuotaService_ReserveQuota(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	resID := uuid.New()
	subID := uuid.New()

	rateKey := model.QuotaRateKeyInput{
		ServiceType:      "VOICE",
		SourceType:       "HOME",
		ServiceDirection: "MO",
		ServiceCategory:  "LOCAL",
	}

	t.Run("delegates to ReserveQuota and returns grantedUnits", func(t *testing.T) {
		mgr := new(mockQuotaManager)
		svc := NewQuotaService(mgr)

		mgr.On("ReserveQuota",
			ctx, mock.AnythingOfType("time.Time"),
			resID, subID,
			quota.ReasonCode("SERVICE_USAGE"),
			mock.AnythingOfType("charging.RateKey"),
			charging.MONETARY,
			int64(100),
			dec("0.50"),
			decimal.NewFromInt(1),
			60*time.Second,
			false,
		).Return(int64(100), nil)

		result, err := svc.ReserveQuota(ctx, now, resID.String(), subID.String(),
			model.ReasonCodeServiceUsage, rateKey, model.UnitTypeMonetary,
			100, "0.50", 60, false)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 100, result.GrantedUnits)
		mgr.AssertExpectations(t)
	})

	t.Run("returns error when requestedUnits is zero", func(t *testing.T) {
		svc := NewQuotaService(new(mockQuotaManager))
		_, err := svc.ReserveQuota(ctx, now, resID.String(), subID.String(),
			model.ReasonCodeServiceUsage, rateKey, model.UnitTypeMonetary,
			0, "0.50", 60, false)
		assert.Error(t, err)
	})

	t.Run("returns error on invalid unitPrice", func(t *testing.T) {
		svc := NewQuotaService(new(mockQuotaManager))
		_, err := svc.ReserveQuota(ctx, now, resID.String(), subID.String(),
			model.ReasonCodeServiceUsage, rateKey, model.UnitTypeMonetary,
			100, "not-a-number", 60, false)
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// DebitQuota tests
// ---------------------------------------------------------------------------

func TestQuotaService_DebitQuota(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	resID := uuid.New()
	subID := uuid.New()

	t.Run("delegates to Debit and maps response", func(t *testing.T) {
		mgr := new(mockQuotaManager)
		svc := NewQuotaService(mgr)

		debitResp := quota.NewDebitResponse(80, dec("40.00"), 80, 0)
		mgr.On("Debit",
			ctx, mock.AnythingOfType("time.Time"),
			subID, resID.String(), resID,
			int64(80), charging.UNITS, true,
		).Return(debitResp, nil)

		result, err := svc.DebitQuota(ctx, now, subID.String(), resID.String(), 80, model.UnitTypeUnits, true)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 80, result.UnitsDebited)
		assert.Equal(t, "40", result.UnitsValue)
		assert.Equal(t, 80, result.ValueUnits)
		assert.Equal(t, 0, result.UnaccountedUnits)
		mgr.AssertExpectations(t)
	})

	t.Run("returns error when usedUnits is zero", func(t *testing.T) {
		svc := NewQuotaService(new(mockQuotaManager))
		_, err := svc.DebitQuota(ctx, now, subID.String(), resID.String(), 0, model.UnitTypeUnits, false)
		assert.Error(t, err)
	})

	t.Run("propagates manager error", func(t *testing.T) {
		mgr := new(mockQuotaManager)
		svc := NewQuotaService(mgr)

		mgr.On("Debit", ctx, mock.AnythingOfType("time.Time"),
			subID, resID.String(), resID,
			int64(10), charging.MONETARY, false,
		).Return((*quota.DebitResponse)(nil), errors.New("quota error"))

		_, err := svc.DebitQuota(ctx, now, subID.String(), resID.String(), 10, model.UnitTypeMonetary, false)
		assert.Error(t, err)
		mgr.AssertExpectations(t)
	})
}
