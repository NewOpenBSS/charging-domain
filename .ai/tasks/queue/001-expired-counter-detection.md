# Task 001 — Expired counter detection and loan-aware cleanup in RemoveExpiredEntries

**Feature:** F-008 — Counter Expiry Cleanup with Quota Journal
**Sequence:** 1 of 2
**Date:** 2026-03-22
**Status:** Active

---

## Objective

Extend `RemoveExpiredEntries` in `internal/quota/loadedquota.go` to detect expired counters,
apply loan-aware and reservation-aware removal logic, and return a snapshot of every counter
whose balance was written off so that the caller can publish journal events.

No journal events are published in this task — that is Task 002. This task concerns only the
domain layer change: the new type, the new logic, and the updated tests.

---

## Scope

**In scope:**
- New `ExpiredCounterEntry` struct in `internal/quota/loadedquota.go`
- Updated `RemoveExpiredEntries` signature: `func (l *LoadedQuota) RemoveExpiredEntries(now time.Time) []ExpiredCounterEntry`
- Full 4-case expiry logic (see below)
- Update caller in `internal/quota/manager.go` to discard the return value with `_` (keeps build passing; Task 002 will use it)
- Update existing `loadedquota_test.go` tests to handle the new return type
- New table-driven tests for all 4 expiry cases and the loan-retention case

**Out of scope:**
- Publishing journal events (Task 002)
- Any change to `QuotaJournalEvent` or `PublishJournalEvent`
- Any change to loan repayment / clawback logic

---

## Context

### File to modify
`internal/quota/loadedquota.go`

### Current signature
```go
func (l *LoadedQuota) RemoveExpiredEntries(now time.Time)
```

### `LoadedQuota` struct
`LoadedQuota` holds a `*Quota` (which has `QuotaID uuid.UUID`) and `Counters []Counter`.
The current method iterates `l.Counters` and rebuilds the slice in-place.

### `Counter` fields relevant to this task
- `Expiry *time.Time` — nil means never expires
- `Balance *decimal.Decimal` — current balance (always non-nil in practice; guard defensively)
- `Reservations map[uuid.UUID]Reservation` — keyed by reservation ID
- `Reservation.Expiry time.Time` — required, never nil
- `Loan *Loan` — nil if no loan
- `Loan.LoanBalance decimal.Decimal` — outstanding principal

### Four-case logic for expired counters
A counter is **expired** when: `counter.Expiry != nil && !counter.Expiry.After(now)`

Process each counter in order:

**Step 1 — Prune expired reservations** (for ALL counters, expired or not — existing behaviour)

**Step 2 — Apply case logic:**

| Case | Condition | Action |
|---|---|---|
| A | Expired AND has ≥1 unexpired reservation | Keep counter unchanged. No entry returned. |
| B | Expired AND no unexpired reservations AND `Loan.LoanBalance > 0` | Zero `counter.Balance`. Return `ExpiredCounterEntry` with `BalanceAtExpiry` = original balance. Keep counter in slice. |
| C | Expired AND no unexpired reservations AND no outstanding loan AND `Balance > 0` | Return `ExpiredCounterEntry` with `BalanceAtExpiry` = original balance. Remove counter from slice. |
| D | Expired AND no unexpired reservations AND no outstanding loan AND `Balance == 0` | Remove counter silently. No entry returned. |

**Zero-balance non-expired counters** (existing behaviour, updated):
- Remove silently if `Balance == 0` AND (`Loan == nil` OR `Loan.LoanBalance.IsZero()`)
- Keep if `Balance == 0` AND `Loan != nil` AND `Loan.LoanBalance > 0` (loan still outstanding)

### `ExpiredCounterEntry` struct to add
```go
// ExpiredCounterEntry records a counter whose balance was written off due to expiry.
// It contains a copy of the counter (with Balance zeroed to reflect the post-expiry state)
// and the balance that was present at the moment of expiry, for use as AdjustedUnits in the
// QUOTA_EXPIRY journal event.
type ExpiredCounterEntry struct {
    Counter        Counter         // copy of the counter; Balance is zero (post-expiry state)
    BalanceAtExpiry decimal.Decimal // the balance written off; used as AdjustedUnits in journal
    QuotaID        uuid.UUID       // quota this counter belongs to; needed for journal event
}
```

**Important — pointer fields when making the copy:**
`Counter.Balance` is `*decimal.Decimal`. When building the `ExpiredCounterEntry`, set the copy's
`Balance` field to point to a new zero `decimal.Decimal` (not the same pointer as the original).
Do not share pointer fields between the copy and the live counter.

### Caller update in manager.go
`RemoveExpiredEntries` is called twice in `executeWithQuota` (lines ~87 and ~96). Update both
call sites to:
```go
_ = loaded.RemoveExpiredEntries(now)
```
This keeps the build clean. Task 002 replaces `_` with the actual consumption logic.

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| `ExpiredCounterEntry` holds a counter copy, not a pointer | The counter may be removed from the slice; a pointer into the slice would dangle. A value copy is safe. |
| `Balance` in the copy is zeroed | `PublishJournalEvent` reads `counter.Balance` to populate the journal `Balance` field. Post-expiry balance is 0. |
| `QuotaID` included in the entry | `PublishJournalEvent` requires `quotaId` as a separate parameter; including it in the entry avoids threading extra state through the call in Task 002. |
| Callers discard return value with `_` in this task | Keeps the build passing independently. Task 002 adds consumption. |
| Unexpired reservations block expiry | Upstream services may still report usage against an unexpired reservation — dropping the counter would cause the debit to fail. |
| Loan outstanding blocks removal but not balance zeroing | The balance is expired (income should be recognised), but the counter must remain so loan repayment can proceed through the normal clawback path. |

---

## Acceptance Criteria

- [ ] `ExpiredCounterEntry` struct is defined in `loadedquota.go` with a Go doc comment
- [ ] `RemoveExpiredEntries` returns `[]ExpiredCounterEntry` (empty slice, never nil, when nothing expired)
- [ ] Case A (unexpired reservations): counter retained, no entry returned
- [ ] Case B (expired, loan outstanding): counter balance zeroed, entry returned, counter kept in slice
- [ ] Case C (expired, no loan, balance > 0): entry returned, counter removed from slice
- [ ] Case D (expired, no loan, zero balance): counter removed silently, no entry returned
- [ ] Zero-balance, non-expired counter with no outstanding loan: removed silently
- [ ] Zero-balance, non-expired counter WITH outstanding loan: retained
- [ ] `BalanceAtExpiry` in returned entry equals the counter's balance before zeroing/removal
- [ ] `Counter.Balance` in returned entry is zero
- [ ] `QuotaID` in returned entry matches `l.Quota.QuotaID`
- [ ] Both call sites in `manager.go` updated to `_ = loaded.RemoveExpiredEntries(now)`
- [ ] Existing `loadedquota_test.go` tests pass with the updated signature
- [ ] New table-driven tests cover all cases above
- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes

---

## Risk Assessment

**Medium** — `RemoveExpiredEntries` is on the hot path; it is called before and after every quota
operation. Incorrect retention logic (e.g. failing to keep a counter with an outstanding loan)
could cause loan state corruption. The existing test coverage plus the new table-driven tests
mitigate this.

---

## Notes

- `Reservation.Expiry` is `time.Time` (not a pointer). An unexpired reservation satisfies
  `reservation.Expiry.After(now)`.
- Guard against nil `counter.Balance` defensively (treat as zero) even though it should not
  occur in practice.
- Do not change `RemoveExpiredEntries` for zero-balance counters that are NOT expired —
  the existing silent removal behaviour is correct and expected (they carry no balance to journal).
