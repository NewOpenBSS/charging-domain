# Project Status

_Updated by Claude Code at the end of every task. Source of truth for current implementation state._
_Last updated: 2026-03-31 (F-009 Task 004 complete)_

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
| `internal/backend/consumer` | ✅ Complete | SubscriberEventConsumer + WholesaleContractConsumer + QuotaProvisioningConsumer with dispatch logic, unit tests |
| F-005 wiring | ✅ Complete | Consumer wired into AppContext + main.go; subscriber-event topic in config |
| F-006 wiring | ✅ Complete | WholesaleContractConsumer wired into AppContext + main.go; wholesale-contract-event topic in config |
| F-007 wiring | ✅ Complete | QuotaProvisioningConsumer wired into AppContext + main.go; quota-provisioning topic in config |
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
| **Counter Expiry Journal** | ✅ Complete | QUOTA_EXPIRY events on counter expiry (F-008) |
| **DestinationGroupResource** | ✅ Complete | Full CRUD via GraphQL — six operations (F-002) |

### Database

| Component | Status | Notes |
|---|---|---|
| Schema migrations | ✅ Complete | 9 migrations applied |
| sqlc generated queries | ✅ Complete | All resources have static sqlc queries |
| Seed data | ✅ Complete | Test carriers (NZ, NG, ZA), home network config |

---

### charging-housekeeping (`cmd/charging-housekeeping` — F-009)

| Component | Status | Notes |
|---|---|---|
| Housekeeping SQL queries | ✅ Complete | 5 new sqlc queries: FindExpiredQuotaSubscribers, DeleteStaleChargingData, DeleteOldChargingTrace, ListSupersededRatePlanVersions, DeleteRatePlanVersionById |
| QuotaManager.ProcessExpiredQuota | ✅ Complete | Delegates to executeWithQuota with no-op; tested |
| HousekeepingService | ✅ Complete | CleanStaleSessions, PurgeOldTraces, CleanupSupersededRatePlans; tested |
| charging-housekeeping binary | ✅ Complete | Run-once CronJob binary with 4 tasks; tested |

---

## Current Focus

No active feature. F-003 and F-009 both merged. Next up: F-004 (GraphQL API Test Files) or a new feature.

---

## Recently Completed

- **F-009 Task 001** — SQL queries and sqlc regeneration for housekeeping operations. 5 new queries added. Completed 2026-03-31.
- **F-008** — Counter Expiry Cleanup with Quota Journal. `QUOTA_EXPIRY` journal events on counter expiry. Merged 2026-03-29.
- **F-002 Task 003** — DestinationGroupService, resolvers, and AppContext wiring. All six GraphQL operations functional. Completed 2026-03-29.
- **F-001 Task 003** — Resolvers, AppContext, and GraphQL router wired for ChargingTrace. All three query methods functional end-to-end. Completed 2026-03-20.

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
