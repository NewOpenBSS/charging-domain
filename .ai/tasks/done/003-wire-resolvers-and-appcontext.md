# Task 003 — Wire Resolvers, AppContext, and GraphQL Router

**Feature:** F-001 — ChargingTraceResource
**Sequence:** 003 of 003
**Date:** 2026-03-20
**Status:** Active

---

## Objective

Complete the feature by wiring `ChargingTraceService` into the dependency injection
container and implementing the three GraphQL resolver methods. After this task the
`chargingTraceList`, `countChargingTrace`, and `chargingTraceById` operations are
fully functional end-to-end.

---

## Scope

**In scope:**
- Add `ChargingTraceSvc *services.ChargingTraceService` field to `resolvers.Resolver`
  in `internal/backend/resolvers/resolver.go`
- Implement the three resolver methods in the gqlgen-generated
  `internal/backend/resolvers/charging_trace.resolvers.go` (replace the panic stubs
  with real delegations to `r.ChargingTraceSvc`)
- Add `ChargingTraceSvc *services.ChargingTraceService` field to `AppContext` in
  `internal/backend/appcontext/context.go`
- Instantiate `ChargingTraceSvc` in `NewAppContext` using `services.NewChargingTraceService(s)`
- Wire `ChargingTraceSvc` into the `Resolver` in
  `internal/backend/handlers/graphql/router.go`
- Run `go build ./...` and `go test ./...` — all must pass

**Out of scope:**
- Any mutations — the resolver file must not add mutation methods
- Changes to the store layer (complete since Task 001)
- Changes to the service implementation (complete since Task 002)

---

## Context

### Files to modify

| File | Change |
|---|---|
| `internal/backend/resolvers/resolver.go` | Add `ChargingTraceSvc *services.ChargingTraceService` |
| `internal/backend/resolvers/charging_trace.resolvers.go` | Replace panic stubs with service delegation |
| `internal/backend/appcontext/context.go` | Add `ChargingTraceSvc` field + instantiate in `NewAppContext` |
| `internal/backend/handlers/graphql/router.go` | Add `ChargingTraceSvc: appCtx.ChargingTraceSvc` to Resolver literal |

### Resolver delegation pattern

The resolver methods must be thin — delegate directly to the service:

```go
func (r *queryResolver) ChargingTraceList(ctx context.Context, page *model.PageRequest, filter *model.FilterRequest) ([]*model.ChargingTrace, error) {
    return r.ChargingTraceSvc.ListChargingTraces(ctx, page, filter)
}

func (r *queryResolver) CountChargingTrace(ctx context.Context, filter *model.FilterRequest) (int, error) {
    return r.ChargingTraceSvc.CountChargingTrace(ctx, filter)
}

func (r *queryResolver) ChargingTraceById(ctx context.Context, traceID string) (*model.ChargingTrace, error) {
    return r.ChargingTraceSvc.ChargingTraceById(ctx, traceID)
}
```

Note: gqlgen generates method names from the GraphQL field names. The generated argument
name for `chargingTraceById(traceId: String!)` will be `traceID` (Go convention).
Confirm the exact generated signature in `charging_trace.resolvers.go` before implementing.

### Reference implementations

- `internal/backend/resolvers/charging.resolvers.go` — Carrier resolver (same delegation pattern)
- `internal/backend/resolvers/resolver.go` — existing service fields to follow
- `internal/backend/appcontext/context.go` — existing `NewAppContext` instantiation pattern
- `internal/backend/handlers/graphql/router.go` — existing Resolver wiring pattern

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| No dedicated resolver test file | Resolver methods are one-line delegations with no logic; the service tests in Task 002 provide the meaningful coverage. Testing the resolver would require a gqlgen test server setup that adds complexity without proportional value. |
| `ChargingTraceSvc` added to `AppContext` (not constructed in router) | Consistent with every other service in the application — all services live on `AppContext`, the router receives `AppContext` and unpacks it. |

---

## Acceptance Criteria

- [ ] `resolvers.Resolver` has a `ChargingTraceSvc` field
- [ ] The three resolver methods in `charging_trace.resolvers.go` delegate to the service
  with no business logic in the resolver body
- [ ] `AppContext` has a `ChargingTraceSvc` field instantiated in `NewAppContext`
- [ ] The GraphQL router wires `ChargingTraceSvc` into the `Resolver` struct literal
- [ ] `go build ./...` passes cleanly
- [ ] `go test ./...` passes cleanly (no regressions, no panic stubs remaining)
- [ ] `go test -race ./...` passes

---

## Risk Assessment

Low. All changes are additive wiring — no existing logic is modified. The resolver
methods contain no business logic. The only risk is a naming mismatch between the
gqlgen-generated method signatures and the service method signatures; mitigate by
reading the generated stub file carefully before implementing.

No charging pipeline, quota, or financial logic is touched.

---

## Notes

After completing this task, update `FEATURES.md` to set F-001 status to "In Review"
and perform the final commit per the memory lifecycle protocol.
