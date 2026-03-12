package quota

import (
	"go-ocs/internal/charging"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type IntervalPeriod string

const (
	IntervalWeekly      IntervalPeriod = "WEEKLY"
	IntervalFortnightly IntervalPeriod = "FORTNIGHTLY"
	IntervalMonthly     IntervalPeriod = "MONTHLY"
)

// QuotaJournalEvent represents a journal entry for quota adjustments,
// including usage and provisioning events.
type QuotaJournalEvent struct {

	// Unique reference ID for this journal entry
	JournalID uuid.UUID `json:"journalId"`

	// The transaction id linked to the event
	TransactionID string `json:"transactionId"`

	// Identifier of the quota affected by this event
	QuotaID uuid.UUID `json:"quotaId"`

	// Identifier of the counter affected by this event
	CounterID uuid.UUID `json:"counterId"`

	// Subscriber associated with the quota adjustment
	SubscriberID uuid.UUID `json:"subscriberId"`

	// Number of units adjusted (positive or negative)
	AdjustedUnits decimal.Decimal `json:"adjustedUnits"`

	// Resulting balance after the adjustment
	Balance decimal.Decimal `json:"balance"`

	// Type of units involved (e.g. OCTETS, MINUTES)
	UnitType charging.UnitType `json:"unitType"`

	// Monetary value of the adjustment excluding tax
	ValueExTax decimal.Decimal `json:"valueExTax"`

	// Tax amount applied to the adjustment
	TaxAmount decimal.Decimal `json:"taxAmount"`

	// Reason code for the adjustment
	ReasonCode ReasonCode `json:"reasonCode"`

	// Timestamp when the adjustment occurred
	Timestamp time.Time `json:"timestamp"`

	// External reference associated with the event
	ExternalReference string `json:"externalReference"`

	// Product affected by the event
	ProductID uuid.UUID `json:"productId"`

	ProductName string `json:"productName"`

	// Metadata related to the counter
	CounterMetaData *CounterEvent `json:"counterMetaData"`
}

// CounterEvent represents metadata about a quota counter at the time of an event.
type CounterEvent struct {

	// Unique identifier for the product.
	ProductID uuid.UUID `json:"productId"`

	// Name of the product.
	ProductName string `json:"productName"`

	// Human-readable description for the counter.
	Description string `json:"description"`

	// Type of unit this counter tracks.
	UnitType charging.UnitType `json:"unitType"`

	// Priority of the counter. Higher priority counters are preferred for consumption.
	Priority int `json:"priority"`

	// Initial balance of the counter.
	InitialBalance decimal.Decimal `json:"initialBalance"`

	// Remaining balance available for consumption.
	Balance decimal.Decimal `json:"balance"`

	// Expiration timestamp of the counter.
	Expiry time.Time `json:"expiry"`

	// Number of times the counter can renew.
	NrRenewals int `json:"nrRenewals"`

	// Renewal interval (e.g. weekly, monthly).
	RenewalInterval IntervalPeriod `json:"renewalInterval"`

	// Day of the week or month when renewal occurs.
	RenewalDay int `json:"renewalDay"`

	// Duration before the renewed counter expires.
	ExpiryDuration time.Duration `json:"expiryDuration"`

	// Timestamp of the next scheduled renewal.
	NextRenewalDate time.Time `json:"nextRenewalDate"`

	// Indicates if renewal can be used to repay a loan.
	CanRepayLoan bool `json:"canRepayLoan"`

	// Indicates if balance can be transferred.
	CanTransfer bool `json:"canTransfer"`

	// Indicates if balance can be converted.
	CanConvert bool `json:"canConvert"`

	// Selection keys determining which usage applies to this counter.
	CounterSelectionKeys []charging.RateKey `json:"counterSelectionKeys"`
}

func PublishJournalEvent(manager *QuotaManager, quotaId uuid.UUID, transactionId string, counter *Counter, reasonCode ReasonCode, adjustedUnits decimal.Decimal, unitType charging.UnitType, taxCalculation TaxCalculation, subscriberId uuid.UUID, metaData *CounterEvent) {

	journalId := uuid.New()
	event := QuotaJournalEvent{
		JournalID:         journalId,
		SubscriberID:      subscriberId,
		TransactionID:     transactionId,
		QuotaID:           quotaId,
		CounterID:         counter.CounterID,
		ProductID:         counter.ProductID,
		ProductName:       counter.ProductName,
		ReasonCode:        reasonCode,
		Timestamp:         time.Now(),
		ExternalReference: counter.ExternalReference,
		UnitType:          unitType,
		Balance:           *counter.Balance,
		AdjustedUnits:     adjustedUnits,
		TaxAmount:         taxCalculation.TaxAmount,
		ValueExTax:        taxCalculation.ExTaxValue,
		CounterMetaData:   metaData,
	}

	manager.kafkaManager.PublishEvent("quota-journal", journalId.String(), event)
}
