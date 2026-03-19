# Charging Backend - Application Design

## 1. Overview

The `charging-backend` application provides REST and GraphQL **administration endpoints**
for the charging domain. It is a sibling application to `charging-engine` and follows
the same structural patterns established in that application.

### Key Characteristics

- **REST endpoints** mounted at `/api/charging`
- **GraphQL endpoint** mounted at `/api/charging/graphql`
- **OAuth2-secured** via Keycloak using `github.com/Nerzal/gocloak/v13`
- Business logic lives under `internal/backend`
- Shared infrastructure: `internal/auth`, `internal/store`, `internal/logging`, `internal/baseconfig`, `internal/appl`

---

## 2. Directory Structure

```
go-ocs/
├── cmd/
│   ├── charging-engine/                   # Existing
│   ├── charging-dra/                      # Existing
│   └── charging-backend/                  # NEW
│       ├── main.go                        # Application entry point
│       └── backend-config.yaml           # Configuration file (dev)
│
├── internal/
│   ├── baseconfig/                        # Existing - shared config loading
│   ├── appl/                              # Existing - shared app utilities
│   ├── logging/                           # Existing - shared structured logging
│   │
│   ├── auth/                              # NEW - Shared OAuth2/Keycloak layer
│   │   │                                  # (reusable by ALL applications)
│   │   ├── config/
│   │   │   └── oauth2_config.go           # KeycloakConfig struct
│   │   │
│   │   ├── interfaces/
│   │   │   ├── authenticator.go           # Authenticator interface
│   │   │   └── authorizer.go             # Authorizer interface
│   │   │
│   │   └── keycloak/
│   │       ├── client.go                  # gocloak client initialisation
│   │       ├── claims.go                  # KeycloakClaims struct & extraction
│   │       ├── middleware.go             # Chi-compatible auth middleware
│   │       └── user_service.go           # Fetch user/role attributes via admin API
│   │
│   ├── backend/                           # NEW - Charging Backend business logic
│   │   ├── appcontext/
│   │   │   ├── config.go                  # BackendConfig (Server + Auth + Base)
│   │   │   ├── context.go                 # AppContext struct (DI container)
│   │   │   └── metrics.go                 # Prometheus metrics definitions
│   │   │
│   │   ├── handlers/
│   │   │   ├── rest/
│   │   │   │   └── router.go              # Chi router setup for /api/charging
│   │   │   └── graphql/
│   │   │       └── router.go              # GraphQL handler on /api/charging/graphql
│   │   │
│   │   ├── services/                      # Business logic (to be implemented)
│   │   │   └── .gitkeep
│   │   │
│   │   └── resolvers/                     # GraphQL resolvers (to be implemented)
│   │       └── .gitkeep
│   │
│   ├── chargeengine/                      # Existing
│   ├── quota/                             # Existing
│   ├── store/                             # Existing
│   └── events/                            # Existing
│
├── gql/                                   # NEW - GraphQL schema (gqlgen)
│   ├── schema/
│   │   ├── schema.graphql                 # Root schema
│   │   └── charging.graphql               # Charging domain types
│   └── gqlgen.yml                         # gqlgen code generation config
│
└── db/                                    # Existing (shared)
```

---

## 3. Configuration Design

### 3.1 Shared Auth Config

```go
// internal/auth/config/oauth2_config.go

package config

import "time"

type KeycloakConfig struct {
	Enabled   bool   `yaml:"enabled"`   // toggle auth on/off (false for dev/test)
	IssuerURL string `yaml:"issuerUrl"` // Keycloak realm URL
	// e.g. https://keycloak.example.com/realms/charging
	ClientID      string         `yaml:"clientId"`      // OAuth2 client ID registered in Keycloak
	ClientSecret  string         `yaml:"clientSecret"`  // Client secret (confidential clients)
	Audience      string         `yaml:"audience"`      // Expected 'aud' claim value in JWT
	SkipTLSVerify bool           `yaml:"skipTLSVerify"` // true only for local dev with self-signed certs
	JWKSExpiry    *time.Duration `yaml:"jwksExpiry"`    // optional: JWKS key cache TTL (default: 1h)
}

func NewKeycloakConfig() KeycloakConfig {
	return KeycloakConfig{
		Enabled: true,
	}
}
```

### 3.2 Backend Application Config

```go
// internal/backend/appcontext/config.go

package appcontext

import (
	"time"
	"go-ocs/internal/baseconfig"
	authconfig "go-ocs/internal/auth/config"
	"go-ocs/internal/logging"
)

type BackendConfig struct {
	Base   baseconfig.BaseConfig     `yaml:"base"`   // Metrics, Database, Logging
	Server ServerConfig              `yaml:"server"` // HTTP listener and paths
	Auth   authconfig.KeycloakConfig `yaml:"auth"`   // OAuth2 / Keycloak settings
}

type ServerConfig struct {
	Addr         string        `yaml:"addr"`         // default: ":8081"
	RestPath     string        `yaml:"restPath"`     // default: "/api/charging"
	GraphqlPath  string        `yaml:"graphqlPath"`  // default: "/api/charging/graphql"
	ReadTimeout  time.Duration `yaml:"readTimeout"`  // default: 15s
	WriteTimeout time.Duration `yaml:"writeTimeout"` // default: 15s
}

func NewConfig(configFilename string) *BackendConfig {
	cfg := &BackendConfig{
		Server: ServerConfig{
			Addr:         ":8081",
			RestPath:     "/api/charging",
			GraphqlPath:  "/api/charging/graphql",
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
		Auth: authconfig.NewKeycloakConfig(),
	}

	if err := baseconfig.LoadConfig(configFilename, cfg); err != nil {
		logging.Fatal("Failed to load backend config", "err", err)
	}

	if cfg.Auth.Enabled {
		if cfg.Auth.IssuerURL == "" {
			logging.Fatal("auth.issuerUrl is required when auth is enabled")
		}
		if cfg.Auth.ClientID == "" {
			logging.Fatal("auth.clientId is required when auth is enabled")
		}
	}

	return cfg
}
```

### 3.3 Application Context

```go
// internal/backend/appcontext/context.go

package appcontext

import (
	"go-ocs/internal/auth/keycloak"
	"go-ocs/internal/store"
)

type AppContext struct {
	Config  *BackendConfig
	Metrics *AppMetrics
	Store   *store.Store
	Auth    *keycloak.Client // nil when auth.enabled = false
}
```

### 3.4 Example Configuration File

```yaml
# cmd/charging-backend/backend-config.yaml

base:
  appName: "charging-backend"

  metrics:
    enabled: true
    addr: ":9091"
    path: "/metrics"

  logging:
    level: info
    format: json

  database:
    url: "postgres://gobss:gobss@localhost:5432/gobss?sslmode=disable&search_path=charging"

server:
  addr: ":8081"
  restPath: "/api/charging"
  graphqlPath: "/api/charging/graphql"
  readTimeout: "15s"
  writeTimeout: "15s"

auth:
  enabled: true
  issuerUrl: "https://keycloak.example.com/realms/charging-realm"
  clientId: "charging-backend-client"
  audience: "charging-api"
  skipTLSVerify: false
  # jwksExpiry: "1h"   # optional, gocloak handles caching internally
```

---

## 4. Authentication Design (internal/auth)

### 4.1 Library: `github.com/Nerzal/gocloak/v13`

**Selected because:**

- Full Keycloak REST API client (introspection, user management)
- OIDC provider discovery with automatic JWKS resolution
- Built-in key rotation and caching
- Extracts Keycloak-specific claims (`realm_access`, `resource_access`, custom attributes)
- Actively maintained

### 4.2 Keycloak Claims Structure

Keycloak JWTs contain both standard OIDC claims and Keycloak-specific role/attribute claims:

```go
// internal/auth/keycloak/claims.go

package keycloak

import "github.com/golang-jwt/jwt/v5"

// KeycloakClaims represents the full set of claims extracted from a Keycloak JWT.
// Standard JWT claims are embedded; Keycloak-specific claims are mapped explicitly.
type KeycloakClaims struct {
	jwt.RegisteredClaims

	// Roles assigned at the Keycloak realm level
	RealmAccess RealmAccess `json:"realm_access"`

	// Roles assigned per OAuth2 client/resource
	ResourceAccess map[string]ResourceAccess `json:"resource_access"`

	// Standard OIDC user info
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`

	// Custom user/group attributes (populated via Keycloak token mappers)
	Groups []string `json:"groups"`
}

type RealmAccess struct {
	Roles []string `json:"roles"`
}

type ResourceAccess struct {
	Roles []string `json:"roles"`
}

// HasRealmRole checks whether the token contains the given realm-level role.
func (c *KeycloakClaims) HasRealmRole(role string) bool {
	for _, r := range c.RealmAccess.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasClientRole checks whether the token contains the given role for a specific client.
func (c *KeycloakClaims) HasClientRole(clientID, role string) bool {
	access, ok := c.ResourceAccess[clientID]
	if !ok {
		return false
	}
	for _, r := range access.Roles {
		if r == role {
			return true
		}
	}
	return false
}
```

### 4.3 Interfaces

```go
// internal/auth/interfaces/authenticator.go

package interfaces

import (
	"context"
	"go-ocs/internal/auth/keycloak"
)

// Authenticator validates a raw Bearer token and returns the extracted claims.
type Authenticator interface {
	ValidateToken(ctx context.Context, rawToken string) (*keycloak.KeycloakClaims, error)
}
```

```go
// internal/auth/interfaces/authorizer.go

package interfaces

import (
	"context"
	"go-ocs/internal/auth/keycloak"
)

// Authorizer performs role-based access checks against extracted claims.
type Authorizer interface {
	RequireRealmRole(ctx context.Context, claims *keycloak.KeycloakClaims, role string) error
	RequireClientRole(ctx context.Context, claims *keycloak.KeycloakClaims, clientID, role string) error
}
```

### 4.4 Keycloak Client

```go
// internal/auth/keycloak/client.go

package keycloak

import (
	"context"
	"fmt"
	"go-ocs/internal/auth/config"
	"go-ocs/internal/logging"

	"github.com/Nerzal/gocloak/v13"
)

// Client wraps gocloak and provides token validation and user attribute lookup.
type Client struct {
	gocloak gocloak.GoCloak
	config  config.KeycloakConfig
	realm   string // extracted from the issuer URL
}

// NewClient initialises the gocloak client using the provided KeycloakConfig.
func NewClient(cfg config.KeycloakConfig) (*Client, error) {
	if !cfg.Enabled {
		logging.Warn("Keycloak auth is DISABLED - all requests will be unauthenticated")
		return nil, nil
	}

	gc := gocloak.NewClient(cfg.IssuerURL)
	realm := extractRealm(cfg.IssuerURL)

	logging.Info("Keycloak client initialised", "issuer", cfg.IssuerURL, "realm", realm, "clientId", cfg.ClientID)

	return &Client{
		gocloak: gc,
		config:  cfg,
		realm:   realm,
	}, nil
}

// ValidateToken introspects the token via Keycloak and returns extracted claims.
func (c *Client) ValidateToken(ctx context.Context, rawToken string) (*KeycloakClaims, error) {
	result, err := c.gocloak.RetrospectToken(ctx, rawToken, c.config.ClientID, c.config.ClientSecret, c.realm)
	if err != nil {
		return nil, fmt.Errorf("token introspection failed: %w", err)
	}

	if result.Active == nil || !*result.Active {
		return nil, fmt.Errorf("token is not active")
	}

	// Decode and return the full claims including Keycloak-specific fields
	claims, err := decodeKeycloakClaims(rawToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}

	return claims, nil
}

// extractRealm parses the realm name from a Keycloak issuer URL.
// e.g. https://keycloak.example.com/realms/charging-realm -> "charging-realm"
func extractRealm(issuerURL string) string {
	const prefix = "/realms/"
	idx := len(issuerURL)
	for i := len(issuerURL) - 1; i >= 0; i-- {
		if issuerURL[i] == '/' {
			idx = i
			break
		}
	}
	return issuerURL[idx+1:]
}
```

### 4.5 Chi Authentication Middleware

```go
// internal/auth/keycloak/middleware.go

package keycloak

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const ClaimsContextKey contextKey = "keycloak_claims"

// Middleware returns a Chi-compatible HTTP middleware that validates Bearer tokens.
// If auth is disabled (client is nil) the middleware is a no-op pass-through.
func Middleware(client *Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Auth disabled - pass through
			if client == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Extract Bearer token
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing Authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				http.Error(w, "invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			rawToken := parts[1]

			// Validate token via Keycloak
			claims, err := client.ValidateToken(r.Context(), rawToken)
			if err != nil {
				http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// Inject claims into request context
			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext retrieves KeycloakClaims previously injected by Middleware.
func ClaimsFromContext(ctx context.Context) (*KeycloakClaims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*KeycloakClaims)
	return claims, ok
}
```

### 4.6 User Service (Fetch Keycloak Attributes)

```go
// internal/auth/keycloak/user_service.go

package keycloak

import (
	"context"
	"fmt"
	"go-ocs/internal/auth/config"

	"github.com/Nerzal/gocloak/v13"
)

// UserService provides access to Keycloak admin API for fetching
// user attributes and role details that are not embedded in the JWT.
type UserService struct {
	gocloak gocloak.GoCloak
	config  config.KeycloakConfig
	realm   string
}

// NewUserService creates a UserService using admin credentials from config.
func NewUserService(cfg config.KeycloakConfig) *UserService {
	gc := gocloak.NewClient(cfg.IssuerURL)
	return &UserService{
		gocloak: gc,
		config:  cfg,
		realm:   extractRealm(cfg.IssuerURL),
	}
}

// GetUserAttributes fetches the attributes map for a given Keycloak user ID.
func (s *UserService) GetUserAttributes(ctx context.Context, adminToken, userID string) (map[string][]string, error) {
	user, err := s.gocloak.GetUserByID(ctx, adminToken, s.realm, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user %s: %w", userID, err)
	}

	if user.Attributes == nil {
		return map[string][]string{}, nil
	}

	return *user.Attributes, nil
}

// GetRoleAttributes fetches the attributes defined on a Keycloak role.
func (s *UserService) GetRoleAttributes(ctx context.Context, adminToken, roleName string) (map[string][]string, error) {
	role, err := s.gocloak.GetRealmRole(ctx, adminToken, s.realm, roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch role %s: %w", roleName, err)
	}

	if role.Attributes == nil {
		return map[string][]string{}, nil
	}

	return *role.Attributes, nil
}
```

---

## 5. REST & GraphQL Framework Design

### 5.1 GraphQL Library: `github.com/99designs/gqlgen`

**Selected because:**

- Most widely adopted Go GraphQL library
- Schema-first code generation (generates type-safe resolver interfaces)
- Integrates cleanly with Chi router via `http.Handler`
- Supports subscriptions, dataloaders, middleware
- Actively maintained with excellent documentation

### 5.2 REST + GraphQL Router Setup

```go
// internal/backend/handlers/rest/router.go

package rest

import (
	"net/http"
	"time"
	"go-ocs/internal/backend/appcontext"
	"go-ocs/internal/auth/keycloak"
	"go-ocs/internal/logging"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter builds the Chi router for the REST API.
// At this stage it only sets up the framework with auth middleware.
// Actual endpoints will be added in subsequent iterations.
func NewRouter(appCtx *appcontext.AppContext) http.Handler {
	r := chi.NewRouter()

	// Common middleware (mirrors charging-engine pattern)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))
	r.Use(logging.Middleware)

	// OAuth2 authentication middleware (no-op when auth.enabled = false)
	r.Use(keycloak.Middleware(appCtx.Auth))

	// REST routes - placeholder, endpoints to be added
	r.Route(appCtx.Config.Server.RestPath, func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
		// Future REST endpoints will be mounted here
	})

	return r
}
```

```go
// internal/backend/handlers/graphql/router.go

package graphql

import (
	"net/http"
	"go-ocs/internal/backend/appcontext"
	"go-ocs/internal/auth/keycloak"
	"go-ocs/internal/logging"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
)

// NewRouter builds the Chi router for the GraphQL endpoint.
// At this stage it mounts the gqlgen handler and playground only.
// Resolvers will be implemented in subsequent iterations.
func NewRouter(appCtx *appcontext.AppContext) http.Handler {
	r := chi.NewRouter()

	// Common middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(logging.Middleware)

	// OAuth2 authentication middleware
	r.Use(keycloak.Middleware(appCtx.Auth))

	graphqlPath := appCtx.Config.Server.GraphqlPath

	// GraphQL handler (schema and resolvers to be wired in later)
	// srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
	//     Resolvers: &resolvers.Resolver{AppCtx: appCtx},
	// }))

	// Placeholder until gqlgen schema is generated
	r.Get(graphqlPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"GraphQL endpoint - schema not yet defined"}`))
	})

	// GraphQL Playground (dev only - disable in production via config)
	r.Get(graphqlPath+"/playground", playground.Handler("Charging Admin", graphqlPath))

	logging.Info("GraphQL router configured", "path", graphqlPath)
	return r
}
```

---

## 6. Application Entry Point

```go
// cmd/charging-backend/main.go

package main

import (
	"context"
	"net/http"
	"os"
	"go-ocs/internal/appl"
	"go-ocs/internal/backend/appcontext"
	"go-ocs/internal/backend/handlers/graphql"
	"go-ocs/internal/backend/handlers/rest"
	"go-ocs/internal/auth/keycloak"
	"go-ocs/internal/logging"
	"go-ocs/internal/store"
)

func main() {
	// Bootstrap early logging
	logging.Bootstrap()

	// Load configuration
	configFile := os.Getenv("BACKEND_CONFIG")
	if configFile == "" {
		configFile = "backend-config.yaml"
	}
	cfg := appcontext.NewConfig(configFile)

	// Configure structured logging from loaded config
	logging.Configure(&cfg.Base.Logging)

	logging.Info("Starting charging-backend", "addr", cfg.Server.Addr)

	// Initialise database
	db := store.NewStore(cfg.Base.Database.URL)
	defer db.DB.Close()

	// Initialise Keycloak auth client (returns nil if auth.enabled = false)
	authClient, err := keycloak.NewClient(cfg.Auth)
	if err != nil {
		logging.Fatal("Failed to initialise Keycloak client", "err", err)
	}

	// Build application context
	appCtx := &appcontext.AppContext{
		Config:  cfg,
		Metrics: appcontext.NewMetrics(),
		Store:   db,
		Auth:    authClient,
	}

	// Start Prometheus metrics server (reuses shared appl utility)
	metricsSrv := appl.StartMetricsServer(&cfg.Base)
	defer func() {
		if metricsSrv != nil {
			_ = metricsSrv.Shutdown(context.Background())
		}
	}()

	// Build REST and GraphQL routers
	restHandler := rest.NewRouter(appCtx)
	graphqlHandler := graphql.NewRouter(appCtx)

	// Mount both routers onto a single HTTP server
	mux := http.NewServeMux()
	mux.Handle(cfg.Server.GraphqlPath+"/", graphqlHandler)
	mux.Handle(cfg.Server.RestPath+"/", restHandler)

	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start the server in a goroutine
	go func() {
		logging.Info("charging-backend listening", "addr", cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Fatal("Server error", "err", err)
		}
	}()

	// Wait for shutdown signal (SIGTERM, SIGINT etc.)
	sig := appl.WaitForSignal()
	logging.Info("Shutdown signal received", "signal", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 15*cfg.Server.WriteTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logging.Error("Graceful shutdown failed", "err", err)
	}

	logging.Info("charging-backend stopped")
}
```

---

## 7. Metrics Design

Following the same pattern as `charging-engine`:

```go
// internal/backend/appcontext/metrics.go

package appcontext

import "github.com/prometheus/client_golang/prometheus"

type AppMetrics struct {
	Runtime   *prometheus.HistogramVec // Request duration
	Rate      *prometheus.CounterVec   // Request count
	ErrorRate *prometheus.CounterVec   // Error count
}

func NewMetrics() *AppMetrics {
	m := &AppMetrics{
		Runtime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "charging_backend_runtime_seconds",
				Help:    "Charging backend request duration.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		Rate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "charging_backend_rate_total",
				Help: "Total charging backend requests.",
			},
			[]string{"method", "path"},
		),
		ErrorRate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "charging_backend_error_rate_total",
				Help: "Total charging backend errors.",
			},
			[]string{"method", "path"},
		),
	}

	prometheus.DefaultRegisterer.MustRegister(
		m.Runtime,
		m.Rate,
		m.ErrorRate,
	)

	return m
}
```

---

## 8. New Go Module Dependencies

The following dependencies will be added to `go.mod`:

| Dependency                       | Version | Purpose                                         |
|----------------------------------|---------|-------------------------------------------------|
| `github.com/Nerzal/gocloak/v13`  | v13.x   | Keycloak OAuth2 client & token introspection    |
| `github.com/99designs/gqlgen`    | v0.17.x | GraphQL server with code generation             |
| `github.com/golang-jwt/jwt/v5`   | v5.x    | JWT claims parsing (used internally by gocloak) |
| `github.com/vektah/gqlparser/v2` | v2.x    | GraphQL schema parsing (gqlgen dependency)      |

---

## 9. Implementation Phases

| Phase       | Scope                                         | Status              |
|-------------|-----------------------------------------------|---------------------|
| **Phase 1** | Framework scaffold (this design)              | 🔲 Pending approval |
| **Phase 2** | GraphQL schema definition + gqlgen generation | 🔲 Future           |
| **Phase 3** | First REST endpoints (carriers, subscribers)  | 🔲 Future           |
| **Phase 4** | First GraphQL resolvers                       | 🔲 Future           |
| **Phase 5** | Role-based authorisation on endpoints         | 🔲 Future           |

---

## 10. Design Rules (mirroring existing architectural rules)

1. **No business logic in transport handlers** - handlers delegate to services
2. **Auth middleware is the single enforcement point** - handlers assume a valid, injected claims context
3. **Auth is shared** - `internal/auth` is never imported by a specific app's `appcontext`; it's a shared library
4. **GraphQL resolvers delegate to services** - resolvers are thin wrappers over `internal/backend/services`
5. **All endpoints instrumented** - Prometheus metrics on every REST and GraphQL operation
6. **Auth can be toggled off** - `auth.enabled: false` for local development without Keycloak running
7. **Configuration loaded at startup only** - no runtime config reloading
