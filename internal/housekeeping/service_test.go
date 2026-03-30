package housekeeping

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

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
// Mock Rows — satisfies pgx.Rows for :many queries
// ---------------------------------------------------------------------------

type mockRows struct {
	rows    [][]interface{}
	current int
}

func newMockRows(rows [][]interface{}) *mockRows {
	return &mockRows{rows: rows}
}

func (m *mockRows) Close()                                        {}
func (m *mockRows) Err() error                                    { return nil }
func (m *mockRows) CommandTag() pgconn.CommandTag                 { return pgconn.CommandTag{} }
func (m *mockRows) FieldDescriptions() []pgconn.FieldDescription  { return nil }
func (m *mockRows) Values() ([]interface{}, error)                { return nil, nil }
func (m *mockRows) RawValues() [][]byte                           { return nil }
func (m *mockRows) Conn() *pgx.Conn                               { return nil }

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
		case *pgtype.UUID:
			*ptr = row[i].(pgtype.UUID)
		case *pgtype.Timestamptz:
			*ptr = row[i].(pgtype.Timestamptz)
		case *pgtype.Text:
			*ptr = row[i].(pgtype.Text)
		case *[]byte:
			*ptr = row[i].([]byte)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestService(mockDB *mockDBTX) *HousekeepingService {
	return NewHousekeepingService(store.NewTestStore(mockDB, mockDB))
}

// ---------------------------------------------------------------------------
// CleanStaleSessions
// ---------------------------------------------------------------------------

func TestCleanStaleSessions(t *testing.T) {
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	threshold := 24 * time.Hour

	tests := []struct {
		name      string
		tag       pgconn.CommandTag
		execErr   error
		wantCount int64
		wantErr   bool
	}{
		{
			name:      "deletes N rows successfully",
			tag:       pgconn.NewCommandTag("DELETE 5"),
			execErr:   nil,
			wantCount: 5,
			wantErr:   false,
		},
		{
			name:      "zero rows deleted",
			tag:       pgconn.NewCommandTag("DELETE 0"),
			execErr:   nil,
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "store error is propagated",
			tag:       pgconn.CommandTag{},
			execErr:   errors.New("connection lost"),
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := &mockDBTX{}
			mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).
				Return(tc.tag, tc.execErr)

			svc := newTestService(mockDB)
			count, err := svc.CleanStaleSessions(context.Background(), now, threshold)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "delete stale charging data")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantCount, count)
			}
			mockDB.AssertExpectations(t)
		})
	}
}

// ---------------------------------------------------------------------------
// PurgeOldTraces
// ---------------------------------------------------------------------------

func TestPurgeOldTraces(t *testing.T) {
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	threshold := 36 * time.Hour

	tests := []struct {
		name      string
		tag       pgconn.CommandTag
		execErr   error
		wantCount int64
		wantErr   bool
	}{
		{
			name:      "deletes N rows successfully",
			tag:       pgconn.NewCommandTag("DELETE 12"),
			execErr:   nil,
			wantCount: 12,
			wantErr:   false,
		},
		{
			name:      "zero rows deleted",
			tag:       pgconn.NewCommandTag("DELETE 0"),
			execErr:   nil,
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "store error is propagated",
			tag:       pgconn.CommandTag{},
			execErr:   errors.New("timeout"),
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := &mockDBTX{}
			mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).
				Return(tc.tag, tc.execErr)

			svc := newTestService(mockDB)
			count, err := svc.PurgeOldTraces(context.Background(), now, threshold)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "delete old charging trace")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantCount, count)
			}
			mockDB.AssertExpectations(t)
		})
	}
}

// ---------------------------------------------------------------------------
// CleanupSupersededRatePlans
// ---------------------------------------------------------------------------

func TestCleanupSupersededRatePlans(t *testing.T) {
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	threshold := 30 * 24 * time.Hour

	t.Run("deletes multiple superseded versions", func(t *testing.T) {
		mockDB := &mockDBTX{}
		planID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
		ts := pgtype.Timestamptz{Time: now.Add(-60 * 24 * time.Hour), Valid: true}

		// ListSupersededRatePlanVersions returns 2 rows (11 columns each per Rateplan model)
		rows := newMockRows([][]interface{}{
			{int64(10), planID, ts, "SETTLEMENT", pgtype.UUID{}, "Plan A v1", []byte("{}"), "ACTIVE", "admin", pgtype.Text{}, ts},
			{int64(20), planID, ts, "SETTLEMENT", pgtype.UUID{}, "Plan A v2", []byte("{}"), "ACTIVE", "admin", pgtype.Text{}, ts},
		})

		mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).
			Return(pgx.Rows(rows), nil)

		// Two Exec calls for DeleteRatePlanVersionById (id=10, id=20)
		mockDB.On("Exec", mock.Anything, mock.Anything, int64(10)).
			Return(pgconn.NewCommandTag("DELETE 1"), nil)
		mockDB.On("Exec", mock.Anything, mock.Anything, int64(20)).
			Return(pgconn.NewCommandTag("DELETE 1"), nil)

		svc := newTestService(mockDB)
		count, err := svc.CleanupSupersededRatePlans(context.Background(), now, threshold)

		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
		mockDB.AssertExpectations(t)
	})

	t.Run("zero versions returns 0 with no error", func(t *testing.T) {
		mockDB := &mockDBTX{}
		rows := newMockRows(nil) // no rows

		mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).
			Return(pgx.Rows(rows), nil)

		svc := newTestService(mockDB)
		count, err := svc.CleanupSupersededRatePlans(context.Background(), now, threshold)

		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
		mockDB.AssertExpectations(t)
	})

	t.Run("list query error is propagated", func(t *testing.T) {
		mockDB := &mockDBTX{}
		rows := newMockRows(nil)

		mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).
			Return(pgx.Rows(rows), errors.New("db error"))

		svc := newTestService(mockDB)
		count, err := svc.CleanupSupersededRatePlans(context.Background(), now, threshold)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "list superseded rate plan versions")
		assert.Equal(t, int64(0), count)
		mockDB.AssertExpectations(t)
	})

	t.Run("stop on first delete error", func(t *testing.T) {
		mockDB := &mockDBTX{}
		planID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
		ts := pgtype.Timestamptz{Time: now.Add(-60 * 24 * time.Hour), Valid: true}

		rows := newMockRows([][]interface{}{
			{int64(10), planID, ts, "SETTLEMENT", pgtype.UUID{}, "Plan A v1", []byte("{}"), "ACTIVE", "admin", pgtype.Text{}, ts},
			{int64(20), planID, ts, "SETTLEMENT", pgtype.UUID{}, "Plan A v2", []byte("{}"), "ACTIVE", "admin", pgtype.Text{}, ts},
		})

		mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).
			Return(pgx.Rows(rows), nil)

		// First delete succeeds, second fails
		mockDB.On("Exec", mock.Anything, mock.Anything, int64(10)).
			Return(pgconn.NewCommandTag("DELETE 1"), nil)
		mockDB.On("Exec", mock.Anything, mock.Anything, int64(20)).
			Return(pgconn.CommandTag{}, errors.New("constraint violation"))

		svc := newTestService(mockDB)
		count, err := svc.CleanupSupersededRatePlans(context.Background(), now, threshold)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete rate plan version 20")
		// First delete succeeded before the error
		assert.Equal(t, int64(1), count)
		mockDB.AssertExpectations(t)
	})
}
