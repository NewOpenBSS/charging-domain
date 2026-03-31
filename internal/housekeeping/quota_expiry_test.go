package housekeeping

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock QuotaExpirer
// ---------------------------------------------------------------------------

type mockQuotaExpirer struct {
	mock.Mock
}

func (m *mockQuotaExpirer) ProcessExpiredQuota(ctx context.Context, now time.Time, subscriberID uuid.UUID) error {
	return m.Called(ctx, now, subscriberID).Error(0)
}

// ---------------------------------------------------------------------------
// ExpireQuotas tests
// ---------------------------------------------------------------------------

func TestExpireQuotas(t *testing.T) {
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)

	t.Run("no expired subscribers returns zero", func(t *testing.T) {
		mockDB := &mockDBTX{}
		mockQE := &mockQuotaExpirer{}

		rows := newMockRows(nil)
		mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).
			Return(pgx.Rows(rows), nil)

		svc := newTestServiceWithQuotaExpirer(mockDB, mockQE)
		count, err := svc.ExpireQuotas(context.Background(), now)

		require.NoError(t, err)
		assert.Equal(t, 0, count)
		mockDB.AssertExpectations(t)
	})

	t.Run("processes multiple subscribers successfully", func(t *testing.T) {
		mockDB := &mockDBTX{}
		mockQE := &mockQuotaExpirer{}

		sub1 := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
		sub2 := pgtype.UUID{Bytes: [16]byte{2}, Valid: true}

		rows := newMockRows([][]interface{}{
			{sub1},
			{sub2},
		})
		mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).
			Return(pgx.Rows(rows), nil)

		mockQE.On("ProcessExpiredQuota", mock.Anything, now, uuid.UUID(sub1.Bytes)).Return(nil)
		mockQE.On("ProcessExpiredQuota", mock.Anything, now, uuid.UUID(sub2.Bytes)).Return(nil)

		svc := newTestServiceWithQuotaExpirer(mockDB, mockQE)
		count, err := svc.ExpireQuotas(context.Background(), now)

		require.NoError(t, err)
		assert.Equal(t, 2, count)
		mockDB.AssertExpectations(t)
		mockQE.AssertExpectations(t)
	})

	t.Run("find query error is propagated", func(t *testing.T) {
		mockDB := &mockDBTX{}
		mockQE := &mockQuotaExpirer{}

		rows := newMockRows(nil)
		mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).
			Return(pgx.Rows(rows), errors.New("db down"))

		svc := newTestServiceWithQuotaExpirer(mockDB, mockQE)
		count, err := svc.ExpireQuotas(context.Background(), now)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "find expired quota subscribers")
		assert.Equal(t, 0, count)
		mockDB.AssertExpectations(t)
	})

	t.Run("individual subscriber failure continues processing", func(t *testing.T) {
		mockDB := &mockDBTX{}
		mockQE := &mockQuotaExpirer{}

		sub1 := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
		sub2 := pgtype.UUID{Bytes: [16]byte{2}, Valid: true}

		rows := newMockRows([][]interface{}{
			{sub1},
			{sub2},
		})
		mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything).
			Return(pgx.Rows(rows), nil)

		// First subscriber fails, second succeeds
		mockQE.On("ProcessExpiredQuota", mock.Anything, now, uuid.UUID(sub1.Bytes)).
			Return(errors.New("process failed"))
		mockQE.On("ProcessExpiredQuota", mock.Anything, now, uuid.UUID(sub2.Bytes)).
			Return(nil)

		svc := newTestServiceWithQuotaExpirer(mockDB, mockQE)
		count, err := svc.ExpireQuotas(context.Background(), now)

		require.Error(t, err)
		assert.Equal(t, 1, count)
		mockDB.AssertExpectations(t)
		mockQE.AssertExpectations(t)
	})

	t.Run("nil quota expirer returns error", func(t *testing.T) {
		mockDB := &mockDBTX{}

		svc := newTestService(mockDB) // nil quota expirer
		count, err := svc.ExpireQuotas(context.Background(), now)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "quota expirer not configured")
		assert.Equal(t, 0, count)
	})
}
