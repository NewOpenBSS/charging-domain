# Task 004 — Extend subscriber delete with wholesaler cascade

**Feature:** F-006 — WholesaleContractConsumer
**Sequence:** 4 of 5
**Date:** 2026-03-21
**Status:** Active

---

## Objective

Extend the existing `SubscriberEventConsumer` to cascade-delete an inactive wholesaler with zero
remaining subscribers when a subscriber is removed. The `WholesaleContractProvisionedEvent` deregistering
path handles wholesaler deactivation, but the final cleanup — deleting the wholesaler row when the
last subscriber on an inactive wholesaler is removed — must be triggered from the subscriber delete path.

This task modifies `SubscriberStorer.DeleteSubscriber` to also accept the `wholesaleID`, updates the
`StoreSubscriberAdapter` to call `DeleteInactiveWholesalerIfEmpty` after the subscriber row is deleted,
and updates the subscriber consumer's `handleRecord` to pass `event.WholesaleID`. The interface change
is the only modification to a public-facing type; all other changes are in the same package.

---

## Scope

**In scope:**
- `internal/backend/consumer/subscriber_consumer.go`:
  - Update `SubscriberStorer.DeleteSubscriber` interface method signature:
    `DeleteSubscriber(ctx context.Context, subscriberID uuid.UUID, wholesaleID uuid.UUID) error`
  - Update `handleRecord` case for `events.SubscriberEventDeleted`:
    `c.storer.DeleteSubscriber(ctx, event.SubscriberID, event.WholesaleID)`
- `internal/backend/consumer/subscriber_storer.go`:
  - Update `StoreSubscriberAdapter.DeleteSubscriber` to:
    1. Call `a.s.Q.DeleteSubscriber(ctx, uuidToPgtype(subscriberID))` — delete the subscriber row
    2. Call `a.s.Q.DeleteInactiveWholesalerIfEmpty(ctx, uuidToPgtype(wholesaleID))` — atomic no-op if wholesaler still active or has subscribers
    3. Return the first non-nil error
- `internal/backend/consumer/subscriber_consumer_test.go`:
  - Update mock `DeleteSubscriber` expectations to include `wholesaleID` argument
  - Add test case: subscriber deleted → wholesaler cascade called with correct wholesaleID
  - Ensure all existing tests remain passing with the updated signature

**Out of scope:**
- Any changes to the WholesaleContractConsumer or WholesaleStorer
- Changes to the subscriber SQL queries (the `DeleteInactiveWholesalerIfEmpty` query was added in Task 001)
- Transaction wrapping — the two DB calls are intentionally sequential, not atomic (same pattern as deregister)

---

## Context

- `SubscriberEvent.WholesaleID uuid.UUID` is already present on the event struct — it just isn't
  currently passed to DeleteSubscriber
- `DeleteInactiveWholesalerIfEmpty` SQL (from Task 001): atomically deletes wholesaler only if
  `active = false` AND `(SELECT COUNT(*) FROM subscriber WHERE wholesale_id = $1) = 0`; if the
  wholesaler is still active or still has subscribers, it is a no-op
- The non-atomicity between the subscriber delete and the wholesaler cascade is acceptable — if the
  wholesaler is still active, the conditional delete is a guaranteed no-op; the only edge case is
  concurrent subscriber deletes, which is extremely low probability in the wholesale domain
- `subscriber_consumer_test.go` uses `testify/mock` — update mock interface and test expectations
  in the same file; do not create a new test file

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| Pass wholesaleID through DeleteSubscriber rather than a separate method | Keeps the cascade co-located with the delete; avoids two consumer-level calls |
| `DeleteInactiveWholesalerIfEmpty` runs unconditionally after subscriber delete | The SQL is a no-op when the condition isn't met; simplest correct implementation |
| No transaction wrapping for the two-step delete | Matches the Java reference; race condition is extremely low probability and tolerated |

---

## Acceptance Criteria

- [ ] `SubscriberStorer.DeleteSubscriber` has updated signature with `wholesaleID uuid.UUID`
- [ ] `handleRecord` passes `event.WholesaleID` to `DeleteSubscriber`
- [ ] `StoreSubscriberAdapter.DeleteSubscriber` calls `DeleteInactiveWholesalerIfEmpty` after subscriber delete
- [ ] Existing subscriber consumer tests updated and passing
- [ ] New test: DELETE event with an inactive wholesaler having no remaining subscribers triggers cascade
- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes

---

## Risk Assessment

Medium. This modifies the existing `SubscriberEventConsumer` and its storer interface, which was
implemented and tested in F-005. The change is minimal and additive (new argument, one extra DB call
after delete), but incorrect implementation could lead to premature wholesaler row deletion (if the
cascade condition check is wrong) or lost wholesale data. The atomic SQL in
`DeleteInactiveWholesalerIfEmpty` mitigates this — verify the SQL condition before committing.
