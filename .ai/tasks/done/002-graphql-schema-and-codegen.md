# Task 002 — GraphQL schema and gqlgen code generation for DestinationGroup

**Feature:** F-002 — DestinationGroupResource
**Sequence:** 002 of 003
**Date:** 2026-03-20
**Status:** Active

---

## Objective

Write the GraphQL schema file for the DestinationGroup resource and run `gqlgen generate` to
produce the model types and resolver stub file.  After this task the build must pass with stub
resolver methods (the generated stubs return `nil, nil` or equivalent), ready for Task 003 to
fill in the real implementations.

---

## Scope

**In scope:**
- `gql/schema/destination_group.graphql` — defines `DestinationGroup` type,
  `DestinationGroupInput`, six operations on `extend type Query` and `extend type Mutation`
- Run `go run github.com/99designs/gqlgen generate` — updates:
  - `internal/backend/graphql/model/models_gen.go` (adds DestinationGroup, DestinationGroupInput)
  - `internal/backend/graphql/generated/generated.go` (adds resolver dispatch)
  - `internal/backend/resolvers/destination_group.resolvers.go` (stub methods, created fresh)
- `go build ./...` must pass after generation

**Out of scope:**
- Service implementation, resolver logic, AppContext wiring (all in Task 003)
- Any changes to existing `.graphql` files or existing resolver files

---

## Context

- **Required GraphQL operation names** (must match Java service exactly):
  - Queries: `destinationGroupList`, `countDestinationGroup`, `destinationGroupByGroupName`
  - Mutations: `createDestinationGroup`, `updateDestinationGroup`, `deleteDestinationGroup`
- **Type fields:**
  - `DestinationGroup`: `groupName: String!`, `region: String!`
  - `DestinationGroupInput`: `groupName: String!`, `region: String!`
  - No `modifiedOn` — the `carrier_destination_group` table has no timestamp column
- **Schema pattern:** Follow `gql/schema/charging.graphql` (CarrierResource) exactly:
  - `extend type Query { ... }` and `extend type Mutation { ... }`
  - List query: `destinationGroupList(page: PageRequest, filter: FilterRequest): [DestinationGroup!]!`
  - Count query: `countDestinationGroup(filter: FilterRequest): Int!`
  - By-key query: `destinationGroupByGroupName(groupName: String!): DestinationGroup`
  - Create: `createDestinationGroup(destinationGroup: DestinationGroupInput!): DestinationGroup!`
  - Update: `updateDestinationGroup(groupName: String!, destinationGroup: DestinationGroupInput!): DestinationGroup!`
  - Delete: `deleteDestinationGroup(groupName: String!): Boolean!`
- **gqlgen command:** `go run github.com/99designs/gqlgen generate` from repo root
- **gqlgen config:** `gql/gqlgen.yml` — resolver layout is `follow-schema`, so a new
  `destination_group.resolvers.go` file will be created in `internal/backend/resolvers/`
- Files marked `// Code generated ... DO NOT EDIT` must not be manually modified

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| Input argument name `destinationGroup` | Consistent with Java API surface; gqlgen generates method signature from this |
| No `modifiedOn` field | The `carrier_destination_group` table has no timestamp column |
| `deleteDestinationGroup` returns `Boolean!` | Consistent with `deleteCarrier` pattern |

---

## Acceptance Criteria

- [ ] `gql/schema/destination_group.graphql` exists with the correct type, input, and all six operations
- [ ] `go run github.com/99designs/gqlgen generate` completes without errors
- [ ] `internal/backend/graphql/model/models_gen.go` contains `DestinationGroup` and `DestinationGroupInput`
- [ ] `internal/backend/resolvers/destination_group.resolvers.go` exists with six stub methods
- [ ] `go build ./...` passes (stub methods satisfy the generated interface)
- [ ] `go test -race ./...` passes

---

## Risk Assessment

None — gqlgen only generates new types and stub methods; it does not modify existing resolver
implementations. The `generated.go` file is regenerated but existing logic is unchanged.

---

## Notes

After running gqlgen, verify that the resolver stub file has exactly six methods:
three in `queryResolver` and three in `mutationResolver`.
Do not manually edit any `// Code generated` file.
