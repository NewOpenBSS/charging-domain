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

<!-- Features being worked on go here -->

---

## Backlog

<!-- Approved Features waiting to be started go here -->

## F-001 — ChargingTraceResource

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
