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
)

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
