# Task 001 — Wholesaler SQL queries and sqlc regeneration

**Feature:** F-006 — WholesaleContractConsumer
**Sequence:** 1 of 5
**Date:** 2026-03-21
**Status:** Active

---

## Objective

Add the five SQL queries the WholesaleContractConsumer needs to `internal/store/queries/wholesaler.sql`
and regenerate the sqlc type-safe Go layer. The new queries cover: UPSERT on provisioning, set-active
flag on deregistering/suspend, hard delete, count-by-wholesaler for deregistering logic, and an atomic
conditional delete used for the subscriber cascade. This task is purely additive — no application code
changes, no wiring.

---

## Scope

**In scope:**
- Add to `internal/store/queries/wholesaler.sql`:
  - `UpsertWholesaler :exec` — INSERT … ON CONFLICT (id) DO UPDATE for all DB-mapped fields
  - `SetWholesalerActive :exec` — UPDATE active = $2 WHERE id = $1
  - `DeleteWholesaler :exec` — hard DELETE WHERE id = $1
  - `CountSubscribersByWholesaler :one` — SELECT COUNT(*) FROM subscriber WHERE wholesale_id = $1
  - `DeleteInactiveWholesalerIfEmpty :exec` — atomic DELETE WHERE id = $1 AND active = false AND (SELECT COUNT(*) FROM subscriber WHERE wholesale_id = $1) = 0
- Run `sqlc generate` to regenerate `internal/store/sqlc/`
- Verify `go build ./...` and `go test ./...` pass

**Out of scope:**
- Any application code, consumers, event types, or wiring — that is in later tasks
- New database migrations — the `wholesaler` table already exists

---

## Context

- Wholesaler table schema (from `db/migrations/`): `id uuid PK`, `modified_on TIMESTAMPTZ`, `active boolean`,
  `legal_name varchar`, `display_name varchar`, `realm varchar`, `hosts text[]`, `nchfUrl varchar`,
  `rateLimit numeric`, `contract_id uuid`, `rateplan_id uuid`
- Existing query in `internal/store/queries/wholesaler.sql`: `AllWholesalers` — do NOT remove it
- Subscriber table has column `wholesale_id uuid` used in `CountSubscribersByWholesaler`
- sqlc config in `sqlc.yaml`: `query_parameter_limit: 4` — queries with >4 params produce a `*Params` struct
- Pattern reference: `internal/store/queries/subscriber.sql` for single-exec and named struct patterns
- `sqlc generate` command: `sqlc generate` (requires sqlc tool on PATH)

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| `UpsertWholesaler` uses INSERT … ON CONFLICT DO UPDATE | Idempotent handling of re-provisioning events without explicit check-then-insert |
| `modified_on` updated to NOW() in upsert | Keeps the modified timestamp accurate for all write paths |
| `DeleteInactiveWholesalerIfEmpty` is a single atomic SQL statement | Prevents race conditions between the count check and delete in the application layer |
| No new DB migration required | The wholesaler table already contains all required columns |

---

## Acceptance Criteria

- [ ] `internal/store/queries/wholesaler.sql` contains all five new named queries
- [ ] `sqlc generate` completes without errors
- [ ] `internal/store/sqlc/` regenerated files include `UpsertWholesaler`, `SetWholesalerActive`, `DeleteWholesaler`, `CountSubscribersByWholesaler`, `DeleteInactiveWholesalerIfEmpty`
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

---

## Risk Assessment

Low. SQL-only change. The wholesaler table exists and no existing queries are modified.
`DeleteInactiveWholesalerIfEmpty` uses a subquery with a correlated count — verify it produces the
correct plan by reading the generated Go carefully. No charging, quota, or rating logic is affected.

---

## Notes

The sqlc `query_parameter_limit: 4` setting means `UpsertWholesaler` will generate a
`UpsertWholesalerParams` struct (11 fields). This is expected and follows the same pattern as
`InsertSubscriberParams` in `internal/store/sqlc/`.
