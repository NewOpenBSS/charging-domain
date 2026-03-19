# Architecture Decision Log

Append-only record of significant decisions made during the development of go-ocs.
Each entry records what was decided, why, and what alternatives were considered.
Entries are never edited — superseded decisions are marked with a reference to the
entry that replaces them.

---

## ADR-001 — Language: Go over Java

**Status:** Accepted
**Area:** Core platform

**Decision:** Rewrite the OCS platform in Go rather than continuing Java development.

**Rationale:** Performance characteristics better suited to real-time charging workloads.
Simpler deployment (single binary). Stronger concurrency primitives for concurrent quota
operations. Smaller runtime footprint.

**Consequences:** New toolchain (sqlc, gqlgen). Java domain knowledge preserved in port.

---

## ADR-002 — Three-application architecture

**Status:** Accepted
**Area:** System structure

**Decision:** Split into three independent applications sharing a database and Kafka:
`charging-engine`, `charging-dra`, `charging-backend`.

**Rationale:** Each has a distinct protocol and operational profile. Charging engine
handles high-volume real-time NCHF HTTP. DRA handles Diameter Ro for wholesale carriers.
Backend is low-volume admin/config. Separation allows independent scaling and deployment.

**Consequences:** Shared `internal/` packages for common concerns (store, auth, logging,
baseconfig, events). Each application has its own `appcontext` config struct.

---

## ADR-003 — Database access: sqlc for static queries, raw pgx for dynamic queries

**Status:** Accepted
**Area:** Persistence layer

**Decision:** Use sqlc for type-safe Go from static SQL. Use raw pgx for dynamic queries
where WHERE/ORDER BY are runtime-constructed (filtered list endpoints).

**Rationale:** sqlc provides compile-time SQL validation. Dynamic filter/pagination
queries cannot be expressed as static sqlc queries — parameterised pgx is the safe
alternative. ORMs rejected due to complexity and poor fit with financial determinism.

**Consequences:** `internal/store/sqlc/` is generated — never manually edited.
Dynamic query methods live on the `Store` struct. SQL injection prevention relies on
per-resource column allowlists in the filter builder.

---

## ADR-004 — Financial values: shopspring/decimal throughout

**Status:** Accepted
**Area:** Domain / charging correctness

**Decision:** All financial values use `github.com/shopspring/decimal`. Native float types
are prohibited for financial arithmetic.

**Rationale:** IEEE 754 floating-point produces incorrect monetary results. Hard
requirement for a system processing real money.

**Consequences:** `DecimalDigits int32` config field required in every application
performing financial arithmetic. Default precision 22 digits. Must be propagated
explicitly through call chains — never hardcoded at call sites.

---

## ADR-005 — GraphQL for backend admin API: gqlgen (schema-first)

**Status:** Accepted
**Area:** charging-backend API layer

**Decision:** Use `github.com/99designs/gqlgen` with schema-first approach.
Define `.graphql` schema files, generate type-safe resolver interfaces via `gqlgen generate`.

**Alternatives considered:** `graph-gophers/graphql-go` (resolver-first) — rejected
because schema-first aligns better with the design-first workflow.

**Consequences:** `gql/schema/` is source of truth. `internal/backend/graphql/generated/`
is generated — never edited. Resolvers are thin wrappers; logic lives in services.

---

## ADR-006 — OAuth2/Keycloak for backend authentication

**Status:** Accepted
**Area:** charging-backend security

**Decision:** Use `github.com/Nerzal/gocloak/v13` for Keycloak token introspection.
Auth is togglable via `auth.enabled` config flag for local development.

**Rationale:** gocloak provides JWKS resolution, key rotation caching, and
Keycloak-specific claims (realm roles, client roles, custom attributes).
Toggle allows development without Keycloak running locally.

**Consequences:** `internal/auth/` is a shared package. Middleware is the single
enforcement point. Auth not applied to charging-engine or charging-dra.

---

## ADR-007 — Domain model package location: internal/model

**Status:** Accepted
**Area:** Package structure
**Supersedes:** Original location `internal/chargeengine/model`

**Decision:** Move shared domain model types to `internal/model/` to allow sharing
between charging-engine and charging-backend.

**Rationale:** charging-backend needed the same domain types. Keeping them under
`internal/chargeengine/` incorrectly implied they were private to that application.

**Consequences:** 25 files updated with new import path. `Plan` struct renamed to
`ClassificationPlan` to eliminate ambiguity. Done as a single dedicated commit.

---

## ADR-008 — Configuration: YAML only, no environment variables in application code

**Status:** Accepted
**Area:** Configuration

**Decision:** All configuration loaded from YAML files. Environment variables not read
directly in application code.

**Rationale:** Consistent, inspectable configuration. Avoids implicit global state of
environment variables.

**Consequences:** Each application has a `*-config.yaml` in its `cmd/` directory.
`internal/baseconfig/` provides shared YAML loading.

---

## ADR-009 — Quota operations: optimistic locking with retry

**Status:** Accepted
**Area:** Quota management / concurrency

**Decision:** Quota updates use optimistic locking with configurable retry (default 3)
rather than pessimistic row locking.

**Rationale:** Pessimistic locking would serialise all quota updates per subscriber.
Conflicts are rare — optimistic locking handles the common case efficiently.

**Consequences:** `QuotaManager.executeWithQuota` implements the retry loop.
`ErrConflict` and `ErrRetryLimitExceeded` are the defined failure modes.
All quota operations must go through this manager.

---

## ADR-010 — Filter/pagination: generic builder with per-resource column allowlist

**Status:** Accepted
**Area:** charging-backend GraphQL list endpoints

**Decision:** Generic `internal/backend/filter` package builds parameterised SQL WHERE
and ORDER BY from GraphQL input types. Each resource provides a column allowlist
(GraphQL field name → SQL column name).

**Rationale:** All four admin resources require identical filter/sort/pagination behaviour.
Shared builder eliminates repetition and centralises the security boundary.

**Consequences:** User-supplied field names validated against allowlist before SQL use.
Values always passed as positional `$N` args — never interpolated. New resources must
define their own `allowedCols` map.
