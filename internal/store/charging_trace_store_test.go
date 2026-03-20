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
// Mock DBQuerier — satisfies the DBQuerier interface (Query + QueryRow)
// ---------------------------------------------------------------------------

type mockDBQuerier struct {
	mock.Mock
}

func (m *mockDBQuerier) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	callArgs := []any{ctx, sql}
	callArgs = append(callArgs, args...)
	ret := m.Called(callArgs...)
	return ret.Get(0).(pgx.Rows), ret.Error(1)
}

func (m *mockDBQuerier) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	callArgs := []any{ctx, sql}
	callArgs = append(callArgs, args...)
	return m.Called(callArgs...).Get(0).(pgx.Row)
}

// ---------------------------------------------------------------------------
// Mock pgx.Row
// ---------------------------------------------------------------------------

type storeMockRow struct {
	mock.Mock
}

func (m *storeMockRow) Scan(dest ...any) error {
	return m.Called(dest...).Error(0)
}

// ---------------------------------------------------------------------------
// Mock pgx.Rows
// ---------------------------------------------------------------------------

type storeMockRows struct {
	rows    [][]any
	current int
	closed  bool
}

func newMockRows(rows [][]any) *storeMockRows {
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
	return m.current < len(m.rows)
}

func (m *storeMockRows) Scan(dest ...any) error {
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

func (m *storeMockRows) Values() ([]any, error) {
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

func sampleTraceRow() []any {
	uid := pgtype.UUID{}
	copy(uid.Bytes[:], []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	uid.Valid = true

	ts := pgtype.Timestamptz{Time: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC), Valid: true}

	return []any{
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

// newTraceStore builds a Store with an injected querier mock for unit tests.
// DB and Q are left nil because dynamic store methods use s.querier exclusively.
func newTraceStore(q *mockDBQuerier) *Store {
	return &Store{querier: q}
}

// ---------------------------------------------------------------------------
// ListChargingTraces
// ---------------------------------------------------------------------------

func TestListChargingTraces_Success(t *testing.T) {
	q := &mockDBQuerier{}
	rows := newMockRows([][]any{sampleTraceRow()})

	q.On("Query",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(pgx.Rows(rows), nil)

	s := newTraceStore(q)
	result, err := s.ListChargingTraces(context.Background(), ListChargingTracesParams{
		Limit:  10,
		Offset: 0,
	})

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "CHG-001", result[0].ChargingID)
	assert.Equal(t, "27820001001", result[0].Msisdn)
	assert.Equal(t, int64(42), result[0].ExecutionTime)
	q.AssertExpectations(t)
}

func TestListChargingTraces_EmptyResult(t *testing.T) {
	q := &mockDBQuerier{}
	rows := newMockRows(nil)

	q.On("Query",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(pgx.Rows(rows), nil)

	s := newTraceStore(q)
	result, err := s.ListChargingTraces(context.Background(), ListChargingTracesParams{
		Limit:  10,
		Offset: 0,
	})

	require.NoError(t, err)
	assert.Empty(t, result)
	q.AssertExpectations(t)
}

func TestListChargingTraces_QueryError(t *testing.T) {
	q := &mockDBQuerier{}
	dbErr := errors.New("connection refused")

	q.On("Query",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(pgx.Rows(newMockRows(nil)), dbErr)

	s := newTraceStore(q)
	result, err := s.ListChargingTraces(context.Background(), ListChargingTracesParams{
		Limit: 10,
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
	q.AssertExpectations(t)
}

func TestListChargingTraces_FilterApplied(t *testing.T) {
	q := &mockDBQuerier{}
	rows := newMockRows(nil)

	// Capture the SQL to verify filter is included
	var capturedSQL string
	q.On("Query",
		mock.Anything, mock.MatchedBy(func(sql string) bool {
			capturedSQL = sql
			return true
		}),
		mock.Anything, // WHERE arg ($1)
		mock.Anything, // LIMIT ($2)
		mock.Anything, // OFFSET ($3)
	).Return(pgx.Rows(rows), nil)

	s := newTraceStore(q)
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
	q.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CountChargingTraces
// ---------------------------------------------------------------------------

func TestCountChargingTraces_Success(t *testing.T) {
	q := &mockDBQuerier{}
	mockRow := &storeMockRow{}

	q.On("QueryRow", mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		*(args[0].(*int64)) = 5
	}).Return(nil)

	s := newTraceStore(q)
	count, err := s.CountChargingTraces(context.Background(), "", nil)

	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
	q.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestCountChargingTraces_WithFilter(t *testing.T) {
	q := &mockDBQuerier{}
	mockRow := &storeMockRow{}

	q.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		*(args[0].(*int64)) = 3
	}).Return(nil)

	s := newTraceStore(q)
	count, err := s.CountChargingTraces(context.Background(), "WHERE charging_id = $1", []any{"CHG-001"})

	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
	q.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestCountChargingTraces_QueryError(t *testing.T) {
	q := &mockDBQuerier{}
	mockRow := &storeMockRow{}
	dbErr := errors.New("timeout")

	q.On("QueryRow", mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan", mock.Anything).Return(dbErr)

	s := newTraceStore(q)
	count, err := s.CountChargingTraces(context.Background(), "", nil)

	require.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.Equal(t, dbErr, err)
	q.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}
