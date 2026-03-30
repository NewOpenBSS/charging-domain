package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"go-ocs/cmd/charging-housekeeping/appcontext"
	"go-ocs/internal/events"
	"go-ocs/internal/housekeeping"
	"go-ocs/internal/quota"
	"go-ocs/internal/store"
)

// ---------------------------------------------------------------------------
// Mock DBTX — satisfies sqlc.DBTX and store.DBQuerier
// ---------------------------------------------------------------------------

type mockDBTX struct {
	mock.Mock
}

func (m *mockDBTX) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	callArgs := []interface{}{ctx, sql}
	callArgs = append(callArgs, args...)
	ret := m.Called(callArgs...)
	return ret.Get(0).(pgconn.CommandTag), ret.Error(1)
}

func (m *mockDBTX) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	callArgs := []interface{}{ctx, sql}
	callArgs = append(callArgs, args...)
	ret := m.Called(callArgs...)
	return ret.Get(0).(pgx.Rows), ret.Error(1)
}

func (m *mockDBTX) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	callArgs := []interface{}{ctx, sql}
	callArgs = append(callArgs, args...)
	return m.Called(callArgs...).Get(0).(pgx.Row)
}

// ---------------------------------------------------------------------------
// Mock Rows — satisfies pgx.Rows
// ---------------------------------------------------------------------------

type mockRows struct {
	rows    [][]interface{}
	current int
}

func newMockRows(rows [][]interface{}) *mockRows {
	return &mockRows{rows: rows}
}

func (m *mockRows) Close()                                       {}
func (m *mockRows) Err() error                                   { return nil }
func (m *mockRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (m *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (m *mockRows) Values() ([]interface{}, error)               { return nil, nil }
func (m *mockRows) RawValues() [][]byte                          { return nil }
func (m *mockRows) Conn() *pgx.Conn                              { return nil }

func (m *mockRows) Next() bool {
	return m.current < len(m.rows)
}

func (m *mockRows) Scan(dest ...interface{}) error {
	if m.current >= len(m.rows) {
		return errors.New("no more rows")
	}
	row := m.rows[m.current]
	m.current++
	for i, d := range dest {
		switch ptr := d.(type) {
		case *int64:
			*ptr = row[i].(int64)
		case *string:
			*ptr = row[i].(string)
		default:
			// For pgtype types, use reflect-based assignment not needed in basic tests
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func noopKafka() *events.KafkaManager {
	return &events.KafkaManager{KafkaConfig: events.KafkaConfig{}, KafkaClient: nil}
}

func defaultConfig() *appcontext.Config {
	return &appcontext.Config{
		StaleSessions:   24 * time.Hour,
		TracePurge:      36 * time.Hour,
		RatePlanCleanup: 30 * 24 * time.Hour,
	}
}

// ---------------------------------------------------------------------------
// Tests for run()
// ---------------------------------------------------------------------------

func TestRun_AllTasksSucceed_ExitZero(t *testing.T) {
	mockDB := &mockDBTX{}
	s := store.NewTestStore(mockDB, mockDB)
	km := noopKafka()
	qm := quota.NewQuotaManager(*s, 3, km)
	svc := housekeeping.NewHousekeepingService(s)
	cfg := defaultConfig()
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)

	// FindExpiredQuotaSubscribers — returns empty (no expired subscribers)
	mockDB.On("Query", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return contains(sql, "next_action_time")
	}), mock.Anything).Return(pgx.Rows(newMockRows(nil)), nil).Once()

	// DeleteStaleChargingData
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return contains(sql, "charging_data")
	}), mock.Anything).Return(pgconn.NewCommandTag("DELETE 3"), nil).Once()

	// DeleteOldChargingTrace
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return contains(sql, "charging_trace")
	}), mock.Anything).Return(pgconn.NewCommandTag("DELETE 7"), nil).Once()

	// ListSupersededRatePlanVersions — returns empty
	mockDB.On("Query", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return contains(sql, "rateplan")
	}), mock.Anything).Return(pgx.Rows(newMockRows(nil)), nil).Once()

	exitCode := run(context.Background(), now, cfg, s, qm, svc)

	assert.Equal(t, 0, exitCode)
	mockDB.AssertExpectations(t)
}

func TestRun_QuotaExpiryError_ExitOne_OtherTasksStillRun(t *testing.T) {
	mockDB := &mockDBTX{}
	s := store.NewTestStore(mockDB, mockDB)
	km := noopKafka()
	qm := quota.NewQuotaManager(*s, 3, km)
	svc := housekeeping.NewHousekeepingService(s)
	cfg := defaultConfig()
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)

	// FindExpiredQuotaSubscribers — error
	mockDB.On("Query", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return contains(sql, "next_action_time")
	}), mock.Anything).Return(pgx.Rows(newMockRows(nil)), errors.New("db down")).Once()

	// Other tasks still run:
	// DeleteStaleChargingData
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return contains(sql, "charging_data")
	}), mock.Anything).Return(pgconn.NewCommandTag("DELETE 0"), nil).Once()

	// DeleteOldChargingTrace
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return contains(sql, "charging_trace")
	}), mock.Anything).Return(pgconn.NewCommandTag("DELETE 0"), nil).Once()

	// ListSupersededRatePlanVersions
	mockDB.On("Query", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return contains(sql, "rateplan")
	}), mock.Anything).Return(pgx.Rows(newMockRows(nil)), nil).Once()

	exitCode := run(context.Background(), now, cfg, s, qm, svc)

	assert.Equal(t, 1, exitCode)
	mockDB.AssertExpectations(t)
}

func TestRun_CleanupError_ExitOne(t *testing.T) {
	mockDB := &mockDBTX{}
	s := store.NewTestStore(mockDB, mockDB)
	km := noopKafka()
	qm := quota.NewQuotaManager(*s, 3, km)
	svc := housekeeping.NewHousekeepingService(s)
	cfg := defaultConfig()
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)

	// FindExpiredQuotaSubscribers — success, empty
	mockDB.On("Query", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return contains(sql, "next_action_time")
	}), mock.Anything).Return(pgx.Rows(newMockRows(nil)), nil).Once()

	// DeleteStaleChargingData — error
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return contains(sql, "charging_data")
	}), mock.Anything).Return(pgconn.CommandTag{}, errors.New("timeout")).Once()

	// DeleteOldChargingTrace — still runs, succeeds
	mockDB.On("Exec", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return contains(sql, "charging_trace")
	}), mock.Anything).Return(pgconn.NewCommandTag("DELETE 2"), nil).Once()

	// ListSupersededRatePlanVersions — still runs, succeeds
	mockDB.On("Query", mock.Anything, mock.MatchedBy(func(sql string) bool {
		return contains(sql, "rateplan")
	}), mock.Anything).Return(pgx.Rows(newMockRows(nil)), nil).Once()

	exitCode := run(context.Background(), now, cfg, s, qm, svc)

	assert.Equal(t, 1, exitCode)
	mockDB.AssertExpectations(t)
}

// contains is a helper to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
