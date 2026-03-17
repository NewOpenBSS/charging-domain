AI Guidance – Development Rules

This document defines how AI assistants should behave when working in this repository.

AI tools must follow these rules to ensure safe and predictable development.

⸻

Session Initialization

Before performing any task in this repository:

1. Read `PROJECT_BRIEF.md` to understand the project purpose and domain context.
2. Read `ARCHITECTURE.md` to understand system structure and architectural boundaries.
3. Read `AI_GUIDANCE.md` to understand the development rules for this repository.
4. Inspect the repository structure and relevant files before making assumptions about behaviour or architecture.

Never assume project structure or behaviour without inspecting the codebase first.

⸻

General Working Style

When performing tasks:
• Analyse the problem before modifying code.
• Prefer small, incremental changes.
• Explain reasoning and assumptions clearly.
• Surface ambiguity rather than guessing missing requirements.
• When uncertain, ask for clarification instead of inventing behaviour.

The goal is correctness and maintainability, not cleverness.

⸻

Tool Usage Rules

Before invoking any tool:

1. Inspect the tool metadata.
2. Use only the parameters defined in the tool schema.
3. Do not invent tool arguments.
4. If a tool call fails, re-read the tool schema and retry.

Never assume parameter names or payload structure.

⸻

Code Modification Rules

When editing code:
• Modify the smallest amount of code necessary.
• Preserve existing project conventions.
• Avoid unnecessary abstractions.
• Do not introduce new frameworks or patterns unless explicitly requested.
• Do not refactor unrelated code during a task.

Large refactors must be proposed and approved before implementation.

⸻

Architectural Rules

Follow the architectural boundaries defined in ARCHITECTURE.md.

In particular:
• Keep transport layers thin.
• Place business logic in service-layer components.
• Keep domain logic separate from infrastructure code.
• Kafka consumers should delegate business logic to services rather than implement it directly.

Do not introduce architecture that conflicts with the existing design.

⸻

Domain Safety Rules

This system contains charging and quota logic that must remain correct.

The following rules must always be respected:
• Charging behaviour must remain deterministic.
• Quota counters must never become negative.
• Duplicate or replayed events must not cause double charging.
• Billing semantics must never be invented or inferred.

If a change could affect these areas, clearly explain the risk.

⸻

Validation Requirements

After modifying code:
• Run relevant tests where possible.
• If tests are unavailable, explicitly state that validation could not be performed.
• Describe what behaviour was validated.

Never claim functionality works without verification.

Go Project Build Verification

For any task that modifies Go code, imports, or dependencies, the assistant must verify that the project builds correctly before reporting success.

Required verification steps:
• Run `go mod tidy` to ensure dependencies are resolved and the module file is consistent.
• If `go mod tidy` reports an import path, module path, or version error, investigate the cause and attempt to resolve it before claiming success.
• When dependency resolution fails, the assistant must determine a strategy to fix the issue rather than stopping at the first error.
• The assistant may resolve the issue by correcting import paths, adjusting module versions, replacing an invalid package, or applying another
appropriate fix supported by evidence.
• If internet search capability is available, use it to investigate the error and look for authoritative guidance before retrying dependency
resolution.
• After resolving dependency issues, rerun `go mod tidy` until it succeeds or clearly report why it could not be made to succeed.
• Run `go build ./...` to verify the entire repository compiles.
• Run `go test ./...` where tests exist.

Additional rules:
• Never claim the application is "working" unless the build step completed successfully.
• If `go mod tidy` fails, do not simply repeat the command without analysis.
• Investigate dependency errors and choose a reasonable solution strategy based on the evidence available.
• If internet access is available, search for the specific error, package, or module involved and use authoritative sources to guide the fix.
• Do not invent module paths, package paths, or versions.
• If any command fails, report the exact command and the error output.
• If build or tests cannot be executed, explicitly state that verification could not be performed and explain why.

These steps are mandatory for all Go-related changes to prevent unverified code from being reported as complete.

⸻

Go File Testing Workflow

For any task involving Go implementation work, the assistant must follow this file-by-file workflow before claiming success:

1. Identify each Go source file that contains executable functions or methods.
2. Ignore files that only declare structs, constants, types, or interfaces and contain no functions or methods requiring behavioural verification.
3. For each relevant Go file, create or extend unit tests that cover the expected behaviour of the functions in that file.
4. The generated tests must cover the meaningful scenarios the code is expected to handle, including success cases, failure cases, edge cases, and any
   important branching behaviour.
5. Run the unit tests for that file or package immediately after creating or modifying the tests.
6. If any test fails, or if the code does not compile, stop progressing to the next Go file and fix the issue first.
7. Only continue to the next Go file once the current file’s tests pass.
8. Do not leave a file partially implemented, partially tested, or with failing tests while claiming the task is complete.

The assistant must not claim that code works merely because tests were written. The tests must be executed and must pass.

After all relevant Go files have passing unit tests:
• verify that the full application builds cleanly
• verify that the application can start where startup verification is reasonably possible
• investigate and fix any build, startup, or dependency issues before reporting success
• repeat the verification and fix cycle until there is a clean build

Never report a Go implementation task as complete while tests are failing, while files remain partially verified, or while the application still has
unresolved build issues.

⸻

File and Repository Safety

AI assistants may:
• read any repository file
• propose design changes
• implement small scoped changes when requested

AI assistants must ask before:
• deleting files
• performing broad refactors
• changing public APIs
• modifying core charging logic
• introducing new dependencies

⸻

Communication Style

When reporting results:
• clearly explain what changed
• reference specific files or packages
• explain reasoning behind design decisions
• highlight potential risks

Prefer clarity over brevity.

⸻

Branching and Git Workflow
• Never make changes directly on the main branch.
• Always create a new branch for each feature or task.
• Branch names should clearly reflect the purpose of the change.
• Add all the files that are part of the change to the branch.
• Once a feature is completed, commit the changes to git and create a pull request.
• Never merge a pull request. Pull requests must be reviewed and merged by a human developer.

⸻

Testing and Documentation Requirements
• All public functions must include a short comment explaining the purpose of the function.
• All Go source files created that contain functions or methods must include accompanying unit tests.
• When modifying existing Go files that contain functions or methods, add or update unit tests so the changed behaviour is verified.

⸻

Dependency Management
• Prefer libraries that are already used in the project.
• Introduce new dependencies only when there is a clear and justified reason.

Import and Dependency Verification
• When introducing a new third-party import, verify that the module path exists and is correct.
• Check authoritative sources such as pkg.go.dev, proxy.golang.org, or the upstream repository to confirm the correct module path and current
recommended version.
• If internet search capability is available, use it to confirm the latest stable version of the dependency before adding it.
• Do not assume import paths or module versions.
• If internet access is not available, explicitly state that dependency verification could not be performed.

This verification must occur before reporting that a change involving new imports or dependencies is complete.

⸻

Application Consistency
• Any new application created must follow the same general structure and patterns used in charging-dra and charging-engine.

⸻

Goal

The goal of AI assistance in this repository is to support careful, maintainable development while respecting the architecture and domain constraints
of the system.
