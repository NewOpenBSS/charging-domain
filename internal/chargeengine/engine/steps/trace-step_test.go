package steps

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/model"
	"go-ocs/internal/chargeengine/ocserrors"
	"go-ocs/internal/nchf"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// buildTraceDC constructs a ChargingContext wired to a mock DB for CreateTrace tests.
func buildTraceDC(chargingID string, seqNr int64, msisdn string, mockDB *stepsMockDBTX) *engine.ChargingContext {
	req := nchf.NewChargingDataRequest()
	req.ChargingId = &chargingID
	req.InvocationSequenceNumber = &seqNr

	cd := model.NewChargingData()
	cd.Subscriber = &model.Subscriber{Msisdn: msisdn}

	appCtx := &appcontext.AppContext{
		Config: &appcontext.Config{},
		Store:  &store.Store{Q: sqlc.New(mockDB)},
	}

	resp := nchf.NewChargingDataResponse()

	return &engine.ChargingContext{
		StartTime:    time.Now().Add(-10 * time.Millisecond), // ensure non-zero execution time
		AppContext:   appCtx,
		Request:      req,
		Response:     resp,
		ChargingData: cd,
	}
}

func TestCreateTrace_Success(t *testing.T) {
	mockDB := &stepsMockDBTX{}
	mockRow := &stepsMockRow{}

	// CreateChargingTrace returns trace_id via QueryRow/Scan
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)
	mockRow.On("Scan", mock.Anything).Return(nil)

	dc := buildTraceDC("cid-trace", 1, "0211234567", mockDB)

	err := CreateTrace(dc)

	require.NoError(t, err)
	// Runtime should be set on the response
	assert.NotNil(t, dc.Response.Runtime)
	assert.GreaterOrEqual(t, *dc.Response.Runtime, int64(0))
	mockDB.AssertExpectations(t)
}

func TestCreateTrace_DBError(t *testing.T) {
	mockDB := &stepsMockDBTX{}
	mockRow := &stepsMockRow{}

	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)
	mockRow.On("Scan", mock.Anything).Return(errors.New("db write failed"))

	dc := buildTraceDC("cid-trace-err", 1, "0211234567", mockDB)

	err := CreateTrace(dc)

	require.Error(t, err)
	var ocsErr *ocserrors.OcsError
	require.True(t, errors.As(err, &ocsErr))
	assert.Equal(t, ocserrors.CodeGeneralError, ocsErr.Code)
	mockDB.AssertExpectations(t)
}

func TestCreateTrace_RuntimeRecorded(t *testing.T) {
	mockDB := &stepsMockDBTX{}
	mockRow := &stepsMockRow{}

	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)
	mockRow.On("Scan", mock.Anything).Return(nil)

	// Start 50ms in the past to guarantee a positive execution time.
	dc := buildTraceDC("cid-trace-rt", 2, "0211234567", mockDB)
	dc.StartTime = time.Now().Add(-50 * time.Millisecond)

	err := CreateTrace(dc)

	require.NoError(t, err)
	require.NotNil(t, dc.Response.Runtime)
	assert.Greater(t, *dc.Response.Runtime, int64(0))
}
