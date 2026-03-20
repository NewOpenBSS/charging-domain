# Task 003 — SubscriberEventConsumer

## Feature
F-005 — SubscriberEventConsumer

## What to Build
A Kafka consumer that reads `SubscriberEvent` messages and applies the
appropriate DB operation. Lives in `internal/backend/consumer/`.

## Files to Create
- `internal/backend/consumer/subscriber_consumer.go`
- `internal/backend/consumer/subscriber_consumer_test.go`

## Design

### SubscriberStorer interface (defined in consumer package)
```go
type SubscriberStorer interface {
    InsertSubscriber(ctx, *events.SubscriberEvent) error
    UpdateSubscriber(ctx, *events.SubscriberEvent) error
    DeleteSubscriber(ctx, subscriberID uuid.UUID) error
}
```

### SubscriberEventConsumer struct
- Takes `*events.KafkaConfig`, `SubscriberStorer`, and topic string
- When Kafka disabled: Start() logs and returns immediately; Stop() is a no-op
- When Kafka enabled: creates a dedicated kgo.Client with consumer group
  `charging-backend-subscriber`, consuming the given topic

### Start(ctx context.Context)
Launches background goroutine that:
- Calls PollFetches(ctx) in a loop
- Returns when client is closed or ctx is cancelled
- Logs fetch errors but continues
- Calls handleRecord for each fetched record

### handleRecord
- Unmarshal record Value into SubscriberEvent
- Malformed JSON → log Warn + return nil (skip, do not crash)
- CREATED → InsertSubscriber
- UPDATED / MSISDN_SWAP / SIM_SWAP → UpdateSubscriber
- DELETED → DeleteSubscriber
- Unknown type → log Warn + return nil

### Stop()
Closes the kgo.Client if non-nil.

## Tests
- handleRecord with CREATED event → calls InsertSubscriber
- handleRecord with UPDATED event → calls UpdateSubscriber
- handleRecord with MSISDN_SWAP event → calls UpdateSubscriber
- handleRecord with SIM_SWAP event → calls UpdateSubscriber
- handleRecord with DELETED event → calls DeleteSubscriber
- handleRecord with malformed JSON → returns nil, no store call
- handleRecord with unknown event type → returns nil, no store call
- Start with disabled Kafka → returns immediately, no goroutine panic
- Stop with nil client → no panic

## Acceptance Criteria
- [ ] Consumer dispatches correctly for all five event types
- [ ] Malformed events are skipped without crashing
- [ ] Unknown event types are skipped without crashing
- [ ] Disabled-Kafka path works without panicking
- [ ] All unit tests pass with race detector
- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes
