package events

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fixed UUIDs used across all SubscriberEvent tests.
var (
	testSubscriberID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testRatePlanID   = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	testCustomerID   = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	testWholesaleID  = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	testContractID   = uuid.MustParse("55555555-5555-5555-5555-555555555555")
)

// newTestSubscriberEvent returns a SubscriberEvent populated with fixed test values.
func newTestSubscriberEvent(eventType SubscriberEventType) SubscriberEvent {
	return SubscriberEvent{
		EventType:        eventType,
		SubscriberID:     testSubscriberID,
		RatePlanID:       testRatePlanID,
		CustomerID:       testCustomerID,
		WholesaleID:      testWholesaleID,
		Msisdn:           "27820001001",
		Iccid:            "8927001234567890",
		ContractID:       testContractID,
		Status:           "ACTIVE",
		AllowOobCharging: true,
	}
}

// ---------------------------------------------------------------------------
// Event type constants
// ---------------------------------------------------------------------------

func TestSubscriberEventType_Constants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, SubscriberEventType("CREATED"), SubscriberEventCreated)
	assert.Equal(t, SubscriberEventType("UPDATED"), SubscriberEventUpdated)
	assert.Equal(t, SubscriberEventType("MSISDN_SWAP"), SubscriberEventMsisdnSwap)
	assert.Equal(t, SubscriberEventType("SIM_SWAP"), SubscriberEventSimSwap)
	assert.Equal(t, SubscriberEventType("DELETED"), SubscriberEventDeleted)
}

// ---------------------------------------------------------------------------
// JSON round-trip — one test per event type
// ---------------------------------------------------------------------------

func TestSubscriberEvent_MarshalUnmarshal_Created(t *testing.T) {
	t.Parallel()
	original := newTestSubscriberEvent(SubscriberEventCreated)

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded SubscriberEvent
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, SubscriberEventCreated, decoded.EventType)
	assert.Equal(t, testSubscriberID, decoded.SubscriberID)
	assert.Equal(t, testRatePlanID, decoded.RatePlanID)
	assert.Equal(t, testCustomerID, decoded.CustomerID)
	assert.Equal(t, testWholesaleID, decoded.WholesaleID)
	assert.Equal(t, "27820001001", decoded.Msisdn)
	assert.Equal(t, "8927001234567890", decoded.Iccid)
	assert.Equal(t, testContractID, decoded.ContractID)
	assert.Equal(t, "ACTIVE", decoded.Status)
	assert.True(t, decoded.AllowOobCharging)
}

func TestSubscriberEvent_MarshalUnmarshal_Updated(t *testing.T) {
	t.Parallel()
	original := newTestSubscriberEvent(SubscriberEventUpdated)
	data, err := json.Marshal(original)
	require.NoError(t, err)
	var decoded SubscriberEvent
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, SubscriberEventUpdated, decoded.EventType)
}

func TestSubscriberEvent_MarshalUnmarshal_MsisdnSwap(t *testing.T) {
	t.Parallel()
	original := newTestSubscriberEvent(SubscriberEventMsisdnSwap)
	data, err := json.Marshal(original)
	require.NoError(t, err)
	var decoded SubscriberEvent
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, SubscriberEventMsisdnSwap, decoded.EventType)
}

func TestSubscriberEvent_MarshalUnmarshal_SimSwap(t *testing.T) {
	t.Parallel()
	original := newTestSubscriberEvent(SubscriberEventSimSwap)
	data, err := json.Marshal(original)
	require.NoError(t, err)
	var decoded SubscriberEvent
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, SubscriberEventSimSwap, decoded.EventType)
}

func TestSubscriberEvent_MarshalUnmarshal_Deleted(t *testing.T) {
	t.Parallel()
	original := newTestSubscriberEvent(SubscriberEventDeleted)
	original.AllowOobCharging = false
	data, err := json.Marshal(original)
	require.NoError(t, err)
	var decoded SubscriberEvent
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, SubscriberEventDeleted, decoded.EventType)
	assert.False(t, decoded.AllowOobCharging)
}

// ---------------------------------------------------------------------------
// JSON field name verification — ensure tag names are correct
// ---------------------------------------------------------------------------

func TestSubscriberEvent_JSONFieldNames(t *testing.T) {
	t.Parallel()
	event := newTestSubscriberEvent(SubscriberEventCreated)
	data, err := json.Marshal(event)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))

	expectedKeys := []string{
		"eventType", "subscriberId", "ratePlanId", "customerId",
		"wholesaleId", "msisdn", "iccid", "contractId", "status", "allowOobCharging",
	}
	for _, key := range expectedKeys {
		assert.Contains(t, raw, key, "expected JSON field %q to be present", key)
	}
}
