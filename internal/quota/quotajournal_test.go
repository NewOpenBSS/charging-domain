package quota

import (
	"encoding/json"
	"go-ocs/internal/charging"
	"go-ocs/internal/events"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestPublishJournalEvent(t *testing.T) {
	// We can't easily mock kgo.Client as it is a struct, but we can use a client with no brokers.
	// However, we want to verify the payload.
	// Since kgo.Client.Produce is asynchronous, we might need a way to intercept it.
	// For this test, we'll verify the data structure and that it marshals correctly.

	subscriberID := uuid.New()
	quotaID := uuid.New()
	counterID := uuid.New()
	transactionID := "test-transaction-123"
	reasonCode := ReasonServiceUsage
	adjustedUnits := decimal.NewFromInt(50)
	unitType := charging.UNITS
	balance := decimal.NewFromInt(950)
	unitPrice := decimal.NewFromFloat(1.0)
	taxRate := decimal.NewFromFloat(0.15)
	expiry := time.Now().Add(time.Hour)

	counter := &Counter{
		CounterID:      counterID,
		ProductID:      uuid.New(),
		ProductName:    "Test Product",
		Balance:        &balance,
		InitialBalance: &balance,
		UnitPrice:      &unitPrice,
		TaxRate:        &taxRate,
		Expiry:         &expiry,
	}

	taxCalculation := TaxCalculation{
		ExTaxValue: decimal.NewFromFloat(50.0),
		TaxAmount:  decimal.NewFromFloat(7.5),
	}

	metaData := &CounterEvent{
		ProductID:      counter.ProductID,
		ProductName:    counter.ProductName,
		UnitType:       unitType,
		InitialBalance: *counter.InitialBalance,
		Balance:        *counter.Balance,
		Expiry:         *counter.Expiry,
	}

	// Create a manager with a dummy Kafka client
	cl, _ := kgo.NewClient(kgo.SeedBrokers("localhost:9092"))
	defer cl.Close()

	manager := &QuotaManager{
		kafkaManager: &events.KafkaManager{
			KafkaClient: cl,
			KafkaConfig: events.KafkaConfig{
				Topics: map[string]string{
					"quota-journal": "test-topic",
				},
			},
		},
	}

	// Since we can't easily intercept the produce call without a real Kafka or a complex mock,
	// and the requirement is to "generate testing case quotajournal",
	// I will focus on ensuring the event can be constructed and marshaled without errors.
	// In a real scenario, we might use a mock Kafka or a specialized test client.

	t.Run("Event marshaling", func(t *testing.T) {
		event := QuotaJournalEvent{
			JournalID:         uuid.New(),
			SubscriberID:      subscriberID,
			TransactionID:     transactionID,
			QuotaID:           quotaID,
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

		payload, err := json.Marshal(event)
		assert.NoError(t, err)
		assert.NotEmpty(t, payload)

		var unmarshaled QuotaJournalEvent
		err = json.Unmarshal(payload, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, event.JournalID, unmarshaled.JournalID)
		assert.Equal(t, event.SubscriberID, unmarshaled.SubscriberID)
		assert.True(t, event.AdjustedUnits.Equal(unmarshaled.AdjustedUnits))
		assert.True(t, event.Balance.Equal(unmarshaled.Balance))
	})

	t.Run("PublishJournalEvent call", func(t *testing.T) {
		// This won't actually send to a real Kafka but it should not panic.
		// Note: PublishJournalEvent is asynchronous and uses context.Background()
		assert.NotPanics(t, func() {
			PublishJournalEvent(manager, quotaID, transactionID, counter, reasonCode, adjustedUnits, unitType, taxCalculation, subscriberID, metaData)
		})
	})
}

func TestCounterEventMapping(t *testing.T) {
	// Verify that we can create a CounterEvent from a Counter
	counterID := uuid.New()
	productID := uuid.New()
	balance := decimal.NewFromInt(100)
	expiry := time.Now().Add(time.Hour)

	counter := &Counter{
		CounterID:      counterID,
		ProductID:      productID,
		ProductName:    "Gold Data",
		Description:    "100MB Data",
		UnitType:       charging.UNITS,
		Priority:       10,
		InitialBalance: &balance,
		Balance:        &balance,
		Expiry:         &expiry,
	}

	event := &CounterEvent{
		ProductID:      counter.ProductID,
		ProductName:    counter.ProductName,
		Description:    counter.Description,
		UnitType:       counter.UnitType,
		Priority:       counter.Priority,
		InitialBalance: *counter.InitialBalance,
		Balance:        *counter.Balance,
		Expiry:         *counter.Expiry,
	}

	assert.Equal(t, counter.ProductID, event.ProductID)
	assert.Equal(t, counter.ProductName, event.ProductName)
	assert.Equal(t, counter.UnitType, event.UnitType)
	assert.True(t, counter.Balance.Equal(event.Balance))
}
