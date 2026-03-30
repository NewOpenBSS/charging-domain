# Task 003 — Housekeeping service: stale sessions, trace purge, rate plan cleanup

**Feature:** F-009 — Charging Domain Housekeeping
**Sequence:** 003 of 004
**Date:** 2026-03-30
**Status:** Active

---

## Objective

Create the `internal/housekeeping/` package containing a `HousekeepingService` that
implements the three non-quota housekeeping operations: deleting stale charging sessions,
purging old trace records, and removing superseded ACTIVE rate plan versions. Each method
calls the sqlc-generated store functions added in Task 001 and returns the count of rows
affected for summary logging.

---

## Scope

**In scope:**
- Create `internal/housekeeping/service.go` — `HousekeepingService` struct with three methods
- Create `internal/housekeeping/service_test.go` — unit tests using mocked store
- `go build ./...` and `go test -race ./...` must pass

**Out of scope:**
- Quota expiry (handled by `QuotaManager.ProcessExpiredQuota` — Task 002)
- The binary entrypoint and config (Task 004)
- Any Kafka or HTTP dependencies — this service only needs the store

---

## Context

**Package location:** `internal/housekeeping/` — new package, does not exist yet

**Store access pattern:**
The service receives a `*store.Store` and calls sqlc-generated methods directly via
`store.Q.XXX(ctx, ...)`. No dynamic store wrapper methods are needed for these operations.

The three sqlc functions available after Task 001:
- `store.Q.DeleteStaleChargingData(ctx, beforeTime pgtype.Timestamptz) (int64, error)` — `:execrows`
- `store.Q.DeleteOldChargingTrace(ctx, beforeTime pgtype.Timestamptz) (int64, error)` — `:execrows`
- `store.Q.ListSupersededRatePlanVersions(ctx, beforeTime pgtype.Timestamptz) ([]sqlc.Rateplan, error)`
- `store.Q.DeleteRatePlanVersionById(ctx, id int64) error`

**Note on pgtype.Timestamptz:** sqlc maps `TIMESTAMPTZ` parameters to `pgtype.Timestamptz`.
Convert a `time.Time` to `pgtype.Timestamptz` using:
```go
pgtype.Timestamptz{Time: t, Valid: true}
```

**Test mocking pattern:**
Follow the existing pattern in `internal/store/store.go` — `NewTestStore(dbtx, querier)`.
Use a simple mock struct implementing the `sqlc.DBTX` interface and `store.DBQuerier` interface,
or use `gomock` if already used in the project. Check `internal/backend/services/` for the
established mocking pattern.

**Method signatures for `HousekeepingService`:**

```go
// Store is the minimal interface consumed by HousekeepingService. Defined at point of
// consumption per Go interface conventions.
type Store interface {
    // ... narrow interface with only the methods HousekeepingService needs
}

type HousekeepingService struct {
    store *store.Store
}

func NewHousekeepingService(s *store.Store) *HousekeepingService

// CleanStaleSessions deletes charging_data rows with modified_on older than the threshold
// (i.e. before now - threshold). Returns the number of rows deleted.
func (s *HousekeepingService) CleanStaleSessions(ctx context.Context, now time.Time, threshold time.Duration) (int64, error)

// PurgeOldTraces deletes charging_trace rows with created_at older than the threshold
// (i.e. before now - threshold). Returns the number of rows deleted.
func (s *HousekeepingService) PurgeOldTraces(ctx context.Context, now time.Time, threshold time.Duration) (int64, error)

// CleanupSupersededRatePlans deletes superseded ACTIVE rate plan versions whose effective_at
// is older than the threshold and which have been replaced by a newer ACTIVE version for the
// same plan_id. DRAFT and PENDING versions are never touched. Returns the number of rows deleted.
func (s *HousekeepingService) CleanupSupersededRatePlans(ctx context.Context, now time.Time, threshold time.Duration) (int64, error)
```

**CleanStaleSessions implementation outline:**
```go
func (s *HousekeepingService) CleanStaleSessions(ctx context.Context, now time.Time, threshold time.Duration) (int64, error) {
    before := pgtype.Timestamptz{Time: now.Add(-threshold), Valid: true}
    n, err := s.store.Q.DeleteStaleChargingData(ctx, before)
    if err != nil {
        return 0, fmt.Errorf("delete stale charging data: %w", err)
    }
    return n, nil
}
```

**CleanupSupersededRatePlans implementation outline:**
```go
func (s *HousekeepingService) CleanupSupersededRatePlans(ctx context.Context, now time.Time, threshold time.Duration) (int64, error) {
    before := pgtype.Timestamptz{Time: now.Add(-threshold), Valid: true}
    versions, err := s.store.Q.ListSupersededRatePlanVersions(ctx, before)
    if err != nil {
        return 0, fmt.Errorf("list superseded rate plan versions: %w", err)
    }
    var deleted int64
    for _, v := range versions {
        logging.Info("Deleting superseded rate plan version", "id", v.ID, "plan_id", v.PlanID, "effective_at", v.EffectiveAt)
        if err := s.store.Q.DeleteRatePlanVersionById(ctx, v.ID); err != nil {
            return deleted, fmt.Errorf("delete rate plan version %d: %w", v.ID, err)
        }
        deleted++
    }
    return deleted, nil
}
```

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| Accept `now time.Time` and `threshold time.Duration` as parameters, not absolute `before time.Time` | Matches the feature spec ("older than a configurable threshold"); enables deterministic testing by injecting `now` |
| Log each rate plan version before deleting | Rate plan deletion is high-value data loss; a log entry per version aids audit and recovery |
| Return first error on rate plan loop and stop | Partial deletion is undesirable for rate plans; fail fast and let the binary report the error |
| `fmt.Errorf` for infrastructure wrapping | Go standard: `fmt.Errorf` is permitted for wrapping DB/infrastructure errors |
| Do NOT define `Store` as an interface in this package | The service holds `*store.Store` directly; mock via `NewTestStore` in tests (established project pattern) |

---

## Acceptance Criteria

- [ ] `internal/housekeeping/service.go` exists with `HousekeepingService`, `NewHousekeepingService`,
  and three exported methods with Go doc comments
- [ ] Unit tests cover:
  - `CleanStaleSessions` — success path (N rows deleted), store error is propagated
  - `PurgeOldTraces` — success path (N rows deleted), store error is propagated
  - `CleanupSupersededRatePlans` — success path with multiple versions, stop-on-first-error behaviour,
    zero versions returns 0 with no error
- [ ] All tests use table-driven structure where multiple input/output combinations are tested
- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes

---

## Risk Assessment

**Low risk.** This task introduces a new package with no coupling to existing charging pipeline
code. The `CleanStaleSessions` and `PurgeOldTraces` methods issue bulk deletes scoped by
timestamp — they cannot affect rows that are newer than the threshold. The `CleanupSupersededRatePlans`
method issues targeted single-row deletes, each guarded by `plan_status = 'ACTIVE'` in the SQL
(from Task 001). DRAFT and PENDING rate plans cannot be deleted by this path. The logging before
each rate plan delete provides an audit trail. No quota, charging, or financial logic is touched.

---

## Notes

- The `pgtype` package is already used throughout the store layer — no new dependency needed
- `internal/housekeeping/` will be a new top-level package under `internal/`; follow the
  naming convention of other `internal/` packages

