package quota

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"go-ocs/internal/charging"
)

// ProvisionCounterRequest carries all parameters for a single counter provisioning operation.
// The caller supplies Now as the reference time to ensure deterministic journal timestamps.
type ProvisionCounterRequest struct {
	// SubscriberID is the owner of the quota receiving the new counter.
	SubscriberID uuid.UUID

	// CounterID is the deterministic identifier for the new counter.
	// If a counter with this ID already exists the operation is a no-op.
	CounterID uuid.UUID

	// ProductID is the product linked to this counter.
	ProductID uuid.UUID

	// ProductName is the human-readable product name.
	ProductName string

	// Description is a human-readable description for the counter.
	Description string

	// UnitType is the type of units tracked by this counter.
	UnitType charging.UnitType

	// Priority determines counter selection order — higher value is preferred.
	Priority int

	// InitialBalance is the starting balance loaded onto the counter.
	InitialBalance decimal.Decimal

	// ExpiryDate is the optional expiry time for the counter.
	ExpiryDate *time.Time

	// CanRepayLoan indicates whether this counter's balance triggers clawback of
	// outstanding loans. Forced to false when LoanInfo is non-nil.
	CanRepayLoan bool

	// CanTransfer indicates whether this counter's balance can be transferred.
	CanTransfer bool

	// CanConvert indicates whether this counter's balance can be converted.
	CanConvert bool

	// UnitPrice is the optional monetary price per service unit.
	UnitPrice *decimal.Decimal

	// TaxRate is the optional tax rate applied when this counter is debited.
	TaxRate *decimal.Decimal

	// CounterSelectionKeys are the rate keys used to match this counter to usage.
	CounterSelectionKeys []charging.RateKey

	// ExternalReference is an optional external system identifier.
	ExternalReference string

	// ReasonCode is the business reason code for the provisioning journal event.
	ReasonCode ReasonCode

	// LoanInfo is optional. When non-nil, the counter receives a Loan with
	// loanBalance = initialBalance and transactFee = initialBalance.
	// CanRepayLoan is forced to false when LoanInfo is provided.
	LoanInfo *LoanProvisionInfo

	// Now is the caller-supplied reference time for journal timestamps and expiry.
	Now time.Time

	// TransactionID links all journal events published for this provisioning.
	TransactionID string
}

// LoanProvisionInfo carries the loan configuration from the provisioning event.
type LoanProvisionInfo struct {
	// TransactionFee is the fee charged for issuing this loan. It is collected first
	// during clawback before any principal repayment is applied.
	TransactionFee decimal.Decimal

	// MinRepayment is the minimum repayment amount per clawback cycle.
	MinRepayment decimal.Decimal

	// ClawbackPercentage is the fraction of incoming balance used to repay this loan (0–1).
	ClawbackPercentage decimal.Decimal
}

// ProvisionCounter provisions a new counter onto the subscriber's quota.
//
// The operation is idempotent: if a counter with the same CounterID already exists,
// the method returns nil without making any changes.
//
// When LoanInfo is non-nil, a Loan is attached to the counter with
// loanBalance = initialBalance and transactFee = initialBalance; CanRepayLoan is
// forced to false.
//
// A journal event is published for the provisioned counter. When CanRepayLoan is true,
// clawback is triggered against all outstanding loan counters oldest-first, publishing
// TRANSACTION_FEE and LOAN_REPAYMENT journal events per loan with outstanding balance.
func (m *QuotaManager) ProvisionCounter(ctx context.Context, req ProvisionCounterRequest) error {
	return m.executeWithQuota(ctx, req.Now, req.SubscriberID, func(q *Quota) error {
		return m.applyProvisioning(q, req)
	})
}

// applyProvisioning performs the counter provisioning mutation on the in-memory Quota.
// It is called inside the executeWithQuota retry loop which handles load, save, and
// optimistic-lock retries.
func (m *QuotaManager) applyProvisioning(q *Quota, req ProvisionCounterRequest) error {
	// Idempotency check — skip silently if counter already exists.
	if q.FindCounterByID(req.CounterID) != nil {
		return nil
	}

	// Build the initial balance pointer — Counter fields are pointer types.
	balance := req.InitialBalance

	counter := Counter{
		CounterID:            req.CounterID,
		ProductID:            req.ProductID,
		ProductName:          req.ProductName,
		Description:          req.Description,
		UnitType:             req.UnitType,
		Priority:             req.Priority,
		InitialBalance:       &balance,
		Balance:              &balance,
		Expiry:               req.ExpiryDate,
		CanTransfer:          req.CanTransfer,
		CanConvert:           req.CanConvert,
		UnitPrice:            req.UnitPrice,
		TaxRate:              req.TaxRate,
		CounterSelectionKeys: req.CounterSelectionKeys,
		ExternalReference:    req.ExternalReference,
		Reservations:         make(map[uuid.UUID]Reservation),
	}

	// Attach loan if provided.
	if req.LoanInfo != nil {
		counter.Loan = &Loan{
			LoanBalance:        req.InitialBalance,
			TransactFee:        req.LoanInfo.TransactionFee,
			MinRepayment:       req.LoanInfo.MinRepayment,
			ClawbackPercentage: req.LoanInfo.ClawbackPercentage,
		}
	}

	q.AddCounter(counter)

	// Obtain a stable pointer to the newly added counter for all subsequent operations.
	newCounter := &q.Counters[len(q.Counters)-1]

	// Determine metadata for the provisioning journal event.
	var metaData *CounterEvent
	if req.ReasonCode == ReasonQuotaProvisioned {
		metaData = buildCounterMetaData(newCounter, req)
	}

	// Publish the provisioning journal event.
	PublishJournalEvent(
		m,
		q.QuotaID,
		req.TransactionID,
		newCounter,
		req.ReasonCode,
		req.InitialBalance,
		req.UnitType,
		TaxCalculation{},
		req.SubscriberID,
		metaData,
		req.Now,
	)

	// Trigger clawback when the provisioning request permits loan repayment
	// and the counter is not itself a loaned counter.
	if req.CanRepayLoan && req.LoanInfo == nil {
		m.applyClawback(q, newCounter, req)
	}

	return nil
}

// applyClawback iterates all loan counters oldest-first and claws back outstanding
// balances from newCounter, publishing TRANSACTION_FEE and LOAN_REPAYMENT journal
// events per loan until the new counter's balance is exhausted.
func (m *QuotaManager) applyClawback(q *Quota, newCounter *Counter, req ProvisionCounterRequest) {
	remaining := req.InitialBalance

	for _, loanCounter := range q.FindCountersWithLoans() {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}

		loanPaid, feePaid := loanCounter.Loan.Clawback(remaining)

		if feePaid.GreaterThan(decimal.Zero) {
			newCounter.DebitBalance(feePaid)
			remaining = remaining.Sub(feePaid)
			PublishJournalEvent(
				m,
				q.QuotaID,
				req.TransactionID,
				newCounter,
				ReasonTransactionFee,
				feePaid.Neg(),
				req.UnitType,
				TaxCalculation{},
				req.SubscriberID,
				nil,
				req.Now,
			)
		}

		if loanPaid.GreaterThan(decimal.Zero) {
			newCounter.DebitBalance(loanPaid)
			remaining = remaining.Sub(loanPaid)
			PublishJournalEvent(
				m,
				q.QuotaID,
				req.TransactionID,
				newCounter,
				ReasonLoanRepayment,
				loanPaid.Neg(),
				req.UnitType,
				TaxCalculation{},
				req.SubscriberID,
				nil,
				req.Now,
			)
		}
	}
}

// buildCounterMetaData constructs the CounterEvent metadata payload included in
// QUOTA_PROVISIONED journal events.
func buildCounterMetaData(c *Counter, req ProvisionCounterRequest) *CounterEvent {
	var expiry time.Time
	if c.Expiry != nil {
		expiry = *c.Expiry
	}

	var initialBalance decimal.Decimal
	if c.InitialBalance != nil {
		initialBalance = *c.InitialBalance
	}

	var balance decimal.Decimal
	if c.Balance != nil {
		balance = *c.Balance
	}

	return &CounterEvent{
		ProductID:            c.ProductID,
		ProductName:          c.ProductName,
		Description:          c.Description,
		UnitType:             c.UnitType,
		Priority:             c.Priority,
		InitialBalance:       initialBalance,
		Balance:              balance,
		Expiry:               expiry,
		CanRepayLoan:         req.CanRepayLoan && req.LoanInfo == nil,
		CanTransfer:          c.CanTransfer,
		CanConvert:           c.CanConvert,
		CounterSelectionKeys: req.CounterSelectionKeys,
	}
}
