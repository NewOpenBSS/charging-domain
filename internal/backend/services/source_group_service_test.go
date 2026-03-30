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

// newSourceGroupService wires a SourceGroupService backed by a mock DBTX.
// Uses NewTestStore so both static (Q) and dynamic (querier) paths are covered.
func newSourceGroupService(mockDB *servicesMockDBTX) *SourceGroupService {
	return NewSourceGroupService(store.NewTestStore(mockDB, mockDB))
}

// populateSourceGroupScan fills the 2 Scan destinations with a deterministic group.
func populateSourceGroupScan(groupName, region string) func(mock.Arguments) {
	return func(args mock.Arguments) {
		*(args[0].(*string)) = groupName // GroupName
		*(args[1].(*string)) = region    // Region
	}
}

// anyQueryRow1SG registers a QueryRow expectation for source group single-row queries
// (1 SQL arg: group_name).
func anyQueryRow1SG(mockDB *servicesMockDBTX, row *servicesMockRow) {
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(row)
}

// anyQueryRow2SG registers a QueryRow expectation for source group create/update
// (2 SQL args: group_name, region).
func anyQueryRow2SG(mockDB *servicesMockDBTX, row *servicesMockRow) {
	mockDB.On("QueryRow",
		mock.Anything, mock.Anything,
		mock.Anything, mock.Anything,
	).Return(row)
}

// ---------------------------------------------------------------------------
// sourceGroupToModel
// ---------------------------------------------------------------------------

func TestSourceGroupToModel_MapsCorrectly(t *testing.T) {
	g := sqlc.CarrierSourceGroup{
		GroupName: "HOME",
		Region:    "Local",
	}

	m := sourceGroupToModel(g)

	assert.Equal(t, "HOME", m.GroupName)
	assert.Equal(t, "Local", m.Region)
}

// ---------------------------------------------------------------------------
// SourceGroupByGroupName
// ---------------------------------------------------------------------------

func TestSourceGroupByGroupName_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow1SG(mockDB, mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything).
		Run(populateSourceGroupScan("HOME", "Local")).
		Return(nil)

	svc := newSourceGroupService(mockDB)
	g, err := svc.SourceGroupByGroupName(context.Background(), "HOME")

	require.NoError(t, err)
	require.NotNil(t, g)
	assert.Equal(t, "HOME", g.GroupName)
	assert.Equal(t, "Local", g.Region)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestSourceGroupByGroupName_NotFound(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow1SG(mockDB, mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything).Return(pgx.ErrNoRows)

	svc := newSourceGroupService(mockDB)
	_, err := svc.SourceGroupByGroupName(context.Background(), "UNKNOWN")

	require.Error(t, err)
	assert.Equal(t, pgx.ErrNoRows, err)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CreateSourceGroup
// ---------------------------------------------------------------------------

func TestCreateSourceGroup_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow2SG(mockDB, mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything).
		Run(populateSourceGroupScan("EUROPE", "Europe")).
		Return(nil)

	svc := newSourceGroupService(mockDB)
	input := model.SourceGroupInput{GroupName: "EUROPE", Region: "Europe"}
	g, err := svc.CreateSourceGroup(context.Background(), input)

	require.NoError(t, err)
	require.NotNil(t, g)
	assert.Equal(t, "EUROPE", g.GroupName)
	assert.Equal(t, "Europe", g.Region)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestCreateSourceGroup_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow2SG(mockDB, mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything).Return(errors.New("unique constraint violation"))

	svc := newSourceGroupService(mockDB)
	input := model.SourceGroupInput{GroupName: "HOME", Region: "Local"}
	_, err := svc.CreateSourceGroup(context.Background(), input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unique constraint violation")
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// UpdateSourceGroup
// ---------------------------------------------------------------------------

func TestUpdateSourceGroup_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow2SG(mockDB, mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything).
		Run(populateSourceGroupScan("HOME", "Local (Updated)")).
		Return(nil)

	svc := newSourceGroupService(mockDB)
	input := model.SourceGroupInput{GroupName: "HOME", Region: "Local (Updated)"}
	g, err := svc.UpdateSourceGroup(context.Background(), "HOME", input)

	require.NoError(t, err)
	require.NotNil(t, g)
	assert.Equal(t, "HOME", g.GroupName)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestUpdateSourceGroup_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow2SG(mockDB, mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything).Return(errors.New("record not found"))

	svc := newSourceGroupService(mockDB)
	input := model.SourceGroupInput{GroupName: "UNKNOWN", Region: "Nowhere"}
	_, err := svc.UpdateSourceGroup(context.Background(), "UNKNOWN", input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "update source group UNKNOWN")
	assert.Contains(t, err.Error(), "record not found")
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// DeleteSourceGroup
// ---------------------------------------------------------------------------

func TestDeleteSourceGroup_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).
		Return(pgconn.CommandTag{}, nil)

	svc := newSourceGroupService(mockDB)
	ok, err := svc.DeleteSourceGroup(context.Background(), "HOME")

	require.NoError(t, err)
	assert.True(t, ok)
	mockDB.AssertExpectations(t)
}

func TestDeleteSourceGroup_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).
		Return(pgconn.CommandTag{}, errors.New("foreign key violation"))

	svc := newSourceGroupService(mockDB)
	ok, err := svc.DeleteSourceGroup(context.Background(), "HOME")

	require.Error(t, err)
	assert.False(t, ok)
	assert.Contains(t, err.Error(), "delete source group HOME")
	assert.Contains(t, err.Error(), "foreign key violation")
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// ListSourceGroups (dynamic — uses s.store.querier)
// ---------------------------------------------------------------------------

func TestListSourceGroups_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	rows := newServicesMockRows([][]interface{}{
		{"HOME", "Local"},
		{"WORLD", "Global"},
	})

	mockDB.On("Query",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(pgx.Rows(rows), nil)

	svc := newSourceGroupService(mockDB)
	result, err := svc.ListSourceGroups(context.Background(), nil, nil)

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "HOME", result[0].GroupName)
	assert.Equal(t, "Local", result[0].Region)
	mockDB.AssertExpectations(t)
}

func TestListSourceGroups_EmptyResult(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	rows := newServicesMockRows(nil)

	mockDB.On("Query",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(pgx.Rows(rows), nil)

	svc := newSourceGroupService(mockDB)
	result, err := svc.ListSourceGroups(context.Background(), nil, nil)

	require.NoError(t, err)
	assert.Empty(t, result)
	mockDB.AssertExpectations(t)
}

func TestListSourceGroups_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	dbErr := errors.New("connection refused")

	mockDB.On("Query",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(pgx.Rows(newServicesMockRows(nil)), dbErr)

	svc := newSourceGroupService(mockDB)
	result, err := svc.ListSourceGroups(context.Background(), nil, nil)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CountSourceGroups (dynamic — uses s.store.querier)
// ---------------------------------------------------------------------------

func TestCountSourceGroups_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	mockDB.On("QueryRow", mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		*(args[0].(*int64)) = 9
	}).Return(nil)

	svc := newSourceGroupService(mockDB)
	count, err := svc.CountSourceGroups(context.Background(), nil)

	require.NoError(t, err)
	assert.Equal(t, 9, count)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestCountSourceGroups_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}
	dbErr := errors.New("timeout")

	mockDB.On("QueryRow", mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan", mock.Anything).Return(dbErr)

	svc := newSourceGroupService(mockDB)
	count, err := svc.CountSourceGroups(context.Background(), nil)

	require.Error(t, err)
	assert.Equal(t, 0, count)
	assert.Equal(t, dbErr, err)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}
