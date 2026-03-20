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

---

## Backlog

<!-- Requirements ready or nearly ready for scoping -->

## R-001 — Port the Charging Backend Service to Go

**Status:** Ready for Scoping
**Priority:** High
**Created:** 2026-03-20
**Features:** — (populated after scoping)

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

## Backlog

<!-- Requirements ready or nearly ready for scoping -->

---

## Draft

<!-- Requirements captured but not yet refined -->

---

## Deferred

<!-- Requirements parked for future consideration -->

---

## Done

<!-- Completed requirements — kept for reference -->

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
