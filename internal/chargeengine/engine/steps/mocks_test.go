package steps

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"
)

// stepsMockDBTX provides a testify mock for the pgx DBTX interface used by sqlc.
type stepsMockDBTX struct {
	mock.Mock
}

func (m *stepsMockDBTX) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	a := m.Called(ctx, query, args)
	return a.Get(0).(pgconn.CommandTag), a.Error(1)
}

func (m *stepsMockDBTX) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	a := m.Called(ctx, query, args)
	return a.Get(0).(pgx.Rows), a.Error(1)
}

func (m *stepsMockDBTX) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	a := m.Called(ctx, query, args)
	return a.Get(0).(pgx.Row)
}

// stepsMockRow provides a testify mock for pgx.Row used by sqlc scan operations.
type stepsMockRow struct {
	mock.Mock
}

func (m *stepsMockRow) Scan(dest ...interface{}) error {
	return m.Called(dest...).Error(0)
}
