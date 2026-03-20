package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock DBTX — satisfies PoolDB (sqlc.DBTX + Close)
// ---------------------------------------------------------------------------

type storeMockDB struct {
	mock.Mock
}

func (m *storeMockDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	callArgs := []interface{}{ctx, sql}
	callArgs = append(callArgs, args...)
	ret := m.Called(callArgs...)
	return ret.Get(0).(pgconn.CommandTag), ret.Error(1)
}

func (m *storeMockDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	callArgs := []interface{}{ctx, sql}
	callArgs = append(callArgs, args...)
	ret := m.Called(callArgs...)
	return ret.Get(0).(pgx.Rows), ret.Error(1)
}

func (m *storeMockDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	callArgs := []interface{}{ctx, sql}
	callArgs = append(callArgs, args...)
	return m.Called(callArgs...).Get(0).(pgx.Row)
}

func (m *storeMockDB) Close() {
	m.Called()
}

// ---------------------------------------------------------------------------
// Mock pgx.Row
// ---------------------------------------------------------------------------

type storeMockRow struct {
	mock.Mock
}

func (m *storeMockRow) Scan(dest ...interface{}) error {
	return m.Called(dest...).Error(0)
}

// ---------------------------------------------------------------------------
// Mock pgx.Rows
// ---------------------------------------------------------------------------

type storeMockRows struct {
	mock.Mock
	rows    [][]interface{}
	current int
	closed  bool
}

func newMockRows(rows [][]interface{}) *storeMockRows {
	return &storeMockRows{rows: rows}
}

func (m *storeMockRows) Close() {
	m.closed = true
}

func (m *storeMockRows) Err() error {
	return nil
}

func (m *storeMockRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (m *storeMockRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (m *storeMockRows) Next() bool {
	if m.current < len(m.rows) {
		return true
	}
	return false
}

func (m *storeMockRows) Scan(dest ...interface{}) error {
	if m.current >= len(m.rows) {
		return errors.New("no more rows")
	}
	row := m.rows[m.current]
	m.current++
	for i, d := range dest {
		if i >= len(row) {
			break
		}
		switch v := d.(type) {
		case *pgtype.UUID:
			if src, ok := row[i].(pgtype.UUID); ok {
				*v = src
			}
		case *pgtype.Timestamptz:
			if src, ok := row[i].(pgtype.Timestamptz); ok {
				*v = src
			}
		case *[]byte:
			if src, ok := row[i].([]byte); ok {
				*v = src
			}
		case *int64:
			if src, ok := row[i].(int64); ok {
				*v = src
			}
		case *string:
			if src, ok := row[i].(string); ok {
				*v = src
			}
		case *int32:
			if src, ok := row[i].(int32); ok {
				*v = src
			}
		}
	}
	return nil
}

func (m *storeMockRows) Values() ([]interface{}, error) {
	return nil, nil
}

func (m *storeMockRows) RawValues() [][]byte {
	return nil
}

func (m *storeMockRows) Conn() *pgx.Conn {
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func sampleTraceRow() []interface{} {
	uid := pgtype.UUID{}
	copy(uid.Bytes[:], []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	uid.Valid = true

	ts := pgtype.Timestamptz{Time: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC), Valid: true}

	return []interface{}{
		uid,
		ts,
		[]byte(`{"service":"data"}`),
		[]byte(`{"granted":1024}`),
		int64(42),
		"CHG-001",
		int32(1),
		"27820001001",
	}
}

func newTraceStore(mockDB *storeMockDB) *Store {
	return &Store{DB: mockDB}
}

// ---------------------------------------------------------------------------
// ListChargingTraces
// ---------------------------------------------------------------------------

func TestListChargingTraces_Success(t *testing.T) {
	mockDB := &storeMockDB{}
	rows := newMockRows([][]interface{}{sampleTraceRow()})

	mockDB.On("Query",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(pgx.Rows(rows), nil)

	s := newTraceStore(mockDB)
	result, err := s.ListChargingTraces(context.Background(), ListChargingTracesParams{
		Limit:  10,
		Offset: 0,
	})

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "CHG-001", result[0].ChargingID)
	assert.Equal(t, "27820001001", result[0].Msisdn)
	assert.Equal(t, int64(42), result[0].ExecutionTime)
	mockDB.AssertExpectations(t)
}

func TestListChargingTraces_EmptyResult(t *testing.T) {
	mockDB := &storeMockDB{}
	rows := newMockRows(nil)

	mockDB.On("Query",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(pgx.Rows(rows), nil)

	s := newTraceStore(mockDB)
	result, err := s.ListChargingTraces(context.Background(), ListChargingTracesParams{
		Limit:  10,
		Offset: 0,
	})

	require.NoError(t, err)
	assert.Empty(t, result)
	mockDB.AssertExpectations(t)
}

func TestListChargingTraces_QueryError(t *testing.T) {
	mockDB := &storeMockDB{}
	dbErr := errors.New("connection refused")

	mockDB.On("Query",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(pgx.Rows(newMockRows(nil)), dbErr)

	s := newTraceStore(mockDB)
	result, err := s.ListChargingTraces(context.Background(), ListChargingTracesParams{
		Limit: 10,
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
	mockDB.AssertExpectations(t)
}

func TestListChargingTraces_FilterApplied(t *testing.T) {
	mockDB := &storeMockDB{}
	rows := newMockRows(nil)

	// Capture the SQL to verify filter is included
	var capturedSQL string
	mockDB.On("Query",
		mock.Anything, mock.MatchedBy(func(sql string) bool {
			capturedSQL = sql
			return true
		}),
		mock.Anything, // WHERE arg ($1)
		mock.Anything, // LIMIT ($2)
		mock.Anything, // OFFSET ($3)
	).Return(pgx.Rows(rows), nil)

	s := newTraceStore(mockDB)
	_, err := s.ListChargingTraces(context.Background(), ListChargingTracesParams{
		WhereSQL: "WHERE charging_id ILIKE $1",
		Args:     []any{"CHG%"},
		OrderSQL: "ORDER BY created_at DESC",
		Limit:    5,
		Offset:   10,
	})

	require.NoError(t, err)
	assert.True(t, strings.Contains(capturedSQL, "WHERE charging_id ILIKE $1"), "query must include WHERE clause")
	assert.True(t, strings.Contains(capturedSQL, "ORDER BY created_at DESC"), "query must include ORDER BY clause")
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CountChargingTraces
// ---------------------------------------------------------------------------

func TestCountChargingTraces_Success(t *testing.T) {
	mockDB := &storeMockDB{}
	mockRow := &storeMockRow{}

	mockDB.On("QueryRow", mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		*(args[0].(*int64)) = 5
	}).Return(nil)

	s := newTraceStore(mockDB)
	count, err := s.CountChargingTraces(context.Background(), "", nil)

	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestCountChargingTraces_EmptyResult(t *testing.T) {
	mockDB := &storeMockDB{}
	mockRow := &storeMockRow{}

	mockDB.On("QueryRow", mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		*(args[0].(*int64)) = 0
	}).Return(nil)

	s := newTraceStore(mockDB)
	count, err := s.CountChargingTraces(context.Background(), "", nil)

	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestCountChargingTraces_QueryError(t *testing.T) {
	mockDB := &storeMockDB{}
	mockRow := &storeMockRow{}
	dbErr := errors.New("timeout")

	mockDB.On("QueryRow", mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan", mock.Anything).Return(dbErr)

	s := newTraceStore(mockDB)
	count, err := s.CountChargingTraces(context.Background(), "", nil)

	require.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.Equal(t, dbErr, err)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}
