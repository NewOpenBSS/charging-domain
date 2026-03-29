package store

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func sampleDestinationGroupRows() [][]any {
	return [][]any{
		{"NZ", "New Zealand"},
		{"AU", "Australia"},
	}
}

// newDestinationGroupStore builds a Store with an injected querier mock for unit tests.
func newDestinationGroupStore(q *mockDBQuerier) *Store {
	return &Store{querier: q}
}

// ---------------------------------------------------------------------------
// ListDestinationGroups
// ---------------------------------------------------------------------------

func TestListDestinationGroups_Success(t *testing.T) {
	q := &mockDBQuerier{}
	rows := newMockRows(sampleDestinationGroupRows())

	q.On("Query",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(pgx.Rows(rows), nil)

	s := newDestinationGroupStore(q)
	result, err := s.ListDestinationGroups(context.Background(), ListDestinationGroupsParams{
		Limit:  10,
		Offset: 0,
	})

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "NZ", result[0].GroupName)
	assert.Equal(t, "New Zealand", result[0].Region)
	assert.Equal(t, "AU", result[1].GroupName)
	assert.Equal(t, "Australia", result[1].Region)
	q.AssertExpectations(t)
}

func TestListDestinationGroups_EmptyResult(t *testing.T) {
	q := &mockDBQuerier{}
	rows := newMockRows(nil)

	q.On("Query",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(pgx.Rows(rows), nil)

	s := newDestinationGroupStore(q)
	result, err := s.ListDestinationGroups(context.Background(), ListDestinationGroupsParams{
		Limit:  10,
		Offset: 0,
	})

	require.NoError(t, err)
	assert.Empty(t, result)
	q.AssertExpectations(t)
}

func TestListDestinationGroups_QueryError(t *testing.T) {
	q := &mockDBQuerier{}
	dbErr := errors.New("connection refused")

	q.On("Query",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(pgx.Rows(newMockRows(nil)), dbErr)

	s := newDestinationGroupStore(q)
	result, err := s.ListDestinationGroups(context.Background(), ListDestinationGroupsParams{
		Limit: 10,
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
	q.AssertExpectations(t)
}

func TestListDestinationGroups_FilterApplied(t *testing.T) {
	q := &mockDBQuerier{}
	rows := newMockRows(nil)

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

	s := newDestinationGroupStore(q)
	_, err := s.ListDestinationGroups(context.Background(), ListDestinationGroupsParams{
		WhereSQL: "WHERE group_name ILIKE $1",
		Args:     []any{"NZ%"},
		OrderSQL: "ORDER BY group_name ASC",
		Limit:    5,
		Offset:   10,
	})

	require.NoError(t, err)
	assert.True(t, strings.Contains(capturedSQL, "WHERE group_name ILIKE $1"), "query must include WHERE clause")
	assert.True(t, strings.Contains(capturedSQL, "ORDER BY group_name ASC"), "query must include ORDER BY clause")
	q.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CountDestinationGroups
// ---------------------------------------------------------------------------

func TestCountDestinationGroups_Success(t *testing.T) {
	q := &mockDBQuerier{}
	mockRow := &storeMockRow{}

	q.On("QueryRow", mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		*(args[0].(*int64)) = 7
	}).Return(nil)

	s := newDestinationGroupStore(q)
	count, err := s.CountDestinationGroups(context.Background(), "", nil)

	require.NoError(t, err)
	assert.Equal(t, int64(7), count)
	q.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestCountDestinationGroups_WithFilter(t *testing.T) {
	q := &mockDBQuerier{}
	mockRow := &storeMockRow{}

	q.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		*(args[0].(*int64)) = 2
	}).Return(nil)

	s := newDestinationGroupStore(q)
	count, err := s.CountDestinationGroups(context.Background(), "WHERE region = $1", []any{"New Zealand"})

	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
	q.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestCountDestinationGroups_QueryError(t *testing.T) {
	q := &mockDBQuerier{}
	mockRow := &storeMockRow{}
	dbErr := errors.New("timeout")

	q.On("QueryRow", mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan", mock.Anything).Return(dbErr)

	s := newDestinationGroupStore(q)
	count, err := s.CountDestinationGroups(context.Background(), "", nil)

	require.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.Equal(t, dbErr, err)
	q.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}
