package services

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go-ocs/internal/backend/graphql/model"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newDestinationGroupService wires a DestinationGroupService backed by a mock DBTX.
// Uses NewTestStore so both static (Q) and dynamic (querier) paths are covered.
func newDestinationGroupService(mockDB *servicesMockDBTX) *DestinationGroupService {
	return NewDestinationGroupService(store.NewTestStore(mockDB, mockDB))
}

// populateDestinationGroupScan fills the 2 Scan destinations with a deterministic group.
func populateDestinationGroupScan(groupName, region string) func(mock.Arguments) {
	return func(args mock.Arguments) {
		*(args[0].(*string)) = groupName // GroupName
		*(args[1].(*string)) = region    // Region
	}
}

// anyQueryRow1DG registers a QueryRow expectation for destination group single-row queries
// (1 SQL arg: group_name).
func anyQueryRow1DG(mockDB *servicesMockDBTX, row *servicesMockRow) {
	// ctx, sql, 1 SQL param
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(row)
}

// anyQueryRow2DG registers a QueryRow expectation for destination group create/update
// (2 SQL args: group_name, region).
func anyQueryRow2DG(mockDB *servicesMockDBTX, row *servicesMockRow) {
	// ctx, sql, 2 SQL params
	mockDB.On("QueryRow",
		mock.Anything, mock.Anything,
		mock.Anything, mock.Anything,
	).Return(row)
}

// ---------------------------------------------------------------------------
// destinationGroupToModel
// ---------------------------------------------------------------------------

func TestDestinationGroupToModel_MapsCorrectly(t *testing.T) {
	g := sqlc.CarrierDestinationGroup{
		GroupName: "NZ",
		Region:    "New Zealand",
	}

	m := destinationGroupToModel(g)

	assert.Equal(t, "NZ", m.GroupName)
	assert.Equal(t, "New Zealand", m.Region)
}

// ---------------------------------------------------------------------------
// DestinationGroupByGroupName
// ---------------------------------------------------------------------------

func TestDestinationGroupByGroupName_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow1DG(mockDB, mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything).
		Run(populateDestinationGroupScan("NZ", "New Zealand")).
		Return(nil)

	svc := newDestinationGroupService(mockDB)
	g, err := svc.DestinationGroupByGroupName(context.Background(), "NZ")

	require.NoError(t, err)
	require.NotNil(t, g)
	assert.Equal(t, "NZ", g.GroupName)
	assert.Equal(t, "New Zealand", g.Region)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestDestinationGroupByGroupName_NotFound(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow1DG(mockDB, mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything).Return(pgx.ErrNoRows)

	svc := newDestinationGroupService(mockDB)
	_, err := svc.DestinationGroupByGroupName(context.Background(), "UNKNOWN")

	require.Error(t, err)
	assert.Equal(t, pgx.ErrNoRows, err)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CreateDestinationGroup
// ---------------------------------------------------------------------------

func TestCreateDestinationGroup_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow2DG(mockDB, mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything).
		Run(populateDestinationGroupScan("AU", "Australia")).
		Return(nil)

	svc := newDestinationGroupService(mockDB)
	input := model.DestinationGroupInput{GroupName: "AU", Region: "Australia"}
	g, err := svc.CreateDestinationGroup(context.Background(), input)

	require.NoError(t, err)
	require.NotNil(t, g)
	assert.Equal(t, "AU", g.GroupName)
	assert.Equal(t, "Australia", g.Region)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestCreateDestinationGroup_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow2DG(mockDB, mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything).Return(errors.New("unique constraint violation"))

	svc := newDestinationGroupService(mockDB)
	input := model.DestinationGroupInput{GroupName: "AU", Region: "Australia"}
	_, err := svc.CreateDestinationGroup(context.Background(), input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unique constraint violation")
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// UpdateDestinationGroup
// ---------------------------------------------------------------------------

func TestUpdateDestinationGroup_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow2DG(mockDB, mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything).
		Run(populateDestinationGroupScan("NZ", "New Zealand (Updated)")).
		Return(nil)

	svc := newDestinationGroupService(mockDB)
	input := model.DestinationGroupInput{GroupName: "NZ", Region: "New Zealand (Updated)"}
	g, err := svc.UpdateDestinationGroup(context.Background(), "NZ", input)

	require.NoError(t, err)
	require.NotNil(t, g)
	assert.Equal(t, "NZ", g.GroupName)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestUpdateDestinationGroup_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow2DG(mockDB, mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything).Return(errors.New("record not found"))

	svc := newDestinationGroupService(mockDB)
	input := model.DestinationGroupInput{GroupName: "UNKNOWN", Region: "Nowhere"}
	_, err := svc.UpdateDestinationGroup(context.Background(), "UNKNOWN", input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "update destination group UNKNOWN")
	assert.Contains(t, err.Error(), "record not found")
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// DeleteDestinationGroup
// ---------------------------------------------------------------------------

func TestDeleteDestinationGroup_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).
		Return(pgconn.CommandTag{}, nil)

	svc := newDestinationGroupService(mockDB)
	ok, err := svc.DeleteDestinationGroup(context.Background(), "NZ")

	require.NoError(t, err)
	assert.True(t, ok)
	mockDB.AssertExpectations(t)
}

func TestDeleteDestinationGroup_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).
		Return(pgconn.CommandTag{}, errors.New("foreign key violation"))

	svc := newDestinationGroupService(mockDB)
	ok, err := svc.DeleteDestinationGroup(context.Background(), "NZ")

	require.Error(t, err)
	assert.False(t, ok)
	assert.Contains(t, err.Error(), "delete destination group NZ")
	assert.Contains(t, err.Error(), "foreign key violation")
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// ListDestinationGroups (dynamic — uses s.store.querier)
// ---------------------------------------------------------------------------

type servicesMockRows struct {
	rows    [][]interface{}
	current int
}

func newServicesMockRows(rows [][]interface{}) *servicesMockRows {
	return &servicesMockRows{rows: rows}
}

func (m *servicesMockRows) Close() {}

func (m *servicesMockRows) Err() error { return nil }

func (m *servicesMockRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (m *servicesMockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (m *servicesMockRows) Next() bool {
	return m.current < len(m.rows)
}

func (m *servicesMockRows) Scan(dest ...interface{}) error {
	if m.current >= len(m.rows) {
		return errors.New("no more rows")
	}
	row := m.rows[m.current]
	m.current++
	for i, d := range dest {
		if i >= len(row) {
			break
		}
		if v, ok := d.(*string); ok {
			if s, ok := row[i].(string); ok {
				*v = s
			}
		}
	}
	return nil
}

func (m *servicesMockRows) Values() ([]interface{}, error) { return nil, nil }

func (m *servicesMockRows) RawValues() [][]byte { return nil }

func (m *servicesMockRows) Conn() *pgx.Conn { return nil }

func TestListDestinationGroups_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	rows := newServicesMockRows([][]interface{}{
		{"AU", "Australia"},
		{"NZ", "New Zealand"},
	})

	mockDB.On("Query",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(pgx.Rows(rows), nil)

	svc := newDestinationGroupService(mockDB)
	result, err := svc.ListDestinationGroups(context.Background(), nil, nil)

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "AU", result[0].GroupName)
	assert.Equal(t, "Australia", result[0].Region)
	mockDB.AssertExpectations(t)
}

func TestListDestinationGroups_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	dbErr := errors.New("connection refused")

	mockDB.On("Query",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(pgx.Rows(newServicesMockRows(nil)), dbErr)

	svc := newDestinationGroupService(mockDB)
	result, err := svc.ListDestinationGroups(context.Background(), nil, nil)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CountDestinationGroups (dynamic — uses s.store.querier)
// ---------------------------------------------------------------------------

func TestCountDestinationGroups_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	mockDB.On("QueryRow", mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		*(args[0].(*int64)) = 5
	}).Return(nil)

	svc := newDestinationGroupService(mockDB)
	count, err := svc.CountDestinationGroups(context.Background(), nil)

	require.NoError(t, err)
	assert.Equal(t, 5, count)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestCountDestinationGroups_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}
	dbErr := errors.New("timeout")

	mockDB.On("QueryRow", mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan", mock.Anything).Return(dbErr)

	svc := newDestinationGroupService(mockDB)
	count, err := svc.CountDestinationGroups(context.Background(), nil)

	require.Error(t, err)
	assert.Equal(t, 0, count)
	assert.Equal(t, dbErr, err)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}
