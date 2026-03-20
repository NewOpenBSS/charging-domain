# Task 002 — SubscriberEvent Message Model

## Feature
F-005 — SubscriberEventConsumer

## What to Build
Define the `SubscriberEvent` message struct and event type constants in
`internal/events/`. This is the Go representation of the Kafka message payload
consumed from `public.subscriber-event`.

## Files to Create
- `internal/events/subscriber_event.go` — event type constants + struct
- `internal/events/subscriber_event_test.go` — unit tests

## Event Type Constants
```
SubscriberEventType string
CREATED, UPDATED, MSISDN_SWAP, SIM_SWAP, DELETED
```

## SubscriberEvent Struct
Fields matching the subscriber table (all required):
- EventType SubscriberEventType `json:"eventType"`
- SubscriberID uuid.UUID `json:"subscriberId"`
- RatePlanID uuid.UUID `json:"ratePlanId"`
- CustomerID uuid.UUID `json:"customerId"`
- WholesaleID uuid.UUID `json:"wholesaleId"`
- Msisdn string `json:"msisdn"`
- Iccid string `json:"iccid"`
- ContractID uuid.UUID `json:"contractId"`
- Status string `json:"status"`
- AllowOobCharging bool `json:"allowOobCharging"`

## Acceptance Criteria
- [ ] All five event type constants defined
- [ ] SubscriberEvent struct defined with JSON tags
- [ ] Unit tests cover JSON marshal/unmarshal round-trip for each event type
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
