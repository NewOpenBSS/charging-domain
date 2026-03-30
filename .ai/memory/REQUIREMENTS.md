# Requirements

This is the process-controlled register of all requirements.
Managed by the requirements-session recipe.
Do not edit directly — use the requirements-session recipe.

## Status Values

- **Draft** — captured, needs refinement before scoping
- **Ready for Scoping** — sufficiently detailed, next in line
- **In Scoping** — scoping session currently running
- **Scoped** — one or more Features exist in FEATURES.md
- **In Progress** — Features actively being developed
- **Done** — all Features merged and delivered
- **Deferred** — parked, not on current roadmap

---

## Active Requirements

<!-- Requirements currently being worked on -->

## R-004 — SourceGroupResource GraphQL Endpoint

**Status:** In Progress
**Priority:** High
**Created:** 2026-03-20
**Features:** F-003

### The Idea
Add a SourceGroupResource to the charging-backend GraphQL API, following the same pattern as the existing resources — specifically mirroring DestinationGroupResource exactly.

### The Problem
Source groups classify the roaming origin of a carrier and drive the `sourceType` dimension of a `RateKey` in the charging engine. The `carrier_source_group` reference table exists and is seeded at DB init time, but there is no admin API to manage its entries. Admins cannot add new regions, rename existing ones, or remove obsolete entries without a database migration. The Java charging-backend exposed a full CRUD GraphQL resource for this; the Go port is missing it.

### Wishlist
An admin can create, read, update, and delete source group entries via the GraphQL API — exactly as they manage destination groups today. The API surface is indistinguishable from the Java original so that existing admin console clients require no changes.

### Notes
- The `carrier_source_group` table has two columns: `group_name` (PK) and `region`.
- Structure and operations are identical to DestinationGroup — plain CRUD, no state machine.
- Operation naming follows the same convention: `sourceGroupList`, `countSourceGroup`, `sourceGroupByGroupName`, `createSourceGroup`, `updateSourceGroup`, `deleteSourceGroup`.
- F-003 also delivers `SourceGroupGraphQL.http` in `api-tests/` (co-located with implementation).
- Completes the set of 8 resources required for the charging-backend port (R-001).

---

## Backlog

<!-- Requirements ready or nearly ready for scoping -->

## R-001 — Port the Charging Backend Service to Go

**Status:** Scoped
**Priority:** High
**Created:** 2026-03-20
**Features:** F-001, F-002, F-003, F-004

### The Idea
Port the charging backend service from Java to Go. Not a line-for-line translation — the goal is to do it the Go way, taking advantage of what Go is good at rather than reproducing Java patterns in a different language.

### The Problem
The Java implementation exists but Go is the target platform for the OCS. The existing admin console UI relies on a backend service to administer the charging domain — carriers, classifications, number plans, rate plans, quotas, and related entities. Without this service, the admin console has nothing to talk to.

Hard constraints: the database schema, data model, REST endpoints, and GraphQL API cannot change. External systems depend on them.

### Wishlist
The REST and GraphQL endpoints implemented in Go are indistinguishable from the Java originals — same API surface, same behaviour, same contracts. Any client or external system that worked against the Java service works against the Go service without modification.

Beyond API compatibility, the implementation takes advantage of Go's strengths: simplicity, explicit error handling, and lightweight concurrency — rather than carrying over Java patterns that don't belong in Go.

### Notes
- The port is already underway. Four of five core resources are complete (Carrier, Classification, NumberPlan, RatePlan). QuotaResource is in progress.
- Remaining work tracked in `.ai/memory/STATUS.md` under Known Deferred Items.
- The scoping session will identify the specific outstanding resources and capabilities needed to call this requirement fully delivered.

---

## Draft

<!-- Requirements captured but not yet refined -->

---

## R-009 — Charging Domain Housekeeping

**Status:** Scoped
**Priority:** High
**Created:** 2026-03-30
**Features:** F-009

### The Idea
A housekeeping job that cleans up stale data across the charging domain — expired quota counters, orphaned sessions, old trace records, and superseded rate plan versions.

### The Problem
Several tables accumulate stale data that is never cleaned up without active user traffic. Dormant subscribers cause quota counters to silently pass their expiry without being processed. The `charging_data` table can hold orphaned sessions that were never terminated cleanly. The `charging_trace` table grows unboundedly — every request and response is recorded with no TTL. Superseded rate plan versions linger indefinitely. Left unaddressed, this affects data correctness (quota state for dormant subscribers) and storage and performance over time.

### Wishlist
A standalone application runs as a Kubernetes CronJob. Kubernetes owns the schedule; the application runs all four checks top to bottom each time it is invoked. No scheduler embedded in the application. Each threshold is configurable, with sensible defaults so it works out of the box.

### Notes
Four housekeeping tasks:
1. **Quota expiry** — Find quotas where `next_action_time` is in the past and open each one. The existing quota logic handles the expiry work; no new logic needed.
2. **Stale sessions** — Remove rows from `charging_data` where `modified_on` is older than a configurable period (default: 24 hours). The table should naturally clean itself as sessions terminate; this catches anything that didn't.
3. **Trace purge** — Remove rows from `charging_trace` older than a configurable period (default: 36 hours). CDR events already preserve the full detail, so nothing is lost.
4. **Rate plan cleanup** — Delete superseded ACTIVE rate versions that have been replaced by a newer version for longer than a configurable period (default: 30 days). DRAFT and PENDING versions are never touched.

---

## Deferred

<!-- Requirements parked for future consideration -->

---

## Done

<!-- Completed requirements — kept for reference -->

## R-002 — ChargingTraceResource GraphQL Endpoint

**Status:** Done
**Priority:** High
**Created:** 2026-03-20
**Features:** F-001

### The Idea
Add a ChargingTraceResource to the charging-backend GraphQL API, following the same pattern as the existing resources (Carrier, Classification, NumberPlan, RatePlan, Quota).

### The Problem
— (needs clarification: what does an admin need to do with charging traces? Read-only audit queries, or more?)

### Wishlist
—

### Notes
- Charging trace records are already written to the DB by the charging-engine pipeline.
- Needs clarification on: what queries/mutations are required, who uses it and for what purpose.

---

## R-003 — DestinationGroupResource GraphQL Endpoint

**Status:** Done
**Delivered:** 2026-03-29
**Priority:** High
**Created:** 2026-03-20
**Features:** F-002

### The Idea
Add a DestinationGroupResource to the charging-backend GraphQL API, following the same pattern as the existing resources.

### The Problem
— (needs clarification: what is a destination group, what does an admin need to do with it?)

### Wishlist
—

### Notes
- Needs clarification on: DB table/schema, what operations are required (CRUD? state machine?), relationship to other resources (NumberPlan? RatePlan?).

---

## R-005 — QuotaEventConsumer

**Status:** Done
**Priority:** High
**Created:** 2026-03-20
**Features:** F-007

### The Idea
Add a Kafka consumer for quota events.

### The Problem
— (needs clarification: what events are consumed, what does the consumer do with them, which service hosts it?)

### Wishlist
—

### Notes
- Kafka producer for quota journal events already exists in charging-engine.
- Needs clarification on: topic name, event schema, consumer action (update DB? trigger downstream?), which application hosts the consumer.

---

## R-006 — SubscriberEventConsumer

**Status:** Done
**Priority:** High
**Created:** 2026-03-20
**Features:** F-005

### The Idea
Add a Kafka consumer for subscriber events.

### The Problem
— (needs clarification: what subscriber events are consumed, what triggers them, what the consumer does with them?)

### Wishlist
—

### Notes
- Subscriber table exists in DB but no GraphQL resource or consumer yet.
- Needs clarification on: event source, topic name, event schema, consumer action, which application hosts the consumer.

---

## R-007 — WholesaleContractConsumer

**Status:** Done
**Priority:** High
**Created:** 2026-03-20
**Features:** F-006

### The Idea
Add a Kafka consumer for wholesale contract events.

### The Problem
— (needs clarification: what wholesale contract events are consumed, what triggers them, what the consumer does with them?)

### Wishlist
—

### Notes
- Wholesaler table exists in DB but no consumer yet.
- Needs clarification on: event source, topic name, event schema, consumer action, which application hosts the consumer.

---

## R-008 — API Test Files for New GraphQL Resources

**Status:** Done
**Priority:** High
**Created:** 2026-03-20
**Features:** F-004

### The Idea
Create `.http` API test files in `api-tests/` for each new GraphQL resource, following the pattern of the existing files.

### The Problem
Developers have no way to manually exercise the new GraphQL endpoints from their IDE. QuotaResource is also missing a test file despite being complete.

### Wishlist
One `.http` file per resource, covering every operation with realistic sample data.

### Notes
- Follows the pattern of `CarrierGraphQL.http`, `ClassficationGraphQL.http`, etc.
- Covers: QuotaResource, ChargingTraceResource, DestinationGroupResource, SourceGroupResource.

---

## Requirement Template

```markdown
## R-NNN — [Title]

**Status:** Draft
**Priority:** High / Medium / Low
**Created:** YYYY-MM-DD
**Features:** — (populated after scoping)

### The Idea
What the person wants. In their own words. No structure imposed.
Do not translate into technical language here.

### The Problem
What is broken, missing, or painful right now.
Who experiences it.

### Wishlist
High-level aspirations — not criteria, not requirements.
What would "great" look like if everything went perfectly?
Unconstrained thinking is welcome here.

### Notes
Anything else worth capturing. Open questions. Context.
Constraints if known.
```
