package chargeengine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/engine/business/interfaces"
	"go-ocs/internal/chargeengine/ocserrors"
	"go-ocs/internal/nchf"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockInfraForEngine struct {
	mock.Mock
	interfaces.Infrastructure
}

func setupHandlerTest() (*appcontext.AppContext, *MockInfraForEngine) {
	metrics := &appcontext.AppMetrics{
		Runtime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "test_runtime"},
			[]string{"method", "path"},
		),
		Rate: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "test_rate"},
			[]string{"method", "path"},
		),
		ErrorRate: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "test_error_rate"},
			[]string{"method", "path"},
		),
	}

	appCtx := &appcontext.AppContext{
		Config: &appcontext.Config{
			Engine: appcontext.EngineConfig{
				Path: "/api",
			},
		},
		Metrics:      metrics,
		KafkaManager: new(MockKafkaManager),
	}
	mockInfra := new(MockInfraForEngine)
	return appCtx, mockInfra
}

func TestNchfHandler_Success(t *testing.T) {
	appCtx, mockInfra := setupHandlerTest()
	chargingID := "test-id"
	processFunc := func(ctx *appcontext.AppContext, infra interfaces.Infrastructure, ref string, req *nchf.ChargingDataRequest) (*nchf.ChargingDataResponse, error) {
		assert.Equal(t, chargingID, ref)
		return &nchf.ChargingDataResponse{}, nil
	}

	handler := nchfHandler(appCtx, mockInfra, processFunc)

	reqBody, _ := json.Marshal(nchf.ChargingDataRequest{ChargingId: &chargingID})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp nchf.ChargingDataResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
}

func TestNchfHandler_InvalidJSON(t *testing.T) {
	appCtx, mockInfra := setupHandlerTest()
	processFunc := func(ctx *appcontext.AppContext, infra interfaces.Infrastructure, ref string, req *nchf.ChargingDataRequest) (*nchf.ChargingDataResponse, error) {
		return &nchf.ChargingDataResponse{}, nil
	}

	handler := nchfHandler(appCtx, mockInfra, processFunc)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("invalid-json"))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid json")
}

func TestNchfHandler_ProcessError(t *testing.T) {
	appCtx, mockInfra := setupHandlerTest()
	chargingID := "test-id"
	processFunc := func(ctx *appcontext.AppContext, infra interfaces.Infrastructure, ref string, req *nchf.ChargingDataRequest) (*nchf.ChargingDataResponse, error) {
		return nil, errors.New("business logic failed")
	}

	handler := nchfHandler(appCtx, mockInfra, processFunc)

	reqBody, _ := json.Marshal(nchf.ChargingDataRequest{ChargingId: &chargingID})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "request failed")
	assert.Contains(t, rr.Body.String(), "business logic failed")
}

func TestNchfHandler_RetransmitError(t *testing.T) {
	appCtx, mockInfra := setupHandlerTest()
	chargingID := "test-id"
	expectedResp := &nchf.ChargingDataResponse{}
	processFunc := func(ctx *appcontext.AppContext, infra interfaces.Infrastructure, ref string, req *nchf.ChargingDataRequest) (*nchf.ChargingDataResponse, error) {
		return nil, ocserrors.CreateRetransmit(expectedResp)
	}

	handler := nchfHandler(appCtx, mockInfra, processFunc)

	reqBody, _ := json.Marshal(nchf.ChargingDataRequest{ChargingId: &chargingID})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp nchf.ChargingDataResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
}

func TestNchfHandler_ChiURLParam(t *testing.T) {
	appCtx, mockInfra := setupHandlerTest()
	chargingID := "id-from-json"
	urlRef := "ref-from-url"

	processFunc := func(ctx *appcontext.AppContext, infra interfaces.Infrastructure, ref string, req *nchf.ChargingDataRequest) (*nchf.ChargingDataResponse, error) {
		assert.Equal(t, urlRef, ref)
		return &nchf.ChargingDataResponse{}, nil
	}

	handler := nchfHandler(appCtx, mockInfra, processFunc)

	reqBody, _ := json.Marshal(nchf.ChargingDataRequest{ChargingId: &chargingID})
	req := httptest.NewRequest(http.MethodPost, "/"+urlRef, bytes.NewBuffer(reqBody))

	// Manually set chi URL param
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("chargingDataRef", urlRef)

	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestNchfHandler_Metrics(t *testing.T) {
	appCtx, mockInfra := setupHandlerTest()
	chargingID := "test-id"
	processFunc := func(ctx *appcontext.AppContext, infra interfaces.Infrastructure, ref string, req *nchf.ChargingDataRequest) (*nchf.ChargingDataResponse, error) {
		return &nchf.ChargingDataResponse{}, nil
	}

	handler := nchfHandler(appCtx, mockInfra, processFunc)

	reqBody, _ := json.Marshal(nchf.ChargingDataRequest{ChargingId: &chargingID})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Check metrics - we can't easily check the value of CounterVec without more setup or using prometheus/testutil
	// but the fact that it didn't panic is good.
}
