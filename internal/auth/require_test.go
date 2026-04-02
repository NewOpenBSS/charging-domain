package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go-ocs/internal/auth/keycloak"
)

// injectClaims returns a request with KeycloakClaims stored in context.
func injectClaims(r *http.Request, claims *keycloak.KeycloakClaims) *http.Request {
	ctx := context.WithValue(r.Context(), keycloak.ClaimsContextKey, claims)
	return r.WithContext(ctx)
}

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func TestRequire_Authorised(t *testing.T) {
	middleware := Require(true, "read", "write")
	handler := middleware(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = injectClaims(req, &keycloak.KeycloakClaims{
		Permissions: []string{"read"},
	})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequire_Unauthorised_MissingPermission(t *testing.T) {
	middleware := Require(true, "admin")
	handler := middleware(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = injectClaims(req, &keycloak.KeycloakClaims{
		Permissions: []string{"read", "write"},
	})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequire_Unauthenticated_NoClaims(t *testing.T) {
	middleware := Require(true, "read")
	handler := middleware(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequire_AuthDisabled_Bypass(t *testing.T) {
	middleware := Require(false, "admin")
	handler := middleware(http.HandlerFunc(okHandler))

	// No claims in context — should still pass when auth is disabled.
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequire_MultiplePermissions_AnyMatch(t *testing.T) {
	middleware := Require(true, "admin", "superadmin")
	handler := middleware(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = injectClaims(req, &keycloak.KeycloakClaims{
		Permissions: []string{"superadmin"},
	})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequire_EmptyPermissionsList(t *testing.T) {
	// No permissions required — but claims must be present when auth enabled.
	middleware := Require(true)
	handler := middleware(http.HandlerFunc(okHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = injectClaims(req, &keycloak.KeycloakClaims{
		Permissions: []string{"read"},
	})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	// No permissions to match against — should be forbidden.
	assert.Equal(t, http.StatusForbidden, rr.Code)
}
