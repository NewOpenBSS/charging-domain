package chargeengine

import (
	"context"
	"go-ocs/internal/charging"
	"go-ocs/internal/quota"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/twmb/franz-go/pkg/kgo"
)

type MockKafkaManager struct {
	mock.Mock
}

func (m *MockKafkaManager) Produce(ctx context.Context, r *kgo.Record, promise func(*kgo.Record, error)) {
	m.Called(ctx, r, promise)
}

func (m *MockKafkaManager) PublishEvent(topicName string, key string, event any) {
	m.Called(topicName, key, event)
}

type MockQuotaManager struct {
	mock.Mock
}

func (m *MockQuotaManager) ReserveQuota(ctx context.Context, now time.Time, reservationId uuid.UUID, subscriberId uuid.UUID, reason quota.ReasonCode, rateKey charging.RateKey, unitType charging.UnitType, requestedUnits int64, unitPrice decimal.Decimal, multiplier decimal.Decimal, validityTime time.Duration, allowOOBCharging bool) (int64, error) {
	args := m.Called(ctx, now, reservationId, subscriberId, reason, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, allowOOBCharging)
	if fn, ok := args.Get(0).(func(context.Context, time.Time, uuid.UUID, uuid.UUID, quota.ReasonCode, charging.RateKey, charging.UnitType, int64, decimal.Decimal, decimal.Decimal, time.Duration, bool) int64); ok {
		return fn(ctx, now, reservationId, subscriberId, reason, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, allowOOBCharging), args.Error(1)
	}
	res := args.Get(0)
	if res == nil {
		return 0, args.Error(1)
	}
	if i, ok := res.(int); ok {
		return int64(i), args.Error(1)
	}
	return res.(int64), args.Error(1)
}

func (m *MockQuotaManager) Debit(ctx context.Context, now time.Time, subscriberID uuid.UUID, requestId string, reservationId uuid.UUID, usedUnits int64, unitType charging.UnitType, reclaimUnusedUnits bool) (*quota.DebitResponse, error) {
	args := m.Called(ctx, now, subscriberID, requestId, reservationId, usedUnits, unitType, reclaimUnusedUnits)
	return args.Get(0).(*quota.DebitResponse), args.Error(1)
}

func (m *MockQuotaManager) Release(ctx context.Context, subscriberId uuid.UUID, reservationId uuid.UUID) error {
	args := m.Called(ctx, subscriberId, reservationId)
	return args.Error(0)
}

func (m *MockQuotaManager) GetBalance(ctx context.Context, now time.Time, subscriberID uuid.UUID, query quota.BalanceQuery) ([]*quota.CounterBalance, error) {
	args := m.Called(ctx, now, subscriberID, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*quota.CounterBalance), args.Error(1)
}

type MockDBTX struct {
	mock.Mock
}

func (m *MockDBTX) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	a := m.Called(ctx, query, args)
	return a.Get(0).(pgconn.CommandTag), a.Error(1)
}

func (m *MockDBTX) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	a := m.Called(ctx, query, args)
	return a.Get(0).(pgx.Rows), a.Error(1)
}

func (m *MockDBTX) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	a := m.Called(ctx, query, args)
	return a.Get(0).(pgx.Row)
}

type MockRow struct {
	mock.Mock
}

func (m *MockRow) Scan(dest ...interface{}) error {
	return m.Called(dest...).Error(0)
}
