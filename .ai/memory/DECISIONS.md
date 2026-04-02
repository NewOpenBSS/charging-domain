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

## ADR-011 — PoolDB interface for testable store dynamic queries
**Status:** Accepted
**Area:** internal/store
**Decision:** Introduced PoolDB interface (sqlc.DBTX + Close) and changed Store.DB from *pgxpool.Pool to PoolDB.
**Rationale:** Dynamic store methods (ListX, CountX) use Store.DB directly rather than sqlc.Queries. Without an interface, unit tests cannot mock the DB and would require a live PostgreSQL instance, violating the no-external-services rule for unit tests. *pgxpool.Pool satisfies PoolDB so all production code is unaffected.
**Consequences:** New dynamic store methods must use Store.DB (not Store.Q). Tests in the store package can create Store{DB: mockDB} with a mock that satisfies PoolDB.

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

---

## ADR-012 — store.NewTestStore for cross-package service test injection
**Status:** Accepted
**Area:** internal/store, internal/backend/services
**Decision:** Added `NewTestStore(dbtx sqlc.DBTX, q DBQuerier) *Store` to the store package to enable service-layer unit tests to inject mock implementations without exposing the unexported `querier` field.
**Rationale:** The `querier` field is unexported so it cannot be set from outside the `store` package. Service tests (in `services` package) need to mock both the static sqlc path (via `Q`) and the dynamic query path (via `querier`). A named constructor is cleaner than exporting the field and makes the testing intent explicit.
**Consequences:** A new exported function in the store package. Must only be called in test code. Any new dynamic store methods that services need to test will benefit from this constructor automatically.

---

## ADR-010 — DBQuerier interface for testable dynamic store methods
**Status:** Accepted
**Area:** internal/store
**Decision:** Added a `DBQuerier` interface to store.go (with `Query` and `QueryRow` methods) and an unexported `querier` field on `Store` that defaults to the `*pgxpool.Pool`. Dynamic store methods (`ListChargingTraces`, `CountChargingTraces`) use `s.querier` rather than `s.DB` directly.
**Rationale:** `*pgxpool.Pool` is a concrete type with no lifecycle-independent interface in pgx/v5. Without an interface, unit tests for dynamic store methods require a real PostgreSQL connection, violating the "unit tests must not require external services" rule. The `querier` field allows tests in `package store` to substitute a testify mock, keeping the production code path unchanged. The exported `DB *pgxpool.Pool` field is preserved so existing callers (e.g. `db.DB.Close()` in main.go) are unaffected.
**Consequences:** New dynamic store methods should use `s.querier` rather than `s.DB`. Existing `carrier_store.go` (and similar) can be migrated to `s.querier` in a future cleanup task.

## ADR-013 — SubscriberEventConsumer: consumer package in internal/backend/

**Status:** Accepted
**Area:** internal/backend/consumer

**Decision:** The Kafka consumer for SubscriberEvent messages lives in a new
`internal/backend/consumer/` package. A `SubscriberStorer` interface (defined
at the point of consumption) is satisfied by `StoreSubscriberAdapter` which wraps
`*store.Store`. The consumer takes a `SubscriberStorer` rather than `*store.Store`
directly so it can be tested without a real database.

**Rationale:** Placing the consumer in `internal/backend/` keeps it alongside the
rest of the backend concerns (services, resolvers, appcontext) and avoids a circular
dependency between `internal/events` (which would need to import `internal/store`)
and `internal/backend`. The narrow `SubscriberStorer` interface keeps the consumer
unit-testable without PostgreSQL or Kafka. The adapter pattern isolates the sqlc
type conversions (uuid.UUID → pgtype.UUID) in one file.

**Consequences:** Future Kafka consumers for the backend should follow the same
pattern: consumer package in `internal/backend/consumer/`, narrow interface,
store adapter, wired via AppContext.
## ADR-012 — GitHub Organisation: NewOpenBSS
**Status:** Accepted
**Area:** Infrastructure / Repository Structure
**Decision:** Create GitHub organisation `NewOpenBSS` to house all delivery repos and a shared requirements repo.
**Rationale:** Separates requirements/roadmap from code. A dedicated requirements repo under the org serves multiple delivery projects. Continues the OpenBSS brand (the Java organisation) while signalling the new Go platform.
**Consequences:** Requirements and Features will migrate from REQUIREMENTS.md and FEATURES.md to GitHub Issues in a new `NewOpenBSS/requirements` repo. Recipes will be updated to use `gh issue` commands with `--repo NewOpenBSS/requirements`. go-ocs will eventually move to `NewOpenBSS/go-ocs`.

---

## ADR-014 — WholesaleContractConsumer: UpsertWholesaler with INSERT ON CONFLICT

**Status:** Accepted
**Area:** Kafka consumer / wholesaler shadow table
**Decision:** Use INSERT … ON CONFLICT (id) DO UPDATE for all provisioned events rather than a separate insert-or-update flow at the application level.
**Rationale:** Idempotent handling of re-provisioning events without explicit check-then-insert. Reduces round-trips and eliminates a race condition that could occur between a check and an insert under concurrent events.
**Consequences:** Modified_on is always updated to NOW() on upsert, keeping the timestamp accurate for all write paths.

---

## ADR-015 — DeleteInactiveWholesalerIfEmpty: atomic single SQL statement

**Status:** Accepted
**Area:** Kafka consumer / wholesaler cascade delete
**Decision:** Implement the "delete wholesaler if inactive and has no subscribers" check as a single SQL DELETE with embedded subquery rather than as two application-level statements (count then delete).
**Rationale:** A two-step approach would have a race window between the count query and the delete where a concurrent subscriber insert could result in an orphaned wholesaler delete. The single-statement SQL eliminates this window entirely.
**Consequences:** The cascade delete is always a no-op unless both conditions are satisfied simultaneously. Slightly more complex SQL but safer than any application-level alternative.

---

## ADR-016 — DeregisterWholesaler: count-then-act without transaction

**Status:** Accepted
**Area:** Kafka consumer / wholesaler deregistration
**Decision:** The DeregisterWholesaler adapter method performs count then delete-or-deactivate as two sequential DB calls without a wrapping transaction.
**Rationale:** Matches the Java reference implementation behaviour. A race between two simultaneous deregistering events for the same wholesaler is extremely low probability in the wholesale domain. The complexity of a transaction for this path is not warranted at this stage.
**Consequences:** In the rare case of concurrent deregistering events, the wholesaler may be left in an intermediate state (active=false with zero subscribers). The DeleteInactiveWholesalerIfEmpty SQL path will eventually clean it up on next subscriber delete.

---

## ADR-017 — floatToNumeric: decimal string conversion for pgtype.Numeric

**Status:** Accepted
**Area:** Kafka consumer / type conversion
**Decision:** Convert RateLimit float64 to pgtype.Numeric by routing through shopspring/decimal.NewFromFloat().String() and scanning the string into pgtype.Numeric.
**Rationale:** Direct float64-to-Numeric conversion risks floating-point representation artefacts. The decimal string representation is exact for the values used (rate limits are whole numbers or simple decimals). This avoids importing additional conversion libraries and matches the project's existing decimal handling patterns.
**Consequences:** floatToNumeric is a private helper in the consumer package. Any error from Scan (unreachable for valid finite float64 inputs) is propagated as a wrapped error from UpsertWholesaler.

## ADR-018 — ProvisionCounter: QuotaManager method with QuotaProvisioner interface

**Status:** Accepted
**Area:** F-007 — QuotaProvisioningEventConsumer / quota package
**Decision:** Implement provisioning business logic as a `ProvisionCounter` method on
`*QuotaManager`, exposed to the consumer via a narrow `QuotaProvisioner` interface defined
in the consumer package.
**Rationale:** All quota business logic (load/save with optimistic retry, journal publishing,
clawback) already lives in `internal/quota`. The provisioning operation uses the same
`executeWithQuota` retry loop as Reserve, Debit, and Release. Keeping it in the quota package
avoids leaking quota internals into the consumer layer. The narrow interface at the point of
consumption follows the project's interface-at-consumer-side convention.
**Consequences:** `QuotaManager` grows a new exported method. The `QuotaProvisioner` interface
requires no changes to AppContext wiring beyond passing the existing `*QuotaManager`.

## ADR-019 — RemoveExpiredEntries: nil Expiry treated as never-expires

**Status:** Accepted
**Area:** internal/quota — LoadedQuota
**Decision:** Fix `RemoveExpiredEntries` to treat nil `Expiry` as "never expires" by
changing the condition from `c.Expiry.After(now)` to `c.Expiry == nil || c.Expiry.After(now)`.
**Rationale:** The provisioning event supports counters with no expiry date. The existing
code panicked on nil Expiry. The fix is consistent with how `GetBalance` handles nil Expiry
(uses the same `c.Expiry != nil && !c.Expiry.After(now)` guard).
**Consequences:** Previously all test counters included an expiry; the fix is backward-
compatible and enables non-expiring counters throughout the quota system.

## ADR-021 — SourceGroupResource: mirrors DestinationGroupResource exactly (F-003)
**Status:** Accepted
**Area:** `internal/store/queries/source_groups.sql`, `internal/store/source_group_store.go`
**Decision:** `carrier_source_group` has the same schema as `carrier_destination_group` (group_name, region).
All store, service, schema, and resolver code is a mechanical rename of the DestinationGroup equivalents.
No new migration is required — the table was created in `000001_init.up.sql`.
sqlc generates positional-arg signatures (no params struct) for the same reason as DestinationGroup —
only 2 columns, below the `query_parameter_limit: 4` threshold.
**Rationale:** F-003 acceptance criteria explicitly require matching Java operation names and mirroring
the DestinationGroupResource pattern. Mechanical mirroring minimises risk of divergence.
**Consequences:** Any schema change to `carrier_source_group` that adds a third column will cause
sqlc to switch to a params struct, requiring service-layer call-site updates.

## ADR-020 — DestinationGroupResource: positional args for Create/Update (F-002)
**Status:** Accepted
**Area:** `internal/store/sqlc/destination_groups.sql.go`, `internal/backend/services/destination_group_service.go`
**Decision:** sqlc generated positional-argument function signatures for `CreateDestinationGroup` and
`UpdateDestinationGroup` (no params struct) because `carrier_destination_group` has only 2 columns,
which is below the `query_parameter_limit: 4` configured in `sqlc.yaml`. The service calls these
with two plain `string` arguments rather than a `CreateDestinationGroupParams` struct.
**Rationale:** This is the behaviour specified by sqlc's `query_parameter_limit` setting — no change
was needed; the task spec noted this was likely and required verification after generation.
**Consequences:** If a third column is ever added to `carrier_destination_group`, `sqlc generate`
will automatically switch to a struct arg. Callers in `destination_group_service.go` will need
to be updated at that point.

## ADR-022 — Housekeeping thresholds via env vars (exception to ADR-008) (F-009)
**Status:** Accepted
**Area:** `cmd/charging-housekeeping/appcontext/config.go`
**Decision:** The three operational thresholds (stale sessions, trace purge, rate plan cleanup) are read from environment variables with duration-string defaults, not from YAML config. This is a deliberate exception to ADR-008 (YAML-only configuration).
**Rationale:** For a Kubernetes CronJob, environment variables are the standard Kubernetes-native configuration mechanism — injected via the CronJob spec without requiring a ConfigMap or volume mount for each threshold change. YAML config is used for structural/connection settings (DB URL, Kafka brokers, logging); per-environment operational thresholds are a better fit for env vars.
**Consequences:** Operators tune thresholds by changing the CronJob env vars in the Helm chart or kubectl patch. Invalid values produce a warning log and fall back to safe defaults.

## ADR-023 — Four-application architecture: charging-housekeeping added (F-009)
**Status:** Accepted
**Area:** System structure
**Decision:** Add `cmd/charging-housekeeping` as a fourth application — a run-once binary designed for Kubernetes CronJob invocation. It connects to DB and Kafka, runs four housekeeping tasks sequentially (quota expiry, stale sessions, trace purge, rate plan cleanup), logs a summary, and exits.
**Rationale:** Housekeeping operations (expired quota processing, stale data cleanup) must run on a schedule but do not belong in the long-running charging-engine or charging-backend. A separate binary keeps the operational profile clean and enables independent scaling/scheduling.
**Consequences:** Extends ADR-002 from three to four applications. Shares the same `internal/` packages. Requires a Kubernetes CronJob manifest (deferred to a future feature).

---

## ADR-024 — JWT validation via JWKS instead of token introspection (F-010)
**Status:** Accepted
**Area:** `internal/auth/keycloak`
**Decision:** Replace Keycloak token introspection (`RetrospectToken` via `gocloak`) with local JWT signature verification using Keycloak's JWKS endpoint (`/protocol/openid-connect/certs`) via `github.com/MicahParks/keyfunc/v2`.
**Rationale:** Token introspection requires the calling client to be confidential and have introspection permissions granted in Keycloak. The `portal` client is a public OAuth2 client with no secret, making introspection impossible. JWKS-based validation works for both public and confidential clients, requires no client secret, and is lower latency (local verification vs. a network round-trip to Keycloak per request). Keys are cached and refreshed automatically in the background.
**Consequences:** `clientId` and `clientSecret` removed from `KeycloakConfig` — no credentials needed for token validation. `gocloak` is retained only for `UserService` (admin API calls). The `keyfunc/v2` dependency is added. Token expiry and signature are enforced by `golang-jwt/jwt/v5` during local parse.

## ADR-025 — Permission enforcement framework: @auth directive + SecureRouter (F-011)
**Status:** Accepted
**Area:** `internal/auth/`, GraphQL schema, REST router
**Decision:** Implement a reusable permission enforcement framework with three components:
1. `Permission` type + `HasPermission` helper extracting permissions from JWT `permissions` claim
2. `SecureRouter` for REST — wraps chi.Router, requires explicit permission declaration per route (compile error if omitted). `Public()` marker for unauthenticated endpoints.
3. `@auth(permissions: [...])` GraphQL directive — enforces permissions per field. `DenyByDefaultFieldMiddleware` rejects unannotated Query/Mutation fields.

All three components respect `auth.enabled: false` for local dev bypass.
**Rationale:** Keycloak middleware validates *who* the caller is but not *what* they can do. The framework establishes deny-by-default semantics without coupling to any specific permission constants (which are domain-specific). Using generic `read`/`write` placeholder permissions allows the framework to be deployed immediately; fine-grained permissions (e.g. `carrier:create`, `rateplan:approve`) are a future feature.
**Consequences:** Every new REST route must use `SecureRouter` methods. Every new GraphQL query/mutation field must include `@auth(permissions: [...])` or the deny-default middleware will reject it. The `permissions` JWT claim must be configured in Keycloak token mappers for production use.
