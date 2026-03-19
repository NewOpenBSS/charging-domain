package business

import (
	"context"
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/charging"
	"go-ocs/internal/quota"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/twmb/franz-go/pkg/kgo"
)

type mockKafkaForAccounting struct {
	mock.Mock
}

func (m *mockKafkaForAccounting) Produce(ctx context.Context, r *kgo.Record, promise func(*kgo.Record, error)) {
	m.Called(ctx, r, promise)
}

func (m *mockKafkaForAccounting) PublishEvent(topicName string, key string, event any) {
	m.Called(topicName, key, event)
}

type mockQuotaManager struct {
	mock.Mock
}

func (m *mockQuotaManager) ReserveQuota(ctx context.Context, now time.Time, reservationId uuid.UUID, subscriberId uuid.UUID, reason quota.ReasonCode, rateKey charging.RateKey, unitType charging.UnitType, requestedUnits int64, unitPrice decimal.Decimal, multiplier decimal.Decimal, validityTime time.Duration, allowOOBCharging bool) (int64, error) {
	args := m.Called(ctx, now, reservationId, subscriberId, reason, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, allowOOBCharging)
	return int64(args.Int(0)), args.Error(1)
}

func (m *mockQuotaManager) Debit(ctx context.Context, now time.Time, subscriberID uuid.UUID, requestId string, reservationId uuid.UUID, usedUnits int64, unitType charging.UnitType, reclaimUnusedUnits bool) (*quota.DebitResponse, error) {
	args := m.Called(ctx, now, subscriberID, requestId, reservationId, usedUnits, unitType, reclaimUnusedUnits)
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

func TestReserveQuota_MonetaryUnits(t *testing.T) {
	reservationId := uuid.New()
	subscriberId := uuid.New()
	reason := quota.ReasonServiceUsage
	rateKey := charging.RateKey{
		ServiceType:      "VOICE",
		SourceType:       "HOME",
		ServiceDirection: charging.MO,
		ServiceCategory:  "NATIONAL",
	}
	unitType := charging.MONETARY
	requestedUnits := int64(60)
	unitPrice := decimal.NewFromFloat(0.1)
	multiplier := decimal.NewFromInt(1)
	validityTime := 3600 * time.Second
	allowOOBCharging := false

	mockQM := new(mockQuotaManager)
	dc := &engine.ChargingContext{
		AppContext: &appcontext.AppContext{
			QuotaManager: mockQM,
			KafkaManager: new(mockKafkaForAccounting),
		},
	}

	mockQM.On("ReserveQuota", mock.Anything, mock.Anything, reservationId, subscriberId, reason, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, allowOOBCharging).
		Return(int(requestedUnits), nil)

	grantedUnits, err := ReserveQuota(
		dc,
		reservationId,
		subscriberId,
		reason,
		rateKey,
		unitType,
		requestedUnits,
		unitPrice,
		multiplier,
		validityTime,
		allowOOBCharging,
	)

	assert.NoError(t, err)
	assert.Equal(t, requestedUnits, grantedUnits)
	mockQM.AssertExpectations(t)
}

func TestDebitQuota(t *testing.T) {
	subscriberId := uuid.New()
	requestId := "test-request-id"
	reservationId := uuid.New()
	usedUnits := int64(30)
	unitType := charging.SECONDS
	reclaimUnusedUnits := true

	mockQM := new(mockQuotaManager)
	dc := &engine.ChargingContext{
		AppContext: &appcontext.AppContext{
			QuotaManager: mockQM,
			KafkaManager: new(mockKafkaForAccounting),
		},
	}

	mockResp := &quota.DebitResponse{
		UnitsDebited:     usedUnits,
		UnitsValue:       decimal.NewFromInt(0),
		UnaccountedUnits: 0,
	}

	mockQM.On("Debit", mock.Anything, mock.Anything, subscriberId, requestId, reservationId, usedUnits, unitType, reclaimUnusedUnits).
		Return(mockResp, nil)

	result, err := DebitQuota(
		dc,
		subscriberId,
		requestId,
		reservationId,
		usedUnits,
		unitType,
		reclaimUnusedUnits,
	)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, usedUnits, result.DebitedUnits)
	assert.Equal(t, int64(0), result.UnaccountedUnits)
	assert.True(t, result.MonetaryValue.Equal(decimal.NewFromInt(0)))
	assert.Equal(t, usedUnits, result.MonetaryUnits)
	mockQM.AssertExpectations(t)
}

func TestReserveQuota_ServiceUnits(t *testing.T) {
	reservationId := uuid.New()
	subscriberId := uuid.New()
	reason := quota.ReasonServiceUsage
	rateKey := charging.RateKey{
		ServiceType:      "VOICE",
		SourceType:       "HOME",
		ServiceDirection: charging.MO,
		ServiceCategory:  "NATIONAL",
	}
	unitType := charging.SECONDS
	requestedUnits := int64(60)
	unitPrice := decimal.NewFromFloat(0.1)
	multiplier := decimal.NewFromInt(1)
	validityTime := 3600 * time.Second
	allowOOBCharging := false

	mockQM := new(mockQuotaManager)
	dc := &engine.ChargingContext{
		AppContext: &appcontext.AppContext{
			QuotaManager: mockQM,
			KafkaManager: new(mockKafkaForAccounting),
		},
	}

	mockQM.On("ReserveQuota", mock.Anything, mock.Anything, reservationId, subscriberId, reason, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, allowOOBCharging).
		Return(int(requestedUnits), nil)

	grantedUnits, err := ReserveQuota(
		dc,
		reservationId,
		subscriberId,
		reason,
		rateKey,
		unitType,
		requestedUnits,
		unitPrice,
		multiplier,
		validityTime,
		allowOOBCharging,
	)

	assert.NoError(t, err)
	assert.Equal(t, requestedUnits, grantedUnits)
	mockQM.AssertExpectations(t)
}
