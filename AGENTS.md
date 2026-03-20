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

There are three distinct session types. The human will state which one applies.

### Design Session

Runs in the **main repository directory** on `main`. Its only job is to create
the worktree and produce a task summary.

1. Read context files in order (see Session Initialisation above)
2. Confirm current branch is `main`: `git branch --show-current`
   - If NOT on `main`: stop and tell the human
3. Ask the human for the feature name if not already provided
4. Create the worktree:
   ```bash
   git worktree add ~/Development/goplay/branches/<repo>-<feature> feature/<feature>
   ```
5. Produce a brief task summary (1 paragraph) describing what will be built
6. Tell the human the worktree path and to open it in their preferred
   editor and agent tool for the implementation session
7. Stop — do not write any code or files in the main repo


### Spec and Implementation Session

Runs inside the **feature worktree**. Clarifies the requirement, confirms understanding,
then proceeds fully autonomously through implementation to PR.

1. Read context files in order (see Session Initialisation above)
2. Confirm current branch is NOT `main`: `git branch --show-current`
   - If on `main`: stop immediately and tell the human
3. Analyse the requirement against the existing codebase and context files
4. If anything is ambiguous, ask focused clarifying questions — one exchange only
5. Summarise in plain language what will be built, including:
   - Whether the requirement decomposes into multiple sequential tasks
   - The order tasks will run if more than one
   - Any risks to critical business logic
6. **Wait for human confirmation — this is the only pause in the session**
7. Once confirmed, proceed fully autonomously through all remaining steps
   with no further interruptions:

   **For each task (repeat if requirement decomposes into multiple):**
   - Write `.ai/tasks/CURRENT.md` using the template in `.ai/tasks/TASK_TEMPLATE.md`
   - Implement the full task
   - Build and test — fix any failures before proceeding
   - Update `.ai/memory/STATUS.md`
   - Append any decisions to `.ai/memory/DECISIONS.md`
   - Move `.ai/tasks/CURRENT.md` to `.ai/tasks/done/NNN-short-description.md`

   **After all tasks complete:**
   - Open a single pull request covering all work
   - Report completion — what was built, what changed, PR link

---

## Git Rules

This project uses **Git worktrees** for feature development. The main repository
directory always tracks `main` and mirrors production. Feature branches live under
a sibling `branches/` directory:

```
~/Development/goplay/
    <repo>/                      ← main, always clean
    branches/
        <repo>-<feature>/        ← feature worktree, own CURRENT.md
```

**Rules:**
- Never commit or make changes on `main` — unconditional
- Never merge pull requests — leave that for human review
- **Always use `git mv` to rename or move tracked files** — never use an OS-level
  `mv` command. OS-level moves cause Git to see a deletion and a new untracked file,
  losing all commit history on the file.
- **Stage new files immediately after creating them** using `git add <file>` — do not
  wait until the end of the task. This keeps the working tree clean and makes it easy
  to see what has been added versus modified in the IDE at any point during development.
- **Every recipe that commits to the repo must also push and open a PR** — no commit
  should be left unpushed and unreviewed. Register updates (REQUIREMENTS.md, FEATURES.md)
  commit directly to the current branch — no separate branch needed. Code changes always
  go via a feature branch and PR — never direct to main.
- Branch names must reflect purpose: `feature/wholesaler-admin`, `fix/rateplan-query`
- PR description must include: what changed, why, packages affected, risks to
  critical business logic, and a testing summary


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

---

## Communication

- Explain what changed, referencing specific files and packages
- Explain reasoning behind design decisions
- Explicitly highlight risks for changes touching critical business logic
- State clearly when a verification step could not be performed
- Prefer clarity over brevity when describing risks

---

## Memory and Task Lifecycle

**At session end, before opening a PR:**
1. Update `.ai/memory/STATUS.md` — reflect what changed, mark completed items
2. Append significant decisions to `.ai/memory/DECISIONS.md` using ADR format
3. Move `.ai/tasks/CURRENT.md` to `.ai/tasks/done/NNN-short-description.md`

**ADR format for DECISIONS.md:**
```
## ADR-NNN — Title
**Status:** Accepted
**Area:** Which part of the system
**Decision:** What was decided
**Rationale:** Why
**Consequences:** What this means going forward
```

