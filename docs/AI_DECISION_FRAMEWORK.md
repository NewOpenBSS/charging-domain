# AI Development Decision Framework

**Version:** 1.0
**Status:** Living document

This document defines the decision-making principles for the agentic development
process. Where SDLC_PROCESS.md describes *what* the process is, this document
describes *how to think* when making decisions within it.

These principles apply to both humans and AI agents operating in the process.

---

## The Fundamental Rule

> **When in doubt, build less, learn fast, adjust.**

Every other principle in this document flows from this one.

---

## Section 1 — Scope Decisions

### 1.1 Always define the MVP first
Before any other design work, define the smallest version of the feature that
delivers real, demonstrable value. If you cannot define it, the idea is not
ready. Everything beyond the MVP is a future Feature.

### 1.2 When scope is uncertain, shrink it
If there is genuine uncertainty about what to include, the decision is always
to build less. The cost of adding later is lower than the cost of building
the wrong thing now.

### 1.3 When in doubt, park it
Ideas that emerge during design but are not essential to the MVP go to the
Parking Lot. They are not discarded — they are deferred. A good idea at the
wrong time is still a good idea.

### 1.4 One Feature, one PR
A Feature should produce work that fits in a single pull request that a human
can review in a single sitting. If it doesn't, split the Feature.


---

## Section 2 — Design Decisions

### 2.1 Human design answers what and why — AI design answers how
These are not interchangeable. The human defines the problem, the constraints,
and the success criteria. The AI defines the technical solution. If the human
is specifying implementation detail, they are working at the wrong level.
If the AI is making business decisions, it is operating outside its authority.

### 2.2 The quality of human design determines the quality of AI output
A vague Feature produces a technically correct but misaligned implementation.
A precise Feature produces exactly what was intended. When AI output is wrong,
look first at the Feature — not the AI.

### 2.3 Good enough to start beats perfect before starting
The design only needs to be complete enough that the AI cannot misinterpret
the intent. It does not need to anticipate every edge case. Edge cases that
matter will surface during implementation or review — and can be addressed
then, with real evidence rather than speculation.

### 2.4 The 30-minute rule
If a design conversation has not produced a clear Feature in 30 minutes, the
idea is not ready. Stop. Park it. Return when thinking is clearer.
Continuing past 30 minutes produces diminishing returns and risks
analysis paralysis.

### 2.5 Acceptance criteria must describe outcomes, not solutions

The most common design failure is acceptance criteria that specify *how* something
should be built rather than *what* it should do. This is called solution bias.

During the design session, the AI must actively challenge any criterion that:
- Names a specific method, class, or component
- Specifies a data format, schema, or structure
- Describes what the code does rather than what the user can do

**The rewrite rule:**
- Wrong: *"[system] must [do something technical]"*
- Right: *"[user] can [achieve something observable]"*
- Right: *"[condition] results in [observable outcome]"*
- Right: *"[thing] must always/never be [measurable state]"*

**The forcing question the AI asks for each criterion:**
*"If someone deleted all the code and started from scratch, would this criterion
tell them whether they'd succeeded?"* If no — rewrite it.


When the AI's technical decomposition doesn't match your expectations, the
first question is: *"Did the Feature fully describe the need?"* In most cases,
the answer is no. Update the Feature and re-run the design. If the Feature was
complete and the AI design is genuinely wrong, that is feedback for improving
the design recipe — not a reason to distrust the AI.

---

## Section 3 — Implementation Decisions

### 3.1 The AI implements — the human does not intervene mid-task
Once implementation begins, the human's role is to wait for the PR.
Interrupting mid-task breaks context and creates inconsistent state.
If the implementation is going wrong, let it finish, reject the PR,
and provide clear feedback for the next run.

### 3.2 Small incremental changes — always
Implement the simplest solution that satisfies the MVP. No more.
The AI will implement whatever is asked — it is the human's responsibility
to ask for the right-sized thing.

### 3.3 Refactor when the need is real, not anticipated
Do not design for future requirements that have not been stated.
When future Features reveal that the current design needs to change,
refactor then — with full context of the actual need. Premature
refactoring is waste disguised as prudence.

### 3.4 Build verification is non-negotiable
The build must pass. Tests must pass. This is not optional and is
not subject to time pressure. A failing build is not a completed task.


---

## Section 4 — Review Decisions

### 4.1 Review against the Feature, not against preference
The PR review question is: *"Does this satisfy the Feature's success criteria?"*
Not: *"Is this how I would have done it?"* Technical style differences that
don't affect correctness, performance, or maintainability are not rejection
reasons.

### 4.2 Clear feedback on rejection
If a PR is rejected, the feedback must be specific enough that the AI can
act on it without asking clarifying questions. Vague feedback produces
vague corrections.

### 4.3 When to reject vs. when to merge and create a new Feature
Reject a PR when: it does not satisfy the success criteria, or it introduces
a defect, or it creates technical debt that would be immediately expensive.
Create a new Feature when: the implementation is correct but you want something
additional or different. Do not use PR rejection to add scope.

### 4.4 The merge is the acceptance
Merging a PR is the human's statement that the work meets the standard.
If it doesn't, don't merge. There is no "merge and fix later" — that
creates invisible debt and erodes the quality signal of the PR process.

---

## Section 5 — Roadmap Decisions

### 5.1 Know the destination — not the full route
The roadmap is the destination. The Features are the next step.
You do not need to design step 4 before taking step 1.
You only need to know what step 1 must not prevent.

### 5.2 The Parking Lot is the roadmap seed
Review the Parking Lot after every merged Feature.
The natural next Feature often comes from there — built from real experience
of the current implementation, not speculation about future needs.

### 5.3 Architectural decisions that foreclose future options are red flags
When reviewing a design or a PR, ask: *"Does this make the next step harder?"*
If yes, it needs to be flagged and discussed before merging — not after.
This is the one forward-looking question that is always worth asking.

### 5.4 Prioritise by what unblocks next
When choosing which Feature to work on, prefer the one that unblocks the most
subsequent work. Build foundations before features that depend on them.

---

## Section 6 — Process Decisions

### 6.1 Follow the process — including yourself
These principles apply to humans as much as to AI agents. The human is not
exempt from the MVP rule, the 30-minute rule, or the small incremental
changes principle. The process works because everyone follows it.

### 6.2 When the process creates friction, improve it — don't bypass it
If a step in the process feels wrong or burdensome, the right response is
to update SDLC_PROCESS.md or the recipes — not to skip the step.
Bypassing the process erodes it silently.

### 6.3 The context layer is shared infrastructure
AGENTS.md, context files, and project memory are not one person's
responsibility. When something goes wrong because the context was missing
or wrong, fixing the context is as important as fixing the output.

### 6.4 Retrospect on the process, not just the output
After each Feature is delivered, the question is not only *"was the output correct?"*
but *"did the process work well?"* Where did the design need revision?
Where did the AI misinterpret? What would have made the review faster?
These answers improve the process for the next Feature.

---

## Quick Reference — Decision Rules

| Situation | Decision rule |
|---|---|
| Scope is unclear | Shrink it — build the smaller version |
| Design conversation exceeds 30 minutes | Stop — idea isn't ready |
| Good idea outside current scope | Park it — don't build it now |
| AI design doesn't match expectation | Check the Feature first |
| Tempted to add to the PR mid-review | Create a new Feature instead |
| Tempted to design for future needs | Don't — refactor when the need is real |
| Process feels burdensome | Improve the process, don't bypass it |
| PR fails success criteria | Reject with specific feedback |
| PR meets criteria but could be better | Merge and create improvement Feature |

---

## Section 7 — Exceptions

### 7.1 Every rule has an exception — but no exception is silent

You can break any rule in this framework. You cannot break a rule silently.

When deviating from a principle, three things must happen:
1. **Name the rule** being broken
2. **State the reason** — what makes this situation genuinely different
3. **Record it** — append an entry to `.ai/memory/DECISIONS.md`

If you cannot articulate the reason clearly enough to write it down,
the deviation is not justified.

### 7.2 Three types of exceptions

**Situational exception**
The rule is right in general but wrong for this specific case.
Deviate, record why, continue. The rule stands for future cases.

*Example: "One Feature, one PR" — but this refactor cannot be split without
creating a broken intermediate state.*

**Process learning**
The rule is producing consistently bad outcomes. Reality has shown the
rule needs adjusting.
Deviate, record why, then update this framework or SDLC_PROCESS.md.
A rule that keeps being broken for the same reason is a wrong rule.

*Example: "30-minute design time box" — but complex domain problems
consistently need 45 minutes to reach clarity.*

**Emergency**
Something critical requires bypassing the normal process entirely.
Deviate, record why, then conduct a retrospective to assess whether
the process needs an expedited path for future emergencies.

*Example: Critical production defect needs immediate fix without full
Feature refinement.*

### 7.3 Exceptions create learning — if recorded

An unrecorded exception is waste. A recorded exception is data.
Patterns in the DECISIONS.md exception log reveal where the framework
needs to evolve. Review exception entries during continuous improvement
cycles and update the framework accordingly.

### 7.4 The meta-rule

> **You can break any rule. You cannot break a rule silently.**

This is the one rule that has no exception.

---

## Status of This Document

Defined during the design of the go-ocs agentic development framework.
Update when new decision patterns emerge from practice.
This document and SDLC_PROCESS.md are read together — neither is complete
without the other.
