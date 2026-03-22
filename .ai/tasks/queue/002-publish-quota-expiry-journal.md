# Task 002 — Publish QUOTA_EXPIRY journal events from executeWithQuota

**Feature:** F-008 — Counter Expiry Cleanup with Quota Journal
**Sequence:** 2 of 2
**Date:** 2026-03-22
**Status:** Active

---

## Objective

Update `executeWithQuota` in `internal/quota/manager.go` to consume the `[]ExpiredCounterEntry`
returned by `RemoveExpiredEntries` (introduced in Task 001) and publish a `QUOTA_EXPIRY`
`QuotaJournalEvent` for each expired counter, using the counter's `TaxRate` (defaulting to `1`
if nil) and `CalculateTax` for the tax calculation.

---

## Scope

**In scope:**
- Update both `RemoveExpiredEntries` call sites in `executeWithQuota` to capture and process
  the returned `[]ExpiredCounterEntry`
- New private helper `publishExpiryJournals` (or inline logic) that calls `PublishJournalEvent`
  per entry
- Tax rate nil-guard: if `counter.TaxRate == nil`, use `decimal.NewFromInt(1)`
- Tests: new test cases exercising the journal publish path via the existing test harness pattern

**Out of scope:**
- Any change to `loadedquota.go` (completed in Task 001)
- Any change to `PublishJournalEvent` signature or `QuotaJournalEvent` struct (contracts)
- Changes to loan repayment / clawback logic

---

## Context

### File to modify
`internal/quota/manager.go`

### Current `executeWithQuota` skeleton (relevant section)
```go
func (m *QuotaManager) executeWithQuota(
    ctx context.Context,
    now time.Time,
    subscriberID uuid.UUID,
    op func(q *Quota) error,
) error {
    for attempt := 0; attempt < m.retryLimit; attempt++ {
        loaded, err := m.repo.Load(ctx, subscriberID)
        // ... error handling ...

        _ = loaded.RemoveExpiredEntries(now)   // Task 001 left _ here

        if err := op(loaded.Quota); err != nil {
            return err
        }

        _ = loaded.RemoveExpiredEntries(now)   // Task 001 left _ here

        // ... CheckForUsageNotifications, Save, retry logic ...
    }
}
```

### What to change
Replace both `_ = loaded.RemoveExpiredEntries(now)` with:
```go
expiredBefore := loaded.RemoveExpiredEntries(now)
```
and
```go
expiredAfter := loaded.RemoveExpiredEntries(now)
```

After the successful `Save`, publish journals for all entries from both slices:
```go
publishExpiryJournals(m, subscriberID, expiredBefore, now)
publishExpiryJournals(m, subscriberID, expiredAfter, now)
```

### `publishExpiryJournals` helper
```go
func publishExpiryJournals(
    m *QuotaManager,
    subscriberID uuid.UUID,
    entries []ExpiredCounterEntry,
    now time.Time,
) {
    for _, entry := range entries {
        taxRate := decimal.NewFromInt(1)
        if entry.Counter.TaxRate != nil {
            taxRate = *entry.Counter.TaxRate
        }
        taxCalc := CalculateTax(entry.BalanceAtExpiry, taxRate)

        PublishJournalEvent(
            m,
            entry.QuotaID,
            "",                        // no natural transactionId for expiry events
            &entry.Counter,
            ReasonQuotaExpiry,
            entry.BalanceAtExpiry,     // AdjustedUnits = balance written off
            entry.Counter.UnitType,
            taxCalc,
            subscriberID,
            nil,                       // no CounterMetaData for expiry events
            now,
        )
    }
}
```

### Key journal event field mapping
| Journal field | Source |
|---|---|
| `AdjustedUnits` | `entry.BalanceAtExpiry` |
| `Balance` | `*entry.Counter.Balance` → 0 (zeroed in Task 001) |
| `TaxAmount` | `CalculateTax(entry.BalanceAtExpiry, taxRate).TaxAmount` |
| `ValueExTax` | `CalculateTax(entry.BalanceAtExpiry, taxRate).ExTaxValue` |
| `ReasonCode` | `ReasonQuotaExpiry` |
| `CounterID` | `entry.Counter.CounterID` |
| `QuotaID` | `entry.QuotaID` |
| `SubscriberID` | `subscriberID` (from `executeWithQuota` parameter) |
| `ProductID` | `entry.Counter.ProductID` |
| `ProductName` | `entry.Counter.ProductName` |
| `UnitType` | `entry.Counter.UnitType` |
| `ExternalReference` | `entry.Counter.ExternalReference` |
| `TransactionID` | `""` (no natural transaction for expiry) |
| `CounterMetaData` | `nil` |
| `Timestamp` | `now` |

### Double-publish safety
`RemoveExpiredEntries` is called twice per attempt. A counter removed (or zeroed) in the first
call will not be present in the second call with a non-zero balance, so it will not generate a
second entry. No additional guard is required.

### Test pattern to follow
Existing provisioning tests (`provisioning_test.go`) use a mock `KafkaManager` injected via
`NewQuotaManager`. Follow the same pattern:
1. Build a quota with one or more counters that have an `Expiry` in the past
2. Call any public `QuotaManager` method that invokes `executeWithQuota` (e.g. `Reserve`,
   `Debit`, or if there is a no-op operation, a simple `OpenQuota` call)
3. Assert that the mock Kafka manager received a `QuotaJournalEvent` with:
   - `ReasonCode == ReasonQuotaExpiry`
   - `AdjustedUnits` equals the counter's original balance
   - `Balance` equals zero
   - `TaxAmount` and `ValueExTax` match `CalculateTax(balance, taxRate)`
4. Test the nil-TaxRate default: counter with `TaxRate == nil` should use rate `1`
5. Test the loan-retention case: expired counter with outstanding loan publishes journal but
   counter remains in quota (observable via a subsequent operation that shows the counter still
   present with zero balance)

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| `transactionId` is empty string for expiry events | Expiry is not triggered by a charging transaction; there is no natural correlation ID. Downstream systems correlate on `CounterID` + `ReasonCode`. |
| `CounterMetaData` is nil | `CounterMetaData` is only populated for `QUOTA_PROVISIONED` events (confirmed by all existing call sites). |
| Journals published after successful `Save` | Consistent with existing pattern — events should not be published if the DB operation fails. |
| Tax rate defaults to `1` | As specified: a nil `TaxRate` on the counter means the provisioning event did not set it; defaulting to `1` results in `TaxAmount = balance × 1 = balance`, `ExTaxValue = balance`. This is the safe fallback. |
| Helper function `publishExpiryJournals` | Keeps `executeWithQuota` readable; the helper is private and has no external callers. |

---

## Acceptance Criteria

- [ ] Both `RemoveExpiredEntries` call sites in `executeWithQuota` capture the return value
- [ ] `publishExpiryJournals` (or equivalent) is called after each `RemoveExpiredEntries` call, after a successful `Save`
- [ ] Journal events are published only on the successful attempt (not on retry attempts that fail with `ErrConflict`)
- [ ] `ReasonCode` in the published event is `ReasonQuotaExpiry`
- [ ] `AdjustedUnits` equals the balance at expiry (not zero)
- [ ] `Balance` in the event equals zero
- [ ] `TaxAmount` and `ValueExTax` match `CalculateTax(balanceAtExpiry, taxRate)`
- [ ] When counter `TaxRate` is nil, tax is calculated with rate `1`
- [ ] `CounterMetaData` is nil in the published event
- [ ] `TransactionID` is empty string in the published event
- [ ] Expired counter with loan: journal is published AND counter is still present in quota with zero balance after the operation
- [ ] No journal published for a zero-balance expired counter (Case D from Task 001)
- [ ] No double-publish when `RemoveExpiredEntries` is called twice in one `executeWithQuota` invocation
- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes

---

## Risk Assessment

**High** — `executeWithQuota` is the single path through which all quota mutations flow.
Any error introduced here affects charging, debiting, and provisioning. The change is additive
(consuming a return value and publishing events) and does not alter the operation or save logic,
but care must be taken to ensure journal publication does not block or fail the operation.
`PublishJournalEvent` already swallows Kafka errors gracefully (confirm before implementing).

---

## Notes

- Confirm that `PublishJournalEvent` does not return an error and does not panic on Kafka
  unavailability before publishing from `executeWithQuota`. If it can fail fatally, wrap it
  defensively with a recover or error log.
- The `CalculateTax` function is defined in `internal/quota/taxcalculator.go` and is already
  in-package — no import needed.
- `decimal.NewFromInt(1)` is the correct way to construct the default tax rate.
