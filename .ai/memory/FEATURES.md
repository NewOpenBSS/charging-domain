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

**Status:** Done
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

**Status:** Done
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

## F-011 — Permission Enforcement Framework

**Status:** Backlog
**Priority:** High
**Created:** 2026-03-31

### Implementation Approval Required
- [x] Yes — pause after AI Design for human review before implementation begins
- [ ] No — proceed to implementation automatically after AI Design

### Feature Switch
None

### Goal
Build a reusable, platform-wide permission enforcement framework. The mechanism — `SecureRouter`, `auth.Require` middleware, GraphQL `@auth` directive, JWT `permissions` claim extraction — is built and tested independently of any actual permission rules. Any domain backend service can adopt it without additional framework work.

### Problem Statement
The Keycloak middleware validates *who the caller is* but does not check *what they are allowed to do*. Any valid token holder can execute any operation including destructive and financial ones. This is a security gap that must be closed before production. The solution must be reusable across all domain backend services in the platform — not just charging.

### Security Model

**Permission storage and resolution (ported from Java `SecurityService` / `PrivilegeAugmentor`):**
- Permissions are **not included in the JWT token**. They are stored as **Keycloak role attributes**.
- A configurable **marker attribute** (default: `GoBSS`) identifies roles that participate in the permission system. Only roles with this attribute are loaded. Other roles are ignored.
- On qualifying roles, the remaining attributes define permissions: **attribute key = resource name, attribute values = actions**. e.g. `{"GoBSS": ["true"], "rateplan": ["admin", "viewer"], "carrier": ["admin"]}`.
- The **domain name** (e.g. `charging`) is configurable. Combined with the attribute key and action, it forms the full permission identifier used in operation declarations.
- On startup and on a **configurable refresh interval**, the permission cache is loaded in sequence: (1) fetch all realms from Keycloak, excluding `master`, (2) for each realm fetch all roles, (3) for each role that carries the marker attribute, read its permission attributes and build the role→permissions mapping. The result is a `map[realm]map[role][]permission`. This is the Go equivalent of Java's `@Scheduled loadRolesAndPermissions()`. New realms (new wholesaler tenants) are automatically picked up on the next refresh cycle — no restart required. The cache pattern follows the existing `TenantResolver` design.
- On each authenticated request, the caller's effective permissions are resolved by: (1) extracting realm roles from `KeycloakClaims.RealmAccess.Roles`, (2) looking up each role in the cache, (3) aggregating all permissions across all roles. Resolved permissions are injected into the request context — analogous to Java's `PrivilegeAugmentor`.
- The cache is **realm-aware** (multi-tenant): permissions are keyed by realm. The realm is derived directly from the JWT `iss` claim using the existing `extractRealm` function — no workarounds needed. In the Java port, the realm was packed into a pipe-delimited `tenant-id` attribute (`realmName|tenantId`) due to Quarkus's single general-purpose identity attribute limitation. Go has no such constraint and reads the realm cleanly from the JWT issuer.
- `UserService.GetRoleAttributes` (already implemented in Go) provides the Keycloak admin API call. A **separate confidential admin client** must be configured in Keycloak with read access to realm roles and their attributes. Its credentials (`adminClientId`, `adminClientSecret`) are added to `KeycloakConfig` as a distinct section — entirely separate from the JWKS validation config which requires no credentials.
- **Go enforcement differs from Java** — Quarkus uses annotation processors (`@PermissionsAllowed`) which have no Go equivalent. The Go port uses `@auth` schema directives (GraphQL) and `SecureRouter` (REST) instead. The Keycloak data model is identical; only the enforcement mechanism changes.

**Permission model:**
- **Permissions** are domain-defined, resource-scoped, and action-oriented. Naming convention: `{domain}.{resource}.{action}` — e.g. `charging.rateplan.admin`, `charging.rateplan.approver`, `charging.rateplan.viewer`. CRUD-style permissions are avoided in favour of business-meaningful actions.
- **Roles** are Keycloak realm roles (schema roles). A role is a named collection of explicitly listed permissions stored as role attributes — no wildcard or implicit grants. Roles are assigned to **users only** — permissions are never assigned to a user directly. The chain is always: User → Role → Permissions (via role attributes). Every permission a user holds is traceable to a deliberate Keycloak admin decision.
- **Super admin** is simply a role that has all permissions listed in its attributes — no special framework treatment needed.
- The backend **never checks role names** — only the resolved permissions. Keycloak's role structure can evolve without touching the codebase.
- **OR semantics** — a caller is granted access if they hold any one of the listed permissions for that operation. No AND logic required.
- **Permissions are reused across operations** — `charging.rateplan.viewer` covers `ratePlanList`, `countRatePlans`, and `ratePlanById`. Many-to-many: one permission applies to many operations, one operation accepts many permissions.
- **Reserved permissions:** `public` (no token required), `authenticated` (valid token, no specific permission required).
- **Deny by default** — an operation with no permission declaration is denied and a warning is logged. A missing declaration is a configuration error, never a silent grant of access.
- **Ingress/egress boundary** — only `/api/xxx` paths are externally accessible. Other paths (`/health`, `/metrics`) are internal, controlled by network policy, no code-level security applied.

### MVP
Every REST and GraphQL operation in charging-backend has an explicit permission declaration. Undeclared operations are denied. Auth-disabled mode bypasses all checks for local dev.

### Acceptance Criteria
- [ ] JWT `permissions` claim is populated in Keycloak via a token mapper and extracted into `KeycloakClaims`
- [ ] Every REST `/api/` route declares permissions at registration via `SecureRouter` — compile error if omitted
- [ ] Every GraphQL operation declares permissions via `@auth` schema directive — deny + warn if missing at runtime
- [ ] A caller with a valid token and a matching permission is granted access
- [ ] A caller with a valid token but no matching permission receives HTTP 403 — the resolver does not execute
- [ ] A caller with no token on a non-`public` endpoint receives HTTP 401
- [ ] `auth.enabled: false` bypasses all permission checks — local dev behaviour preserved
- [ ] The framework lives in `internal/auth/` and has no dependency on the charging domain
- [ ] Unit tests cover: authorised, unauthorised (wrong permission), unauthenticated, public endpoint, and auth-disabled pass-through

### Constraints
- Permission names for the charging domain must be agreed with the Keycloak administrator before implementation
- Must not require changes to the GraphQL schema contracts (field names, types, operation names)
- **REST enforcement — `SecureRouter`:** a wrapper around `chi.Router` that requires `[]string` permissions on every route registration method. Enforces deny-by-default at compile time.
- **GraphQL enforcement — schema directives:** `@auth(permissions: [String!]!)` declared on each operation in the `.graphql` schema. gqlgen generates enforcement. Deny-by-default (missing directive = deny + warn) to be confirmed as supportable during scoping.
- **Shared framework:** `SecureRouter`, `auth.Require` middleware, and GraphQL directive enforcement in `internal/auth/` — reusable across all domain backend services.
- The Keycloak middleware remains responsible for token validation only. Permission enforcement is a separate, downstream concern.

### Out of Scope
- Resource-level authorisation (a user may only access their own subscriber's data) — deferred
- UI/portal role management — Keycloak admin console manages role and permission assignment
- Permission enforcement in services other than charging-backend — the framework will be reusable but other domains are out of scope for this feature

### Design Proof of Concept — Approval Workflows
The state machine operations (RatePlan, Classification) serve as the canonical test of the permission model's expressiveness. A correctly designed permission set enforces separation of duties without any application logic:

> **Note:** Permission names below are illustrative only — used to explain the model. Actual permission names must be taken from the Java codebase to ensure the Go service uses the same permissions already configured in Keycloak. This is a prerequisite for scoping.

| Operation | Required Permission |
|---|---|
| `createRatePlan`, `updateRatePlan` | `charging.rateplan.admin` |
| `submitRatePlanForApproval` | `charging.rateplan.admin` |
| `approveRatePlan`, `rejectRatePlan` | `charging.rateplan.approver` |
| `ratePlanList`, `ratePlanById`, `countRatePlans` | `charging.rateplan.viewer` |

A user with only `charging.rateplan.admin` can author and submit but cannot approve their own work. A user with only `charging.rateplan.approver` can approve or reject but cannot author. Separation of duties is enforced by the permission model alone.

If the model handles this cleanly, it handles everything else in the platform.

### Parking Lot
- Audit logging of authorisation decisions (who accessed what, when, with which permission)
- Fine-grained field-level access control

### Future Considerations
- As the platform grows, a policy engine (e.g. OPA) may replace the hard-coded permission map
- Client roles (`resource_access`) may be needed if different portal clients require different access levels

---

## F-012 — Charging-Backend Permission Rules

**Status:** Backlog
**Priority:** High
**Created:** 2026-03-31

### Implementation Approval Required
- [x] Yes — pause after AI Design for human review before implementation begins
- [ ] No — proceed to implementation automatically after AI Design

### Feature Switch
None

### Goal
Wire the actual permission declarations into every REST and GraphQL operation in charging-backend using the F-011 framework. Every operation is explicitly assigned one or more permissions from the agreed charging domain permission set.

### Problem Statement
Once the F-011 framework exists, charging-backend operations are still unprotected until each one is wired to a permission. This feature completes the security coverage by declaring permissions on every operation.

### MVP
Every REST `/api/` route and every GraphQL operation in charging-backend has an explicit permission declaration using the actual permission names from the Java codebase. No operation is reachable without a declared permission.

### Acceptance Criteria
- [ ] Every GraphQL operation has an `@auth` directive with real permission names from the Java codebase
- [ ] Every REST `/api/` route is registered via `SecureRouter` with real permission names
- [ ] Approval workflow operations (RatePlan, Classification state transitions) correctly enforce separation of duties via `.approver` vs `.admin` permissions
- [ ] Integration tested against the live Keycloak `gonz` realm with real tokens
- [ ] Auth-disabled mode still bypasses all checks

### Constraints
- **Blocked on F-011** — the framework must exist before rules can be wired
- **Blocked on Java permission extraction** — actual permission names must be sourced from the Java codebase before implementation begins. Do not invent permission names.
- Permission names must match exactly what is configured in Keycloak — the Go service shares the same realm and token mapper as the Java service

### Out of Scope
- Permission rules for services other than charging-backend
- Changes to the framework itself — those belong in F-011

### Parking Lot
None

### Future Considerations
- As new operations are added to charging-backend, each must be wired to a permission at the time of creation — enforced by the `SecureRouter` compile-time check and GraphQL deny-by-default

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
