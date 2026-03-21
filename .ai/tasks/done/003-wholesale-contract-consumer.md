# Task 003 — WholesaleContractConsumer implementation

**Feature:** F-006 — WholesaleContractConsumer
**Sequence:** 3 of 5
**Date:** 2026-03-21
**Status:** Active

---

## Objective

Implement the `WholesaleContractConsumer` in `internal/backend/consumer/`, following the
`SubscriberEventConsumer` pattern exactly. This task creates three files: the consumer struct,
the storer interface with its store adapter, and the unit test suite. No wiring into AppContext
or main.go yet — that is Task 005.

---

## Scope

**In scope:**
- `internal/backend/consumer/wholesale_consumer.go`:
  - `WholesaleContractConsumer` struct with `client *kgo.Client`, `storer WholesaleStorer`, `topic string`
  - `NewWholesaleContractConsumer(cfg *events.KafkaConfig, storer WholesaleStorer, topic string)` constructor
  - `Start(ctx context.Context)` — launches background goroutine; no-op when Kafka disabled
  - `Stop()` — closes Kafka client; safe when disabled
  - `run(ctx)` private consumer loop using `PollFetches`
  - `handleRecord(ctx, r *kgo.Record)` — unmarshal event type first, then dispatch; skip malformed/unknown
  - Consumer group ID: `"charging-backend-wholesale"`
  - Client ID suffix: `"-wholesale-consumer"`
  - Event type dispatch:
    - `WholesaleContractProvisioned` → `storer.UpsertWholesaler(ctx, event)`
    - `WholesaleContractDeregistering` → `storer.DeregisterWholesaler(ctx, wholesalerID)`
    - `WholesaleContractSuspend` → `storer.SuspendWholesaler(ctx, wholesalerID, event.Suspend)`
    - Unknown → log warn and skip

- `internal/backend/consumer/wholesale_storer.go`:
  - `WholesaleStorer` interface (defined at point of consumption):
    - `UpsertWholesaler(ctx context.Context, event *events.WholesaleContractProvisionedEvent) error`
    - `DeregisterWholesaler(ctx context.Context, wholesalerID uuid.UUID) error`
    - `SuspendWholesaler(ctx context.Context, wholesalerID uuid.UUID, suspend bool) error`
  - `StoreWholesaleAdapter` struct wrapping `*store.Store`
  - `NewStoreWholesaleAdapter(s *store.Store) *StoreWholesaleAdapter`
  - `UpsertWholesaler` — delegates to `s.Q.UpsertWholesaler` with sqlc params; map all fields from event
  - `DeregisterWholesaler` — calls `s.Q.CountSubscribersByWholesaler`; if 0 calls `DeleteWholesaler`, else calls `SetWholesalerActive(id, false)`
  - `SuspendWholesaler` — calls `s.Q.SetWholesalerActive(id, !suspend)`; `suspend=true` → `active=false`, `suspend=false` → `active=true`
  - Reuse `uuidToPgtype` helper already defined in `subscriber_storer.go`

- `internal/backend/consumer/wholesale_consumer_test.go`:
  - Mock `WholesaleStorer` using `testify/mock`
  - Tests for: Provisioned dispatches to UpsertWholesaler, Deregistering dispatches to DeregisterWholesaler,
    Suspend dispatches to SuspendWholesaler with correct `suspend` bool, malformed JSON skipped,
    unknown event type skipped, store error propagated, Kafka disabled returns no-op consumer
  - Table-driven tests where applicable; all subtests use `t.Parallel()`

**Out of scope:**
- Wiring into AppContext or main.go — Task 005
- Subscriber cascade delete — Task 004
- Changes to SubscriberEventConsumer

---

## Context

- Follow `internal/backend/consumer/subscriber_consumer.go` pattern **exactly** — struct layout,
  constructor, Start/Stop, run loop, handleRecord error handling, logging calls
- Follow `internal/backend/consumer/subscriber_storer.go` for the adapter pattern
- Follow `internal/backend/consumer/subscriber_consumer_test.go` for the test pattern
- `uuidToPgtype` helper lives in `subscriber_storer.go` — it is unexported but in the same package,
  so the wholesale storer can use it directly
- The `DeregisterWholesaler` adapter method contains sequenced DB calls (count then delete/update) —
  this is acceptable; the count + delete are NOT wrapped in a transaction (same as Java behaviour),
  meaning a race between two deregistering events is tolerated at this stage
- sqlc generated params structs (from Task 001): `UpsertWholesalerParams`, `SetWholesalerActiveParams`
- `RateLimit` in the event is `float64`; the sqlc param for `rateLimit` is `pgtype.Numeric` —
  use `pgtype.Numeric` with `decimal.NewFromFloat` or convert via string; do NOT use float arithmetic
  for the numeric conversion. Simplest safe approach: use `pgtype.Numeric` with `Val` set from
  `github.com/shopspring/decimal` e.g.:
  ```go
  rateLimit := decimal.NewFromFloat(event.RateLimit)
  pgRateLimit := pgtype.Numeric{...} // follow pattern in codebase for decimal → pgtype.Numeric
  ```
  Inspect existing codebase usage for the correct conversion pattern.
- `hosts text[]` maps to `pgtype.Array[string]` or `[]string` depending on sqlc generation — inspect
  generated params struct after Task 001 to use the correct type

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| `DeregisterWholesaler` does count-then-act in the adapter, not in consumer | Keeps business logic out of the consumer; adapter encapsulates DB-level decision |
| No transaction wrapping on deregister's count+delete | Matches Java behaviour; deregistering race is low-probability and tolerated |
| `SuspendWholesaler` maps `suspend bool` to `active = !suspend` | Cleaner inversion than passing active directly; makes the semantics explicit |
| Reuse `uuidToPgtype` from subscriber_storer.go | Same package, avoids duplication |

---

## Acceptance Criteria

- [ ] `wholesale_consumer.go` exists with correct struct, constructor, Start/Stop, run, handleRecord
- [ ] `wholesale_storer.go` exists with WholesaleStorer interface and StoreWholesaleAdapter
- [ ] All three event types dispatch correctly in handleRecord
- [ ] Malformed JSON is logged and skipped without crashing
- [ ] Unknown event type is logged and skipped without crashing
- [ ] Kafka disabled (nil client) results in no-op Start and safe Stop
- [ ] All new functions have Go doc comments
- [ ] `wholesale_consumer_test.go` exists with tests covering all dispatch paths, error cases, and disabled mode
- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes

---

## Risk Assessment

Low-medium. The `DeregisterWholesaler` adapter has a two-step DB operation (count then delete/update)
that is not atomic. A race between two simultaneous deregistering events could result in incorrect
state. This matches the Java reference implementation and is accepted at this stage. No charging,
quota, or rating logic is touched.
