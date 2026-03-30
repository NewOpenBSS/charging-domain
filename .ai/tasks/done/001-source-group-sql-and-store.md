# Task 001 — SQL Queries, sqlc Generation, Store Layer, and Store Tests

**Feature:** F-003 — SourceGroupResource
**Sequence:** 001 of 003
**Date:** 2026-03-30
**Status:** Active

---

## Objective

Write the static SQL query file for `carrier_source_group`, run `sqlc generate` to produce
the type-safe Go wrappers, implement the dynamic `ListSourceGroups` and `CountSourceGroups`
store methods, and verify everything with unit tests. This task establishes the full
persistence layer that Tasks 002 and 003 depend on.

---

## Scope

**In scope:**
- `internal/store/queries/source_groups.sql` — four named sqlc queries mirroring `destination_groups.sql`
- Run `sqlc generate` to regenerate `internal/store/sqlc/source_groups.sql.go` and update `internal/store/sqlc/models.go`
- `internal/store/source_group_store.go` — `ListSourceGroupsParams`, `ListSourceGroups`, `CountSourceGroups` — mirrors `destination_group_store.go` exactly, replacing table and type names
- `internal/store/source_group_store_test.go` — unit tests covering list (no filter, with filter, empty result) and count (matching and non-matching filter)
- `go build ./...` and `go test ./...` must pass

**Out of scope:**
- GraphQL schema or gqlgen generation (Task 002)
- Service layer (Task 002)
- Resolver wiring (Task 003)

---

## Context

The `carrier_source_group` table already exists — it was created in the initial migration
`db/migrations/000001_init.up.sql` with the identical schema to `carrier_destination_group`:

```sql
create table carrier_source_group
(
    group_name varchar not null
        constraint carrier_source_group_pk
            primary key,
    region     varchar not null
);
```

Seed data is in `db/migrations/000005_source_groups.up.sql`.

**Pattern to follow exactly:** `internal/store/destination_group_store.go` and
`internal/store/queries/destination_groups.sql`. Every name change is mechanical:

| DestinationGroup → SourceGroup     | carrier_destination_group → carrier_source_group |
|-------------------------------------|--------------------------------------------------|
| `ListDestinationGroupsParams`       | `ListSourceGroupsParams`                         |
| `ListDestinationGroups`             | `ListSourceGroups`                               |
| `CountDestinationGroups`            | `CountSourceGroups`                              |
| `CarrierDestinationGroup`           | `CarrierSourceGroup`                             |
| `DestinationGroupByGroupName`       | `SourceGroupByGroupName`                         |
| `CreateDestinationGroup`            | `CreateSourceGroup`                              |
| `UpdateDestinationGroup`            | `UpdateSourceGroup`                              |
| `DeleteDestinationGroup`            | `DeleteSourceGroup`                              |

**sqlc config:** `sqlc.yaml` in the repo root controls generation — read it to confirm
query file location and output paths before writing the `.sql` file. The generated output
goes to `internal/store/sqlc/`.

**Store test pattern:** Follow `internal/store/destination_group_store_test.go` — uses the
`servicesMockDBTX` pattern with `store.NewTestStore`.

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| No new migration required | `carrier_source_group` table already exists in `000001_init.up.sql` |
| Mechanically mirror destination_group_store.go | F-003 constraint: follow DestinationGroupResource pattern exactly |

---

## Acceptance Criteria

- [ ] `internal/store/queries/source_groups.sql` exists with four named queries: `SourceGroupByGroupName`, `CreateSourceGroup`, `UpdateSourceGroup`, `DeleteSourceGroup`
- [ ] `sqlc generate` runs without errors and produces `internal/store/sqlc/source_groups.sql.go` with the four generated methods
- [ ] `internal/store/sqlc/models.go` contains a `CarrierSourceGroup` struct with `GroupName` and `Region` fields
- [ ] `internal/store/source_group_store.go` implements `ListSourceGroupsParams`, `ListSourceGroups`, and `CountSourceGroups`
- [ ] `internal/store/source_group_store_test.go` covers: list with no filter, list with WHERE filter, list returning empty slice, count matching filter, count non-matching filter
- [ ] `go build ./...` passes clean
- [ ] `go test ./...` passes clean (including race detector: `go test -race ./...`)

---

## Risk Assessment

None. This task is purely additive — new files only, no existing code modified.
The `carrier_source_group` table is already present in the DB schema.

---

## Notes

Run `sqlc generate` after writing `source_groups.sql` and before writing any Go code
that imports the generated types — otherwise the build will fail on missing symbols.
