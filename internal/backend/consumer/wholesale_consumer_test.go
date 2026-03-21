package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"

	"go-ocs/internal/events"
)

// ---------------------------------------------------------------------------
// Mock WholesaleStorer
// ---------------------------------------------------------------------------

type mockWholesaleStorer struct {
	mock.Mock
}

func (m *mockWholesaleStorer) UpsertWholesaler(ctx context.Context, event *events.WholesaleContractProvisionedEvent) error {
	return m.Called(ctx, event).Error(0)
}

func (m *mockWholesaleStorer) DeregisterWholesaler(ctx context.Context, wholesalerID uuid.UUID) error {
	return m.Called(ctx, wholesalerID).Error(0)
}

func (m *mockWholesaleStorer) SuspendWholesaler(ctx context.Context, wholesalerID uuid.UUID, suspend bool) error {
	return m.Called(ctx, wholesalerID, suspend).Error(0)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var testWholesalerID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
var testContractID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
var testRatePlanID = uuid.MustParse("33333333-3333-3333-3333-333333333333")

// newWholesaleRecord encodes v as JSON and wraps it in a *kgo.Record.
func newWholesaleRecord(t *testing.T, v any) *kgo.Record {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return &kgo.Record{
		Topic:     "public.wholesale-contract-event",
		Partition: 0,
		Offset:    0,
		Value:     data,
	}
}

// newWholesaleConsumer builds a consumer with a nil kgo.Client (disabled-Kafka mode) for tests.
func newWholesaleConsumer(storer WholesaleStorer) *WholesaleContractConsumer {
	return &WholesaleContractConsumer{
		client: nil,
		storer: storer,
		topic:  "public.wholesale-contract-event",
	}
}

// newProvisionedEvent returns a fully populated WholesaleContractProvisionedEvent.
func newProvisionedEvent() events.WholesaleContractProvisionedEvent {
	return events.WholesaleContractProvisionedEvent{
		EventType:   events.WholesaleContractProvisioned,
		WholesalerID: testWholesalerID,
		ContractID:  testContractID,
		RatePlanID:  testRatePlanID,
		LegalName:   "Acme Wholesale Ltd",
		DisplayName: "Acme",
		Realm:       "acme-realm",
		Hosts:       []string{"acme.example.com"},
		NchfUrl:     "https://acme.example.com/nchf",
		RateLimit:   100.0,
		Active:      true,
	}
}

// ---------------------------------------------------------------------------
// handleRecord — dispatch tests
// ---------------------------------------------------------------------------

func TestWholesaleHandleRecord_Provisioned_CallsUpsert(t *testing.T) {
	t.Parallel()
	storer := &mockWholesaleStorer{}
	event := newProvisionedEvent()
	storer.On("UpsertWholesaler", mock.Anything, &event).Return(nil)

	c := newWholesaleConsumer(storer)
	err := c.handleRecord(context.Background(), newWholesaleRecord(t, event))

	require.NoError(t, err)
	storer.AssertExpectations(t)
}

func TestWholesaleHandleRecord_Deregistering_CallsDeregister(t *testing.T) {
	t.Parallel()
	storer := &mockWholesaleStorer{}
	event := events.WholesaleContractDeregisteringEvent{
		EventType:    events.WholesaleContractDeregistering,
		WholesalerID: testWholesalerID,
	}
	storer.On("DeregisterWholesaler", mock.Anything, testWholesalerID).Return(nil)

	c := newWholesaleConsumer(storer)
	err := c.handleRecord(context.Background(), newWholesaleRecord(t, event))

	require.NoError(t, err)
	storer.AssertExpectations(t)
}

func TestWholesaleHandleRecord_Suspend_True_CallsSuspendWithTrue(t *testing.T) {
	t.Parallel()
	storer := &mockWholesaleStorer{}
	event := events.WholesaleContractSuspendEvent{
		EventType:    events.WholesaleContractSuspend,
		WholesalerID: testWholesalerID,
		Suspend:      true,
	}
	storer.On("SuspendWholesaler", mock.Anything, testWholesalerID, true).Return(nil)

	c := newWholesaleConsumer(storer)
	err := c.handleRecord(context.Background(), newWholesaleRecord(t, event))

	require.NoError(t, err)
	storer.AssertExpectations(t)
}

func TestWholesaleHandleRecord_Suspend_False_CallsSuspendWithFalse(t *testing.T) {
	t.Parallel()
	storer := &mockWholesaleStorer{}
	event := events.WholesaleContractSuspendEvent{
		EventType:    events.WholesaleContractSuspend,
		WholesalerID: testWholesalerID,
		Suspend:      false,
	}
	storer.On("SuspendWholesaler", mock.Anything, testWholesalerID, false).Return(nil)

	c := newWholesaleConsumer(storer)
	err := c.handleRecord(context.Background(), newWholesaleRecord(t, event))

	require.NoError(t, err)
	storer.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// handleRecord — malformed / unknown event
// ---------------------------------------------------------------------------

func TestWholesaleHandleRecord_MalformedJSON_SkipsWithoutError(t *testing.T) {
	t.Parallel()
	storer := &mockWholesaleStorer{}
	c := newWholesaleConsumer(storer)

	r := &kgo.Record{
		Topic: "public.wholesale-contract-event",
		Value: []byte(`{not valid json`),
	}
	err := c.handleRecord(context.Background(), r)

	assert.NoError(t, err, "malformed JSON must be skipped without returning an error")
	storer.AssertNotCalled(t, "UpsertWholesaler")
	storer.AssertNotCalled(t, "DeregisterWholesaler")
	storer.AssertNotCalled(t, "SuspendWholesaler")
}

func TestWholesaleHandleRecord_UnknownEventType_SkipsWithoutError(t *testing.T) {
	t.Parallel()
	storer := &mockWholesaleStorer{}
	c := newWholesaleConsumer(storer)

	type unknownEvent struct {
		EventType string `json:"eventType"`
	}
	r := newWholesaleRecord(t, unknownEvent{EventType: "UNKNOWN_WHOLESALE_OP"})
	err := c.handleRecord(context.Background(), r)

	assert.NoError(t, err, "unknown event type must be skipped without returning an error")
	storer.AssertNotCalled(t, "UpsertWholesaler")
	storer.AssertNotCalled(t, "DeregisterWholesaler")
	storer.AssertNotCalled(t, "SuspendWholesaler")
}

// ---------------------------------------------------------------------------
// handleRecord — store error propagation
// ---------------------------------------------------------------------------

func TestWholesaleHandleRecord_UpsertError_ReturnsError(t *testing.T) {
	t.Parallel()
	storer := &mockWholesaleStorer{}
	event := newProvisionedEvent()
	upsertErr := errors.New("db error")
	storer.On("UpsertWholesaler", mock.Anything, &event).Return(upsertErr)

	c := newWholesaleConsumer(storer)
	err := c.handleRecord(context.Background(), newWholesaleRecord(t, event))

	assert.Equal(t, upsertErr, err)
	storer.AssertExpectations(t)
}

func TestWholesaleHandleRecord_DeregisterError_ReturnsError(t *testing.T) {
	t.Parallel()
	storer := &mockWholesaleStorer{}
	event := events.WholesaleContractDeregisteringEvent{
		EventType:    events.WholesaleContractDeregistering,
		WholesalerID: testWholesalerID,
	}
	deregErr := errors.New("db error")
	storer.On("DeregisterWholesaler", mock.Anything, testWholesalerID).Return(deregErr)

	c := newWholesaleConsumer(storer)
	err := c.handleRecord(context.Background(), newWholesaleRecord(t, event))

	assert.Equal(t, deregErr, err)
	storer.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Start / Stop with disabled Kafka
// ---------------------------------------------------------------------------

func TestWholesaleStart_KafkaDisabled_ReturnsImmediately(t *testing.T) {
	t.Parallel()
	cfg := &events.KafkaConfig{Enabled: false}
	storer := &mockWholesaleStorer{}
	c := NewWholesaleContractConsumer(cfg, storer, "public.wholesale-contract-event")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Start(ctx) // must not block
}

func TestWholesaleStop_NilClient_NoPanic(t *testing.T) {
	t.Parallel()
	storer := &mockWholesaleStorer{}
	c := newWholesaleConsumer(storer)

	assert.NotPanics(t, func() { c.Stop() })
}
