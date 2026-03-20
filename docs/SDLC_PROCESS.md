# Agentic Software Development Lifecycle

**Version:** 1.0
**Status:** Living document — update as the process evolves

---

## Preface

This document defines a software development lifecycle designed from first principles
for agentic AI development. It does not adapt or modify any existing framework such as
Scrum or Kanban. It is built around one central observation:

> **Traditional SDLC processes were designed to manage human implementation capacity
> as the primary constraint. Agentic AI removes that constraint. The process must
> be redesigned accordingly.**

The valuable parts of traditional SDLC are preserved — requirements definition,
architecture decisions, quality gates, and review. The ceremonies and artefacts that
existed solely to manage human capacity are discarded.

---

## Core Principles

**1. Human judgment at boundaries, AI execution between them**
Humans define intent, approve plans, and validate outcomes. AI does everything in between.

**2. The bottleneck is now design and review, not implementation**
Process design must optimise for requirement quality and review throughput,
not for implementation velocity.

**3. Every process step must earn its place**
If a ceremony, artefact, or meeting exists only to coordinate human capacity,
it does not belong in this process.

**4. Autonomy within defined constraints**
AI agents operate fully autonomously within the boundaries established by the
human-approved context layer (AGENTS.md, context files, project memory).

**5. Continuous improvement of the context layer**
The quality of AI output is directly proportional to the quality of the context
it operates within. Improving AGENTS.md and context files is engineering work,
not documentation work.

**6. Human design answers what and why — AI design answers how**
The design process has two distinct parts with different owners. Humans define
the problem, the constraints, and the success criteria. AI defines the technical
solution. Conflating the two produces poor outcomes in both directions.

**7. Define the MVP — then stop**
Every Feature must have a clearly defined Minimum Viable Product. The MVP is the
forcing function that ends the design conversation. If you cannot define the
smallest version of this feature that delivers real value, the Feature is not ready.
Everything beyond the MVP is a future Feature.

**8. Small incremental changes**
Epics should be small enough that their implementation fits in one PR that a
human can review in a single sitting. Large changes are hard to review, hard to
revert, and mask problems. If a Feature feels large, split it. The AI can implement
quickly — but the human must be able to validate what was built. Review capacity
is the real constraint.

**9. Refactor when needed, not upfront**
Implement the simplest solution that satisfies the MVP. Do not design for
future requirements that haven't been asked for yet. When future Features reveal
that the current design needs to change, refactor then — with full context of
the actual need. Premature refactoring optimises for a future that may never
arrive.


---

## The Role of AI in the Design Phase

The AI plays a specific, bounded role in Stage 2. It is not a passive scribe.
It is an active participant with three defined responsibilities:

### 1. Distil the MVP

The AI's job is to push *toward* the smallest viable version of the feature,
not to help add more to it. When the human describes what they want, the AI
asks:

- *"What is the minimum version of this that delivers real value?"*
- *"What parts of this could be deferred to a later Feature without blocking the core need?"*
- *"If we built only X, would that be enough to validate the idea?"*

The AI is the forcing function that prevents scope creep from entering the pipeline.
A feature that is too large gets split — not just documented as large.

### 2. Maintain the Parking Lot

Good ideas that emerge during the design conversation but are out of scope for
the current Feature must not be lost — and must not be built prematurely.

The AI captures these in a **Parking Lot** section within the Feature:

```markdown
### Parking Lot
Ideas that emerged during design but are deferred to future Features:
- [Idea]: [brief description and why it was deferred]
```

At the end of the design session the AI reviews the parking lot with the human
and asks: *"Should any of these become the next Feature?"* Some parking lot items
become the natural next step in the roadmap. Others stay parked indefinitely.

The parking lot is how good ideas survive without derailing the current work.

### 3. Navigate the Roadmap — Next Destination Only

The human knows where the product is going. They do not need to design the
entire journey before starting. The AI helps the human identify the **next
destination** clearly without requiring the full roadmap to be designed upfront.

The questions that drive this:

- *"What is the next meaningful milestone from where we are now?"*
- *"What would need to be true for the feature after this one to be buildable?"*
- *"Are there architectural decisions in this Feature that would make future steps
  harder? If so, we should note them — but not build for them yet."*

The roadmap is the destination. The Features are the steps. You only need to see
the next step clearly.

---


```
STAGE 1: INTENT              Human brings a raw idea
STAGE 2: SCOPING        Human → well-formed Feature (what and why)
STAGE 3: FEATURE DESIGN           AI → technical decomposition + branches (how)
STAGE 4: DEVELOPMENT      AI → code + tests + PR (fully autonomous)
STAGE 5: VALIDATION          Human → PR review → merge
```

**Stage 2 and Stage 3 are deliberately separated.** Human design answers *what to
build and why*. AI design answers *how to build it technically*. These require
different inputs, different outputs, and different quality gates.

There are exactly **three human touch points**:
- End of Stage 2 — approve the Feature (what and why is correct)
- End of Stage 3 — approve the decomposition (how is sensible)
- End of Stage 5 — approve the implementation (what was built is correct)

Everything else is autonomous.

---

## Stage 1 — Intent

**Input:** A raw idea, business need, or problem statement from any stakeholder.
No structure required. Could be a sentence, a paragraph, a voice note transcription.

**Output:** A raw intent statement ready for refinement.

**Human effort:** Minutes.

**Notes:**
Anyone can contribute intent — developers, product owners, users, business stakeholders.
The barrier to entry is deliberately low. Quality comes in Stage 2, not here.

---

## Stage 2 — Scoping

**Input:** Raw intent from Stage 1.
**Output:** A well-formed Feature entry in `.ai/memory/FEATURES.md`.
**Tool:** Goose Desktop — conversational, human-driven.
**Human effort:** 15–30 minutes of focused conversation with AI assistance.

This stage is owned by the human. The AI assists by asking clarifying questions
and helping articulate the Feature precisely — but the human drives the content.

**Time box:** 15–30 minutes maximum. If the Feature cannot be articulated in this
time, the idea is not ready — not the process. Park it and return when thinking
is clearer.

**The "good enough" exit condition — three questions:**
1. Can the AI implement this without inventing requirements?
2. Would you recognise the correct implementation when you see it in a PR?
3. Is there anything that would make you reject the PR regardless of quality?

If the answers are yes, yes, no — the Feature is good enough. Write it and move on.

**The escape hatch:** The Feature can be updated after implementation starts.
If the dev session surfaces a question the Feature didn't answer, the session
pauses, the Feature is updated, and implementation continues. The Feature is a
living document, not a contract carved in stone.

**What this stage must answer:**
- What problem does this solve and for whom?
- What does success look like — specifically?
- What is explicitly out of scope?
- What are the constraints — technical, regulatory, business?
- What future needs might this need to accommodate? *(the most commonly missed question)*
- How does this relate to other Features in the backlog?

**The critical insight:** Most AI design failures trace back here. When the AI
designs something technically correct but wrong for the need, it is almost always
because the Feature was incomplete — not because the AI made a bad technical decision.
The human did not provide the full picture. This stage exists to close that gap.

**The equally important counter-insight:** Perfect is the enemy of shipped.
The AI asks only questions whose answers would materially change the implementation.
Edge cases, refinements, and future considerations beyond the current scope are
explicitly deferred. The cost of adjusting in the next Feature is lower than the
cost of analysis paralysis.

The output is a Feature with:
- A clear goal statement
- **Minimum Viable Product** — the smallest version that delivers real value
- Defined scope and explicit out-of-scope boundaries
- Success criteria specific enough to verify
- Known constraints
- Future considerations that may influence current architecture
- **Parking Lot** — good ideas deferred to future Features (captured, not lost)
- Priority relative to other Features

**Quality gate:** Human reviews and approves the Feature before Stage 3 begins.
A Feature that cannot be clearly stated is not ready for implementation.
If you cannot define the MVP, the Feature is not ready.
If you cannot write the success criteria, the Feature is not ready.


---

## Stage 3 — Feature Design

**Input:** Approved Feature from `.ai/memory/FEATURES.md`.
**Output:** One or more feature branches, each with a `CURRENT.md` task spec.
**Tool:** Feature Design recipe (Goose Desktop or CLI — AI-driven).
**Human effort:** 10–20 minutes review. Rarely requires changes.

This stage is owned by the AI. The human does not drive the design — they review it.

The AI reads the Feature, analyses the existing codebase and architecture, and
decomposes the Feature into implementable features. For each feature it:
- Creates a Git branch: `feature/<epic-id>-<feature-name>`
- Creates a worktree at `~/Development/goplay/branches/<repo>-<feature-name>`
- Writes `.ai/tasks/CURRENT.md` with scope, context, decisions, and acceptance criteria
- Pushes the branch to GitHub (making it visible to all machines)

**What the human is reviewing at this gate:**
- Is the decomposition aligned with the MVP — not more, not less?
- Is the decomposition sensible? Are features the right size?
- Is the sequencing and dependency ordering correct?
- Does the AI's interpretation of the Feature match the intent?

**The critical insight:** In practice, human disagreement at this gate is rare.
When it does occur, it almost always reveals that the Feature in Stage 2 was
incomplete — not that the AI's technical judgment was wrong. If you find yourself
disagreeing with the decomposition, first ask: *did the Feature fully describe the need?*
If not, update the Feature and re-run Stage 3. If the Feature was complete and the
decomposition is genuinely wrong, that is feedback for improving the AI design recipe.

**Quality gate:** Human approves the decomposition before autonomous implementation
begins. This is the last human touch point before PRs appear.

**Simple Features** may produce a single feature. Complex Features may produce several,
potentially with dependencies that determine sequencing.

---

## Stage 4 — Development

**Input:** `CURRENT.md` in the feature worktree.
**Output:** A pull request on GitHub.
**Tool:** Dev Session recipe (Goose CLI — autonomous).
**Human effort:** Zero during execution. Minutes to trigger.

The AI operates fully autonomously:
1. Reads all context files (PROJECT_BRIEF, ARCHITECTURE, PROJECT_STRUCTURE,
   STATUS, and all `.ai/context/` language files)
2. Implements the feature as specified in `CURRENT.md`
3. Builds and tests — fixes failures before proceeding
4. Updates `.ai/memory/STATUS.md` and `.ai/memory/DECISIONS.md`
5. Archives the completed task to `.ai/tasks/done/`
6. Opens a pull request

If a feature was decomposed into multiple tasks, each task runs sequentially.
A single PR covers all tasks.

**No human involvement during this stage.** The human returns to a PR waiting
for review.

---

## Stage 5 — Validation

**Input:** An open pull request on GitHub.
**Output:** Merged code, closed Feature (or ready for next feature).
**Human effort:** Proportional to implementation complexity.

The human reviews the PR against the Feature's success criteria:
- Does it do what was asked?
- Is the code quality acceptable?
- Are the tests adequate?
- Are there risks to critical business logic?

**If approved:** Merge. Update Feature status in `FEATURES.md`.
**If changes needed:** Add review comments. The dev session can be re-triggered
with feedback, or the human can make targeted corrections directly.

This is the final quality gate. The merge is the human's acceptance of the work.


---

## Boundary — Delivery vs Release

This framework governs **delivery** — the process by which code is written,
tested, and merged. It does not govern **release** — the process by which
capability becomes available to users.

These are deliberately separated:

| Delivery | Release |
|---|---|
| Code is written and merged | Feature is enabled for users |
| Governed by this framework | Governed by business and product decisions |
| Ends at a merged PR | Ends when users can access the feature |
| Timeline driven by technical scope | Timeline driven by roadmap, marketing, commercialisation |
| AI-assisted | Human-governed |

**What the delivery framework provides to support release:**

1. **Feature switch** — every user-visible feature is delivered behind a
   switch. Release controls when the switch is enabled, for which tenants,
   and under what conditions.

2. **Delivery deadline** — the date by which a Feature must be merged to
   support the release plan. Set externally by roadmap planning, carried
   by the Feature as a constraint on prioritisation.

**What the delivery framework deliberately does not own:**

- Quarterly release planning
- Roadmap commitments
- Lead time activities (UAT, data migration, marketing, commercialisation)
- Feature switch configuration per tenant
- Release scheduling

**The cross-quarter consideration:** Features needed for a Q2 release may
need to start development in Q1. This is a planning concern — the roadmap
process sets the delivery deadline, and the delivery pipeline honours it.
The delivery framework has no opinion on quarters; it only sees the deadline.

---



| Artefact | Location | Maintained by | Purpose |
|---|---|---|---|
| `FEATURES.md` | `.ai/memory/FEATURES.md` | Human (with AI assistance) | Feature backlog and status |
| `CURRENT.md` | `.ai/tasks/CURRENT.md` | AI | Active task specification |
| `STATUS.md` | `.ai/memory/STATUS.md` | AI | Current implementation state |
| `DECISIONS.md` | `.ai/memory/DECISIONS.md` | AI (append-only) | Architecture decision log |
| `done/` | `.ai/tasks/done/` | AI | Completed task archive |
| Pull Requests | GitHub | AI (opens), Human (merges) | Implementation review and progress |

---

## Progress Visibility

Progress is visible at two levels:

**Feature level:** `FEATURES.md` shows what is planned, in progress, and done.
This is the human-facing backlog — updated by the human after each PR is merged.

**Feature level:** GitHub pull requests. Open PRs = in progress or in review.
Merged PRs = done. The PR history is the complete audit trail of what was built.

There is no sprint board, no story points, no velocity tracking.
The question "what are we building?" is answered by `FEATURES.md`.
The question "what is done?" is answered by merged PRs.

---

## What Was Deliberately Left Out

**Sprint planning:** No fixed time boxes. Work flows continuously from
Feature to implementation. Prioritisation happens in `FEATURES.md`.

**Daily standups:** The AI reports its own status in `STATUS.md` after every task.
No synchronisation meeting needed.

**Story points and estimation:** Irrelevant when implementation is autonomous.
Human effort is measured in design and review time, not implementation hours.

**Velocity:** Velocity measured implementation capacity. That constraint is gone.
What matters now is the throughput of the design-review cycle.

**Handoffs:** There are no handoffs between humans. The context layer (AGENTS.md,
context files, project memory) is the handoff mechanism — it persists across
sessions and operators.

---

## Scaling to a Team

This process scales to a team without structural change. Each team member can:
- Contribute intent (Stage 1) — no process required
- Run refinement sessions (Stage 2) — any team member with domain knowledge
- Review decompositions (Stage 3) — tech lead or architect
- Trigger dev sessions (Stage 4) — any team member
- Review PRs (Stage 5) — any qualified reviewer

The constraint at team scale is **PR review throughput**. The team should
ensure review capacity keeps pace with implementation output, or PRs pile up
and the process stalls.

The context layer (`AGENTS.md`, `.ai/context/`, project memory) is shared via
Git — every team member and every AI session operates from the same context.

---

## Continuous Improvement

The process itself is subject to continuous improvement. After every Feature is
delivered, consider:
- Did the AI understand the requirements correctly?
- Were the acceptance criteria specific enough?
- Did the context layer provide adequate guidance?
- What would have made the decomposition cleaner?

Improvements are made by updating `AGENTS.md`, the context files, or the recipes.
This is the equivalent of a retrospective — but focused on the quality of the
human-AI collaboration, not team dynamics.

---

## Status of This Document

This process was designed and documented while building the go-ocs agentic
development environment. It represents current thinking and will evolve as
the process is used in practice. Update this document when the process changes —
it should always reflect how the system actually works, not how it was originally
designed.

