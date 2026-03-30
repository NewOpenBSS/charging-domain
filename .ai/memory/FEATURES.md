# Features

This is the single source of truth for all active feature work.
Updated by humans after each PR is merged.
Read by AI agents at the start of every design and development session.

Done features are archived to `.ai/memory/archive/FEATURES.md`.

## Status Values
- **Backlog** — defined, not yet started
- **Ready for AI Design** — Feature approved, waiting for technical decomposition
- **In Design** — AI decomposing into tasks
- **In Development** — branch being implemented
- **In Review** — PR open, waiting for human review
- **Done** — all PRs merged

---

## Ready for AI Design

## F-003 — SourceGroupResource

**Status:** Ready for AI Design
**Priority:** High
**Created:** 2026-03-20
**Branch:** feature/F-003-source-group-resource

### Implementation Approval Required
- [ ] Yes — pause after AI Design for human review before implementation begins
- [x] No — proceed to implementation automatically after AI Design

### Feature Switch
None

### Goal
Expose full CRUD for carrier source groups via GraphQL in the Go charging-backend, matching the Java API surface exactly — mirroring the DestinationGroupResource implementation.

### Problem Statement
Source groups classify the roaming origin of a carrier and drive the `sourceType` dimension of a `RateKey` in the charging engine. The `carrier_source_group` reference table exists and is seeded, but there is no admin API to manage its entries. Admins cannot add new regions, rename existing ones, or remove obsolete entries without a DB migration. The Java charging-backend exposed a full CRUD GraphQL resource for this; the Go port is missing it.

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
- [ ] A `SourceGroupGraphQL.http` file is present in `api-tests/` covering all six operations with realistic sample data

### Constraints
- GraphQL operation names, field names, and behaviour must be identical to the Java service — external clients must work without modification
- No state machine or approval workflow — plain CRUD only
- Implementation must follow the DestinationGroupResource pattern exactly (same layering: sqlc queries → store → service → resolvers)

### Out of Scope
- Approval workflows or state machines for source groups
- Bulk import or export of source groups
- Referential integrity enforcement on delete (no FK check against the carrier table — same as DestinationGroup)

### Parking Lot
None

### Future Considerations
- Cursor-based pagination (current OFFSET-based approach degrades on large datasets — acceptable for now)
- Referential integrity: if a source group is deleted while carriers still reference it, those carriers will have a dangling `sourceGroup` value

---

## Backlog

## F-009 — Charging Domain Housekeeping

**Status:** Backlog
**Priority:** High
**Created:** 2026-03-30
**Branch:** feature/F-009-charging-domain-housekeeping

### Implementation Approval Required
- [ ] Yes — pause after AI Design for human review before implementation begins
- [x] No — proceed to implementation automatically after AI Design

### Feature Switch
None

### Goal
A standalone Go binary, invoked as a Kubernetes CronJob, that processes expired quota counters for dormant subscribers, removes orphaned charging sessions, purges old trace records, and deletes superseded rate plan versions.

### Problem Statement
Revenue is silently lost when subscribers become dormant — their quota counters expire without being processed and the unused balance is never recognised as revenue. For active subscribers this happens JIT via the quota manager; for dormant subscribers it never happens. Additionally, `charging_data` accumulates orphaned session rows that were never terminated cleanly, and `charging_trace` grows unboundedly, both creating unnecessary storage overhead with no current cleanup mechanism.

### MVP
All four housekeeping tasks run top to bottom in a single binary invocation:
1. **Quota expiry** — Find all quotas where `next_action_time` is in the past; open each via the existing quota manager and save — existing expiry logic fires and posts revenue journals.
2. **Stale sessions** — Delete rows from `charging_data` where `modified_on` is older than a configurable threshold (default: 24h).
3. **Trace purge** — Delete rows from `charging_trace` older than a configurable threshold (default: 36h).
4. **Rate plan cleanup** — Delete superseded ACTIVE rate versions replaced by a newer version for longer than a configurable threshold (default: 30 days). DRAFT and PENDING versions are never touched.

### Acceptance Criteria
- [ ] A quota with `next_action_time` in the past is opened via the quota manager, saved, and revenue journals are posted — identical to JIT processing for active subscribers
- [ ] Rows in `charging_data` with `modified_on` older than the configured threshold are deleted; newer rows are untouched
- [ ] Rows in `charging_trace` older than the configured threshold are deleted; newer rows are untouched
- [ ] Superseded ACTIVE rate versions older than the configured threshold are deleted; DRAFT and PENDING versions are never deleted regardless of age
- [ ] The three configurable thresholds (stale session, trace purge, rate plan cleanup) are read from environment variables; defaults apply if absent; quota expiry has no configurable threshold
- [ ] The job logs a summary on completion — records processed/deleted per category
- [ ] The job exits with code 0 on success, non-zero on any error

### Constraints
- No scheduler embedded in the application — Kubernetes owns the schedule; the binary runs all four tasks once and exits
- Quota expiry must reuse the existing quota manager logic — no new expiry business logic

### Out of Scope
- Helm chart / Kubernetes CronJob deployment scripts (deferred to a future requirement)
- Alerting or metrics on housekeeping run outcomes
- Manual trigger endpoint

### Parking Lot
- Helm chart / Kubernetes CronJob deployment scripts — deferred to a future requirement

### Future Considerations
- Metrics/alerting on housekeeping outcomes (records processed, errors encountered)

---

## F-004 — GraphQL API Test Files

**Status:** Backlog
**Priority:** High
**Created:** 2026-03-20

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
- Test files for resources already covered

### Parking Lot
None

### Future Considerations
- Automated API test execution in CI pipeline

---

## Feature Template

Copy this template when adding a new Feature:

```markdown
## F-NNN — [Title]

**Status:** Backlog
**Priority:** High / Medium / Low
**Created:** YYYY-MM-DD
**Branch:** (filled in by scoping recipe)

### Implementation Approval Required
- [ ] Yes — pause after AI Design for human review before implementation begins
- [ ] No — proceed to implementation automatically after AI Design

### Feature Switch
<!-- Name of the feature switch if required, or "None" -->

### Goal
One sentence. What this builds and why.

### Problem Statement
What is broken or missing. Who is affected. Current state vs desired state.

### MVP
The smallest version that delivers real value.

### Acceptance Criteria
- [ ] [user/role] can [achieve something observable]
- [ ] [condition] results in [observable outcome]

### Constraints
Technical, regulatory, business, or performance constraints.

### Out of Scope
What is explicitly not included in this Feature.

### Parking Lot
Ideas that emerged during design but are deferred.

### Future Considerations
Architectural decisions this Feature must not foreclose.
```
