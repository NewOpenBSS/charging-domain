# Task 001 — sqlc queries and store dynamic methods for DestinationGroup

**Feature:** F-002 — DestinationGroupResource
**Sequence:** 001 of 003
**Date:** 2026-03-20
**Status:** Active

---

## Objective

Write the sqlc SQL query file for all `carrier_destination_group` operations, run `sqlc generate`
to produce the Go typed query code, then add the dynamic `ListDestinationGroups` and
`CountDestinationGroups` methods to the store layer.  This is purely a persistence layer task —
no GraphQL or service code is written here.

---

## Scope

**In scope:**
- `internal/store/queries/destination_groups.sql` — four named queries:
  `DestinationGroupByGroupName` (:one), `CreateDestinationGroup` (:one),
  `UpdateDestinationGroup` (:one), `DeleteDestinationGroup` (:exec)
- Run `sqlc generate` — updates `internal/store/sqlc/destination_groups.sql.go`
  and verifies `internal/store/sqlc/models.go` already has `CarrierDestinationGroup`
- `internal/store/destination_group_store.go` — `ListDestinationGroups` and
  `CountDestinationGroups` dynamic methods on `*Store`
- `internal/store/destination_group_store_test.go` — unit tests for both dynamic methods
  (success, DB error cases) using the `servicesMockDBTX` pattern from `carrier_store`
- `go build ./...` and `go test -race ./...` must pass

**Out of scope:**
- GraphQL schema, gqlgen codegen, service layer, resolvers, AppContext wiring

---

## Context

- **DB table:** `carrier_destination_group` — columns `group_name` (varchar PK), `region` (varchar NOT NULL).
  No `modified_on` column. Table created in `db/migrations/000001_init.up.sql`.
- **Existing sqlc model:** `sqlc.CarrierDestinationGroup` already exists in
  `internal/store/sqlc/models.go` — do NOT add it again; `sqlc generate` will regenerate it.
- **SQL pattern:** Follow `internal/store/queries/carrriers.sql` — use `RETURNING` on INSERT/UPDATE.
  `DeleteDestinationGroup` is `:exec` with no RETURNING.
- **sqlc generation command:** `sqlc generate` (uses `sqlc.yaml` at repo root).
- **Dynamic store pattern:** Follow `internal/store/carrier_store.go` exactly.
  `ListDestinationGroups` builds a SELECT with WHERE/ORDER/LIMIT/OFFSET.
  `CountDestinationGroups` builds a `SELECT COUNT(*)`.
  Both accept `ListDestinationGroupsParams` (WhereSQL, Args, OrderSQL, Limit, Offset).
- **Test pattern:** Follow `internal/store/charging_trace_store_test.go` or
  `internal/store/carrier_store.go` tests — use `store.NewTestStore` with a mock querier.
- `query_parameter_limit: 4` in sqlc.yaml means queries with >4 params will use a struct arg
  (sqlc auto-generates `CreateDestinationGroupParams`, `UpdateDestinationGroupParams`).

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| No `modified_on` on destination group | The DB table has no such column — do not add one |
| Use dynamic List/Count (not sqlc-generated) | Consistent with all other list resources; supports arbitrary filter/sort from the filter package |
| `DestinationGroupByGroupName` as the single-row query name | Matches the Java API operation name `destinationGroupByGroupName` |

---

## Acceptance Criteria

- [ ] `internal/store/queries/destination_groups.sql` exists with four named queries
- [ ] `sqlc generate` runs cleanly and produces `internal/store/sqlc/destination_groups.sql.go`
- [ ] `internal/store/destination_group_store.go` exists with `ListDestinationGroups` and `CountDestinationGroups`
- [ ] `internal/store/destination_group_store_test.go` exists and covers success + error paths for both dynamic methods
- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes

---

## Risk Assessment

None — this task adds new store code for a new table only. No existing charging, quota, or rating
logic is modified.

---

## Notes

The sqlc model `CarrierDestinationGroup` is already generated (it appears in `models.go`).
Running `sqlc generate` will regenerate `models.go` and add `destination_groups.sql.go`.
Verify the model struct hasn't changed after generation before proceeding.
