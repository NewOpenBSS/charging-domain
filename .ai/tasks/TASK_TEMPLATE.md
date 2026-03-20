# Task: [Short descriptive title]

**Date:** YYYY-MM-DD
**Status:** Active | Complete

---

## Dependencies

List any branches that must be merged into main before this task can start.
Write "None" if this task has no dependencies.

- `feature/F-NNN-name` — reason why this must land first

---

## Objective

One paragraph. What is being built and why. No implementation detail.

---

## Scope

**In scope:**
- Specific thing 1
- Specific thing 2

**Out of scope:**
- Things that are explicitly not part of this task

---

## Context

What Claude Code needs to know that isn't already in the project memory files.
References to relevant existing packages, patterns to follow, constraints to observe.

Example:
- Follow the pattern established in `internal/backend/services/carrier_service.go`
- The `classification` table schema is in `db/migrations/000001_init.up.sql`
- GraphQL schema lives in `gql/schema/` — update the relevant `.graphql` file first

---

## Decisions Made During Design

Key decisions made before implementation started. Will be copied to DECISIONS.md on completion.

| Decision | Rationale |
|---|---|
| Example: Use X over Y | Because Z |

---

## Acceptance Criteria

How to know the task is done. Be specific.

- [ ] All new functions have unit tests
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] PR description includes risk assessment if charging logic is touched

---

## Risk Assessment

Any risks to charging, quota, or rating behaviour. Required field — write "None" if not applicable.

---

## Notes

Anything else relevant. Optional.
