# Task 001 — SQL queries and sqlc regeneration for housekeeping operations

**Feature:** F-009 — Charging Domain Housekeeping
**Sequence:** 001 of 004
**Date:** 2026-03-30
**Status:** Active

---

## Objective

Add five new static SQL queries to the existing sqlc query files covering all four
housekeeping operations, regenerate the sqlc layer, and verify the build is clean.
These generated functions are the foundation on which Tasks 002–004 depend.

---

## Scope

**In scope:**
- Add `FindExpiredQuotaSubscribers :many` to `internal/store/queries/quota.sql`
- Add `DeleteStaleChargingData :execrows` to `internal/store/queries/charging_data.sql`
- Add `DeleteOldChargingTrace :execrows` to `internal/store/queries/charging_trace.sql`
- Add `ListSupersededRatePlanVersions :many` to `internal/store/queries/rateplan.sql`
- Add `DeleteRatePlanVersionById :exec` to `internal/store/queries/rateplan.sql`
- Run `sqlc generate` to regenerate `internal/store/sqlc/`
- Run `go build ./...` to verify no compilation errors

**Out of scope:**
- Store wrapper methods (not needed — all new queries are static, called via `store.Q.XXX`)
- Any application logic, services, or binaries (Tasks 002–004)
- Unit tests (sqlc-generated code is not tested directly; business logic tests are in later tasks)

---

## Context

- SQL files live in `internal/store/queries/`; sqlc-generated Go code is in `internal/store/sqlc/`
- Run sqlc with: `sqlc generate` (requires `sqlc.yaml` at repo root, already configured)
- All queries are static (no dynamic SQL) — the generated `store.Q.XXX` methods are called
  directly by service and manager code; no store wrapper methods are required
- `quota.next_action_time TIMESTAMPTZ` — the column used to find expired quotas
- `charging_data.modified_on TIMESTAMPTZ` — the column used to find stale sessions
- `charging_trace.created_at TIMESTAMPTZ` — the column used to find old trace records
- `rateplan.effective_at TIMESTAMPTZ` — the column used to identify superseded rate plans
- `rateplan.plan_status VARCHAR` — guards on ACTIVE status; DRAFT and PENDING must never be deleted
- `rateplan.id BIGSERIAL` — primary key used to delete a specific superseded version

**Exact SQL to add:**

`quota.sql` — add after the existing `CreateQuota` query:
```sql
-- name: FindExpiredQuotaSubscribers :many
-- Returns the subscriber_id for every quota row whose next_action_time is in the past
-- relative to the given reference time. Used by the housekeeping job to find dormant
-- subscribers with expired counters.
SELECT subscriber_id
FROM quota
WHERE next_action_time < $1;
```

`charging_data.sql` — add after the existing `DeleteChargeDate` query:
```sql
-- name: DeleteStaleChargingData :execrows
-- Deletes all charging_data rows whose modified_on is before the given threshold.
-- Used by the housekeeping job to remove orphaned sessions.
DELETE FROM charging_data
WHERE modified_on < $1;
```

`charging_trace.sql` — add after the existing `FindChargingTraceByTraceId` query:
```sql
-- name: DeleteOldChargingTrace :execrows
-- Deletes all charging_trace rows whose created_at is before the given threshold.
-- Used by the housekeeping job to purge old audit trail records.
DELETE FROM charging_trace
WHERE created_at < $1;
```

`rateplan.sql` — add after the existing `LatestRatePlanByType` query:
```sql
-- name: ListSupersededRatePlanVersions :many
-- Returns ACTIVE rate plan versions that have been superseded by a newer ACTIVE version
-- for the same plan_id, where the older version's effective_at is before the given
-- threshold. Used by the housekeeping job to identify versions safe to delete.
SELECT r1.id,
       r1.plan_id,
       r1.modified_at,
       r1.plan_type,
       r1.wholesale_id,
       r1.plan_name,
       r1.rateplan,
       r1.plan_status,
       r1.created_by,
       r1.approved_by,
       r1.effective_at
FROM rateplan r1
WHERE r1.plan_status = 'ACTIVE'
  AND r1.effective_at < $1
  AND EXISTS (SELECT 1
              FROM rateplan r2
              WHERE r2.plan_id = r1.plan_id
                AND r2.plan_status = 'ACTIVE'
                AND r2.effective_at > r1.effective_at)
ORDER BY r1.plan_id, r1.effective_at ASC;

-- name: DeleteRatePlanVersionById :exec
-- Permanently deletes a specific superseded ACTIVE rate plan version by its primary key id.
-- The plan_status = 'ACTIVE' guard ensures DRAFT and PENDING versions can never be deleted
-- by this query, even if called with the wrong id by mistake.
DELETE FROM rateplan
WHERE id          = $1
  AND plan_status = 'ACTIVE';
```

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| Use static sqlc queries (not dynamic store methods) | All new queries have fixed WHERE clauses with a single timestamp parameter — no runtime column selection needed |
| `DeleteStaleChargingData` returns `:execrows` | Allows the caller to log how many rows were deleted without a separate COUNT query |
| `DeleteOldChargingTrace` returns `:execrows` | Same reason as above |
| `ListSupersededRatePlanVersions` + `DeleteRatePlanVersionById` (two queries, not one DELETE) | Separating list from delete allows the housekeeping service to log which plan IDs are being deleted before executing; also simplifies testing |
| `AND plan_status = 'ACTIVE'` guard on `DeleteRatePlanVersionById` | Safety net: prevents accidental deletion of DRAFT or PENDING versions even if this query is called with the wrong id |

---

## Acceptance Criteria

- [ ] All five SQL queries are added to the correct `.sql` files with accurate doc comments
- [ ] `sqlc generate` completes without errors
- [ ] The five new generated Go functions are present in `internal/store/sqlc/`
- [ ] `go build ./...` passes cleanly with no compilation errors

---

## Risk Assessment

The new SQL queries are read-only (SELECT) or scoped deletes with a timestamp condition.
`DeleteRatePlanVersionById` carries a `plan_status = 'ACTIVE'` guard — it cannot delete
DRAFT or PENDING versions by design. The `ListSupersededRatePlanVersions` self-join
correctly uses `effective_at >` (not `>=`) to compare versions, so a plan with only one
ACTIVE version is never returned. No existing business logic is touched in this task.

---

## Notes

- `sqlc.yaml` at repo root already references `internal/store/queries/` — no config change needed
- The sqlc-generated file `internal/store/sqlc/db.go` must not be manually edited
- If `sqlc generate` fails due to schema mismatch, check `db/migrations/` for the correct column names

