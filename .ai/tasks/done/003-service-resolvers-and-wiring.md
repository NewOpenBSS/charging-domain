# Task 003 — DestinationGroupService, resolver implementations, and AppContext wiring

**Feature:** F-002 — DestinationGroupResource
**Sequence:** 003 of 003
**Date:** 2026-03-20
**Status:** Active

---

## Objective

Implement `DestinationGroupService` with all six CRUD methods, fill in the generated resolver
stubs, and wire the service through `AppContext`, `Resolver`, and the GraphQL router.  After this
task all six GraphQL operations are fully functional end-to-end.

---

## Scope

**In scope:**
- `internal/backend/services/destination_group_service.go` — `DestinationGroupService` struct
  with six methods: `ListDestinationGroups`, `CountDestinationGroups`,
  `DestinationGroupByGroupName`, `CreateDestinationGroup`, `UpdateDestinationGroup`,
  `DeleteDestinationGroup`
- `internal/backend/services/destination_group_service_test.go` — unit tests for all six methods,
  covering success and error paths, using `store.NewTestStore` with `servicesMockDBTX`
- Fill in `internal/backend/resolvers/destination_group.resolvers.go` — replace stubs with real
  delegation to `r.DestinationGroupSvc`
- `internal/backend/resolvers/resolver.go` — add `DestinationGroupSvc *services.DestinationGroupService`
- `internal/backend/appcontext/context.go` — add `DestinationGroupSvc *services.DestinationGroupService`
  field and wire `services.NewDestinationGroupService(s)` in `NewAppContext`
- `internal/backend/handlers/graphql/router.go` — add `DestinationGroupSvc: appCtx.DestinationGroupSvc`
  to the `Resolver` initialisation
- `go build ./...` and `go test -race ./...` must pass

**Out of scope:**
- DB migrations (table already exists from migration 000001)
- sqlc queries or store layer (done in Task 001)
- GraphQL schema or gqlgen codegen (done in Task 002)
- api-tests `.http` file (covered by F-004)

---

## Context

- **Service pattern:** Follow `internal/backend/services/carrier_service.go` exactly.
  - Define `destinationGroupColumns` map (camelCase → snake_case) for filter/sort validation
  - Define `destinationGroupWildcardCols` slice for wildcard filter search (group_name, region)
  - `ListDestinationGroups` calls `s.store.ListDestinationGroups(ctx, store.ListDestinationGroupsParams{...})`
  - `CountDestinationGroups` calls `s.store.CountDestinationGroups(ctx, where.SQL, where.Args)`
  - `DestinationGroupByGroupName` calls `s.store.Q.DestinationGroupByGroupName(ctx, groupName)`
  - `CreateDestinationGroup` calls `s.store.Q.CreateDestinationGroup(ctx, sqlc.CreateDestinationGroupParams{...})`
  - `UpdateDestinationGroup` calls `s.store.Q.UpdateDestinationGroup(ctx, sqlc.UpdateDestinationGroupParams{...})`
  - `DeleteDestinationGroup` calls `s.store.Q.DeleteDestinationGroup(ctx, groupName)` and returns `(bool, error)`
  - `destinationGroupToModel` maps `sqlc.CarrierDestinationGroup` → `*model.DestinationGroup`
- **No nullable fields / no `modified_on`:** `sqlc.CarrierDestinationGroup` has only
  `GroupName string` and `Region string` — no pgtype wrappers needed
- **Test pattern:** Follow `carrier_service_test.go` — `servicesMockDBTX` is already defined there;
  do not redefine it. Instead import it from the same `services` package (same package = same file scope).
  - For `DestinationGroupByGroupName`: mock `QueryRow` with 1 SQL arg, mock `Scan` with 2 dest args
  - For `CreateDestinationGroup` / `UpdateDestinationGroup`: mock `QueryRow` with 2 SQL args
    (group_name, region), mock `Scan` with 2 dest args
  - For `DeleteDestinationGroup`: mock `Exec` with 1 SQL arg
  - For `ListDestinationGroups`: mock `Query` (dynamic); for `CountDestinationGroups` mock `QueryRow`
- **Resolver pattern:** Follow `internal/backend/resolvers/charging.resolvers.go`
- **Wiring pattern:** Follow the existing entries in `resolver.go`, `context.go`, and `router.go`
- **gqlgen-generated resolver file:** The stub file `destination_group.resolvers.go` is generated code
  BUT the resolver implementations inside it are NOT marked DO NOT EDIT — fill them in directly.

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| `destinationGroupWildcardCols = []string{"group_name", "region"}` | Both fields are short strings; matching either is useful for admin search |
| Default sort key: `"group_name"` | PK is the natural default sort; mirrors carrier (plmn) and number_plan (name) patterns |
| `sqlc.CreateDestinationGroupParams` struct arg | sqlc uses struct params when field count > query_parameter_limit (4); 2 fields here will likely use positional args — verify after sqlc generate and adjust accordingly |

---

## Acceptance Criteria

- [ ] `internal/backend/services/destination_group_service.go` exists with six methods
- [ ] `internal/backend/services/destination_group_service_test.go` covers success + error paths for all six methods
- [ ] `internal/backend/resolvers/destination_group.resolvers.go` resolver methods delegate to `r.DestinationGroupSvc`
- [ ] `internal/backend/resolvers/resolver.go` includes `DestinationGroupSvc`
- [ ] `internal/backend/appcontext/context.go` includes `DestinationGroupSvc` field and wires it in `NewAppContext`
- [ ] `internal/backend/handlers/graphql/router.go` passes `DestinationGroupSvc` to the resolver
- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes

---

## Risk Assessment

Low. This task modifies `resolver.go`, `context.go`, and `router.go` — all of which are small
wiring files with no business logic. The changes are additive (new fields and service wiring only).
No charging, quota, or rating logic is touched.

---

## Notes

After completing this task, update `.ai/memory/STATUS.md` to reflect that
`DestinationGroupResource` is complete, and update `.ai/memory/FEATURES.md` to set
F-002 status to "In Review".
