package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	graphqlmodel "go-ocs/internal/backend/graphql/model"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// newNumberPlanService wires a NumberPlanService backed by a mock DBTX.
// servicesMockDBTX is defined in carrier_service_test.go (same package).
func newNumberPlanService(mockDB *servicesMockDBTX) *NumberPlanService {
	return NewNumberPlanService(&store.Store{Q: sqlc.New(mockDB)})
}

// populateNumberPlanScan fills the 6 Scan destinations for a number_plan row.
func populateNumberPlanScan(id int64, name, plmn, numberRange string, numberLength int32) func(mock.Arguments) {
	return func(args mock.Arguments) {
		*(args[0].(*int64)) = id                                                                     // number_id
		*(args[1].(*pgtype.Timestamptz)) = pgtype.Timestamptz{Time: time.Now(), Valid: true}        // modified_on
		*(args[2].(*string)) = name                                                                  // name
		*(args[3].(*string)) = plmn                                                                  // plmn
		*(args[4].(*string)) = numberRange                                                           // number_range
		*(args[5].(*int32)) = numberLength                                                           // number_length
	}
}

const numberPlanScanArgCount = 6

func numberPlanScanMatchers() []interface{} {
	m := make([]interface{}, numberPlanScanArgCount)
	for i := range m {
		m[i] = mock.Anything
	}
	return m
}

// anyQueryRow1NumberPlan registers a 1-arg QueryRow expectation (FindNumberPlanByID).
func anyQueryRow1NumberPlan(mockDB *servicesMockDBTX, row *servicesMockRow) {
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(row)
}

// anyQueryRow4NumberPlan registers a 4-arg QueryRow expectation (CreateNumberPlan).
func anyQueryRow4NumberPlan(mockDB *servicesMockDBTX, row *servicesMockRow) {
	mockDB.On("QueryRow",
		mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(row)
}

// anyQueryRow5NumberPlan registers a 5-arg QueryRow expectation (UpdateNumberPlan).
func anyQueryRow5NumberPlan(mockDB *servicesMockDBTX, row *servicesMockRow) {
	mockDB.On("QueryRow",
		mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(row)
}

// minimalNumberPlanInput returns a valid NumberPlanInput for mutation tests.
func minimalNumberPlanInput() graphqlmodel.NumberPlanInput {
	name := "NZ Numbers"
	return graphqlmodel.NumberPlanInput{
		Name:         &name,
		Plmn:         "53005",
		NumberRange:  "6421",
		NumberLength: 11,
	}
}

// ---------------------------------------------------------------------------
// parseNumberPlanID
// ---------------------------------------------------------------------------

func TestParseNumberPlanID_Valid(t *testing.T) {
	n, err := parseNumberPlanID("42")
	require.NoError(t, err)
	assert.Equal(t, int64(42), n)
}

func TestParseNumberPlanID_Invalid_Error(t *testing.T) {
	_, err := parseNumberPlanID("not-a-number")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid numberId")
}

func TestParseNumberPlanID_Negative(t *testing.T) {
	// Negative values parse fine — the DB constraint would reject them.
	n, err := parseNumberPlanID("-1")
	require.NoError(t, err)
	assert.Equal(t, int64(-1), n)
}

// ---------------------------------------------------------------------------
// numberPlanToModel
// ---------------------------------------------------------------------------

func TestNumberPlanToModel_AllFields(t *testing.T) {
	row := sqlc.NumberPlan{
		NumberID:     7,
		ModifiedOn:   pgtype.Timestamptz{Time: time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC), Valid: true},
		Name:         "NZ Numbers",
		Plmn:         "53005",
		NumberRange:  "6421",
		NumberLength: 11,
	}

	m := numberPlanToModel(row)

	require.NotNil(t, m)
	assert.Equal(t, "7", m.NumberID)
	assert.Equal(t, "NZ Numbers", m.Name)
	assert.Equal(t, "53005", m.Plmn)
	assert.Equal(t, "6421", m.NumberRange)
	assert.Equal(t, 11, m.NumberLength)
	require.NotNil(t, m.ModifiedOn)
	assert.Contains(t, *m.ModifiedOn, "2024-03-01")
}

func TestNumberPlanToModel_ModifiedOnNull(t *testing.T) {
	row := sqlc.NumberPlan{
		NumberID:     1,
		ModifiedOn:   pgtype.Timestamptz{Valid: false},
		Name:         "Test",
		Plmn:         "53005",
		NumberRange:  "64",
		NumberLength: 10,
	}

	m := numberPlanToModel(row)
	assert.Nil(t, m.ModifiedOn)
}

// ---------------------------------------------------------------------------
// GetNumberPlan
// ---------------------------------------------------------------------------

func TestGetNumberPlan_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow1NumberPlan(mockDB, mockRow)
	mockRow.On("Scan", numberPlanScanMatchers()...).
		Run(populateNumberPlanScan(5, "NZ Numbers", "53005", "6421", 11)).
		Return(nil)

	svc := newNumberPlanService(mockDB)
	m, err := svc.GetNumberPlan(context.Background(), "5")

	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, "5", m.NumberID)
	assert.Equal(t, "NZ Numbers", m.Name)
	assert.Equal(t, "53005", m.Plmn)
	mockDB.AssertExpectations(t)
}

func TestGetNumberPlan_InvalidID_Error(t *testing.T) {
	svc := newNumberPlanService(&servicesMockDBTX{})
	_, err := svc.GetNumberPlan(context.Background(), "not-a-number")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid numberId")
}

func TestGetNumberPlan_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow1NumberPlan(mockDB, mockRow)
	mockRow.On("Scan", numberPlanScanMatchers()...).Return(errors.New("connection refused"))

	svc := newNumberPlanService(mockDB)
	_, err := svc.GetNumberPlan(context.Background(), "1")
	require.Error(t, err)
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CreateNumberPlan
// ---------------------------------------------------------------------------

func TestCreateNumberPlan_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow4NumberPlan(mockDB, mockRow)
	mockRow.On("Scan", numberPlanScanMatchers()...).
		Run(populateNumberPlanScan(10, "NZ Numbers", "53005", "6421", 11)).
		Return(nil)

	svc := newNumberPlanService(mockDB)
	m, err := svc.CreateNumberPlan(context.Background(), minimalNumberPlanInput())

	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, "10", m.NumberID)
	assert.Equal(t, "53005", m.Plmn)
	assert.Equal(t, "6421", m.NumberRange)
	assert.Equal(t, 11, m.NumberLength)
	mockDB.AssertExpectations(t)
}

func TestCreateNumberPlan_NilName_DefaultsToEmpty(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow4NumberPlan(mockDB, mockRow)
	mockRow.On("Scan", numberPlanScanMatchers()...).
		Run(populateNumberPlanScan(11, "", "53005", "6421", 11)).
		Return(nil)

	svc := newNumberPlanService(mockDB)
	input := graphqlmodel.NumberPlanInput{
		Name:         nil, // omitted
		Plmn:         "53005",
		NumberRange:  "6421",
		NumberLength: 11,
	}
	m, err := svc.CreateNumberPlan(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "", m.Name)
	mockDB.AssertExpectations(t)
}

func TestCreateNumberPlan_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow4NumberPlan(mockDB, mockRow)
	mockRow.On("Scan", numberPlanScanMatchers()...).Return(errors.New("unique violation"))

	svc := newNumberPlanService(mockDB)
	_, err := svc.CreateNumberPlan(context.Background(), minimalNumberPlanInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create number plan")
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// UpdateNumberPlan
// ---------------------------------------------------------------------------

func TestUpdateNumberPlan_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow5NumberPlan(mockDB, mockRow)
	mockRow.On("Scan", numberPlanScanMatchers()...).
		Run(populateNumberPlanScan(3, "Updated Name", "53005", "6422", 12)).
		Return(nil)

	svc := newNumberPlanService(mockDB)
	name := "Updated Name"
	input := graphqlmodel.NumberPlanInput{
		Name:         &name,
		Plmn:         "53005",
		NumberRange:  "6422",
		NumberLength: 12,
	}
	m, err := svc.UpdateNumberPlan(context.Background(), "3", input)

	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, "3", m.NumberID)
	assert.Equal(t, "Updated Name", m.Name)
	assert.Equal(t, "6422", m.NumberRange)
	mockDB.AssertExpectations(t)
}

func TestUpdateNumberPlan_InvalidID_Error(t *testing.T) {
	svc := newNumberPlanService(&servicesMockDBTX{})
	_, err := svc.UpdateNumberPlan(context.Background(), "bad", minimalNumberPlanInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid numberId")
}

func TestUpdateNumberPlan_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow5NumberPlan(mockDB, mockRow)
	mockRow.On("Scan", numberPlanScanMatchers()...).Return(errors.New("not found"))

	svc := newNumberPlanService(mockDB)
	_, err := svc.UpdateNumberPlan(context.Background(), "99", minimalNumberPlanInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update number plan")
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// DeleteNumberPlan
// ---------------------------------------------------------------------------

func TestDeleteNumberPlan_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).
		Return(pgconn.CommandTag{}, nil)

	svc := newNumberPlanService(mockDB)
	ok, err := svc.DeleteNumberPlan(context.Background(), "1")

	require.NoError(t, err)
	assert.True(t, ok)
	mockDB.AssertExpectations(t)
}

func TestDeleteNumberPlan_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).
		Return(pgconn.CommandTag{}, errors.New("foreign key violation"))

	svc := newNumberPlanService(mockDB)
	ok, err := svc.DeleteNumberPlan(context.Background(), "1")

	require.Error(t, err)
	assert.False(t, ok)
	assert.Contains(t, err.Error(), "delete number plan")
	mockDB.AssertExpectations(t)
}

func TestDeleteNumberPlan_InvalidID_Error(t *testing.T) {
	svc := newNumberPlanService(&servicesMockDBTX{})
	_, err := svc.DeleteNumberPlan(context.Background(), "abc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid numberId")
}
