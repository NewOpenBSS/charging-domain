# Task: Quota Balance Inquiry — QuotaManager Extension

**Date:** 2026-03-19
**Status:** Implementation complete — pending build verification and commit

---

## Objective

Extend `QuotaManager` with a general-purpose balance inquiry capability. The new
`GetBalance` method allows any part of the domain — charging pipeline, DRA server,
backend GraphQL service, future services — to query a subscriber's counter balances
using a composable filter. This is a prerequisite for the `QuotaResource` GraphQL
resource (Task B), but is intentionally designed as a domain primitive, not a
GraphQL-specific helper.

---

## Scope

**In scope:**
- New `BalanceQuery` filter struct in `internal/quota` ✅ done (`balance.go`)
- New `CounterBalance` result struct in `internal/quota` ✅ done (`balance.go`)
- `GetBalance(ctx, now, subscriberID, BalanceQuery) ([]*CounterBalance, error)` method on
  `QuotaManager` ✅ done (`manager.go`)
- Add `GetBalance` to the `QuotaManagerInterface` ✅ done (`manager.go`)
- Unit tests covering all filter combinations and edge cases ✅ done (`balance_test.go`)

**Out of scope:**
- Loan balance inquiry (`GetLoanBalance`) — deferred, separate task
- GraphQL schema and resolver — deferred Task B (`QuotaResource`)
- Any change to the charging pipeline or DRA server
- Any database migration or sqlc query change (balance is derived from the in-memory
  JSONB quota structure, no new SQL needed)

---

## Context

- The quota is stored as a JSONB blob per subscriber in the `quota` table. It is
  loaded via `QuotaRepository.Load()` and deserialised into `*LoadedQuota`.
- `Counter` fields relevant to this task:
  - `UnitType charging.UnitType` — one of `SECONDS`, `OCTETS`, `UNITS`, `MONETARY`
  - `Balance *decimal.Decimal` — total current balance
  - `Reservations map[uuid.UUID]Reservation` — active reservations against this counter
  - `CanTransfer bool` — counter may be transferred to another subscriber
  - `CanConvert bool` — counter may be converted between unit types
  - `Expiry *time.Time` — counter expiry; expired counters must be excluded from results
- Available balance = `Balance` minus sum of all active `Reservation.Value` (for
  MONETARY) or sum of `Reservation.Units` (for service unit types). The existing
  helper methods `Counter.AvailableValue()` and `Counter.AvailableServiceUnits()` already
  compute this — use them.
- `GetBalance` is a **read-only** operation. It must NOT call `executeWithQuota` (which
  is write-oriented and includes a save/retry loop). Load, filter expired entries in
  memory, apply the query, return — no write back.
- Follow the pattern in `internal/quota/manager.go` and `internal/quota/quota.go`.
- All financial values use `github.com/shopspring/decimal` — no float types.
- `now` must be injected as a parameter for expiry comparisons (never call `time.Now()`
  inside business logic).

---

## Design

### BalanceQuery

```go
// BalanceQuery defines the filter criteria for a balance inquiry.
// Nil pointer fields mean "no filter on this dimension".
type BalanceQuery struct {
    // UnitType restricts results to counters of this unit type.
    // nil returns counters of all unit types.
    UnitType *charging.UnitType

    // Transferable restricts results to counters where CanTransfer matches.
    // nil returns counters regardless of their CanTransfer flag.
    Transferable *bool

    // Convertible restricts results to counters where CanConvert matches.
    // nil returns counters regardless of their CanConvert flag.
    Convertible *bool
}
```

### CounterBalance

```go
// CounterBalance is the balance result for a single matching counter.
type CounterBalance struct {
    CounterID        uuid.UUID
    ProductID        uuid.UUID
    ProductName      string
    UnitType         charging.UnitType
    TotalBalance     decimal.Decimal
    AvailableBalance decimal.Decimal
    Expiry           *time.Time
    CanTransfer      bool
    CanConvert       bool
}
```

### GetBalance signature

```go
// GetBalance returns the balances for all non-expired counters matching query
// for the given subscriber. now is the reference time for expiry comparisons.
// Returns an empty slice (not an error) if the subscriber has no quota or no
// counters match the query.
GetBalance(ctx context.Context, now time.Time, subscriberID uuid.UUID, query BalanceQuery) ([]*CounterBalance, error)
```

### QuotaManagerInterface addition

Add `GetBalance` to the existing `QuotaManagerInterface` alongside `ReserveQuota`,
`Debit`, and `Release`.

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| Single `GetBalance` with `BalanceQuery` struct, not three separate functions | The filtering logic is identical across all balance types; callers compose the query they need. Avoids combinatorial explosion as new filter dimensions are added. |
| `GetBalance` on `QuotaManagerInterface` (not a separate interface) | Balance inquiry is a domain primitive needed anywhere quota is used, not just in the backend layer. Keeping it on the main interface avoids interface proliferation. |
| Read-only — no `executeWithQuota`, no save | Balance is derived from the loaded JSONB; no mutation occurs. The optimistic-locking retry loop is only needed for writes. |
| Expired counters excluded silently | An expired counter has no available balance; including it would mislead callers. Mirrors the behaviour of `RemoveExpiredEntries`. |
| `now` injected as parameter | Consistent with the project-wide rule: never call `time.Now()` in business logic. |
| `AvailableBalance` computed from existing helpers | `Counter.AvailableValue()` and `Counter.AvailableServiceUnits()` already implement the reservation-deduction logic correctly. Reuse them rather than duplicating. |
| Loan balance excluded | A loan is separate domain state on a counter. Outstanding loan balance is a different query with different semantics — deferred to a future task. |
| `nil` quota returns empty slice, not error | A subscriber with no quota record is a valid state (quota created on first reservation). Callers should distinguish "no data" from "error". |

---

## Acceptance Criteria

- [ ] `BalanceQuery` and `CounterBalance` types defined in `internal/quota`
- [ ] `GetBalance` implemented on `*QuotaManager`
- [ ] `GetBalance` added to `QuotaManagerInterface`
- [ ] Expired counters excluded from results (expiry before `now`)
- [ ] `BalanceQuery` with all nil fields returns all non-expired counters
- [ ] `BalanceQuery.UnitType` filter applied correctly
- [ ] `BalanceQuery.Transferable` filter applied correctly
- [ ] `BalanceQuery.Convertible` filter applied correctly
- [ ] Combined filters (e.g. `UnitType=MONETARY` + `Transferable=true`) work correctly
- [ ] Subscriber with no quota record returns empty slice, not error
- [ ] `AvailableBalance` uses existing `AvailableValue()` / `AvailableServiceUnits()` helpers
- [ ] Table-driven unit tests for all filter combinations — success, no-match, nil quota, expired counters
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

---

## Risk Assessment

This change is **read-only** and does not touch the charging pipeline, quota write
path, or any database schema. There is no risk to charging, quota mutation, or rating
behaviour.

The only risk is incorrect filter logic returning wrong balances to callers, which
would be a display/reporting error, not a financial mutation. Covered by the
table-driven test requirement above.

---

## Notes

**Task B — QuotaResource GraphQL (deferred):**

The Java contract to be matched in Task B is:

```
Query:    quotaBalance(balanceEnquiryRequest: QuotaBalanceRequestDto): QuotaBalanceResponseDto
Query:    quotaBalances(balanceEnquiryRequest: QuotaBalanceRequestDto): [QuotaBalanceResponseDto]
Mutation: cancelQuotaReservations(reservationId, subscriberId): QuotaOperationResponse
Mutation: reserveQuota(reservationId, subscriberId, reasonCode, rateKey, unitType,
                       requestedUnits, unitPrice, validityTime, allowOOBCharging): ReserveResponse
Mutation: debitQuota(subscriberId, reservationId, usedUnits, unitType,
                     reclaimUnusedUnits, provisioningRequest): DebitResponse
```

`QuotaBalanceRequestDto` fields: `subscriberId UUID`, `unitType UnitType` (optional),
`balanceType BalanceType` (TRANSFERABLE_BALANCE | CONVERTABLE_BALANCE | AVAILABLE_BALANCE)

`QuotaBalanceResponseDto` fields: `unitType UnitType`, `availableBalance decimal`,
`totalValue decimal`

`balanceType` maps to `BalanceQuery` as follows:
- `AVAILABLE_BALANCE` → `BalanceQuery{}` (all nil — no filter)
- `TRANSFERABLE_BALANCE` → `BalanceQuery{Transferable: ptr(true)}`
- `CONVERTABLE_BALANCE` → `BalanceQuery{Convertible: ptr(true)}`
