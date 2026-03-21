# AGENTS.md — Agent Protocol

This file governs how AI agents behave in this repository.
It is language-agnostic and reusable across projects.

---

## Session Initialisation

At the start of every session, read these files in order before doing anything else:

1. `docs/PROJECT_BRIEF.md` — what this system is and why it exists
2. `docs/ARCHITECTURE.md` — system structure and boundaries
3. `docs/PROJECT_STRUCTURE.md` — where things live and what each directory is for
4. `.ai/memory/STATUS.md` — current implementation state and active focus
5. `.ai/context/` — read ALL files in this directory. Each file contains standards
   and conventions for a specific language or framework used in this project.
   Apply rules from every file present.

Do not skip any file. Do not begin work until all five steps are complete.

---

## Session Types

There are three distinct session types.

### Feature Design Session

Triggered automatically by GitHub Actions when a feature branch is pushed.
Runs on the feature branch. Decomposes the Feature into a numbered task queue.

1. Read context files (see Session Initialisation above)
2. Read the Feature definition from `.ai/memory/FEATURES.md`
3. Analyse the codebase to understand what exists and what must be built
4. Write numbered task files to `.ai/tasks/queue/` — one file per task
5. Write `.ai/tasks/READY` sentinel to trigger the Dev Session workflow
6. Commit and push — do not open a PR

### Dev Session

Triggered automatically by GitHub Actions when `.ai/tasks/READY` is pushed.
Runs on the feature branch. Processes the task queue to completion.

1. Read context files (see Session Initialisation above)
2. Process `.ai/tasks/queue/` files in numerical order
3. For each task: implement → build → test → commit → archive to `done/`
4. If build or tests fail — stop and exit. Do not proceed to the next task.
   The workflow turns red. A human investigates and re-triggers.
5. When queue is empty — remove READY, update STATUS.md, exit cleanly
6. Do NOT push or open a PR — the workflow handles this after clean exit

### Interactive Session

Run manually by a human on a feature branch for investigation or manual work.

1. Read context files (see Session Initialisation above)
2. Confirm not on `main` before making any changes
3. Follow the same build/test discipline as Dev Session

---

## Git Rules

One branch per Feature. Tasks are commits on that branch, not separate branches.

**Rules:**
- Never commit or make changes on `main` — unconditional
- Never push from within a recipe — the workflow pushes after the recipe exits cleanly
- Never open a PR from within a recipe — the workflow handles this
- Never merge pull requests — leave that for human review
- **Always use `git mv` to rename or move tracked files** — never OS-level `mv`
- **Stage new files immediately** using `git add <file>` after creating them
- Branch names reflect the Feature: `feature/F-001-charging-trace`
- Commit messages per task: `feat: [task description] — task N of N (F-NNN)`
- PR description: what the Feature delivers, packages affected, risks, testing summary


## Testing — Universal Rules

- Every piece of logic must have tests
- Tests must be executed and must pass — writing tests without running them
  does not satisfy this rule
- Tests must cover: success cases, failure cases, and edge cases
- Never claim a task complete with failing tests
- Fix failing tests before moving to the next step
- Unit tests must not require external services — isolate infrastructure dependencies
- See the relevant file in `.ai/context/` for language-specific test commands,
  frameworks, naming conventions, and patterns

---

## Build Verification — Universal Rules

- The build must pass cleanly before claiming a task complete
- Report exact command output on any failure — diagnose before retrying
- See the relevant file in `.ai/context/` for language-specific build commands

---

## Working Principles

- Analyse the full problem before modifying any code
- Prefer small, incremental changes over large rewrites
- When requirements are ambiguous, ask — never invent behaviour
- Correctness and maintainability take precedence over cleverness
- Do not make changes outside the scope of the current task
- Propose large refactors before implementing them — never execute without approval

---

## Sensitive Operations — Ask Before Proceeding

Always ask a human before:
- Deleting any file
- Broad refactors across multiple packages
- Changing public APIs
- Modifying core business logic (charging, payments, financial calculations)
- Introducing new dependencies
- **Modifying any contract** — see Contract Rules below

---

## Contract Rules

A **contract** is any structure or schema shared with an external system or process.
Contracts must **never be modified without explicit human approval**, regardless of
how minor the change appears.

The meta-rule: **You can never know all consumers of a contract.** A field that
appears unused may be read by a Java service, a database migration, a reporting
tool, or a downstream event processor. Adding, removing, or renaming fields
without approval is always a breaking change risk.

### What counts as a contract

**Kafka event schemas** — any struct that is serialised and published to a Kafka
topic, or deserialised from a Kafka topic. These are consumed by other services
(Java or otherwise) that you cannot see. The schema is defined by the upstream
publisher — the consuming service must accept what it receives, not invent fields
that the publisher does not send.

Examples in this repo: `internal/events/`

**Database-serialised structs** — any struct that is marshalled into a database
column (e.g. as JSON or JSONB). The database is a shared resource — other
applications (Java services, reporting tools, migrations) may read those columns
directly. The column layout is a contract with every reader.

Examples in this repo: quota structures stored as JSONB in the quota table.

**GraphQL schema** — any type, field, query, mutation, or subscription exposed via
the GraphQL API. External clients depend on these names and shapes.

**Store query interfaces** — the `sqlc`-generated query interfaces in
`internal/store/sqlc/`. These are generated from SQL — modify the SQL, not the Go.

### Rules

1. **Never add, remove, or rename fields** on a contract struct without explicit
   human approval. Adding a field is not "safe" — it may break deserialisation in
   consumers that use strict parsing.

2. **Never invent fields** that the upstream publisher does not send. If a field
   does not exist in the upstream schema, do not add it to the consuming struct.
   The classic example: adding an internal ID (e.g. `counterId`) to a provisioning
   event that originates from an external system. That system has no knowledge of
   internal identifiers.

3. **Internal IDs belong in internal structs**, not in contracts. If a domain
   entity needs an ID for internal use, generate it inside the service layer after
   consuming the event — do not add it to the event struct itself.

4. **When in doubt, ask.** If a task requires changing a contract to proceed,
   stop and raise it with the human before making any change.

5. **Document the reason** for any approved contract change in DECISIONS.md with
   an ADR, including which consumers were checked and what the migration plan is.

---

## Communication

- Explain what changed, referencing specific files and packages
- Explain reasoning behind design decisions
- Explicitly highlight risks for changes touching critical business logic
- State clearly when a verification step could not be performed
- Prefer clarity over brevity when describing risks

---

## Memory and Task Lifecycle

**After each task completes (before moving to the next):**
1. Update `.ai/memory/STATUS.md` — reflect what was built
2. Append significant decisions to `.ai/memory/DECISIONS.md` in ADR format
3. Archive the task: `git mv .ai/tasks/queue/NNN-name.md .ai/tasks/done/NNN-name.md`
4. Commit: `git add -A && git commit -m "feat: [description] — task N of N (F-NNN)"`

**When all tasks are complete:**
1. Remove `.ai/tasks/READY` — `git rm .ai/tasks/READY`
2. Update `.ai/memory/FEATURES.md` — set Feature status to "In Review"
3. Final commit: `git add -A && git commit -m "feat: F-NNN complete — all tasks implemented"`
4. Exit cleanly — the workflow pushes and opens the PR

**ADR format for DECISIONS.md:**
```
## ADR-NNN — Title
**Status:** Accepted
**Area:** Which part of the system
**Decision:** What was decided
**Rationale:** Why
**Consequences:** What this means going forward
```

