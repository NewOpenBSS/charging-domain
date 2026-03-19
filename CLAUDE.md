# CLAUDE.md — Agent Protocol for go-ocs

This file governs how Claude Code behaves in this repository.
Read this file first. Then follow the Session Initialisation sequence below.

---

## Session Initialisation

At the start of every session, read these files in order before doing anything else:

1. `docs/PROJECT_BRIEF.md` — what this system is and why it exists
2. `docs/ARCHITECTURE.md` — system structure and package boundaries
3. `docs/PROJECT_STRUCTURE.md` — where things live and what each directory is for
4. `.ai/memory/STATUS.md` — current implementation state and active focus
5. `.ai/tasks/CURRENT.md` — active task spec (if the file exists)

Do not skip any file. Do not begin work until all four have been read.
If `tasks/CURRENT.md` does not exist, ask the human what the current task is.

---

## Session End Protocol

Before opening a pull request, always complete these steps in order:

1. Update `.ai/memory/STATUS.md` — reflect what changed, mark completed items
2. Append any significant decisions to `.ai/memory/DECISIONS.md` using the ADR format
3. Move `.ai/tasks/CURRENT.md` to `.ai/tasks/done/NNN-short-description.md`
4. Open the pull request

Do not open a pull request without completing steps 1–3 first.

---

## Git and Branching

This project uses **Git worktrees** for parallel feature development. Each feature
branch is checked out as its own worktree under `~/Development/goplay/branches/`.

```
~/Development/goplay/
    go-ocs/                          ← main branch, always clean, mirrors production
    branches/
        go-ocs-wholesaler-admin/     ← feature worktree, own CURRENT.md
        go-ocs-fix-rateplan/         ← another feature worktree, isolated
```

**Starting a new feature:**
```bash
git worktree add ~/Development/goplay/branches/go-ocs-<feature-name> feature/<feature-name>
```

**Finishing a feature (after PR is merged):**
```bash
git worktree remove ~/Development/goplay/branches/go-ocs-<feature-name>
```

**Rules:**
- **Never commit or make changes on `master`** — this is unconditional
- Each worktree has its own `.ai/tasks/CURRENT.md` — no coordination needed
- Check current branch before any change: `git branch --show-current`
- If on `master`: stop immediately and ask the human which branch to use
- Branch names must reflect purpose: `feature/add-quota-tax`, `fix/double-charge`
- Never merge pull requests — leave that for human review

**Commit hygiene:**
- Stage all files that are part of the change before opening a PR
- Never stage generated files listed in `.gitignore`
- Never force-add (`git add -f`) a gitignored file without human approval
- PR description must include: what changed, why, which packages affected,
  any risks to charging/quota/rating logic, and a testing summary

---

## Build Verification

After any change to Go source, imports, or dependencies — run in this order:

```bash
go mod tidy
go build ./...
go test ./...
```

Never claim an implementation is complete without all three passing.
Report exact command output on any failure. Diagnose before retrying.

---

## Working Principles

- Analyse the full problem before modifying any code
- Prefer small, incremental changes over large rewrites
- When requirements are ambiguous, ask — never invent behaviour
- Correctness and maintainability take precedence over cleverness
- Do not make changes outside the scope of the current task
- Propose large refactors before implementing them

---

## Go Standards

**Context:** Every function performing I/O must accept `context.Context` as first
parameter. Never use `context.Background()` inside business logic. Never store context
in structs.

**Nil safety:** Check all pointer, interface, slice, and map returns before use.

**Panics:** Never use `panic` in business logic. Only permitted in `main()` for startup
validation. Handlers must recover from unexpected panics.

**Interfaces:** Define at point of consumption, not implementation. Keep them small.
Accept interfaces, return concrete types.

**Structs:** Always use named fields in struct literals. Positional initialisation
is prohibited in all code including tests.

**Constants:** Numeric literals and strings with business meaning must be named
constants. Timeouts, thresholds, and buffer sizes must come from config, not hardcoded.

---

## Error Handling

- Never use `fmt.Errorf` or `errors.New` for domain errors
- All domain errors use typed error structs with a `Code` type and constructor functions
- Reference pattern: `internal/chargeengine/ocserrors/errors.go`
- Error codes must be meaningful stable identifiers: `"UNKNOWN_SUBSCRIBER"`, `"OUT_OF_FUNDS"`
- Use `errors.As` for type assertions — never string comparison
- `fmt.Errorf` is permitted only for wrapping infrastructure errors (DB, network, I/O)

---

## Concurrency

- All shared mutable state must be protected: `sync.Mutex`, `sync.RWMutex`, atomics, or channels
- Every goroutine must have a documented lifecycle and shutdown path
- No goroutine leaks — every goroutine must terminate via context or stop channel
- Always run `go test -race ./...` for any code involving concurrent access
- Never use `init()` to initialise shared state

---

## Time

- Never call `time.Now()` inside business logic — inject time as a parameter
- All timestamps stored or published must be UTC
- Never compare `time.Time` with `==` — use `.Equal()`, `.Before()`, `.After()`

---

## Financial Values

- All financial values use `github.com/shopspring/decimal` — no float types permitted
- Precision comes from `DecimalDigits int32` config field (default: 22)
- Never hardcode precision — always read from config and propagate explicitly

---

## Architecture Boundaries

- Transport handlers must be thin — delegate all logic to services
- No business logic in HTTP or Diameter handlers
- All database access through repository interfaces in `internal/store/`
- Kafka consumers must delegate to services
- Event publishing isolated from core charging logic
- New applications must follow patterns in `charging-engine` and `charging-dra`
- Configuration from YAML only — no environment variables in application code

---

## Domain Safety — Non-Negotiable

This system processes real money. These rules override everything else.

- **Charging must be deterministic** — same inputs always produce same outputs
- **Quota counters must never go negative** — no overdraft under any code path
- **All quota operations must be transactional** — database transactions required
- **Duplicate/replayed events must not cause double charging** — idempotency enforced
- **Never invent billing semantics** — if requirements are unclear, stop and ask
- **Any change affecting charging, quota, or rating** requires a written risk
  explanation before implementation begins

---

## Sensitive Data

- Subscriber identifiers, balances, and transaction amounts must not appear in logs
- Internal state, raw DB errors, and stack traces must not appear in API responses
- Credentials and connection strings in YAML only — never in source code or logs

---

## Testing

- Every Go source file with functions must have a `_test.go` file
- Tests must cover: success, failure, edge cases, and branching paths
- Tests must run and pass — writing without running does not satisfy this rule
- Unit tests must not require external services (PostgreSQL, Kafka)
- Use build tags or naming conventions to isolate integration tests
- Use table-driven tests for functions with multiple input/output combinations:
  name each case, use `t.Run(tc.name, ...)`, follow pattern `TestFn_Scenario_Expected`
- Run `go test -race ./...` for any concurrent code

---

## Database Migrations

- Every migration must be reversible — up migration requires a down migration
- Migrations must be backward compatible with currently deployed code
- Never drop columns or tables still referenced by application code
- Test migrations locally with `make migrate-up` before committing
- Data migrations must be proposed to a human with risk stated explicitly

---

## Documentation

- All public functions and methods must have a Go doc comment
- Comments describe what and why — not a restatement of the code

---

## Dependency Management

- Prefer libraries already in the project
- Verify new module paths on `pkg.go.dev` before adding
- Never modify files marked `// Code generated ... DO NOT EDIT`
- Re-run the generator to update generated files

---

## Sensitive Operations — Ask Before Proceeding

Always ask a human before:
- Deleting any file
- Broad refactors
- Changing public APIs
- Modifying core charging, quota, or rating logic
- Introducing new dependencies

---

## Communication

- Explain what changed, referencing specific files and packages
- Explain reasoning behind design decisions
- Explicitly highlight risks for changes touching charging, quota, or rating
- State clearly when a verification step could not be performed
- Prefer clarity over brevity when describing risks
