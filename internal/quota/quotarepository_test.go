package quota

import (
	"context"
	"encoding/json"
	"errors"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDBTX is a mock of sqlc.DBTX
type MockDBTX struct {
	mock.Mock
}

func (m *MockDBTX) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	called := m.Called(ctx, query, args)
	return called.Get(0).(pgconn.CommandTag), called.Error(1)
}

func (m *MockDBTX) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	called := m.Called(ctx, query, args)
	return called.Get(0).(pgx.Rows), called.Error(1)
}

func (m *MockDBTX) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	called := m.Called(ctx, query, args)
	return called.Get(0).(pgx.Row)
}

// MockRow is a mock of pgx.Row
type MockRow struct {
	mock.Mock
}

func (m *MockRow) Scan(dest ...interface{}) error {
	return m.Called(dest...).Error(0)
}

func TestQuotaRepository_Load(t *testing.T) {
	subscriberID := uuid.New()
	quotaID := uuid.New()
	now := time.Now()

	quota := NewEmptyQuota()
	quota.QuotaID = quotaID
	quotaBytes, _ := json.Marshal(quota)

	t.Run("successful load", func(t *testing.T) {
		mockDB := new(MockDBTX)
		mockRow := new(MockRow)

		queries := sqlc.New(mockDB)
		repo := NewQuotaRepository(store.Store{Q: queries})

		mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)
		mockRow.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			// args[0] is *pgtype.UUID (QuotaID)
			// args[1] is *pgtype.Timestamptz (LastModified)
			// args[2] is *pgtype.UUID (SubscriberID)
			// args[3] is *pgtype.Timestamptz (NextActionTime)
			// args[4] is *[]byte (Quota)
			*(args[0].(*pgtype.UUID)) = pgtype.UUID{Bytes: quotaID, Valid: true}
			*(args[1].(*pgtype.Timestamptz)) = pgtype.Timestamptz{Time: now, Valid: true}
			*(args[2].(*pgtype.UUID)) = pgtype.UUID{Bytes: subscriberID, Valid: true}
			*(args[4].(*[]byte)) = quotaBytes
		}).Return(nil)

		loaded, err := repo.Load(context.Background(), subscriberID)

		assert.NoError(t, err)
		assert.NotNil(t, loaded)
		assert.Equal(t, quotaID, loaded.Quota.QuotaID)
		assert.True(t, now.Equal(loaded.Version))
		mockDB.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockDB := new(MockDBTX)
		mockRow := new(MockRow)

		queries := sqlc.New(mockDB)
		repo := NewQuotaRepository(store.Store{Q: queries})

		mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)
		mockRow.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(pgx.ErrNoRows)

		loaded, err := repo.Load(context.Background(), subscriberID)

		assert.NoError(t, err)
		assert.Nil(t, loaded)
	})

	t.Run("database error", func(t *testing.T) {
		mockDB := new(MockDBTX)
		mockRow := new(MockRow)

		queries := sqlc.New(mockDB)
		repo := NewQuotaRepository(store.Store{Q: queries})

		mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)
		mockRow.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db error"))

		loaded, err := repo.Load(context.Background(), subscriberID)

		assert.Error(t, err)
		assert.Nil(t, loaded)
	})
}

func TestQuotaRepository_Create(t *testing.T) {
	subscriberID := uuid.New()
	now := time.Now()

	t.Run("successful creation", func(t *testing.T) {
		mockDB := new(MockDBTX)
		mockRow := new(MockRow)

		queries := sqlc.New(mockDB)
		repo := NewQuotaRepository(store.Store{Q: queries})

		mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockRow)
		mockRow.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			*(args[1].(*pgtype.Timestamptz)) = pgtype.Timestamptz{Time: now, Valid: true}
		}).Return(nil)

		loaded, err := repo.Create(context.Background(), subscriberID)

		assert.NoError(t, err)
		assert.NotNil(t, loaded)
		assert.True(t, now.Equal(loaded.Version))
		mockDB.AssertExpectations(t)
	})

	t.Run("database error", func(t *testing.T) {
		mockDB := new(MockDBTX)
		mockRow := new(MockRow)

		queries := sqlc.New(mockDB)
		repo := NewQuotaRepository(store.Store{Q: queries})

		mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockRow)
		mockRow.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db error"))

		loaded, err := repo.Create(context.Background(), subscriberID)

		assert.Error(t, err)
		assert.Nil(t, loaded)
	})
}

func TestQuotaRepository_Save(t *testing.T) {
	quotaID := uuid.New()
	oldVersion := time.Now().Add(-1 * time.Hour)
	quota := NewEmptyQuota()
	quota.QuotaID = quotaID
	loaded := &LoadedQuota{
		Quota:   quota,
		Version: oldVersion,
	}

	t.Run("successful save", func(t *testing.T) {
		mockDB := new(MockDBTX)

		queries := sqlc.New(mockDB)
		repo := NewQuotaRepository(store.Store{Q: queries})

		mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag("UPDATE 1"), nil)

		err := repo.Save(context.Background(), loaded)

		assert.NoError(t, err)
		assert.True(t, loaded.Version.After(oldVersion))
		mockDB.AssertExpectations(t)
	})

	t.Run("optimistic lock failure", func(t *testing.T) {
		mockDB := new(MockDBTX)

		queries := sqlc.New(mockDB)
		repo := NewQuotaRepository(store.Store{Q: queries})

		mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag("UPDATE 0"), nil)

		err := repo.Save(context.Background(), loaded)

		assert.Error(t, err)
		assert.IsType(t, &PessimisticLockError{}, err)
	})

	t.Run("database error", func(t *testing.T) {
		mockDB := new(MockDBTX)

		queries := sqlc.New(mockDB)
		repo := NewQuotaRepository(store.Store{Q: queries})

		mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, errors.New("db error"))

		err := repo.Save(context.Background(), loaded)

		assert.Error(t, err)
	})
}
