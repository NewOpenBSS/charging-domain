# Task 002 — GraphQL Schema, gqlgen Generation, Service Layer, and Service Tests

**Feature:** F-003 — SourceGroupResource
**Sequence:** 002 of 003
**Date:** 2026-03-30
**Status:** Active

---

## Objective

Write the GraphQL schema for `SourceGroup`, run `gqlgen generate` to produce model types
and resolver stubs, implement `SourceGroupService` with all six operations, and verify
the service with unit tests. This task establishes the GraphQL contract and business logic
layer that Task 003 wires into the running application.

---

## Scope

**In scope:**
- `gql/schema/source_group.graphql` — `SourceGroup` type, `SourceGroupInput`, and six operations (3 queries + 3 mutations) matching the Java operation names exactly
- Run `gqlgen generate` to regenerate `internal/backend/graphql/generated/generated.go`,  `internal/backend/graphql/model/models_gen.go`, and create `internal/backend/resolvers/source_group.resolvers.go` (stub file)
- `internal/backend/services/source_group_service.go` — `SourceGroupService` with `ListSourceGroups`, `CountSourceGroups`, `SourceGroupByGroupName`, `CreateSourceGroup`, `UpdateSourceGroup`, `DeleteSourceGroup`
- `internal/backend/services/source_group_service_test.go` — unit tests for all six service methods covering success and error paths
- `go build ./...` and `go test ./...` must pass

**Out of scope:**
- AppContext wiring (Task 003)
- Resolver root wiring (Task 003)
- GraphQL router wiring (Task 003)
- `api-tests/` HTTP file (Task 003)

---

## Context

**GraphQL schema pattern:** Mirror `gql/schema/destination_group.graphql` exactly, substituting
all `DestinationGroup` → `SourceGroup` and `destinationGroup` → `sourceGroup` names. The
field names and types on the type itself (`groupName: String!`, `region: String!`) are identical.

Required Java-compatible operation names (from F-003 acceptance criteria):
- Queries: `sourceGroupList`, `countSourceGroup`, `sourceGroupByGroupName`
- Mutations: `createSourceGroup`, `updateSourceGroup`, `deleteSourceGroup`

**gqlgen:** After writing the schema, run:
```bash
cd /path/to/repo && go run github.com/99designs/gqlgen generate
```
This regenerates `internal/backend/graphql/generated/generated.go` (do not edit manually)
and `internal/backend/graphql/model/models_gen.go` (do not edit manually), and creates a
stub resolver file `internal/backend/resolvers/source_group.resolvers.go`.

The stub resolver file will have empty method bodies. Do NOT fill them in yet — that is
Task 003. Simply confirm it compiles.

**Service pattern:** Follow `internal/backend/services/destination_group_service.go` exactly.
Key elements to replicate:
- `sourceGroupColumns` map (camelCase → snake_case)
- `sourceGroupWildcardCols` slice (`["group_name", "region"]`)
- Column map and wildcard cols identical to destination group (same columns, different table)
- Default sort column: `"group_name"`
- `sourceGroupToModel` helper converts `sqlc.CarrierSourceGroup` → `*model.SourceGroup`
- `UpdateSourceGroup` passes `(groupName, input.Region)` to `store.Q.UpdateSourceGroup` — same signature as destination group
- `CreateSourceGroup` passes `(input.GroupName, input.Region)`
- `ByGroupName`, `Delete` use `store.Q` directly (static sqlc queries)
- `List` and `Count` use dynamic store methods from Task 001

**Service test pattern:** Follow `internal/backend/services/destination_group_service_test.go`.
Uses `store.NewTestStore` with `servicesMockDBTX` to mock the `pgxpool.Pool`-level interface.
Tests must cover:
- `ListSourceGroups`: success (rows returned), empty result, store error
- `CountSourceGroups`: success, store error
- `SourceGroupByGroupName`: found, not found
- `CreateSourceGroup`: success, duplicate key error
- `UpdateSourceGroup`: success, not found error
- `DeleteSourceGroup`: success, not found error

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| Wildcard columns identical to destination group | `carrier_source_group` has the same columns — `group_name` and `region` — no difference |
| Stub resolvers left empty after gqlgen | Task 003 fills them in with `r.SourceGroupSvc` calls; implementing them here would require wiring that is Task 003's scope |

---

## Acceptance Criteria

- [ ] `gql/schema/source_group.graphql` defines `SourceGroup` type, `SourceGroupInput`, and all six operations with Java-matching names
- [ ] `gqlgen generate` produces `internal/backend/resolvers/source_group.resolvers.go` with stub method bodies
- [ ] `internal/backend/graphql/model/models_gen.go` contains `SourceGroup` and `SourceGroupInput` structs
- [ ] `internal/backend/services/source_group_service.go` implements all six service methods
- [ ] `internal/backend/services/source_group_service_test.go` covers success and error paths for all six methods
- [ ] `go build ./...` passes clean (stub resolvers return zero values — acceptable for this task)
- [ ] `go test ./...` passes clean including race detector

---

## Risk Assessment

Low. gqlgen generation is an additive code-gen step — the `generated.go` file will be
regenerated wholesale but existing resources are unaffected as long as their schema files
are unchanged. The only risk is a gqlgen version mismatch; if generation fails, check
`gql/gqlgen.yml` for the expected tool version and run via `go run github.com/99designs/gqlgen`.

---

## Notes

The stub resolver file produced by gqlgen will have compile errors if `r.SourceGroupSvc`
is referenced but not yet added to the `Resolver` struct. Leave the stub bodies as `panic("not implemented")`
or returning zero values — gqlgen generates them this way. Task 003 fills them in properly.
