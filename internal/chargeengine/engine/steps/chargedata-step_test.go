package steps

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/model"
	"go-ocs/internal/chargeengine/ocserrors"
	"go-ocs/internal/nchf"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// buildChargeDataDC constructs a ChargingContext wired to a mock DB for chargedata-step tests.
func buildChargeDataDC(chargingID string, seqNr int64, mockDB *stepsMockDBTX) *engine.ChargingContext {
	req := nchf.NewChargingDataRequest()
	req.ChargingId = &chargingID
	req.InvocationSequenceNumber = &seqNr

	appCtx := &appcontext.AppContext{
		Config: &appcontext.Config{},
		Store:  &store.Store{Q: sqlc.New(mockDB)},
	}

	return &engine.ChargingContext{
		StartTime:    time.Now(),
		AppContext:   appCtx,
		Request:      req,
		Response:     nchf.NewChargingDataResponse(),
		ChargingData: model.NewChargingData(),
	}
}

// --- CreateChargeDataStep ---

func TestCreateChargeDataStep_NewSession(t *testing.T) {
	mockDB := &stepsMockDBTX{}
	mockRow := &stepsMockRow{}

	// GetChargingDataByChargeId returns no rows → session does not exist yet
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(pgx.ErrNoRows)

	dc := buildChargeDataDC("cid-001", 0, mockDB)

	err := CreateChargeDataStep(dc)

	require.NoError(t, err)
	assert.NotNil(t, dc.ChargingData)
	mockDB.AssertExpectations(t)
}

func TestCreateChargeDataStep_SessionAlreadyExists(t *testing.T) {
	mockDB := &stepsMockDBTX{}
	mockRow := &stepsMockRow{}

	// GetChargingDataByChargeId succeeds → session already exists
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			*(args[0].(*string)) = "cid-001"
			*(args[1].(*int64)) = 0
		}).Return(nil)

	dc := buildChargeDataDC("cid-001", 0, mockDB)

	err := CreateChargeDataStep(dc)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ChargingData already exists")
}

// --- LoadChargeDataStep ---

func TestLoadChargeDataStep_Success(t *testing.T) {
	chargingID := "cid-load"
	seqNr := int64(1)

	existingCD := model.NewChargingData()
	existingCD.NewRecord = false
	cdBytes, _ := json.Marshal(existingCD)

	mockDB := &stepsMockDBTX{}
	mockRow := &stepsMockRow{}

	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			*(args[0].(*string)) = chargingID
			*(args[1].(*int64)) = 0 // previous sequence number
			*(args[3].(*[]byte)) = cdBytes
		}).Return(nil)

	dc := buildChargeDataDC(chargingID, seqNr, mockDB)

	err := LoadChargeDataStep(dc)

	require.NoError(t, err)
	assert.NotNil(t, dc.ChargingData)
	assert.False(t, dc.ChargingData.NewRecord)
	mockDB.AssertExpectations(t)
}

func TestLoadChargeDataStep_DBError(t *testing.T) {
	mockDB := &stepsMockDBTX{}
	mockRow := &stepsMockRow{}

	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db error"))

	dc := buildChargeDataDC("cid-load", 1, mockDB)

	err := LoadChargeDataStep(dc)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to load ChargingData")
}

func TestLoadChargeDataStep_DuplicateInvocation(t *testing.T) {
	chargingID := "cid-dup"
	seqNr := int64(1) // same as existing sequence → duplicate

	mockDB := &stepsMockDBTX{}
	mockRow := &stepsMockRow{}

	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)
	mockRow.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			*(args[0].(*string)) = chargingID
			*(args[1].(*int64)) = seqNr // existing sequence == requested → duplicate
		}).Return(nil)

	dc := buildChargeDataDC(chargingID, seqNr, mockDB)

	err := LoadChargeDataStep(dc)

	require.Error(t, err)
	var ocsErr *ocserrors.OcsError
	require.True(t, errors.As(err, &ocsErr))
	assert.Equal(t, ocserrors.CodeInvalidReference, ocsErr.Code)
}

// --- SaveChargeDataStep ---

func TestSaveChargeDataStep_NewRecord(t *testing.T) {
	mockDB := &stepsMockDBTX{}
	// CreateChargeData uses Exec
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, nil)

	dc := buildChargeDataDC("cid-save", 1, mockDB)
	dc.ChargingData.NewRecord = true

	err := SaveChargeDataStep(dc)

	require.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestSaveChargeDataStep_ExistingRecord(t *testing.T) {
	mockDB := &stepsMockDBTX{}
	// UpdateChargeData uses Exec
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, nil)

	dc := buildChargeDataDC("cid-update", 2, mockDB)
	dc.ChargingData.NewRecord = false

	err := SaveChargeDataStep(dc)

	require.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestSaveChargeDataStep_DBError(t *testing.T) {
	mockDB := &stepsMockDBTX{}
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, errors.New("write failed"))

	dc := buildChargeDataDC("cid-save-err", 1, mockDB)
	dc.ChargingData.NewRecord = true

	err := SaveChargeDataStep(dc)

	require.Error(t, err)
	var ocsErr *ocserrors.OcsError
	require.True(t, errors.As(err, &ocsErr))
	assert.Equal(t, ocserrors.CodeGeneralError, ocsErr.Code)
}

// --- ReleaseChargeDataStep ---

func TestReleaseChargeDataStep_Success(t *testing.T) {
	mockDB := &stepsMockDBTX{}
	// DeleteChargeDate uses Exec
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, nil)

	dc := buildChargeDataDC("cid-release", 3, mockDB)

	err := ReleaseChargeDataStep(dc)

	require.NoError(t, err)
	mockDB.AssertExpectations(t)
}

func TestReleaseChargeDataStep_DBError(t *testing.T) {
	mockDB := &stepsMockDBTX{}
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, errors.New("delete failed"))

	dc := buildChargeDataDC("cid-release-err", 3, mockDB)

	err := ReleaseChargeDataStep(dc)

	require.Error(t, err)
	var ocsErr *ocserrors.OcsError
	require.True(t, errors.As(err, &ocsErr))
	assert.Equal(t, ocserrors.CodeGeneralError, ocsErr.Code)
}
