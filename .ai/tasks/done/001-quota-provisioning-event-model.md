# Task 001 — QuotaProvisioningEvent model in internal/events/

**Feature:** F-007 — QuotaProvisioningEventConsumer
**Sequence:** 1 of 4
**Date:** 2026-03-21

---

## Objective

Define the `QuotaProvisioningEvent` and `LoanInfo` structs in `internal/events/` so the
consumer can deserialise messages from the `public.quota-provisioning` Kafka topic. This task
is purely a data model — no logic, no tests required (struct-only files are exempt per Go context rules).

---

## Scope

**Create `internal/events/quota_provisioning_event.go`**

### ProvisioningReasonCode type
```go
type ProvisioningReasonCode string

const (
    ProvisioningReasonQuotaProvisioned ProvisioningReasonCode = "QUOTA_PROVISIONED"
    ProvisioningReasonLoanRepayment    ProvisioningReasonCode = "LOAN_REPAYMENT"
    ProvisioningReasonRefund           ProvisioningReasonCode = "REFUND"
    ProvisioningReasonTransferIn       ProvisioningReasonCode = "TRANSFER_IN"
    ProvisioningReasonConversion       ProvisioningReasonCode = "CONVERSION"
)
```

### LoanInfo struct
Carries the loan configuration embedded in a provisioning event. The consumer uses
`initialBalance` from the parent event to set both `loanBalance` and `transactFee`.
```go
type LoanInfo struct {
    MinRepayment       decimal.Decimal `json:"minRepayment"`
    ClawbackPercentage decimal.Decimal `json:"clawbackPercentage"`
}
```

### QuotaProvisioningEvent struct
All counter fields are present in the event — the consumer maps them directly to
a `quota.Counter`. Renewal fields (renewalCount, renewalInterval, renewalDay) are
present for wire compatibility but are explicitly NOT persisted (renewal is deprecated).
```go
type QuotaProvisioningEvent struct {
    // EventID uniquely identifies this provisioning event for idempotency checks.
    EventID uuid.UUID `json:"eventId"`

    // SubscriberID is the subscriber whose quota receives the new counter.
    SubscriberID uuid.UUID `json:"subscriberId"`

    // CounterID is the deterministic ID for the new counter (idempotency key).
    CounterID uuid.UUID `json:"counterId"`

    // ProductID is the product associated with this counter.
    ProductID uuid.UUID `json:"productId"`

    // ProductName is the human-readable product name.
    ProductName string `json:"productName"`

    // Description is a human-readable description for this counter.
    Description string `json:"description"`

    // UnitType is the unit type (MONETARY, OCTETS, SECONDS, etc.).
    UnitType charging.UnitType `json:"unitType"`

    // Priority is the counter priority — higher value is preferred for consumption.
    Priority int `json:"priority"`

    // InitialBalance is the starting balance loaded onto the counter.
    InitialBalance decimal.Decimal `json:"initialBalance"`

    // ExpiryDate is the optional expiry time for the counter.
    ExpiryDate *time.Time `json:"expiryDate,omitempty"`

    // CanRepayLoan indicates whether this counter's balance can be used to clawback
    // outstanding loans on other counters. Forced to false when LoanInfo is present.
    CanRepayLoan bool `json:"canRepayLoan"`

    // CanTransfer indicates whether this counter's balance can be transferred.
    CanTransfer bool `json:"canTransfer"`

    // CanConvert indicates whether this counter's balance can be converted.
    CanConvert bool `json:"canConvert"`

    // UnitPrice is the optional unit price for this counter.
    UnitPrice *decimal.Decimal `json:"unitPrice,omitempty"`

    // TaxRate is the optional tax rate applied to this counter.
    TaxRate *decimal.Decimal `json:"taxRate,omitempty"`

    // CounterSelectionKeys are the rate keys used to match this counter to usage.
    CounterSelectionKeys []charging.RateKey `json:"counterSelectionKeys"`

    // ExternalReference is an optional external system reference.
    ExternalReference string `json:"externalReference,omitempty"`

    // ReasonCode is the provisioning reason, mapped to quota.ReasonCode by the consumer.
    // Unknown values are substituted with QUOTA_PROVISIONED.
    ReasonCode ProvisioningReasonCode `json:"reasonCode"`

    // LoanInfo is optional. When present, the counter receives a Loan with
    // loanBalance = initialBalance and transactFee = initialBalance.
    // canRepayLoan is forced to false when LoanInfo is provided.
    LoanInfo *LoanInfo `json:"loanInfo,omitempty"`

    // RenewalCount, RenewalInterval, RenewalDay — present for wire compatibility.
    // Renewal is deprecated; these fields are intentionally ignored by the consumer.
    RenewalCount    int    `json:"renewalCount,omitempty"`
    RenewalInterval string `json:"renewalInterval,omitempty"`
    RenewalDay      int    `json:"renewalDay,omitempty"`
}
```

**No other files are created or modified in this task.**

---

## Acceptance Criteria

- [ ] `internal/events/quota_provisioning_event.go` exists and compiles
- [ ] `QuotaProvisioningEvent`, `LoanInfo`, and `ProvisioningReasonCode` are exported
- [ ] All struct fields have correct `json:` tags matching the wire format
- [ ] `go build ./...` passes
