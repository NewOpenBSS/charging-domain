# Project Status

_Updated by Claude Code at the end of every task. Source of truth for current implementation state._
_Last updated: 2026-03-20 (F-005)_

---

## Implementation Status

### Infrastructure & Shared Packages

| Component | Status | Notes |
|---|---|---|
| `internal/baseconfig` | ✅ Complete | Shared YAML config loading |
| `internal/logging` | ✅ Complete | Structured logging with Chi middleware |
| `internal/appl` | ✅ Complete | Shared app lifecycle (metrics server, signal handling) |
| `internal/model` | ✅ Complete | Shared domain types (moved from chargeengine/model) |
| `internal/store` | ✅ Complete | pgxpool + sqlc wrapper, dynamic query methods |
| ChargingTrace store layer | ✅ Complete | ListChargingTraces, CountChargingTraces, FindChargingTraceByTraceId |
| `internal/events` | ✅ Complete | Kafka producer via franz-go |
| Subscriber store queries | ✅ Complete | InsertSubscriber, UpdateSubscriber, DeleteSubscriber (sqlc) |
| `internal/events/subscriber_event.go` | ✅ Complete | SubscriberEvent struct + 5 event type constants |
| `internal/backend/consumer` | ✅ Complete | SubscriberEventConsumer + WholesaleContractConsumer with dispatch logic, unit tests |
| F-005 wiring | ✅ Complete | Consumer wired into AppContext + main.go; subscriber-event topic in config |
| F-006 wiring | ✅ Complete | WholesaleContractConsumer wired into AppContext + main.go; wholesale-contract-event topic in config |
| `internal/auth` | ✅ Complete | Keycloak client, JWT middleware, claims extraction |

### charging-engine (`cmd/charging-engine`, port :8080)

| Component | Status | Notes |
|---|---|---|
| NCHF HTTP handler | ✅ Complete | Create, Update, Release, One-time charge |
| Charging pipeline | ✅ Complete | auth → classify → rate → account → chargedata → response |
| Quota management | ✅ Complete | Reserve, Debit, Release with optimistic locking |
| Rule evaluator | ✅ Complete | Policy-based pricing rule evaluation |
| Kafka event publishing | ✅ Complete | Charge records, quota journal, notifications |
| Charging trace | ✅ Complete | Request/response audit trail in DB |
| NCHF protocol models | ✅ Complete | `internal/nchf` — 3GPP compliant |
| Diameter (DRA) | ✅ Complete | `internal/diameter` — Ro interface |

### charging-dra (`cmd/charging-dra`)

| Component | Status | Notes |
|---|---|---|
| DRA server | ✅ Complete | Diameter Ro interface for wholesale carriers |
| Rate limiting | ✅ Complete | Per-carrier rate limiting |
| OCS client | ✅ Complete | Forwards to charging-engine |

### charging-backend (`cmd/charging-backend`, port :8081)

| Component | Status | Notes |
|---|---|---|
| Application scaffold | ✅ Complete | Chi router, metrics, config, Keycloak wiring |
| Auth middleware | ✅ Complete | Bearer token validation, claims injection |
| GraphQL endpoint | ✅ Complete | gqlgen handler mounted at `/api/charging/graphql` |
| REST health endpoint | ✅ Complete | `/api/charging/health` |
| Filter/pagination builder | ✅ Complete | `internal/backend/filter` — shared across all resources |
| **CarrierResource** | ✅ Complete | CRUD via GraphQL + dynamic list/count |
| **ClassificationResource** | ✅ Complete | CRUD + state machine (DRAFT→PENDING→ACTIVE) |
| **NumberPlanResource** | ✅ Complete | CRUD via GraphQL |
| **RatePlanResource** | ✅ Complete | CRUD + state machine (DRAFT→PENDING→ACTIVE) |
| **QuotaResource** | ✅ Complete | Balance queries + reserve/debit/cancel mutations |

### Database

| Component | Status | Notes |
|---|---|---|
| Schema migrations | ✅ Complete | 9 migrations applied |
| sqlc generated queries | ✅ Complete | All resources have static sqlc queries |
| Seed data | ✅ Complete | Test carriers (NZ, NG, ZA), home network config |

---

## Current Focus

F-007 — QuotaProvisioningEventConsumer — **in progress** (task 001 of 4 complete).

---

## Recently Completed

- **F-001 Task 003** — Resolvers, AppContext, and GraphQL router wired for ChargingTrace. All three query methods functional end-to-end. `go test -race ./...` passes. Completed 2026-03-20.
- **F-001 Task 002** — `ChargingTraceService` implemented in `internal/backend/services/`. All three methods (list, count, by-ID) with UUID parsing, column map, and model mapper. Unit tests pass including race detector. `store.NewTestStore` helper added to enable service-level mocking. Completed 2026-03-20.
- **F-001 Task 001** — `gql/schema/charging_trace.graphql` created; gqlgen regenerated `models_gen.go`, `generated.go`, and `charging_trace.resolvers.go` (stubs). Build clean. Completed 2026-03-20.
- **ChargingTrace store layer** — ListChargingTraces, CountChargingTraces, FindChargingTraceByTraceId. Unit tests. Completed 2026-03-20.
- **QuotaResource** — Balance queries + reserve/debit/cancel mutations. GraphQL schema + service + resolvers complete. Kafka wired to charging-backend. Completed 2026-03-20.
- **QuotaManager.GetBalance** — `BalanceQuery`, `CounterBalance`, `GetBalance` added to `internal/quota`. Read-only domain primitive for balance inquiries. Committed 2026-03-19.
- Reverse-engineered project state and created memory/DECISIONS.md and memory/STATUS.md
- RatePlanResource — GraphQL CRUD + approval state machine
- NumberPlanResource — GraphQL CRUD
- ClassificationResource — GraphQL CRUD + approval state machine
- CarrierResource — GraphQL CRUD + dynamic filter/pagination
- charging-backend scaffold — auth, GraphQL handler, REST router
- Model package refactor — moved to internal/model, Plan → ClassificationPlan rename
- charging-engine — full NCHF charging pipeline
- charging-dra — Diameter Ro interface

---

## Open Questions

- RatePlan has two update queries (`UpdateRatePlan` and `UpdateRatePlanRules`) with a
  TODO comment in the SQL — confirm which to keep and remove the other.
- Keycloak `auth.enabled: false` toggle works for local dev — confirm whether this
  is the intended pattern for CI/test environments or if a separate test config is needed.

---

## Known Deferred Items

- GraphQL subscriptions — not implemented, not currently required
- Role-based authorisation on individual endpoints — auth middleware validates the token
  but no per-endpoint role checks are implemented yet
- Wholesaler admin endpoints — wholesaler table exists in DB, no GraphQL resource yet
- Subscriber admin endpoints — subscriber table exists in DB, no GraphQL resource yet
- Pagination optimisation — current implementation uses OFFSET which degrades on large
  datasets; cursor-based pagination not yet implemented
