package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go-ocs/internal/auth/keycloak"
)

func TestSecureRouter_Get_WithPermissions(t *testing.T) {
	r := chi.NewRouter()
	sr := NewSecureRouter(r, true)

	sr.Get("/api/resource", []Permission{"read"}, okHandler)

	// Authorised request.
	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	ctx := context.WithValue(req.Context(), keycloak.ClaimsContextKey, &keycloak.KeycloakClaims{
		Permissions: []string{"read"},
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Unauthorised request.
	req2 := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	ctx2 := context.WithValue(req2.Context(), keycloak.ClaimsContextKey, &keycloak.KeycloakClaims{
		Permissions: []string{"write"},
	})
	req2 = req2.WithContext(ctx2)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusForbidden, rr2.Code)
}

func TestSecureRouter_Public_SkipsAuth(t *testing.T) {
	r := chi.NewRouter()
	sr := NewSecureRouter(r, true)

	sr.Get("/health", Public(), okHandler)

	// No claims — should still pass because it is public.
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestSecureRouter_AuthDisabled_AllPassThrough(t *testing.T) {
	r := chi.NewRouter()
	sr := NewSecureRouter(r, false)

	sr.Get("/api/resource", []Permission{"admin"}, okHandler)

	// No claims, auth disabled — should pass.
	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestSecureRouter_Post_WithPermissions(t *testing.T) {
	r := chi.NewRouter()
	sr := NewSecureRouter(r, true)

	sr.Post("/api/resource", []Permission{"write"}, okHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/resource", nil)
	ctx := context.WithValue(req.Context(), keycloak.ClaimsContextKey, &keycloak.KeycloakClaims{
		Permissions: []string{"write"},
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestSecureRouter_Unauthenticated_Returns401(t *testing.T) {
	r := chi.NewRouter()
	sr := NewSecureRouter(r, true)

	sr.Get("/api/resource", []Permission{"read"}, okHandler)

	// No claims in context.
	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestSecureRouter_Router_ReturnsUnderlying(t *testing.T) {
	r := chi.NewRouter()
	sr := NewSecureRouter(r, true)
	assert.Equal(t, r, sr.Router())
}
