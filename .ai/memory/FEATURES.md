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
