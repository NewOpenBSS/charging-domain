package consumer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"

	"go-ocs/internal/events"
)

// ---------------------------------------------------------------------------
// Mock SubscriberStorer
// ---------------------------------------------------------------------------

type mockSubscriberStorer struct {
	mock.Mock
}

func (m *mockSubscriberStorer) InsertSubscriber(ctx context.Context, event *events.SubscriberEvent) error {
	return m.Called(ctx, event).Error(0)
}

func (m *mockSubscriberStorer) UpdateSubscriber(ctx context.Context, event *events.SubscriberEvent) error {
	return m.Called(ctx, event).Error(0)
}

func (m *mockSubscriberStorer) DeleteSubscriber(ctx context.Context, subscriberID uuid.UUID, wholesaleID uuid.UUID) error {
	return m.Called(ctx, subscriberID, wholesaleID).Error(0)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var testSubID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

// newRecord encodes v as JSON and wraps it in a *kgo.Record.
func newRecord(t *testing.T, v any) *kgo.Record {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return &kgo.Record{
		Topic:     "public.subscriber-event",
		Partition: 0,
		Offset:    0,
		Value:     data,
	}
}

// newBaseEvent returns a SubscriberEvent with all required fields set.
func newBaseEvent(eventType events.SubscriberEventType) events.SubscriberEvent {
	return events.SubscriberEvent{
		EventType:        eventType,
		SubscriberID:     testSubID,
		RatePlanID:       uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		CustomerID:       uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		WholesaleID:      uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"),
		Msisdn:           "27820001001",
		Iccid:            "8927001234567890",
		ContractID:       uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"),
		Status:           "ACTIVE",
		AllowOobCharging: true,
	}
}

// newConsumer builds a consumer with a nil kgo.Client (disabled-Kafka mode) for tests.
func newConsumer(storer SubscriberStorer) *SubscriberEventConsumer {
	return &SubscriberEventConsumer{
		client: nil,
		storer: storer,
		topic:  "public.subscriber-event",
	}
}

// ---------------------------------------------------------------------------
// handleRecord — dispatch tests
// ---------------------------------------------------------------------------

func TestHandleRecord_Created_CallsInsert(t *testing.T) {
	t.Parallel()
	storer := &mockSubscriberStorer{}
	event := newBaseEvent(events.SubscriberEventCreated)
	storer.On("InsertSubscriber", mock.Anything, &event).Return(nil)

	c := newConsumer(storer)
	err := c.handleRecord(context.Background(), newRecord(t, event))

	require.NoError(t, err)
	storer.AssertExpectations(t)
}

func TestHandleRecord_Updated_CallsUpdate(t *testing.T) {
	t.Parallel()
	storer := &mockSubscriberStorer{}
	event := newBaseEvent(events.SubscriberEventUpdated)
	storer.On("UpdateSubscriber", mock.Anything, &event).Return(nil)

	c := newConsumer(storer)
	err := c.handleRecord(context.Background(), newRecord(t, event))

	require.NoError(t, err)
	storer.AssertExpectations(t)
}

func TestHandleRecord_MsisdnSwap_CallsUpdate(t *testing.T) {
	t.Parallel()
	storer := &mockSubscriberStorer{}
	event := newBaseEvent(events.SubscriberEventMsisdnSwap)
	storer.On("UpdateSubscriber", mock.Anything, &event).Return(nil)

	c := newConsumer(storer)
	err := c.handleRecord(context.Background(), newRecord(t, event))

	require.NoError(t, err)
	storer.AssertExpectations(t)
}

func TestHandleRecord_SimSwap_CallsUpdate(t *testing.T) {
	t.Parallel()
	storer := &mockSubscriberStorer{}
	event := newBaseEvent(events.SubscriberEventSimSwap)
	storer.On("UpdateSubscriber", mock.Anything, &event).Return(nil)

	c := newConsumer(storer)
	err := c.handleRecord(context.Background(), newRecord(t, event))

	require.NoError(t, err)
	storer.AssertExpectations(t)
}

func TestHandleRecord_Deleted_CallsDelete(t *testing.T) {
	t.Parallel()
	storer := &mockSubscriberStorer{}
	event := newBaseEvent(events.SubscriberEventDeleted)
	testWholesaleID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	storer.On("DeleteSubscriber", mock.Anything, testSubID, testWholesaleID).Return(nil)

	c := newConsumer(storer)
	err := c.handleRecord(context.Background(), newRecord(t, event))

	require.NoError(t, err)
	storer.AssertExpectations(t)
}

func TestHandleRecord_Deleted_CascadesWholesalerDelete(t *testing.T) {
	t.Parallel()
	storer := &mockSubscriberStorer{}
	event := newBaseEvent(events.SubscriberEventDeleted)
	// WholesaleID is set on the base event to dddddddd-...
	testWholesaleID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	storer.On("DeleteSubscriber", mock.Anything, testSubID, testWholesaleID).Return(nil)

	c := newConsumer(storer)
	err := c.handleRecord(context.Background(), newRecord(t, event))

	require.NoError(t, err)
	// Verify that DeleteSubscriber was called with the correct wholesaleID so the
	// adapter can cascade to DeleteInactiveWholesalerIfEmpty.
	storer.AssertCalled(t, "DeleteSubscriber", mock.Anything, testSubID, testWholesaleID)
}

// ---------------------------------------------------------------------------
// handleRecord — malformed / unknown event
// ---------------------------------------------------------------------------

func TestHandleRecord_MalformedJSON_SkipsWithoutError(t *testing.T) {
	t.Parallel()
	storer := &mockSubscriberStorer{}
	c := newConsumer(storer)

	r := &kgo.Record{
		Topic: "public.subscriber-event",
		Value: []byte(`{not valid json`),
	}
	err := c.handleRecord(context.Background(), r)

	assert.NoError(t, err, "malformed JSON must be skipped without returning an error")
	storer.AssertNotCalled(t, "InsertSubscriber")
	storer.AssertNotCalled(t, "UpdateSubscriber")
	storer.AssertNotCalled(t, "DeleteSubscriber")
}

func TestHandleRecord_UnknownEventType_SkipsWithoutError(t *testing.T) {
	t.Parallel()
	storer := &mockSubscriberStorer{}
	c := newConsumer(storer)

	type unknownEvent struct {
		EventType string `json:"eventType"`
	}
	r := newRecord(t, unknownEvent{EventType: "UNKNOWN_OP"})
	err := c.handleRecord(context.Background(), r)

	assert.NoError(t, err, "unknown event type must be skipped without returning an error")
	storer.AssertNotCalled(t, "InsertSubscriber")
	storer.AssertNotCalled(t, "UpdateSubscriber")
	storer.AssertNotCalled(t, "DeleteSubscriber")
}

// ---------------------------------------------------------------------------
// Start / Stop with disabled Kafka
// ---------------------------------------------------------------------------

func TestStart_KafkaDisabled_ReturnsImmediately(t *testing.T) {
	t.Parallel()
	cfg := &events.KafkaConfig{Enabled: false}
	storer := &mockSubscriberStorer{}
	c := NewSubscriberEventConsumer(cfg, storer, "public.subscriber-event")

	// Start must return immediately — no panic, no goroutine that blocks the test.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Start(ctx) // must not block
}

func TestStop_NilClient_NoPanic(t *testing.T) {
	t.Parallel()
	storer := &mockSubscriberStorer{}
	c := newConsumer(storer)

	assert.NotPanics(t, func() { c.Stop() })
}

// ---------------------------------------------------------------------------
// Store error propagation
// ---------------------------------------------------------------------------

func TestHandleRecord_InsertError_ReturnsError(t *testing.T) {
	t.Parallel()
	storer := &mockSubscriberStorer{}
	event := newBaseEvent(events.SubscriberEventCreated)

	insertErr := assert.AnError
	storer.On("InsertSubscriber", mock.Anything, &event).Return(insertErr)

	c := newConsumer(storer)
	err := c.handleRecord(context.Background(), newRecord(t, event))

	assert.Equal(t, insertErr, err)
	storer.AssertExpectations(t)
}
