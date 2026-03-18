package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go-ocs/internal/backend/graphql/model"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// ---------------------------------------------------------------------------
// Mock DB plumbing (mirrors stepsMockDBTX in the steps package)
// ---------------------------------------------------------------------------

type servicesMockDBTX struct {
	mock.Mock
}

func (m *servicesMockDBTX) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	callArgs := []interface{}{ctx, sql}
	callArgs = append(callArgs, args...)
	ret := m.Called(callArgs...)
	return ret.Get(0).(pgconn.CommandTag), ret.Error(1)
}

func (m *servicesMockDBTX) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	callArgs := []interface{}{ctx, sql}
	callArgs = append(callArgs, args...)
	ret := m.Called(callArgs...)
	return ret.Get(0).(pgx.Rows), ret.Error(1)
}

func (m *servicesMockDBTX) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	callArgs := []interface{}{ctx, sql}
	callArgs = append(callArgs, args...)
	return m.Called(callArgs...).Get(0).(pgx.Row)
}

type servicesMockRow struct {
	mock.Mock
}

func (m *servicesMockRow) Scan(dest ...interface{}) error {
	return m.Called(dest...).Error(0)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func strPtr(s string) *string { return &s }

// newCarrierService wires a CarrierService backed by a mock DBTX.
func newCarrierService(mockDB *servicesMockDBTX) *CarrierService {
	return NewCarrierService(&store.Store{Q: sqlc.New(mockDB)})
}

// populateCarrierScan fills the 9 Scan destinations with a deterministic carrier.
func populateCarrierScan(plmn string) func(mock.Arguments) {
	return func(args mock.Arguments) {
		*(args[0].(*string)) = plmn                                              // Plmn
		*(args[1].(*pgtype.Timestamptz)) = pgtype.Timestamptz{Valid: false}      // ModifiedOn
		*(args[2].(*string)) = "530"                                             // Mcc
		*(args[3].(*pgtype.Text)) = pgtype.Text{String: "01", Valid: true}       // Mnc
		*(args[4].(*string)) = "Test Carrier"                                    // CarrierName
		*(args[5].(*string)) = "Home"                                            // SourceGroup
		*(args[6].(*string)) = "NZ"                                              // DestinationGroup
		*(args[7].(*string)) = "New Zealand"                                     // CountryName
		*(args[8].(*string)) = "NZ"                                              // Iso
	}
}

// ---------------------------------------------------------------------------
// carrierToModel
// ---------------------------------------------------------------------------

func TestCarrierToModel_MncNil(t *testing.T) {
	c := sqlc.Carrier{
		Plmn:             "53001",
		Mcc:              "530",
		Mnc:              pgtype.Text{Valid: false},
		CarrierName:      "Spark",
		SourceGroup:      "Home",
		DestinationGroup: "NZ",
		CountryName:      "New Zealand",
		Iso:              "NZ",
		ModifiedOn:       pgtype.Timestamptz{Valid: false},
	}

	m := carrierToModel(c)

	assert.Equal(t, "53001", m.Plmn)
	assert.Equal(t, "530", m.Mcc)
	assert.Nil(t, m.Mnc)
	assert.Nil(t, m.ModifiedOn)
	assert.Equal(t, "Spark", m.CarrierName)
	assert.Equal(t, "NZ", m.Iso)
}

func TestCarrierToModel_MncAndModifiedOnSet(t *testing.T) {
	ts := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	c := sqlc.Carrier{
		Plmn:             "53002",
		Mcc:              "530",
		Mnc:              pgtype.Text{String: "02", Valid: true},
		CarrierName:      "One NZ",
		SourceGroup:      "Home",
		DestinationGroup: "NZ",
		CountryName:      "New Zealand",
		Iso:              "NZ",
		ModifiedOn:       pgtype.Timestamptz{Time: ts, Valid: true},
	}

	m := carrierToModel(c)

	require.NotNil(t, m.Mnc)
	assert.Equal(t, "02", *m.Mnc)
	require.NotNil(t, m.ModifiedOn)
	assert.Equal(t, ts.Format(time.RFC3339), *m.ModifiedOn)
}

// ---------------------------------------------------------------------------
// textFromPtr
// ---------------------------------------------------------------------------

func TestTextFromPtr_Nil(t *testing.T) {
	result := textFromPtr(nil)
	assert.False(t, result.Valid)
	assert.Empty(t, result.String)
}

func TestTextFromPtr_NonNil(t *testing.T) {
	s := "hello"
	result := textFromPtr(&s)
	assert.True(t, result.Valid)
	assert.Equal(t, "hello", result.String)
}

// anyQueryRow registers a QueryRow expectation that matches any number of SQL parameters.
// testify expands variadic args individually, so we need one mock.Anything per parameter.
// CarrierByPlmn passes 1 SQL arg; CreateCarrier and UpdateCarrier pass 8 SQL args.
func anyQueryRow1(mockDB *servicesMockDBTX, row *servicesMockRow) {
	// ctx, sql, 1 SQL param
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(row)
}

func anyQueryRow8(mockDB *servicesMockDBTX, row *servicesMockRow) {
	// ctx, sql, 8 SQL params
	mockDB.On("QueryRow",
		mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(row)
}

// ---------------------------------------------------------------------------
// CarrierByPlmn
// ---------------------------------------------------------------------------

func TestCarrierByPlmn_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow1(mockDB, mockRow)
	mockRow.On("Scan",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Run(populateCarrierScan("53001")).Return(nil)

	svc := newCarrierService(mockDB)
	carrier, err := svc.CarrierByPlmn(context.Background(), "53001")

	require.NoError(t, err)
	require.NotNil(t, carrier)
	assert.Equal(t, "53001", carrier.Plmn)
	assert.Equal(t, "Test Carrier", carrier.CarrierName)
	mockDB.AssertExpectations(t)
}

func TestCarrierByPlmn_NotFound(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow1(mockDB, mockRow)
	mockRow.On("Scan",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(pgx.ErrNoRows)

	svc := newCarrierService(mockDB)
	_, err := svc.CarrierByPlmn(context.Background(), "99999")

	require.Error(t, err)
	assert.Equal(t, pgx.ErrNoRows, err)
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CreateCarrier
// ---------------------------------------------------------------------------

func TestCreateCarrier_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow8(mockDB, mockRow)
	mockRow.On("Scan",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Run(populateCarrierScan("53003")).Return(nil)

	svc := newCarrierService(mockDB)
	input := model.CarrierInput{
		Plmn:             "53003",
		Mcc:              "530",
		Mnc:              strPtr("03"),
		CarrierName:      "Test Carrier",
		SourceGroup:      "Home",
		DestinationGroup: "NZ",
		CountryName:      "New Zealand",
		Iso:              "NZ",
	}

	carrier, err := svc.CreateCarrier(context.Background(), input)

	require.NoError(t, err)
	require.NotNil(t, carrier)
	assert.Equal(t, "53003", carrier.Plmn)
	mockDB.AssertExpectations(t)
}

func TestCreateCarrier_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow8(mockDB, mockRow)
	mockRow.On("Scan",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(errors.New("unique constraint violation"))

	svc := newCarrierService(mockDB)
	input := model.CarrierInput{Plmn: "53003", Mcc: "530", CarrierName: "Dup", Iso: "NZ"}

	_, err := svc.CreateCarrier(context.Background(), input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unique constraint violation")
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// UpdateCarrier
// ---------------------------------------------------------------------------

func TestUpdateCarrier_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow8(mockDB, mockRow)
	mockRow.On("Scan",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Run(populateCarrierScan("53004")).Return(nil)

	svc := newCarrierService(mockDB)
	input := model.CarrierInput{
		Plmn:             "53004",
		Mcc:              "530",
		CarrierName:      "Updated Carrier",
		SourceGroup:      "Home",
		DestinationGroup: "NZ",
		CountryName:      "New Zealand",
		Iso:              "NZ",
	}

	carrier, err := svc.UpdateCarrier(context.Background(), "53004", input)

	require.NoError(t, err)
	require.NotNil(t, carrier)
	assert.Equal(t, "53004", carrier.Plmn)
	mockDB.AssertExpectations(t)
}

func TestUpdateCarrier_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow8(mockDB, mockRow)
	mockRow.On("Scan",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(errors.New("record not found"))

	svc := newCarrierService(mockDB)
	input := model.CarrierInput{Plmn: "99999", Mcc: "999", CarrierName: "Ghost", Iso: "XX"}

	_, err := svc.UpdateCarrier(context.Background(), "99999", input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "update carrier 99999")
	assert.Contains(t, err.Error(), "record not found")
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// DeleteCarrier
// ---------------------------------------------------------------------------

func TestDeleteCarrier_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, nil)

	svc := newCarrierService(mockDB)
	ok, err := svc.DeleteCarrier(context.Background(), "53001")

	require.NoError(t, err)
	assert.True(t, ok)
	mockDB.AssertExpectations(t)
}

func TestDeleteCarrier_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, errors.New("foreign key violation"))

	svc := newCarrierService(mockDB)
	ok, err := svc.DeleteCarrier(context.Background(), "53001")

	require.Error(t, err)
	assert.False(t, ok)
	assert.Contains(t, err.Error(), "delete carrier 53001")
	assert.Contains(t, err.Error(), "foreign key violation")
	mockDB.AssertExpectations(t)
}
