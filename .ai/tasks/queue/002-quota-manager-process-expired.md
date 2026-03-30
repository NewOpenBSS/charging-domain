# Task 002 — QuotaManager: add ProcessExpiredQuota method

**Feature:** F-009 — Charging Domain Housekeeping
**Sequence:** 002 of 004
**Date:** 2026-03-30
**Status:** Active

---

## Objective

Add a public `ProcessExpiredQuota` method to `QuotaManager` that triggers the existing
JIT expiry logic for a single subscriber on demand. This is the mechanism the housekeeping
binary uses to process dormant subscribers whose quota counters have expired. The method
must reuse `executeWithQuota` — no new expiry business logic is introduced in this task.

---

## Scope

**In scope:**
- Add `ProcessExpiredQuota(ctx context.Context, now time.Time, subscriberID uuid.UUID) error`
  to `QuotaManager` in `internal/quota/manager.go`
- Unit tests for `ProcessExpiredQuota` in `internal/quota/manager_test.go`
- `go build ./...` and `go test -race ./...` must pass

**Out of scope:**
- Modifying `QuotaManagerInterface` — the housekeeping binary will define its own narrow
  interface at point of consumption; existing consumers are not affected
- Any binary or service code (Task 004)
- The loop that iterates over all expired subscribers (Task 004)
- Store-level wrapper methods (not needed — Task 004 calls `store.Q.FindExpiredQuotaSubscribers` directly)

---

## Context

**How `executeWithQuota` works (from `internal/quota/manager.go`):**
```
executeWithQuota(ctx, now, subscriberID, op func(*Quota) error) error
  1. Load quota from repo
  2. If nil → create new quota row
  3. Call loaded.RemoveExpiredEntries(now)  ← expiry fires here (Case B/C produce journals)
  4. Call op(loaded.Quota)                   ← user-supplied operation
  5. Call loaded.RemoveExpiredEntries(now) again (catches any newly expired after op)
  6. Save quota via repo.Save(ctx, loaded, now)
  7. On successful save → publishExpiryJournals for both before/after expired entries
```

For the housekeeping use case the `op` function is a no-op — expiry fires at step 3 and
journals are published at step 7. The no-op avoids any unintended quota mutation.

**Expiry logic is in `internal/quota/loadedquota.go`:**
- `RemoveExpiredEntries(now)` returns `[]ExpiredCounterEntry` for counters that have expired
- Four cases: A (unexpired reservations — skip), B (loan outstanding — zero balance, keep),
  C (no loan, balance > 0 — publish journal, remove), D (no loan, zero balance — remove silently)
- QUOTA_EXPIRY journal events are published for cases B and C only

**Existing test pattern in `internal/quota/manager_test.go`:**
- Tests use `NewQuotaManager` with a mocked `Repository` and a mocked `events.KafkaManager`
- Follow the same mock approach for the new test

**Method signature to add to `manager.go`:**
```go
// ProcessExpiredQuota processes expired counters for a single subscriber by loading
// their quota, triggering expiry logic, and saving back. Journal events are published
// for all expired counters. This is equivalent to the JIT expiry that runs during
// active charging, but invoked explicitly for dormant subscribers by the housekeeping job.
//
// A nil quota row is silently skipped — if the subscriber has no quota record there is
// nothing to expire. This is not an error.
func (m *QuotaManager) ProcessExpiredQuota(ctx context.Context, now time.Time, subscriberID uuid.UUID) error {
    return m.executeWithQuota(ctx, now, subscriberID, func(_ *Quota) error {
        return nil
    })
}
```

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| No-op operation function passed to `executeWithQuota` | Expiry already fires inside `executeWithQuota` before the operation; a no-op avoids any unintended quota mutation |
| Do NOT add `ProcessExpiredQuota` to `QuotaManagerInterface` | The interface is consumed by the charging pipeline, not the housekeeping binary. Adding it would require updating all existing mocks. The housekeeping binary defines its own narrow interface at point of consumption (Go standard: define interfaces where consumed). |
| Single-subscriber method, not bulk | Keeps the method testable in isolation; the loop over all expired subscribers belongs in the housekeeping binary (Task 004) |

---

## Acceptance Criteria

- [ ] `ProcessExpiredQuota` is added to `internal/quota/manager.go` with a Go doc comment
- [ ] Unit tests cover: subscriber with expired counter publishes journal, subscriber with no quota is a no-op (no error), `executeWithQuota` retry behaviour on conflict is inherited (no new test needed — it is tested elsewhere)
- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes (race detector required — quota tests involve concurrent access patterns)

---

## Risk Assessment

**Low risk.** The method delegates entirely to `executeWithQuota` with a no-op. All expiry
logic, optimistic locking, retry behaviour, and journal publishing are handled by code that
is already tested. The only new code is the delegation and the test. No changes to existing
quota paths. No changes to `QuotaManagerInterface` — existing mocks are unaffected.

---

## Notes

- `executeWithQuota` already handles the case where the quota row does not exist (it creates
  a new empty quota). For housekeeping, a newly created quota will have no counters to expire,
  so the method is effectively a no-op in that case. This is acceptable.
- The housekeeping binary's loop logic (`FindExpiredQuotaSubscribers` → iterate → `ProcessExpiredQuota`)
  is in Task 004, not here.

