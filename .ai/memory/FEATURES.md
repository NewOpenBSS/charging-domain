# Features

This is the single source of truth for all feature work.
Updated by humans after each PR is merged.
Read by AI agents at the start of every design and development session.

## Status Values
- **Backlog** — defined, not yet started
- **Ready for AI Design** — Feature approved, waiting for technical decomposition
- **In Design** — AI decomposing into features and branches
- **In Development** — one or more branches being implemented
- **In Review** — PR(s) open, waiting for human review
- **Done** — all PRs merged

---

## Active Features

## F-001 — ChargingTraceResource

**Status:** Done
**Priority:** High
**Created:** 2026-03-20
**Branch:** feature/F-001-charging-trace-resource

### Implementation Approval Required
- [ ] Yes — pause after AI Design for human review before implementation begins
- [x] No — proceed to implementation automatically after AI Design

### Feature Switch
None

### Goal
Expose charging trace records via three read-only GraphQL queries in the Go charging-backend, matching the Java API surface exactly.

### Problem Statement
Operators need to query the charging audit trail to investigate billing disputes and debug charging sessions by MSISDN or charging ID. The Go charging-engine already writes traces to the DB; the Go charging-backend currently has no way to expose them.

### MVP
An admin can query charging traces using three read-only GraphQL operations — list (paginated + filtered), count, and fetch by ID — with no mutations exposed.

### Acceptance Criteria
- [ ] An admin can retrieve a paginated list of charging traces, filtered by `chargingId` or `msisdn` (wildcard match)
- [ ] An admin can count charging traces matching a given filter
- [ ] An admin can retrieve a single charging trace by `traceId`, with `request` and `response` returned as JSON strings
- [ ] No mutations are exposed — the resource is strictly read-only
- [ ] The GraphQL query names (`chargingTraceList`, `countChargingTrace`, `chargingTraceById`) match the Java service exactly

### Constraints
- GraphQL query names, field names, and behaviour must be identical to the Java service — external clients must work without modification
- Read-only — no mutations under any circumstances

### Out of Scope
- Write operations on charging traces
- Subscriptions or real-time streaming of trace events

### Parking Lot
None

### Future Considerations
- Cursor-based pagination (current OFFSET-based approach degrades on large trace tables — acceptable for now)

---

## Backlog

<!-- Approved Features waiting to be started go here -->

## F-002 — DestinationGroupResource

**Status:** Backlog
**Priority:** High
**Created:** 2026-03-20
**Branches:** (filled in by AI during Stage 3)

### Implementation Approval Required
- [ ] Yes — pause after AI Design for human review before implementation begins
- [x] No — proceed to implementation automatically after AI Design

### Feature Switch
None

### Goal
Expose full CRUD for carrier destination groups via GraphQL in the Go charging-backend, matching the Java API surface exactly.

### Problem Statement
Admins need to manage destination groups — named groupings of destinations by region used in the charging domain. The Go charging-backend currently has no way to create, read, update, or delete these records.

### MVP
An admin can perform full CRUD on destination groups via six GraphQL operations — list (paginated + filtered), count, fetch by name, create, update, and delete.

### Acceptance Criteria
- [ ] An admin can retrieve a paginated list of destination groups, filtered by `groupName` or `region` (wildcard match)
- [ ] An admin can count destination groups matching a given filter
- [ ] An admin can retrieve a single destination group by `groupName`
- [ ] An admin can create a new destination group
- [ ] An admin can update an existing destination group
- [ ] An admin can delete a destination group by `groupName`
- [ ] GraphQL operation names (`destinationGroupList`, `countDestinationGroup`, `destinationGroupByGroupName`, `createDestinationGroup`, `updateDestinationGroup`, `deleteDestinationGroup`) match the Java service exactly

### Constraints
- GraphQL operation names, field names, and behaviour must be identical to the Java service — external clients must work without modification
- No state machine or approval workflow — plain CRUD only

### Out of Scope
- Approval workflows or state machines for destination groups
- Bulk import or export of destination groups

### Parking Lot
None

### Future Considerations
- Cursor-based pagination (current OFFSET-based approach degrades on large datasets — acceptable for now)

## F-003 — SourceGroupResource

**Status:** Backlog
**Priority:** High
**Created:** 2026-03-20
**Branches:** (filled in by AI during Stage 3)

### Implementation Approval Required
- [ ] Yes — pause after AI Design for human review before implementation begins
- [x] No — proceed to implementation automatically after AI Design

### Feature Switch
None

### Goal
Expose full CRUD for carrier source groups via GraphQL in the Go charging-backend, matching the Java API surface exactly.

### Problem Statement
Admins need to manage source groups — named groupings of originating sources by region used in the charging domain. The Go charging-backend currently has no way to create, read, update, or delete these records.

### MVP
An admin can perform full CRUD on source groups via six GraphQL operations — list (paginated + filtered), count, fetch by name, create, update, and delete.

### Acceptance Criteria
- [ ] An admin can retrieve a paginated list of source groups, filtered by `groupName` or `region` (wildcard match)
- [ ] An admin can count source groups matching a given filter
- [ ] An admin can retrieve a single source group by `groupName`
- [ ] An admin can create a new source group
- [ ] An admin can update an existing source group
- [ ] An admin can delete a source group by `groupName`
- [ ] GraphQL operation names (`sourceGroupList`, `countSourceGroup`, `sourceGroupByGroupName`, `createSourceGroup`, `updateSourceGroup`, `deleteSourceGroup`) match the Java service exactly

### Constraints
- GraphQL operation names, field names, and behaviour must be identical to the Java service — external clients must work without modification
- No state machine or approval workflow — plain CRUD only

### Out of Scope
- Approval workflows or state machines for source groups
- Bulk import or export of source groups

### Parking Lot
None

### Future Considerations
- Cursor-based pagination (current OFFSET-based approach degrades on large datasets — acceptable for now)

## F-005 — SubscriberEventConsumer

**Status:** In Review
**Priority:** High
**Created:** 2026-03-20
**Branch:** feature/F-005-subscriber-event-consumer

### Implementation Approval Required
- [ ] Yes — pause after AI Design for human review before implementation begins
- [x] No — proceed to implementation automatically after AI Design

### Feature Switch
None — port of existing Java functionality

### Goal
A Kafka consumer in `charging-backend` that processes `SubscriberEvent` messages from the Retail CRM domain and keeps the shadow subscriber table in sync.

### Problem Statement
The shadow `subscriber` table in the charging domain has no automated population mechanism. Without this consumer, subscriber records are never created, updated, or removed when the Retail CRM makes changes — leaving the charging engine with stale or missing subscriber data.

### MVP
`charging-backend` consumes `SubscriberEvent` messages from `public.subscriber-event` and applies one of three DB operations based on event type:
- `CREATED` → INSERT subscriber row
- `UPDATED`, `MSISDN_SWAP`, `SIM_SWAP` → UPDATE all fields by `subscriber_id`
- `DELETED` → hard DELETE by `subscriber_id`

### Acceptance Criteria
- [ ] A `CREATED` event results in a new row inserted into `subscriber` with all fields populated from the event payload
- [ ] A `UPDATED`, `MSISDN_SWAP`, or `SIM_SWAP` event results in the existing subscriber row being updated with all current field values from the payload
- [ ] A `DELETED` event results in the subscriber row being hard-deleted from the `subscriber` table
- [ ] A malformed or unrecognisable event is logged and skipped — the consumer continues processing without crashing
- [ ] The consumer starts automatically with `charging-backend` and reconnects if the Kafka broker is unavailable

### Constraints
- Event schema (`SubscriberEvent`) is fixed — cannot be modified
- Topic name: `public.subscriber-event`
- Implemented in `charging-backend` only

### Out of Scope
- Cache invalidation in DRA or Engine on event receipt
- Treating `MSISDN_SWAP` and `SIM_SWAP` as distinct partial-update operations

### Parking Lot
- **Cache invalidation on event receipt**: DRA/Engine could listen to `SubscriberEvent` and force an immediate cache refresh rather than waiting for TTL — deferred, not worth the effort at this stage

### Future Considerations
- If subscriber deletes become reversible, the hard-delete strategy would need revisiting in favour of soft-delete

---

## F-006 — WholesaleContractConsumer

**Status:** Backlog
**Priority:** High
**Created:** 2026-03-21
**Branch:** feature/F-006-wholesale-contract-consumer

### Implementation Approval Required
- [ ] Yes — pause after AI Design for human review before implementation begins
- [x] No — proceed to implementation automatically after AI Design

### Feature Switch
None — background Kafka consumer, no user-visible behaviour change

### Goal
A Kafka consumer in `charging-backend` that keeps the wholesaler shadow table in sync with the Wholesale CRM domain by processing three contract lifecycle events, including cascaded wholesaler cleanup when the last subscriber of an inactive wholesaler is removed.

### Problem Statement
The wholesaler shadow table in the charging domain has no automated population mechanism. The charging engine depends on wholesaler data — active status, hosts, NCHF URL, rate plan — for tenant resolution and rate lookups. Without this consumer, wholesaler records can never be created, updated, or removed automatically when the Wholesale CRM domain makes changes, leaving the charging engine with stale, missing, or incorrectly active wholesaler entries.

### MVP
`charging-backend` consumes three event types from the Wholesale CRM domain and applies the appropriate DB operation:
- `WholesaleContractProvisionedEvent` → UPSERT wholesaler row
- `WholesaleContractDeregisteringEvent` → DELETE if no subscribers; else mark `active = false`
- `WholesaleContractSuspendEvent` → set `active = !suspend`

Additionally, when a subscriber is deleted and their associated wholesaler is `active = false` with zero remaining subscribers, the wholesaler row is also deleted.

### Acceptance Criteria
- [ ] A `WholesaleContractProvisionedEvent` results in a wholesaler row being inserted (if new) or updated (if existing) with all DB-mapped fields from the event payload
- [ ] A `WholesaleContractDeregisteringEvent` when subscriber count = 0 results in the wholesaler row being deleted
- [ ] A `WholesaleContractDeregisteringEvent` when subscriber count > 0 results in the wholesaler being marked `active = false`
- [ ] A `WholesaleContractSuspendEvent` with `suspend = true` sets `active = false`; with `suspend = false` sets `active = true`
- [ ] When a subscriber is deleted, if the wholesaler is `active = false` and has zero remaining subscribers, the wholesaler row is also deleted
- [ ] A malformed or unrecognisable event is logged and skipped — the consumer continues without crashing
- [ ] When Kafka is disabled (`cfg.Enabled = false`), the consumer starts as a no-op and `Stop()` is safe to call

### Constraints
- Event schemas are fixed — `WholesaleContractProvisionedEvent`, `WholesaleContractDeregisteringEvent`, `WholesaleContractSuspendEvent` defined in the Wholesale CRM API
- Only fields present in the `wholesaler` DB schema are persisted — extra fields in the provisioned event (registrationNumber, taxNumber, addressInfo, etc.) are ignored
- Topic names are supplied via application configuration — same pattern as `SubscriberEventConsumer`
- Implemented in `charging-backend` only; no changes to `charging-engine`
- Follow the `SubscriberEventConsumer` pattern exactly: consumer struct, storer interface, store adapter, separate event file in `internal/events/`

### Out of Scope
- A GraphQL or REST resource for wholesaler management
- Persisting `registrationNumber`, `taxNumber`, `addressInfo`, `contactInfo`, `invoiceMessage` (DB schema change)
- Dead-letter queue or retry on consumer errors
- Replay or backfill of historical wholesale events

### Parking Lot
- **Wholesaler GraphQL resource**: Admin UI needs — separate feature
- **Additional wholesaler fields** (`registrationNumber`, `taxNumber`, `addressInfo`): Requires DB schema change — deferred
- **Dead-letter queue**: Good practice for production hardening — deferred

### Future Considerations
- If wholesaler deregistration becomes reversible, the delete strategy may need revisiting in favour of soft-delete

---

## F-004 — GraphQL API Test Files

**Status:** Backlog
**Priority:** High
**Created:** 2026-03-20
**Branches:** (filled in by AI during Stage 3)

### Implementation Approval Required
- [ ] Yes — pause after AI Design for human review before implementation begins
- [x] No — proceed to implementation automatically after AI Design

### Feature Switch
None

### Goal
Add `.http` API test files for QuotaResource, ChargingTraceResource, DestinationGroupResource, and SourceGroupResource, following the established pattern in `api-tests/`.

### Problem Statement
Developers have no way to manually exercise or quickly verify the new GraphQL endpoints from their IDE. QuotaResource is also missing a test file despite being complete. All existing resources have `.http` files — the new ones should too.

### MVP
Four new `.http` files in `api-tests/`, one per resource, covering every GraphQL operation with realistic sample data.

### Acceptance Criteria
- [ ] A developer can execute every GraphQL operation for QuotaResource, ChargingTraceResource, DestinationGroupResource, and SourceGroupResource directly from the `.http` files
- [ ] Each file covers all operations for that resource (list with default page, list with wildcard, list with filter, get-by-key, count, and create/update/delete where applicable)
- [ ] Sample data in each file is realistic and consistent with seed data in `db/seeds/`
- [ ] Files follow the naming convention `[ResourceName]GraphQL.http`

### Constraints
- Must follow the exact structure and style of existing files in `api-tests/`

### Out of Scope
- Automated test execution or CI integration of `.http` files
- Test files for resources already covered (`CarrierGraphQL.http`, `ClassficationGraphQL.http`, `RatePlanGraphQL.http`, `NumberPlanGraphQL.http`)

### Parking Lot
None

### Future Considerations
- Automated API test execution in CI pipeline

---

## F-005 — SubscriberEventConsumer

**Status:** Backlog
**Priority:** High
**Created:** 2026-03-20
**Branch:** (filled in by scoping)

### Implementation Approval Required
- [ ] Yes — pause after AI Design for human review before implementation begins
- [x] No — proceed to implementation automatically after AI Design

### Feature Switch
None — port of existing Java functionality

### Goal
Implement a Kafka consumer in charging-backend that processes SubscriberEvent messages from the `public.subscriber-event` topic, keeping the shadow subscriber table in sync.

### Problem Statement
The charging-backend has a shadow subscriber table that must stay in sync with the Retail CRM domain. The Java service has a Kafka consumer that handles this. The Go service currently has no equivalent — subscriber data is stale or missing.

### MVP
A Kafka consumer that processes SubscriberEvent messages and applies CREATE, UPDATE, and hard DELETE operations to the shadow subscriber table.

### Acceptance Criteria
- [ ] Consumer subscribes to `public.subscriber-event` topic on startup
- [ ] CREATE events result in a new subscriber record being inserted
- [ ] UPDATE events result in the existing subscriber record being updated
- [ ] DELETE events result in the subscriber record being hard-deleted
- [ ] Consumer handles unknown event types gracefully without crashing
- [ ] Consumer resumes from last committed offset on restart

### Constraints
- Must match the Java service behaviour exactly — same topic, same event schema, same table operations
- Hard delete only — no soft delete or tombstone records

### Out of Scope
- Subscriber query or admin API (separate feature)
- Replay or backfill of historical events

### Parking Lot
None

### Future Considerations
- Dead letter queue for malformed messages

---

## Done

<!-- Completed Features go here — kept for reference -->

---

## Feature Template

Copy this template when adding a new Feature:

```markdown
## F-NNN — [Title]

**Status:** Backlog
**Priority:** High / Medium / Low
**Created:** YYYY-MM-DD
**Branches:** (filled in by AI during Stage 3)

### Implementation Approval Required
- [ ] Yes — pause after AI Design for human review before implementation begins
- [ ] No — proceed to implementation automatically after AI Design

### Delivery Deadline
<!-- Optional. Date by which this Feature must be merged to support release planning. -->
<!-- Format: YYYY-MM-DD -->
<!-- Set by the planning/roadmap process, not the delivery process. -->

### Feature Switch
<!-- Name of the feature switch if required, or "None" if not user-visible -->
<!-- Convention: lowercase_underscore e.g. quota_self_service -->
<!-- If switch infrastructure does not yet exist, note it as a prerequisite -->

### Goal
One sentence. What this builds and why.

### Problem Statement
What is broken or missing. Who is affected. Current state vs desired state.

### MVP
The smallest version that delivers real value.
What a user can do when this is complete.

### Acceptance Criteria
- [ ] [user/role] can [achieve something observable]
- [ ] [condition] results in [observable outcome]
- [ ] [thing] must always/never be [measurable state]

### Constraints
Technical, regulatory, business, or performance constraints.

### Out of Scope
What is explicitly not included in this Feature.

### Parking Lot
Ideas that emerged during design but are deferred:
- [Idea]: [why deferred]

### Future Considerations
Architectural decisions this Feature must not foreclose.
```
