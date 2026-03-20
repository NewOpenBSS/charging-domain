# Task: QuotaResource GraphQL

**Date:** 2026-03-19
**Status:** Active

---

## Objective

Port the Java `QuotaResource` GraphQL API to the Go charging-backend, keeping the
interface identical so that existing callers are not broken. The resource exposes
two balance queries and three quota mutation operations through the gqlgen-generated
GraphQL endpoint at `/api/charging/graphql`.

---

## Scope

**In scope:**

- `gql/schema/quota.graphql` — new schema file with types, enums, queries, mutations
- `internal/backend/services/quota_service.go` — service wrapping `QuotaManagerInterface`
- `internal/backend/services/quota_service_test.go` — unit tests
- `internal/backend/resolvers/quota.resolvers.go` — resolver implementations (post-gqlgen)
- Wire `QuotaSvc` into `AppContext`, `Resolver`, `router.go`
- Add Kafka config to `BackendConfig` and backend wiring (mutations publish journal events)
- Fix `KafkaManager.PublishEvent` nil safety (calls `m.KafkaClient.Produce` directly, panics when disabled)

**Out of scope:**

- Subscriber admin CRUD (separate future task)
- Loan balance operations
- Changes to the charging pipeline or DRA

---

## Context

### Java contract to match (field names authoritative)

**Queries:**

```
quotaBalance(balanceEnquiryRequest: QuotaBalanceRequestInput!): QuotaBalanceResponse
  - subscriberId and unitType are required
  - Returns aggregated balance (sum) across all matching counters for that unitType

quotaBalances(balanceEnquiryRequest: QuotaBalanceRequestInput!): [QuotaBalanceResponse!]!
  - subscriberId is required; unitType is optional
  - Returns one aggregated QuotaBalanceResponse per distinct UnitType
```

**Mutations:**

```
cancelQuotaReservations(reservationId: ID!, subscriberId: ID!): QuotaOperationResponse!
reserveQuota(reservationId, subscriberId, reasonCode, rateKey, unitType,
             requestedUnits, unitPrice, validitySeconds, allowOOBCharging): QuotaReserveResponse!
debitQuota(subscriberId, reservationId, usedUnits, unitType,
           reclaimUnusedUnits): QuotaDebitResponse!
```

**BalanceType → BalanceQuery mapping:**
| Java BalanceType | Go BalanceQuery |
|---|---|
| `AVAILABLE_BALANCE` | `BalanceQuery{}` (no filter) |
| `TRANSFERABLE_BALANCE` | `BalanceQuery{Transferable: ptr(true)}` |
| `CONVERTABLE_BALANCE` | `BalanceQuery{Convertible: ptr(true)}` |

### Key domain types

- `quota.QuotaManagerInterface.GetBalance` — implemented in Task A
- `quota.QuotaManagerInterface.ReserveQuota` — existing
- `quota.QuotaManagerInterface.Debit` — existing
- `quota.QuotaManagerInterface.Release` — existing
- `charging.RateKey` — serialised as dot-separated string e.g. `"VOICE.HOME.MO.LOCAL"`
- `charging.UnitType` — `SECONDS | OCTETS | UNITS | MONETARY`

### Aggregation logic

`quotaBalance` and `quotaBalances` aggregate per UnitType:

- `TotalBalance` = sum of `CounterBalance.TotalBalance` for all matching counters of that UnitType
- `AvailableBalance` = sum of `CounterBalance.AvailableBalance` for all matching counters

### Pattern references

- Schema: `gql/schema/charging.graphql` — follow this style
- Service: `internal/backend/services/carrier_service.go` — follow this pattern
- Resolver: `internal/backend/resolvers/charging.resolvers.go`
- AppContext wiring: `internal/backend/appcontext/context.go`

### Kafka requirement

The mutations (reserve, debit) call `QuotaManager` which publishes journal events.
`QuotaManager` requires a `*events.KafkaManager`. Add Kafka config to `BackendConfig`
and wire it through `AppContext`. With `enabled: false` in dev config, no connection
is made but the struct is non-nil. Fix `PublishEvent` to use the nil-safe `m.Produce()`
instead of calling `m.KafkaClient.Produce()` directly.

---

## Decisions Made During Design

| Decision                                                 | Rationale                                                                                                                                                        |
|----------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Aggregate counters by UnitType in service layer          | Java returns one DTO per UnitType; Go GetBalance returns per-counter slices. Service aggregates to match contract.                                               |
| Decimal values serialised as String in GraphQL           | Consistent with existing RatePlan schema (BaseTariff, Multiplier are strings). Avoids float precision loss.                                                      |
| `validitySeconds: Int!` for reservation duration         | `time.Duration` has no native GraphQL scalar; seconds as Int is simple and unambiguous.                                                                          |
| Java `provisioningRequest` param dropped from debitQuota | Go `Debit()` uses `reservationId.String()` as requestId directly. No equivalent param needed.                                                                    |
| Java `multiplier` hardcoded to 1 in reserveQuota         | Java source hardcodes `BigDecimal.ONE`. Kept in service layer, not exposed in schema.                                                                            |
| Kafka added to charging-backend                          | Reserve/debit mutations call QuotaManager which publishes journal events. Backend must have Kafka to support these operations. `enabled: false` default for dev. |
| Fix PublishEvent nil safety                              | `PublishEvent` calls `m.KafkaClient.Produce()` directly — panics when `KafkaClient` is nil. Use `m.Produce()` which is already nil-safe.                         |

---

## Acceptance Criteria

- [ ] `gql/schema/quota.graphql` defines all types, enums, queries, mutations
- [ ] `gqlgen generate` runs cleanly and updates generated files
- [ ] `QuotaService` wraps `QuotaManagerInterface` and converts types correctly
- [ ] `quotaBalance` requires subscriberId + unitType; returns nil if no match
- [ ] `quotaBalances` requires subscriberId; unitType optional; returns one entry per UnitType
- [ ] Balance results aggregate TotalBalance and AvailableBalance per UnitType
- [ ] `cancelQuotaReservations` delegates to `QuotaManager.Release`
- [ ] `reserveQuota` delegates to `QuotaManager.ReserveQuota`
- [ ] `debitQuota` delegates to `QuotaManager.Debit`
- [ ] `KafkaManager.PublishEvent` is nil-safe
- [ ] Kafka wired into charging-backend (optional, `enabled: false` default)
- [ ] `QuotaSvc` wired into `AppContext`, `Resolver`, `router.go`
- [ ] Unit tests for `QuotaService` covering success, not-found, and error paths
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

---

## Risk Assessment

**Mutations (reserve, debit, cancel) modify quota state.** This is the same mutation
path used by the charging engine. Risk: a malformed GraphQL request could corrupt
quota state or create phantom reservations.

Mitigations:

- Input validation in the service layer mirrors the Java null/range checks
- `QuotaManager` already handles idempotency at the domain level
- `now` is injected via `time.Now()` at the resolver boundary (not inside business logic)
- Kafka publish failures are logged but non-fatal — quota state is committed independently

**Read operations (balance queries) are completely safe** — no mutations.

## end
