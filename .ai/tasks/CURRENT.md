# Task: F-001 ChargingTrace — Store Layer

**Date:** 2026-03-20
**Status:** Active
**Feature:** F-001 — ChargingTraceResource

---

## Objective

Add the database query and store-layer infrastructure needed to serve charging trace
data through the GraphQL API. The `charging_trace` table and its write-path queries
already exist; this task adds a read-by-trace-id sqlc query and the two dynamic store
methods (`ListChargingTraces`, `CountChargingTraces`) that the service layer will call.
No business logic or GraphQL changes are part of this task.

---

## Scope

**In scope:**
- New sqlc query `FindChargingTraceByTraceId` in `internal/store/queries/charging_trace.sql`
- Regenerate sqlc (`sqlc generate`) to produce updated `internal/store/sqlc/charging_trace.sql.go`
- New `internal/store/charging_trace_store.go` with `ListChargingTraces` and `CountChargingTraces`
  following the pattern of `internal/store/carrier_store.go`
- Unit tests for the new store methods (using the mock/stub pattern from existing store tests)

**Out of scope:**
- GraphQL schema changes
- Service or resolver implementation
- Any changes to the write-path queries (`CreateChargingTrace`, `FindChargingTraceByIdSeqNr`)

---

## Context

- **Existing table:** `charging_trace` — defined in `db/migrations/000001_init.up.sql`
  - Columns: `trace_id UUID PK`, `created_at TIMESTAMPTZ`, `request JSONB`, `response JSONB`,
    `execution_time BIGINT`, `charging_id VARCHAR`, `sequence_nr INTEGER`, `msisdn VARCHAR`
- **Existing sqlc queries:** `internal/store/queries/charging_trace.sql`
  — already has `FindChargingTraceByIdSeqNr` and `CreateChargingTrace`
- **Pattern to follow:** `internal/store/carrier_store.go` — `ListCarriers` and `CountCarriers`
  show the exact pattern for dynamic WHERE/ORDER BY construction
- **sqlc config:** `sqlc.yaml` at project root — run `sqlc generate` after adding the new query
- **New query required:**
  ```sql
  -- name: FindChargingTraceByTraceId :one
  SELECT trace_id, created_at, request, response, execution_time,
         charging_id, sequence_nr, msisdn
  FROM charging_trace
  WHERE trace_id = $1
  ```
- **Store params struct** should follow the same shape as `ListCarriersParams`:
  `WhereSQL string`, `Args []any`, `OrderSQL string`, `Limit int`, `Offset int`
- **Wildcard filter columns** for list/count: `charging_id`, `msisdn`
  (matching the Feature acceptance criteria for wildcard match)

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| Add `FindChargingTraceByTraceId` as a new sqlc query | The existing `FindChargingTraceByIdSeqNr` queries by charging_id + sequence_nr; the GraphQL `chargingTraceById` operation must fetch by `traceId` (UUID primary key) — a separate query is required |
| Dynamic store methods for list/count | Consistent with all other resources; centralised filter builder prevents SQL injection |
| Separate task from GraphQL layer | Isolates DB changes from schema changes; allows sqlc generation to complete before gqlgen needs the store interface |

---

## Acceptance Criteria

- [ ] `internal/store/queries/charging_trace.sql` has `FindChargingTraceByTraceId :one` query
- [ ] `sqlc generate` runs cleanly and `internal/store/sqlc/charging_trace.sql.go` contains the new method
- [ ] `internal/store/charging_trace_store.go` exists with `ListChargingTraces` and `CountChargingTraces`
- [ ] Both store methods accept a params struct with `WhereSQL`, `Args`, `OrderSQL`, `Limit`, `Offset`
- [ ] Unit tests cover `ListChargingTraces` and `CountChargingTraces` (success, empty result, filter applied)
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

---

## Risk Assessment

None. This task only adds new read-path queries and store methods. No existing write-path
code (`CreateChargingTrace`, charging pipeline) is touched. No charging, quota, or rating
logic is affected.

---

## Notes

- `request` and `response` columns are `JSONB` — in the sqlc generated struct they appear
  as `[]byte`. The service layer (Task 2) is responsible for converting these to JSON strings
  for the GraphQL response.
- Wildcard columns for filter should be `charging_id` and `msisdn` — matching the Feature
  acceptance criteria (filter by `chargingId` or `msisdn` with wildcard match).
