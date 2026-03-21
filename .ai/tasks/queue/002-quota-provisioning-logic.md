# Task 002 — Quota provisioning logic: Counter fields + QuotaManager.ProvisionCounter

**Feature:** F-007 — QuotaProvisioningEventConsumer
**Sequence:** 2 of 4
**Date:** 2026-03-21

---

## Objective

Add the business logic that provisions a counter onto a subscriber's quota. This requires
two changes to the `internal/quota` package:

1. Add `CanRepayLoan bool` field to `Counter` (needed by provisioning and already expected
   by `CounterEvent` metadata). Also add `FindCountersWithLoans()` to `Quota`.
2. Create `internal/quota/provisioning.go` with `QuotaManager.ProvisionCounter` — the method
   the consumer service will call for each `QuotaProvisioningEvent`.

---

## Scope

### 1 — `internal/quota/quota.go` — add CanRepayLoan to Counter and FindCountersWithLoans to Quota

**Add `CanRepayLoan bool` to Counter struct:**
```go
// CanRepayLoan indicates whether newly provisioned balance on this counter
// is eligible to trigger clawback repayment of outstanding loans on other counters.
// Always false when the counter itself carries a Loan.
CanRepayLoan bool `json:"canRepayLoan"`
```
Place after `CanConvert`. This is backward-compatible: existing JSONB records will
deserialise to `false` (the zero value), which is the correct default.

**Add `FindCountersWithLoans()` to Quota:**
```go
// FindCountersWithLoans returns all counters that have an outstanding loan balance,
// in insertion order (oldest first). The caller must iterate oldest-first to honour
// the loan clawback ordering guarantee.
func (q *Quota) FindCountersWithLoans() []*Counter {
    list := make([]*Counter, 0)
    for i := range q.Counters {
        c := &q.Counters[i]
        if c.Loan != nil && c.Loan.LoanBalance.GreaterThan(decimal.Zero) {
            list = append(list, c)
        }
    }
    return list
}
```

---

### 2 — `internal/quota/provisioning.go` — ProvisionCounter on QuotaManager

```go
// ProvisionCounterRequest carries all parameters for a single counter provisioning operation.
type ProvisionCounterRequest struct {
    SubscriberID         uuid.UUID
    CounterID            uuid.UUID
    ProductID            uuid.UUID
    ProductName          string
    Description          string
    UnitType             charging.UnitType
    Priority             int
    InitialBalance       decimal.Decimal
    ExpiryDate           *time.Time
    CanRepayLoan         bool
    CanTransfer          bool
    CanConvert           bool
    UnitPrice            *decimal.Decimal
    TaxRate              *decimal.Decimal
    CounterSelectionKeys []charging.RateKey
    ExternalReference    string
    ReasonCode           ReasonCode
    LoanInfo             *LoanProvisionInfo  // nil = no loan
    Now                  time.Time           // caller-supplied reference time
    TransactionID        string              // links all journal events for this provision
}

// LoanProvisionInfo carries the loan configuration from the provisioning event.
type LoanProvisionInfo struct {
    MinRepayment       decimal.Decimal
    ClawbackPercentage decimal.Decimal
}

// ProvisionCounter provisions a new counter onto the subscriber's quota.
// It is idempotent: if a counter with the same CounterID already exists it
// returns nil without making any changes.
//
// When LoanInfo is present the counter receives a Loan with loanBalance = initialBalance
// and transactFee = initialBalance; CanRepayLoan is forced to false.
//
// After provisioning, a QUOTA_PROVISIONED (or caller-supplied reason) journal event
// is published. If CanRepayLoan is true, clawback is triggered against all
// outstanding loan counters oldest-first, publishing TRANSACTION_FEE and/or
// LOAN_REPAYMENT journal events per loan.
func (m *QuotaManager) ProvisionCounter(ctx context.Context, req ProvisionCounterRequest) error
```

**Implementation steps inside ProvisionCounter:**

1. Use `m.executeWithQuota(ctx, req.Now, req.SubscriberID, func(q *Quota) error { ... })`

2. Inside the operation:
   a. **Idempotency**: if `q.FindCounterByID(req.CounterID) != nil` → return nil
   b. **Build counter**: construct `Counter` from request fields. `Balance = &req.InitialBalance`.
      `Reservations = make(map[uuid.UUID]Reservation)`.
   c. **Loan attachment**: if `req.LoanInfo != nil`:
      - Set `counter.Loan = &Loan{LoanBalance: req.InitialBalance, TransactFee: req.InitialBalance, MinRepayment: req.LoanInfo.MinRepayment, ClawbackPercentage: req.LoanInfo.ClawbackPercentage}`
      - Force `counter.CanRepayLoan = false`
   d. **Add counter**: `q.AddCounter(counter)`
   e. **Publish provisioning journal event**: call `PublishJournalEvent` with:
      - `reasonCode = req.ReasonCode`
      - `adjustedUnits = req.InitialBalance`
      - `metaData`: only when `req.ReasonCode == ReasonQuotaProvisioned` — build a `*CounterEvent` from the counter fields; otherwise pass `nil`
   f. **Clawback**: if `counter.CanRepayLoan`:
      - `loanCounters := q.FindCountersWithLoans()`
      - `remaining := req.InitialBalance`
      - For each loan counter (oldest-first):
        - `loanPaid, feePaid := loanCounter.Loan.Clawback(remaining)`
        - If `feePaid > 0`: debit new counter by feePaid; publish journal with `ReasonTransactionFee`, `adjustedUnits = -feePaid`
        - If `loanPaid > 0`: debit new counter by loanPaid; publish journal with `ReasonLoanRepayment`, `adjustedUnits = -loanPaid`
        - `remaining = remaining - feePaid - loanPaid`
        - Stop if `remaining <= 0`
   g. Return nil

Note: `TaxCalculation` passed to `PublishJournalEvent` should be zero-value (`TaxCalculation{}`) for provisioning events — provisioning doesn't apply tax.

---

### 3 — `internal/quota/provisioning_test.go`

Table-driven tests using a mock Repository:

```
TestProvisionCounter_NewCounter_CreatesCounterAndPublishesJournal
TestProvisionCounter_DuplicateCounterID_IsIdempotent
TestProvisionCounter_WithLoanInfo_AttachesLoanAndForcesCanRepayLoanFalse
TestProvisionCounter_CanRepayLoan_TriggersClawbackOldestFirst
TestProvisionCounter_CanRepayLoan_StopsWhenRemainingBalanceExhausted
TestProvisionCounter_UnknownReasonCode_SubstitutedToQuotaProvisioned (handled at consumer level, not here)
TestFindCountersWithLoans_ReturnsOnlyLoanCounters
TestFindCountersWithLoans_EmptyWhenNoLoans
```

Use a stub KafkaManager (nil client, no-op publisher) for unit tests. Check that the quota
state is mutated correctly (counter added, loan set, balances adjusted after clawback).

---

## Acceptance Criteria

- [ ] `Counter.CanRepayLoan` field exists and round-trips through JSON correctly
- [ ] `Quota.FindCountersWithLoans()` returns counters with outstanding loans in insertion order
- [ ] `ProvisionCounterRequest` and `LoanProvisionInfo` types are exported
- [ ] `QuotaManager.ProvisionCounter` exists and satisfies all bullet points above
- [ ] Idempotency: second call with same CounterID is a no-op
- [ ] Loan attachment: sets loanBalance = transactFee = initialBalance; forces CanRepayLoan = false
- [ ] Clawback: publishes TRANSACTION_FEE then LOAN_REPAYMENT per loan counter, oldest-first
- [ ] All unit tests pass with `go test -race ./internal/quota/...`
- [ ] `go build ./...` passes
