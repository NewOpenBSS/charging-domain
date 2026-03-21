package events

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"go-ocs/internal/charging"
)

// ProvisioningReasonCode identifies the business reason for a quota provisioning event.
// Unknown values received from upstream systems are substituted with
// ProvisioningReasonQuotaProvisioned by the consumer.
type ProvisioningReasonCode string

const (
	// ProvisioningReasonQuotaProvisioned is the standard reason for a new quota allocation.
	ProvisioningReasonQuotaProvisioned ProvisioningReasonCode = "QUOTA_PROVISIONED"

	// ProvisioningReasonTransferIn signals that balance was transferred in from another counter.
	ProvisioningReasonTransferIn ProvisioningReasonCode = "TRANSFER_IN"
)

// LoanInfo carries the loan configuration embedded in a QuotaProvisioningEvent.
// When present, the consumer creates a Loan on the new counter with
// loanBalance = initialBalance and transactFee = transactionFee.
type LoanInfo struct {
	// TransactionFee is the fee charged for issuing this loan. It is collected first
	// during clawback before any principal repayment is applied.
	TransactionFee decimal.Decimal `json:"transactionFee"`

	// MinRepayment is the minimum amount that must be repaid per clawback cycle.
	MinRepayment decimal.Decimal `json:"minRepayment"`

	// ClawbackPercentage is the fraction of incoming balance used to repay this loan (0–1).
	// A zero value means MinRepayment applies instead.
	ClawbackPercentage decimal.Decimal `json:"clawbackPercentage"`
}

// QuotaProvisioningEvent is the Kafka message payload published on the
// public.quota-provisioning topic by upstream billing or product systems.
// Each event provisions a single counter onto the named subscriber's quota.
type QuotaProvisioningEvent struct {
	// EventID uniquely identifies this provisioning event. Used by the consumer
	// as the idempotency key for the counter — a counter derived from the same
	// EventID is silently skipped if it already exists.
	EventID uuid.UUID `json:"eventId"`

	// SubscriberID is the subscriber whose quota receives the new counter.
	SubscriberID uuid.UUID `json:"subscriberId"`

	// ProductID is the product associated with this counter.
	ProductID uuid.UUID `json:"productId"`

	// ProductName is the human-readable product name.
	ProductName string `json:"productName"`

	// Description is a human-readable description for this counter.
	Description string `json:"description"`

	// UnitType is the type of units tracked by this counter (e.g. MONETARY, OCTETS).
	UnitType charging.UnitType `json:"unitType"`

	// Priority determines counter selection order — higher value is preferred.
	Priority int `json:"priority"`

	// InitialBalance is the starting balance loaded onto the counter.
	InitialBalance decimal.Decimal `json:"initialBalance"`

	// ExpiryDate is the optional timestamp after which this counter expires.
	ExpiryDate *time.Time `json:"expiryDate,omitempty"`

	// CanRepayLoan indicates whether this counter's balance can trigger clawback
	// repayment of outstanding loans on other counters. Forced to false when
	// LoanInfo is present (a counter with its own loan cannot repay others).
	CanRepayLoan bool `json:"canRepayLoan"`

	// CanTransfer indicates whether this counter's balance can be transferred.
	CanTransfer bool `json:"canTransfer"`

	// CanConvert indicates whether this counter's balance can be converted.
	CanConvert bool `json:"canConvert"`

	// UnitPrice is the optional monetary price per service unit for this counter.
	UnitPrice *decimal.Decimal `json:"unitPrice,omitempty"`

	// TaxRate is the optional tax rate applied when this counter is debited.
	TaxRate *decimal.Decimal `json:"taxRate,omitempty"`

	// CounterSelectionKeys are the rate keys used to match this counter to usage events.
	CounterSelectionKeys []charging.RateKey `json:"counterSelectionKeys"`

	// ExternalReference is an optional identifier used by upstream systems.
	ExternalReference string `json:"externalReference,omitempty"`

	// ReasonCode is the business reason for this provisioning. Unknown values are
	// substituted with ProvisioningReasonQuotaProvisioned by the consumer.
	ReasonCode ProvisioningReasonCode `json:"reasonCode"`

	// LoanInfo is optional. When present, the counter receives a Loan with
	// loanBalance = initialBalance and transactFee = initialBalance.
	// canRepayLoan is forced to false when LoanInfo is provided.
	LoanInfo *LoanInfo `json:"loanInfo,omitempty"`

	// RenewalCount, RenewalInterval, RenewalDay — present for wire compatibility only.
	// Renewal is deprecated and these fields are intentionally ignored by the consumer.
	RenewalCount    int    `json:"renewalCount,omitempty"`
	RenewalInterval string `json:"renewalInterval,omitempty"`
	RenewalDay      int    `json:"renewalDay,omitempty"`
}
