# Task 005 — Wire WholesaleContractConsumer into AppContext and main.go

**Feature:** F-006 — WholesaleContractConsumer
**Sequence:** 5 of 5
**Date:** 2026-03-21
**Status:** Active

---

## Objective

Wire the `WholesaleContractConsumer` (built in Task 003) into the charging-backend application:
add it to `AppContext`, register the topic name in configuration, start and stop it in `main.go`.
This is the final integration task that makes the consumer live in the running application. It
follows the `SubscriberEventConsumer` wiring pattern in `appcontext/context.go` and `main.go` exactly.

---

## Scope

**In scope:**
- `internal/backend/appcontext/context.go`:
  - Add `WholesaleConsumer *consumer.WholesaleContractConsumer` field to `AppContext` struct
  - In `NewAppContext`: construct `StoreWholesaleAdapter` and `WholesaleContractConsumer`:
    ```go
    wholesaleStorer := consumer.NewStoreWholesaleAdapter(s)
    WholesaleConsumer: consumer.NewWholesaleContractConsumer(cfg.Kafkaconfig, wholesaleStorer, wholesaleContractEventTopic(cfg.Kafkaconfig)),
    ```
  - Add `wholesaleContractEventTopic(cfg *events.KafkaConfig) string` helper:
    - Key: `"wholesale-contract-event"`
    - Default fallback: `"public.wholesale-contract-event"`
    - Pattern: identical to existing `subscriberEventTopic`

- `cmd/charging-backend/backend-config.yaml`:
  - Add `wholesale-contract-event: "public.wholesale-contract-event"` to the `kafka.topics` map

- `cmd/charging-backend/main.go`:
  - Add a `wholesaleCtx, wholesaleCancel := context.WithCancel(context.Background())` pair
  - Add `defer wholesaleCancel()`
  - Add `defer appCtx.WholesaleConsumer.Stop()`
  - Add `appCtx.WholesaleConsumer.Start(wholesaleCtx)`
  - Follow the same placement and naming pattern as the subscriber consumer block immediately above

**Out of scope:**
- Any changes to consumer logic, storer, or event types — all in earlier tasks
- GraphQL or REST endpoints for wholesalers — out of scope for F-006

---

## Context

- `internal/backend/appcontext/context.go` is the single wiring point — read the full file before editing
- `cmd/charging-backend/main.go` — follow the subscriber consumer block as the direct template
- `subscriberEventTopic` helper in `context.go` is the exact pattern to replicate for `wholesaleContractEventTopic`
- `cmd/charging-backend/backend-config.yaml` already has a `topics:` map under `kafka:` — add one entry
- After wiring, run the full build and test suite; the consumer itself won't connect to Kafka in unit
  tests because Kafka is disabled in the test environment

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| Separate context for WholesaleConsumer in main.go | Matches subscriber consumer pattern; independent cancellation if needed |
| Topic key `"wholesale-contract-event"` | Consistent with `"subscriber-event"` naming convention in the topics map |
| Default topic `"public.wholesale-contract-event"` | Follows Debezium-style topic naming used throughout the project |

---

## Acceptance Criteria

- [ ] `AppContext.WholesaleConsumer` field exists and is populated by `NewAppContext`
- [ ] `wholesaleContractEventTopic` helper function exists in `context.go`
- [ ] `backend-config.yaml` contains `wholesale-contract-event: "public.wholesale-contract-event"` in `kafka.topics`
- [ ] `main.go` starts and stops `WholesaleConsumer` with its own context, following the subscriber consumer pattern
- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes

---

## Risk Assessment

Low. Pure wiring — no new logic introduced. The consumer itself is a no-op when `kafka.enabled = false`
(which is the default in the config file), so starting it unconditionally is safe for all environments
including local development and CI. No charging, quota, or rating logic is affected.

---

## Notes

After this task completes the full F-006 feature is implemented. The dev session lifecycle steps apply:
- Update `.ai/memory/STATUS.md`
- Update `.ai/memory/FEATURES.md` — set F-006 status to "In Review"
- Remove `.ai/tasks/READY`
- Final commit and clean exit
