package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
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
// Helpers
// ---------------------------------------------------------------------------

// sampleUUID is a stable UUID used across ChargingTrace tests.
var sampleUUID = uuid.MustParse("01020304-0506-0708-090a-0b0c0d0e0f10")

// samplePgtypeUUID returns the pgtype.UUID equivalent of sampleUUID.
func samplePgtypeUUID() pgtype.UUID {
	return pgtype.UUID{Bytes: sampleUUID, Valid: true}
}

// newChargingTraceService wires a ChargingTraceService backed by a mock DBTX.
// The mock satisfies both sqlc.DBTX (for Q) and store.DBQuerier (for dynamic methods).
func newChargingTraceService(mockDB *servicesMockDBTX) *ChargingTraceService {
	s := store.NewTestStore(mockDB, mockDB)
	return NewChargingTraceService(s)
}

// populateTraceScan fills the 8 Scan destinations for FindChargingTraceByTraceId.
func populateTraceScan(traceID pgtype.UUID) func(mock.Arguments) {
	return func(args mock.Arguments) {
		*(args[0].(*pgtype.UUID)) = traceID
		*(args[1].(*pgtype.Timestamptz)) = pgtype.Timestamptz{
			Time:  time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
			Valid: true,
		}
		*(args[2].(*[]byte)) = []byte(`{"service":"data"}`)
		*(args[3].(*[]byte)) = []byte(`{"granted":1024}`)
		*(args[4].(*int64)) = int64(42)
		*(args[5].(*string)) = "CHG-001"
		*(args[6].(*int32)) = int32(1)
		*(args[7].(*string)) = "27820001001"
	}
}

// ---------------------------------------------------------------------------
// chargingTraceToModel
// ---------------------------------------------------------------------------

func TestChargingTraceToModel_CreatedAtNil(t *testing.T) {
	t.Parallel()
	row := sqlc.ChargingTrace{
		TraceID:       samplePgtypeUUID(),
		CreatedAt:     pgtype.Timestamptz{Valid: false},
		Request:       []byte(`{"foo":"bar"}`),
		Response:      []byte(`{"ok":true}`),
		ExecutionTime: 99,
		ChargingID:    "CHG-X",
		SequenceNr:    2,
		Msisdn:        "64211111111",
	}

	m := chargingTraceToModel(row)

	assert.Equal(t, sampleUUID.String(), m.TraceID)
	assert.Nil(t, m.CreatedAt)
	assert.Equal(t, `{"foo":"bar"}`, m.Request)
	assert.Equal(t, `{"ok":true}`, m.Response)
	assert.Equal(t, 99, m.ExecutionTime)
	assert.Equal(t, "CHG-X", m.ChargingID)
	assert.Equal(t, 2, m.SequenceNr)
	assert.Equal(t, "64211111111", m.Msisdn)
}

func TestChargingTraceToModel_CreatedAtSet(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 3, 20, 12, 30, 0, 0, time.UTC)
	row := sqlc.ChargingTrace{
		TraceID:       samplePgtypeUUID(),
		CreatedAt:     pgtype.Timestamptz{Time: ts, Valid: true},
		Request:       []byte(`{}`),
		Response:      []byte(`{}`),
		ExecutionTime: 5,
		ChargingID:    "CHG-Y",
		SequenceNr:    3,
		Msisdn:        "64299999999",
	}

	m := chargingTraceToModel(row)

	require.NotNil(t, m.CreatedAt)
	assert.Equal(t, ts.Format(time.RFC3339), *m.CreatedAt)
}

// ---------------------------------------------------------------------------
// ListChargingTraces
// ---------------------------------------------------------------------------

func TestListChargingTraces_Success(t *testing.T) {
	t.Parallel()
	mockDB := &servicesMockDBTX{}

	traceID := samplePgtypeUUID()
	ts := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)

	// servicesMockDBTX.Query is called by the dynamic store method.
	// Return a mock pgx.Rows with one row.
	mockRows := &mockChargingTraceRows{
		rows: []sqlc.ChargingTrace{
			{
				TraceID:       traceID,
				CreatedAt:     pgtype.Timestamptz{Time: ts, Valid: true},
				Request:       []byte(`{"a":1}`),
				Response:      []byte(`{"b":2}`),
				ExecutionTime: 10,
				ChargingID:    "CHG-001",
				SequenceNr:    1,
				Msisdn:        "27820001001",
			},
		},
	}
	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(pgx.Rows(mockRows), nil)

	svc := newChargingTraceService(mockDB)
	result, err := svc.ListChargingTraces(context.Background(), nil, nil)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, sampleUUID.String(), result[0].TraceID)
	assert.Equal(t, "CHG-001", result[0].ChargingID)
	assert.Equal(t, "27820001001", result[0].Msisdn)
	assert.Equal(t, 10, result[0].ExecutionTime)
	mockDB.AssertExpectations(t)
}

func TestListChargingTraces_DBError(t *testing.T) {
	t.Parallel()
	mockDB := &servicesMockDBTX{}
	dbErr := errors.New("connection timeout")

	mockDB.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(pgx.Rows(&mockChargingTraceRows{}), dbErr)

	svc := newChargingTraceService(mockDB)
	result, err := svc.ListChargingTraces(context.Background(), nil, nil)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
	mockDB.AssertExpectations(t)
}

func TestListChargingTraces_FilterError(t *testing.T) {
	t.Parallel()
	mockDB := &servicesMockDBTX{}
	svc := newChargingTraceService(mockDB)

	// Supply a filter key that is not in the column allowlist.
	badFilter := &model.FilterRequest{
		Filters: []*model.FilterInput{
			{Key: "unknownField", Operation: "=", Value: "x"},
		},
	}
	result, err := svc.ListChargingTraces(context.Background(), nil, badFilter)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unknownField")
}

// ---------------------------------------------------------------------------
// CountChargingTrace
// ---------------------------------------------------------------------------

func TestCountChargingTrace_Success(t *testing.T) {
	t.Parallel()
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	mockDB.On("QueryRow", mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		*(args[0].(*int64)) = 7
	}).Return(nil)

	svc := newChargingTraceService(mockDB)
	count, err := svc.CountChargingTrace(context.Background(), nil)

	require.NoError(t, err)
	assert.Equal(t, 7, count)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestCountChargingTrace_DBError(t *testing.T) {
	t.Parallel()
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}
	dbErr := errors.New("db unavailable")

	mockDB.On("QueryRow", mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan", mock.Anything).Return(dbErr)

	svc := newChargingTraceService(mockDB)
	count, err := svc.CountChargingTrace(context.Background(), nil)

	require.Error(t, err)
	assert.Equal(t, 0, count)
	assert.Equal(t, dbErr, err)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestCountChargingTrace_FilterError(t *testing.T) {
	t.Parallel()
	mockDB := &servicesMockDBTX{}
	svc := newChargingTraceService(mockDB)

	badFilter := &model.FilterRequest{
		Filters: []*model.FilterInput{
			{Key: "notAColumn", Operation: "=", Value: "1"},
		},
	}
	count, err := svc.CountChargingTrace(context.Background(), badFilter)

	require.Error(t, err)
	assert.Equal(t, 0, count)
}

// ---------------------------------------------------------------------------
// ChargingTraceById
// ---------------------------------------------------------------------------

func TestChargingTraceById_Success(t *testing.T) {
	t.Parallel()
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	// FindChargingTraceByTraceId calls QueryRow with the UUID as the single arg.
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Run(populateTraceScan(samplePgtypeUUID())).Return(nil)

	svc := newChargingTraceService(mockDB)
	trace, err := svc.ChargingTraceById(context.Background(), sampleUUID.String())

	require.NoError(t, err)
	require.NotNil(t, trace)
	assert.Equal(t, sampleUUID.String(), trace.TraceID)
	assert.Equal(t, "CHG-001", trace.ChargingID)
	assert.Equal(t, "27820001001", trace.Msisdn)
	assert.Equal(t, 42, trace.ExecutionTime)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestChargingTraceById_NotFound(t *testing.T) {
	t.Parallel()
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(pgx.Row(mockRow))
	mockRow.On("Scan",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(pgx.ErrNoRows)

	svc := newChargingTraceService(mockDB)
	trace, err := svc.ChargingTraceById(context.Background(), sampleUUID.String())

	require.Error(t, err)
	assert.Nil(t, trace)
	assert.Equal(t, pgx.ErrNoRows, err)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestChargingTraceById_InvalidUUID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		traceId string
	}{
		{name: "empty string", traceId: ""},
		{name: "non-UUID string", traceId: "not-a-uuid"},
		{name: "partial UUID", traceId: "01020304-0506"},
		{name: "UUID with extra characters", traceId: "01020304-0506-0708-090a-0b0c0d0e0f10-extra"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mockDB := &servicesMockDBTX{}
			svc := newChargingTraceService(mockDB)

			trace, err := svc.ChargingTraceById(context.Background(), tc.traceId)

			require.Error(t, err)
			assert.Nil(t, trace)
			assert.Contains(t, err.Error(), "invalid traceId")
			mockDB.AssertExpectations(t) // no DB calls expected
		})
	}
}

// ---------------------------------------------------------------------------
// mockChargingTraceRows — a minimal pgx.Rows implementation for service tests.
// Unlike the store-layer mock, this operates at the scan level using ChargingTrace
// structs so we avoid duplicating the low-level scan logic.
// ---------------------------------------------------------------------------

type mockChargingTraceRows struct {
	rows    []sqlc.ChargingTrace
	current int
	err     error
}

func (m *mockChargingTraceRows) Close() {}

func (m *mockChargingTraceRows) Err() error { return m.err }

func (m *mockChargingTraceRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (m *mockChargingTraceRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (m *mockChargingTraceRows) Next() bool {
	return m.current < len(m.rows)
}

func (m *mockChargingTraceRows) Scan(dest ...any) error {
	if m.current >= len(m.rows) {
		return errors.New("no more rows")
	}
	row := m.rows[m.current]
	m.current++

	*(dest[0].(*pgtype.UUID)) = row.TraceID
	*(dest[1].(*pgtype.Timestamptz)) = row.CreatedAt
	*(dest[2].(*[]byte)) = row.Request
	*(dest[3].(*[]byte)) = row.Response
	*(dest[4].(*int64)) = row.ExecutionTime
	*(dest[5].(*string)) = row.ChargingID
	*(dest[6].(*int32)) = row.SequenceNr
	*(dest[7].(*string)) = row.Msisdn
	return nil
}

func (m *mockChargingTraceRows) Values() ([]any, error) { return nil, nil }

func (m *mockChargingTraceRows) RawValues() [][]byte { return nil }

func (m *mockChargingTraceRows) Conn() *pgx.Conn { return nil }
