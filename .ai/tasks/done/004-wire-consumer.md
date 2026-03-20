# Task 004 — Wire SubscriberEventConsumer into charging-backend

## Feature
F-005 — SubscriberEventConsumer

## What to Build
Wire the SubscriberEventConsumer into AppContext and start/stop it in main.go.
Add the subscriber-event topic to the YAML config.

## Files to Modify
- `cmd/charging-backend/backend-config.yaml` — add subscriber-event topic
- `internal/backend/appcontext/context.go` — add SubscriberConsumer field,
  implement SubscriberStorer adapter on *store.Store, wire in NewAppContext
- `cmd/charging-backend/main.go` — start consumer with context, defer stop

## Implementation Notes

### StoreSubscriberAdapter
Implement `consumer.SubscriberStorer` on `*store.Store` via a thin adapter
in `internal/backend/consumer/` (or inline in the consumer package) that calls
`s.Q.InsertSubscriber`, `s.Q.UpdateSubscriber`, `s.Q.DeleteSubscriber`.
Convert uuid.UUID → pgtype.UUID using `pgtype.UUID{Bytes: [16]byte(id), Valid: true}`.

### Config YAML
Add under kafka.topics:
  subscriber-event: "public.subscriber-event"

### AppContext
Add field: `SubscriberConsumer *consumer.SubscriberEventConsumer`
Wire in NewAppContext using cfg.Kafkaconfig, store, and topic alias.

### main.go
After appCtx is created:
```go
consumerCtx, consumerCancel := context.WithCancel(context.Background())
defer consumerCancel()
defer appCtx.SubscriberConsumer.Stop()
appCtx.SubscriberConsumer.Start(consumerCtx)
```

## Acceptance Criteria
- [ ] `public.subscriber-event` topic configured in backend-config.yaml
- [ ] SubscriberEventConsumer started in main.go with a cancellable context
- [ ] Consumer stopped gracefully via deferred Stop()
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
