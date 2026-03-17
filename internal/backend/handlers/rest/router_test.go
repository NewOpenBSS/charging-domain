package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-ocs/internal/backend/appcontext"

	"github.com/stretchr/testify/assert"
)

func TestNewRouter_HealthEndpoint(t *testing.T) {
	cfg := &appcontext.BackendConfig{
		Server: appcontext.ServerConfig{
			RestPath: "/api/charging",
		},
	}
	appCtx := &appcontext.AppContext{
		Config: cfg,
		Auth:   nil, // auth disabled
	}

	handler := NewRouter(appCtx)

	req := httptest.NewRequest(http.MethodGet, "/api/charging/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"status":"ok"`)
}
