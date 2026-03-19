# Design Framework

**Version:** 1.0
**Status:** Living document

This document defines the artefact chain for the design process.
Each artefact is the output of one transformation and the input into the next.
Skipping a step produces incomplete input for the step that follows.

---

## The Chain

```
Raw Idea
    ↓
Problem Statement
    ↓
Feature Definition
    ↓
MVP Scope
    ↓
Acceptance Criteria + Constraints
    ↓
Parking Lot
    ↓
FEATURES.md entry          ← handoff into AI development process
    ↓
Technical Decomposition  ← AI-owned from here
    ↓
CURRENT.md
    ↓
Pull Request
    ↓
Merged Code
```

The line between Parking Lot and FEATURES.md entry is the boundary between
human design and AI development. Everything above it is human territory.
Everything below it is AI territory.

---

## Artefact 1 — Raw Idea

**Input:** A thought, observation, problem, or opportunity. No structure required.
**Output:** A sentence or paragraph that can be handed to the design process.
**Owner:** Anyone — developer, user, stakeholder, product owner.
**Transformation question:** *"What prompted this? What would be different if it existed?"*

**Complete when:** The idea can be stated in plain language.
**Not complete when:** It is still a feeling or a vague dissatisfaction.

**Example:**
> *"Users can't see their current quota balance without calling support."*

---

## Artefact 2 — Problem Statement

**Input:** Raw Idea.
**Output:** A clear statement of the problem — not the solution.
**Owner:** Human (with AI assistance).
**Transformation question:** *"What is broken, missing, or painful right now? Who experiences it?"*

**Complete when:**
- The problem is stated without reference to a solution
- The affected user or system is identified
- The current state and the desired state are both clear

**Not complete when:** It describes a solution rather than a problem.

**Example:**
> *"Subscribers currently have no self-service way to check their remaining quota.
> They must contact support, which creates unnecessary load and degrades
> subscriber experience. The desired state: a subscriber can see their
> balance without human intervention."*

**Watch for:** Jumping straight from Raw Idea to solution. The problem statement
forces clarity about what is actually wrong before any solution is considered.

---

## Artefact 3 — Feature Definition

**Input:** Problem Statement.
**Output:** A named, bounded Feature with a clear goal.
**Owner:** Human (with AI assistance).
**Transformation question:** *"What capability, when delivered, resolves this problem?"*

**Complete when:**
- The Feature has a name that a team member can refer to without confusion
- The goal is a single sentence describing the outcome
- It is clear whether this is a new capability, an enhancement, or a fix

**Not complete when:** The goal statement contains "and" — that's two Features.

**Example:**
> *"F-00EP-005 — Subscriber Quota Self-Service*
> Goal: Allow subscribers to view their current quota balance via the admin portal
> without contacting support."*


## Artefact 4 — MVP Scope

**Input:** Feature Definition.
**Output:** The smallest version of the Feature that delivers real value.
**Owner:** Human (AI actively pushes toward smaller).
**Transformation question:** *"What is the minimum version that makes the problem go away?"*

**Complete when:**
- Described in terms of what a user can do, not what the system does
- Small enough to fit in one PR
- Anything beyond this boundary is explicitly deferred

**Not complete when:** It still contains "nice to have" features or edge case handling
that isn't essential to the core value delivery.

**Example:**
> *"MVP: A subscriber can log into the admin portal and see their current
> quota balance for each unit type. Read-only. No history, no alerts,
> no adjustment capability."*

**The AI's role here:** Actively challenge scope. Ask:
*"If we removed X, would the core problem still be solved?"*
If yes, remove X.

---

## Artefact 5 — Acceptance Criteria + Constraints

**Input:** MVP Scope.
**Output:** Verifiable criteria and known boundaries.
**Owner:** Human (with AI assistance).
**Transformation questions:**
- *"How will we know the implementation is correct?"*
- *"What must always be true? What must never happen?"*
- *"What are the non-negotiable constraints — technical, regulatory, business?"*

### The Solution Bias Problem

This is the most common failure point in the design process. People naturally
think in terms of solutions — and acceptance criteria written from a solution
mindset describe *how* something should be built, not *what* it should do.

**Solution criteria (wrong):**
> *"Must use the QuotaManager.GetBalance method"*
> *"The UI must show a table with columns: Unit Type, Total Balance, Available Balance"*
> *"Must return JSON in this format: {...}"*

These specify implementation. The AI will follow them literally — including
when a better approach exists. They foreclose good solutions before the AI
has a chance to find them.

**Outcome criteria (right):**
> *"A subscriber can see their current balance for each unit type they have quota for"*
> *"The balance reflects the state after the most recent quota operation"*
> *"A subscriber without an active session cannot access the balance"*

These specify what must be true. The AI decides how to make them true.

### The AI's Active Role in Criteria Quality

The AI must actively counter solution bias during the design conversation.
This is not optional — it is a core responsibility of the design session.

**Step 1 — Detect solution thinking**
Flag any criterion that:
- Names a specific component, method, class, or endpoint
- Specifies a data format, schema, table layout, or UI structure
- Describes what the system does rather than what the user achieves

**Step 2 — Challenge with curiosity, not correction**
Do not say: *"That is a solution criterion, please rewrite it."*
Instead ask: *"What would a user be able to do when this works correctly?"*

The question produces the outcome criterion naturally. The user answers it
and arrives at the right form themselves, without feeling corrected.

**Step 3 — Test whether it is actually a constraint**
Ask: *"Is there a reason it must be done this specific way, or is this
how you imagined it?"*

- If there is a real reason (regulatory, integration dependency, existing
  contract, technical boundary) — it is a legitimate constraint.
  Move it to the Constraints section, not Acceptance Criteria.
- If there is no real reason — it is solution bias. Rewrite it as an outcome.

**Step 4 — Respect the exception**
If the human insists the how is important after the challenge — that is their
decision to make. The AI records it as a noted solution criterion with the
stated reason, and moves on. The exception rule applies. The AI does not break
the rule silently, but the human can invoke the exception explicitly.

**The distinction in practice:**

| Criterion | Type | Action |
|---|---|---|
| "Must use QuotaManager.GetBalance" | Solution — no stated reason | Challenge → rewrite to outcome |
| "Must use existing Keycloak auth" | Constraint — integration boundary | Keep in Constraints section |
| "Table must have columns X, Y, Z" | Solution — UI preference | Challenge → what does user need to see? |
| "Response time under 500ms" | Measurable outcome | Keep as acceptance criterion |
| "Must not expose raw quota IDs" | Security constraint | Keep in Constraints section |



> *"If someone deleted all the code and implemented this from scratch,
> would this criterion still tell them whether they succeeded?"*

If no — it describes a specific implementation. Rewrite it.

### The Rewrite Rule

```
Solution criterion:  "[system/component] must [do something technical]"

Outcome criterion:   "[user/role] can [achieve something observable]"
                  OR "[condition] results in [observable outcome]"
                  OR "[thing] must always/never be [measurable state]"
```

### Acceptance Criteria Are Complete When

- Each criterion can be verified by a third party without asking questions
- They cover: the happy path, the failure path, and boundary conditions
- None of them specify how — only what
- Two developers reading them independently would know when they'd succeeded

### Constraints Are Complete When

- Performance requirements are stated (if relevant)
- Security considerations are noted
- Integration boundaries are identified
- Regulatory or compliance requirements are captured

**Example:**
> *Acceptance criteria:*
> *- Authenticated subscriber can view current balance per unit type*
> *- Balance reflects the state as of the last quota operation*
> *- Unauthenticated access is rejected*
> *- Display is read-only — no modification capability*
>
> *Constraints:*
> *- Must use existing Keycloak authentication*
> *- Balance sourced from existing quota store — no new data store*
> *- Response time under 500ms at p95*

**The stress point:** A Feature with clear scope but vague or solution-biased
acceptance criteria produces an implementation that cannot be validated.
Both the what and the how-to-verify must be complete — and the criteria
must describe outcomes, not solutions.


## Artefact 6 — Parking Lot

**Input:** Everything that came up during Artefacts 1–5 but is out of scope.
**Output:** A structured list of deferred ideas.
**Owner:** AI captures during the conversation, human approves.
**Transformation question:** *"What good ideas came up that we are deliberately not building now?"*

**Complete when:**
- Every idea that was considered and deferred is captured
- Each entry has a brief note on why it was deferred
- The parking lot has been reviewed and nothing essential was accidentally parked

**Example:**
> *Parking Lot:*
> *- Quota balance history / trends — deferred, not essential to MVP*
> *- Alert when balance drops below threshold — deferred, separate Feature*
> *- Subscriber ability to request quota increase — deferred, requires approval workflow*
> *- Balance visible in mobile app — deferred, mobile not in scope*

**The AI's role here:** At the end of the design session, review the parking lot
with the human and ask: *"Should any of these become the next Feature?"*
The parking lot seeds the roadmap.

---

## Artefact 7 — FEATURES.md Entry

**Input:** Artefacts 1–6 (all of the above, structured).
**Output:** A single entry in `.ai/memory/FEATURES.md`.
**Owner:** AI writes, human approves.
**Transformation:** The AI synthesises all previous artefacts into the standard Feature format.

**Complete when:** A competent developer could implement this Feature from the
FEATURES.md entry alone, without asking a single clarifying question.

**This is the handoff point.** Once approved, the Feature enters the AI development
process. Human design is done.

**Standard format:**

```markdown
## F-NNN — [Title]

**Status:** Ready for AI Design
**Priority:** High / Medium / Low
**Created:** YYYY-MM-DD

### Goal
One sentence. What this builds and why.

### Problem Statement
What is broken or missing. Who is affected.

### MVP
The smallest version that delivers real value.
What a user can do when this is complete.

### Acceptance Criteria
- [ ] Verifiable criterion (third party can check without asking questions)
- [ ] Failure path criterion
- [ ] Boundary condition criterion

### Constraints
Technical, regulatory, business, or performance constraints.

### Out of Scope
What is explicitly not included in this Feature.

### Parking Lot
- [Idea]: [why deferred]

### Future Considerations
Architectural decisions this Feature must not foreclose.
```

---

## The Handoff Checkpoint

Before a Feature moves from human design (Artefact 7) to AI design (Technical
Decomposition), apply this test:

**The three-question checkpoint:**
1. Can the AI implement this without inventing requirements?
2. Would you recognise the correct implementation when you see it in a PR?
3. Is there anything that would make you reject the PR regardless of quality?

If the answers are yes, yes, no — the Feature is ready.
If any answer fails — identify which artefact is incomplete and return to it.


## AI-Owned Artefacts (For Reference)

These artefacts are produced by the AI development process. They are included
here for completeness of the chain — they are not part of the human design process.

**Technical Decomposition** — AI reads the Feature and the codebase, breaks the
Feature into implementable features, identifies dependencies and sequencing.
Output reviewed by human before implementation begins.

**CURRENT.md** — AI-produced task specification for each feature. Scope, context,
decisions made, acceptance criteria mapped to implementation tasks.
One CURRENT.md per feature branch.

**Pull Request** — AI-produced implementation against CURRENT.md.
Human validates against the Feature's acceptance criteria.

---

## Common Design Failures and Their Root Cause

| Symptom | Root cause | Return to |
|---|---|---|
| AI implemented something technically correct but wrong | Problem statement was unclear | Artefact 2 |
| Feature was too large to implement cleanly | MVP scope was not defined | Artefact 4 |
| PR was built correctly but can't be verified | Acceptance criteria were missing | Artefact 5 |
| Good ideas keep creeping back into scope | Parking lot was not maintained | Artefact 6 |
| AI decomposition doesn't match intent | Feature definition was ambiguous | Artefact 3 |
| Implementation went in the wrong direction | Constraints were unstated | Artefact 5 |

When a PR is rejected, use this table to identify which artefact needs
improvement before re-running the design or implementation.

---

## The Design Session Time Box

The entire human design process (Artefacts 1–7) should take 15–30 minutes.

| Artefact | Time |
|---|---|
| Raw Idea → Problem Statement | 2–3 min |
| Problem Statement → Feature Definition | 3–5 min |
| Feature Definition → MVP Scope | 5–8 min |
| MVP Scope → Acceptance Criteria + Constraints | 5–8 min |
| Parking Lot + FEATURES.md entry | 3–5 min |
| **Total** | **18–29 min** |

If any single step is taking significantly longer, the idea is not ready.
Park it and return when thinking is clearer.

---

## Status of This Document

Defined during the design of the go-ocs agentic development framework.
Update when new patterns emerge from practice.
Read alongside SDLC_PROCESS.md and AI_DECISION_FRAMEWORK.md.
