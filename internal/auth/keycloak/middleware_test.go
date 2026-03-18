package keycloak

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMiddleware_AuthDisabled(t *testing.T) {
	mw := Middleware(nil)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMiddleware_MissingAuthorizationHeader(t *testing.T) {
	dummyClient := &Client{}
	mw := Middleware(dummyClient)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestMiddleware_InvalidAuthorizationHeaderFormat(t *testing.T) {
	dummyClient := &Client{}
	mw := Middleware(dummyClient)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name   string
		header string
	}{
		{"no space", "sometoken"},
		{"wrong scheme", "Basic dXNlcjpwYXNz"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", tc.header)
			rr := httptest.NewRecorder()
			mw(next).ServeHTTP(rr, req)
			assert.Equal(t, http.StatusUnauthorized, rr.Code)
		})
	}
}

func TestClaimsFromContext_Present(t *testing.T) {
	claims := &KeycloakClaims{PreferredUsername: "alice"}
	ctx := context.WithValue(context.Background(), ClaimsContextKey, claims)

	got, ok := ClaimsFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, claims, got)
}

func TestClaimsFromContext_Absent(t *testing.T) {
	claims, ok := ClaimsFromContext(context.Background())
	assert.False(t, ok)
	assert.Nil(t, claims)
}
