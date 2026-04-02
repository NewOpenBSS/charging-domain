package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// SecureRouter wraps a chi.Router and requires every route registration to
// declare the permissions it needs. This ensures that omitting permissions is
// a compile error (the caller must pass the permissions parameter).
//
// Use Public() to obtain a nil permission slice for endpoints that do not
// require authentication (e.g. /health).
type SecureRouter struct {
	router      chi.Router
	authEnabled bool
}

// NewSecureRouter creates a SecureRouter that wraps the given chi.Router.
// When authEnabled is false, permission checks are bypassed.
func NewSecureRouter(r chi.Router, authEnabled bool) *SecureRouter {
	return &SecureRouter{
		router:      r,
		authEnabled: authEnabled,
	}
}

// Router returns the underlying chi.Router for use where the raw router is needed
// (e.g. mounting sub-routers or passing to http.ListenAndServe).
func (sr *SecureRouter) Router() chi.Router {
	return sr.router
}

// Public returns a nil permission slice, signalling that a route does not
// require authentication. Use as: sr.Get("/health", Public(), handler).
func Public() []Permission {
	return nil
}

// Get registers an HTTP GET route with permission enforcement.
func (sr *SecureRouter) Get(pattern string, permissions []Permission, handlerFn http.HandlerFunc) {
	sr.handle(pattern, permissions, handlerFn, sr.router.Get)
}

// Post registers an HTTP POST route with permission enforcement.
func (sr *SecureRouter) Post(pattern string, permissions []Permission, handlerFn http.HandlerFunc) {
	sr.handle(pattern, permissions, handlerFn, sr.router.Post)
}

// Put registers an HTTP PUT route with permission enforcement.
func (sr *SecureRouter) Put(pattern string, permissions []Permission, handlerFn http.HandlerFunc) {
	sr.handle(pattern, permissions, handlerFn, sr.router.Put)
}

// Delete registers an HTTP DELETE route with permission enforcement.
func (sr *SecureRouter) Delete(pattern string, permissions []Permission, handlerFn http.HandlerFunc) {
	sr.handle(pattern, permissions, handlerFn, sr.router.Delete)
}

// Patch registers an HTTP PATCH route with permission enforcement.
func (sr *SecureRouter) Patch(pattern string, permissions []Permission, handlerFn http.HandlerFunc) {
	sr.handle(pattern, permissions, handlerFn, sr.router.Patch)
}

// handle applies permission middleware to a route registration. If permissions
// is nil (Public), no Require middleware is applied.
func (sr *SecureRouter) handle(pattern string, permissions []Permission, handlerFn http.HandlerFunc, register func(string, http.HandlerFunc)) {
	if permissions == nil {
		register(pattern, handlerFn)
		return
	}

	wrapped := Require(sr.authEnabled, permissions...)(handlerFn)
	register(pattern, wrapped.ServeHTTP)
}
