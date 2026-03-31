package housekeeping

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/mock"

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
	return NewHousekeepingService(store.NewTestStore(mockDB, mockDB), nil)
}

func newTestServiceWithQuotaExpirer(mockDB *mockDBTX, qe QuotaExpirer) *HousekeepingService {
	return NewHousekeepingService(store.NewTestStore(mockDB, mockDB), qe)
}
