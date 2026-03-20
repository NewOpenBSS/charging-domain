# Task 001 — ChargingTrace GraphQL Schema and Code Generation

**Feature:** F-001 — ChargingTraceResource
**Sequence:** 001 of 003
**Date:** 2026-03-20
**Status:** Active

---

## Objective

Define the GraphQL schema for the `ChargingTrace` resource and run gqlgen to regenerate
the models and resolver stubs. This task produces the generated code that subsequent
tasks depend on. No business logic is written here — only the schema declaration and
verification that the generated output compiles cleanly.

---

## Scope

**In scope:**
- Create `gql/schema/charging_trace.graphql` with the `ChargingTrace` type and three
  read-only query extensions (`chargingTraceList`, `countChargingTrace`, `chargingTraceById`)
- Run `go run github.com/99designs/gqlgen generate` to regenerate:
  - `internal/backend/graphql/model/models_gen.go` (adds `ChargingTrace` and input types)
  - `internal/backend/graphql/generated/generated.go` (adds resolver interface methods)
  - `internal/backend/resolvers/charging_trace.resolvers.go` (new stub file)
- Verify `go build ./...` passes after code generation

**Out of scope:**
- Service implementation (Task 002)
- Resolver implementation and wiring (Task 003)
- Any mutations — the resource is strictly read-only

---

## Context

- Schema files live in `gql/schema/` — each resource has its own `.graphql` file.
  Refer to `gql/schema/charging.graphql` for the Carrier resource as the closest
  structural model (list + count + get-by-key, same PageRequest/FilterRequest inputs).
- gqlgen config is at `gql/gqlgen.yml`. Code generation command:
  `go run github.com/99designs/gqlgen generate`
- gqlgen uses `follow-schema` resolver layout, so the new schema file will produce a
  new resolver file: `internal/backend/resolvers/charging_trace.resolvers.go`.
- The `ChargingTrace` sqlc model is in `internal/store/sqlc/models.go` — fields:
  `TraceID` (pgtype.UUID), `CreatedAt` (pgtype.Timestamptz), `Request` ([]byte JSONB),
  `Response` ([]byte JSONB), `ExecutionTime` (int64 ms), `ChargingID` (string),
  `SequenceNr` (int32), `Msisdn` (string).
- GraphQL field names must match the Java service exactly (per Feature constraint):
  `traceId`, `createdAt`, `request`, `response`, `executionTime`, `chargingId`,
  `sequenceNr`, `msisdn`.
- Query names must match Java exactly: `chargingTraceList`, `countChargingTrace`,
  `chargingTraceById`. Note: `countChargingTrace` is singular (not `countChargingTraces`).
- `request` and `response` must be `String!` — JSONB payloads serialised to JSON string.
- `traceId` must be `String!` — UUID serialised to string.
- `createdAt` must be `DateTime` (the existing DateTime scalar from `schema.graphql`).

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| `request` and `response` as `String!` (not a custom scalar) | Java API returns raw JSON strings; clients parse them. GraphQL custom JSONB scalar adds complexity for no benefit at this layer. Feature spec says "returned as JSON strings". |
| `traceId` as `String!` (not `ID!`) | Consistent with Java API surface; avoids any gqlgen ID coercion behaviour. |
| `executionTime` as `Int!` | Java returns milliseconds as a long integer. GraphQL `Int` is 32-bit but execution times in ms will not exceed 2^31; acceptable for now. |
| No mutations | Feature is strictly read-only — enforced at schema level. |
| Query name `countChargingTrace` (singular) | Must match Java service exactly per Feature constraint. |
| `chargingTraceById(traceId: String!)` | UUID primary key passed as string; consistent with Java. |

---

## Acceptance Criteria

- [ ] `gql/schema/charging_trace.graphql` exists and defines:
  - `ChargingTrace` type with all eight fields (`traceId`, `createdAt`, `request`,
    `response`, `executionTime`, `chargingId`, `sequenceNr`, `msisdn`)
  - `extend type Query` with exactly three operations: `chargingTraceList`,
    `countChargingTrace`, `chargingTraceById`
  - No `extend type Mutation` block
- [ ] `go run github.com/99designs/gqlgen generate` completes without errors
- [ ] `internal/backend/resolvers/charging_trace.resolvers.go` is generated with
  stub implementations for the three query methods
- [ ] `go build ./...` passes cleanly after code generation

---

## Risk Assessment

None. This task only adds new schema types and generated code. It does not modify
any existing files beyond what gqlgen overwrites in its generated output
(`models_gen.go` and `generated.go`), which are always safe to regenerate.
The new resolver stub file will contain `panic("not implemented")` stubs — these
are replaced in Task 003 before the feature is complete.

---

## Notes

After running gqlgen, the generated `charging_trace.resolvers.go` will have unimplemented
stubs. The build will pass because the stub bodies contain `panic`. Task 003 replaces
them with real implementations before the feature is complete.
