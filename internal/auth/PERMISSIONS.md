# Permission Enforcement Framework

This document explains how the permission enforcement system works across both
the REST API and the GraphQL API. It is the single reference for developers adding
new endpoints or modifying existing permission checks.

---

## Overview

The framework enforces a **default-deny** security model. Every endpoint — REST or
GraphQL — must explicitly declare the permissions it requires. Endpoints that omit
a permission declaration are rejected at runtime (GraphQL) or fail to compile (REST).

Permissions are simple strings (e.g. `"read"`, `"write"`, `"admin"`) carried in the
JWT token issued by Keycloak. The `Permissions` field in the JWT claims contains a
flat list of permission strings assigned to the authenticated user.

When multiple permissions are declared on a single endpoint, **OR logic** applies:
the caller needs at least one of the listed permissions to proceed.

When authentication is disabled (`auth.enabled: false` in config), all permission
checks are bypassed — every request is allowed through. This is intended for local
development only.

---

## How Permissions Flow

```
HTTP Request
  │
  ▼
keycloak.Middleware          ← extracts Bearer token, validates JWT, injects claims into context
  │
  ▼
Permission Check             ← REST: Require() middleware  /  GraphQL: @auth directive
  │
  ├─ No claims → 401 Unauthorized / UNAUTHENTICATED
  ├─ Claims present, no matching permission → 403 Forbidden / FORBIDDEN
  └─ Permission matched → proceed to handler / resolver
```

---

## Part 1: REST API Permissions

### SecureRouter

`SecureRouter` wraps a standard `chi.Router` and forces every route registration
to include a permissions parameter. This is enforced at **compile time** — if you
call `sr.Get(pattern, handler)` without the permissions argument, the code will not
compile.

#### Creating a SecureRouter

```go
import "go-ocs/internal/auth"

// In your router setup function:
authEnabled := appCtx.Auth != nil
sr := auth.NewSecureRouter(r, authEnabled)
```

#### Registering a Protected Route

Pass a slice of `auth.Permission` as the second argument. The caller must hold at
least one of the listed permissions.

```go
// Single permission — caller must have "read"
sr.Get("/api/charging/carriers", []auth.Permission{"read"}, listCarriersHandler)

// Single permission — caller must have "write"
sr.Post("/api/charging/carriers", []auth.Permission{"write"}, createCarrierHandler)
sr.Put("/api/charging/carriers/{plmn}", []auth.Permission{"write"}, updateCarrierHandler)
sr.Delete("/api/charging/carriers/{plmn}", []auth.Permission{"write"}, deleteCarrierHandler)
```

#### Registering a Public Route (No Auth Required)

Use `auth.Public()` to signal that the endpoint does not require authentication.
This returns a nil permission slice, which tells the router to skip the Require
middleware entirely.

```go
// Health check — no authentication needed
sr.Get("/api/charging/health", auth.Public(), healthCheckHandler)
```

#### Multiple Permissions (OR Logic)

When you pass multiple permissions, the caller needs **any one** of them:

```go
// Caller needs either "write" or "admin" to proceed
sr.Delete("/api/charging/carriers/{plmn}", []auth.Permission{"write", "admin"}, deleteHandler)
```

#### How It Works Internally

When you register a route with permissions:

1. `SecureRouter.Get()` (or Post, Put, etc.) calls the internal `handle()` method
2. `handle()` wraps your handler with `auth.Require(authEnabled, permissions...)`
3. `Require()` returns a Chi-compatible middleware that:
   - Returns **401** if no claims are found in the request context
   - Returns **403** if claims are present but none of the required permissions match
   - Calls the next handler if at least one permission matches

When you register a route with `auth.Public()`:

1. `handle()` sees that permissions is nil
2. The handler is registered directly — no Require middleware is applied

#### Complete REST Router Example

```go
package rest

import (
    "net/http"

    "go-ocs/internal/auth"
    "go-ocs/internal/auth/keycloak"
    "go-ocs/internal/backend/appcontext"

    "github.com/go-chi/chi/v5"
)

func NewRouter(appCtx *appcontext.AppContext) http.Handler {
    r := chi.NewRouter()

    // Middleware chain — keycloak.Middleware must come before SecureRouter
    r.Use(keycloak.Middleware(appCtx.Auth))

    authEnabled := appCtx.Auth != nil

    r.Route("/api/charging", func(r chi.Router) {
        sr := auth.NewSecureRouter(r, authEnabled)

        // Public endpoints
        sr.Get("/health", auth.Public(), healthHandler)

        // Protected endpoints
        sr.Get("/carriers", []auth.Permission{"read"}, listCarriersHandler)
        sr.Post("/carriers", []auth.Permission{"write"}, createCarrierHandler)
        sr.Put("/carriers/{plmn}", []auth.Permission{"write"}, updateCarrierHandler)
        sr.Delete("/carriers/{plmn}", []auth.Permission{"write", "admin"}, deleteCarrierHandler)
    })

    return r
}
```

---

## Part 2: GraphQL API Permissions

### The @auth Directive

GraphQL endpoints use a schema-level `@auth` directive to declare permissions.
The directive is defined in `gql/schema/schema.graphql`:

```graphql
directive @auth(permissions: [String!]!) on FIELD_DEFINITION
```

#### Annotating Query Fields

Add `@auth(permissions: [...])` to every Query and Mutation field:

```graphql
extend type Query {
  # Read operations require "read" permission
  carrierList(page: PageRequest, filter: FilterRequest): [Carrier!]!
    @auth(permissions: ["read"])

  carrierByPlmn(plmn: String!): Carrier
    @auth(permissions: ["read"])

  countCarriers(filter: FilterRequest): Int!
    @auth(permissions: ["read"])
}
```

#### Annotating Mutation Fields

```graphql
extend type Mutation {
  createCarrier(carrier: CarrierInput!): Carrier!
    @auth(permissions: ["write"])

  updateCarrier(plmn: String!, carrier: CarrierInput!): Carrier!
    @auth(permissions: ["write"])

  deleteCarrier(plmn: String!): Boolean!
    @auth(permissions: ["write"])
}
```

#### Multiple Permissions (OR Logic)

Same as REST — the caller needs at least one of the listed permissions:

```graphql
extend type Mutation {
  # Caller needs either "write" or "admin"
  deleteCarrier(plmn: String!): Boolean!
    @auth(permissions: ["write", "admin"])
}
```

### Deny-by-Default Middleware

In addition to the `@auth` directive, the GraphQL server uses a **deny-by-default
field middleware** (`auth.DenyByDefaultFieldMiddleware`). This middleware rejects any
top-level Query or Mutation field that is **not** annotated with `@auth`.

This means:
- If you add a new Query or Mutation field but forget `@auth`, it will be
  **automatically rejected** at runtime with a 403 FORBIDDEN error
- You do not need to remember to add the directive — the system will tell you if
  you forget

**Exempt fields** (not subject to deny-by-default):
- Introspection fields: `__schema`, `__type`, and any field starting with `__`
- The `_empty` placeholder field on the root Query and Mutation types
- Nested object fields (only top-level Query/Mutation fields are enforced)

### How It Works Internally

The GraphQL permission enforcement has two layers:

**Layer 1 — DenyByDefaultFieldMiddleware:**
1. Runs on every field resolution
2. For top-level Query/Mutation fields, checks if the field definition has an `@auth` directive
3. If no `@auth` directive is found, rejects with `FORBIDDEN`
4. Nested types and exempt fields pass through

**Layer 2 — @auth Directive Handler:**
1. Runs when gqlgen encounters a field annotated with `@auth`
2. Extracts `KeycloakClaims` from the request context
3. If no claims are found, returns `UNAUTHENTICATED` error
4. Checks if any of the declared permissions match the caller's claims
5. If no match, returns `FORBIDDEN` error
6. If a match is found, proceeds to the resolver

### GraphQL Error Responses

When permission checks fail, the GraphQL API returns structured errors:

**Unauthenticated (no valid token):**
```json
{
  "errors": [{
    "message": "unauthenticated",
    "extensions": { "code": "UNAUTHENTICATED" }
  }]
}
```

**Forbidden (valid token, insufficient permissions):**
```json
{
  "errors": [{
    "message": "forbidden: insufficient permissions",
    "extensions": { "code": "FORBIDDEN" }
  }]
}
```

**Missing @auth directive (developer error):**
```json
{
  "errors": [{
    "message": "forbidden: operation not annotated with @auth directive",
    "extensions": { "code": "FORBIDDEN" }
  }]
}
```

### Complete GraphQL Schema Example

```graphql
# In gql/schema/my_resource.graphql

type MyResource {
  id: ID!
  name: String!
  status: String!
}

input MyResourceInput {
  name: String!
}

extend type Query {
  myResourceList(page: PageRequest, filter: FilterRequest): [MyResource!]!
    @auth(permissions: ["read"])

  myResourceById(id: ID!): MyResource
    @auth(permissions: ["read"])

  countMyResources(filter: FilterRequest): Int!
    @auth(permissions: ["read"])
}

extend type Mutation {
  createMyResource(input: MyResourceInput!): MyResource!
    @auth(permissions: ["write"])

  updateMyResource(id: ID!, input: MyResourceInput!): MyResource!
    @auth(permissions: ["write"])

  deleteMyResource(id: ID!): Boolean!
    @auth(permissions: ["write"])
}
```

### Wiring the GraphQL Server

The GraphQL router setup in `internal/backend/handlers/graphql/router.go` wires
both layers automatically:

```go
// 1. Configure the @auth directive handler
cfg := generated.Config{
    Resolvers:  resolver,
    Directives: auth.NewGraphQLDirectiveConfig(authEnabled),
}

// 2. Create the server
srv := handler.New(generated.NewExecutableSchema(cfg))

// 3. Enable deny-by-default field middleware
srv.AroundFields(auth.DenyByDefaultFieldMiddleware(authEnabled))
```

No additional wiring is needed — adding `@auth` to a schema field is all that is
required for a new endpoint.

---

## Part 3: Permission Strings and Keycloak

### Where Permissions Come From

Permissions are extracted from the JWT token issued by Keycloak. The token contains
a `permissions` claim — a flat array of strings:

```json
{
  "permissions": ["read", "write", "admin"]
}
```

These are mapped to Keycloak via client scopes, protocol mappers, or authorization
services. The exact Keycloak configuration is outside the scope of this document.

### The Permission Type

The `auth.Permission` type is a simple string alias:

```go
type Permission string
```

Permission constants are **not** defined centrally in the auth package. Each domain
package or schema file defines the permission strings it uses. Currently the
codebase uses:
- `"read"` — for query/list operations
- `"write"` — for create/update/delete mutations

### Checking Permissions Programmatically

If you need to check permissions outside of middleware (e.g. in a service layer),
use `auth.HasPermission`:

```go
import (
    "go-ocs/internal/auth"
    "go-ocs/internal/auth/keycloak"
)

func myServiceMethod(ctx context.Context) error {
    claims, ok := keycloak.ClaimsFromContext(ctx)
    if !ok {
        return errors.New("unauthenticated")
    }

    if !auth.HasPermission(claims, "admin") {
        return errors.New("forbidden: admin permission required")
    }

    // proceed with admin-only logic
    return nil
}
```

---

## Quick Reference

| Task | REST | GraphQL |
|---|---|---|
| Declare a protected endpoint | `sr.Get(path, []auth.Permission{"read"}, handler)` | `@auth(permissions: ["read"])` |
| Declare a public endpoint | `sr.Get(path, auth.Public(), handler)` | N/A (use `_empty` or introspection) |
| Require one of many permissions | `[]auth.Permission{"write", "admin"}` | `@auth(permissions: ["write", "admin"])` |
| Check permission in code | `auth.HasPermission(claims, "admin")` | `auth.HasPermission(claims, "admin")` |
| Unauthenticated response | HTTP 401 | `UNAUTHENTICATED` error |
| Forbidden response | HTTP 403 | `FORBIDDEN` error |
| Auth disabled behaviour | All checks bypassed | All checks bypassed |

---

## Checklist for Adding a New Endpoint

### REST

1. Ensure `keycloak.Middleware` is in the middleware chain (already done in router setup)
2. Use `auth.NewSecureRouter(r, authEnabled)` instead of the raw `chi.Router`
3. Register routes with explicit permissions: `sr.Get(path, []auth.Permission{...}, handler)`
4. Use `auth.Public()` only for genuinely public endpoints (health checks, metrics)

### GraphQL

1. Add `@auth(permissions: [...])` to every new Query and Mutation field
2. Run `go generate ./...` to regenerate the gqlgen code if the schema changed
3. The deny-by-default middleware will catch any field you forget to annotate
4. Nested object types do not need `@auth` — only top-level Query/Mutation fields
