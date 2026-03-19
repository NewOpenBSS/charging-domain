# AI Guidance — Development Rules

This document defines the mandatory rules for AI assistants working in this repository.
All rules apply unless explicitly overridden by a human developer for a specific task.

Rule priority is expressed using RFC 2119 keywords: **MUST** (mandatory), **MUST NOT** (prohibited), **SHOULD** (strongly recommended), **MAY** (permitted but optional).

---

## 1. Session Initialization

Before performing any task, read the following files in order:

1. `PROJECT_BRIEF.md` — project purpose and domain context
2. `ARCHITECTURE.md` — system structure and architectural boundaries
3. `AI_GUIDANCE.md` — this document

**Rule 1.1** — The assistant MUST NOT make assumptions about project structure, behaviour, or architecture without first inspecting the relevant source files.

**Rule 1.2** — If the task involves a specific package, the assistant MUST read the relevant source files in that package before modifying any code.

---

## 2. Working Principles

**Rule 2.1** — Analyse the full problem before modifying any code.

**Rule 2.2** — Prefer small, incremental changes over large rewrites.

**Rule 2.3** — When requirements are ambiguous or missing, ask for clarification rather than inventing behaviour.

**Rule 2.4** — Correctness and maintainability take precedence over cleverness or elegance.

**Rule 2.5** — Do not make changes outside the scope of the requested task.

---

## 3. Tool Usage

**Rule 3.1** — Before invoking any tool, inspect its schema and use only the parameters defined in that schema.

**Rule 3.2** — Do not invent tool arguments, parameter names, or payload structures.

**Rule 3.3** — If a tool call fails, re-read the tool schema before retrying. Do not retry without analysis.

---

## 4. Code Modification

**Rule 4.1** — Modify the minimum amount of code necessary to fulfil the task.

**Rule 4.2** — Preserve existing project conventions and code style.

**Rule 4.3** — Do not introduce new frameworks, patterns, or libraries unless explicitly requested.

**Rule 4.4** — Do not refactor unrelated code while implementing a task.

**Rule 4.5** — Large refactors MUST be proposed and explicitly approved by a human developer before implementation begins.

---

## 5. Go Coding Standards

**Rule 5.1 — Context propagation**
Every function that performs I/O (database, Kafka, HTTP, or any downstream call) MUST accept `context.Context` as its first parameter and propagate it to all downstream calls. `context.Background()` MUST NOT be used inside business logic — it is only permitted at the top of a request entry point (e.g., HTTP handler, Kafka consumer). Context MUST NOT be stored in structs.

**Rule 5.2 — Nil safety**
All pointer, interface, slice, and map values returned from functions MUST be checked for nil before use. Do not assume a returned value is non-nil without explicit documentation stating so.

**Rule 5.3 — Panic prevention**
`panic` MUST NOT be used in business logic or service code. It is only permitted in `main()` for startup validation (e.g., config parsing failures). Service entry points (HTTP handlers, Kafka consumers) MUST recover from unexpected panics and return a safe, controlled error response rather than crashing the process.

**Rule 5.4 — Interface design**
Interfaces MUST be defined at the point of consumption (the calling package), not at the point of implementation. Interfaces SHOULD be small and focused — prefer single-method or narrow interfaces over large, monolithic ones. Function signatures SHOULD accept interfaces and return concrete types.

**Rule 5.5 — Struct initialisation**
Struct literals MUST always use named fields. Positional struct initialisation is prohibited. This applies to both production code and test code. Named fields make code resilient to struct additions and make intent explicit.

```go
// Correct
foo := MyStruct{
    Field1: "value",
    Field2: 42,
}

// Prohibited
foo := MyStruct{"value", 42}
```

**Rule 5.6 — Hardcoded values**
Numeric literals and string constants with business or operational meaning MUST be defined as named constants, not inlined at call sites. Timeouts, retry counts, buffer sizes, and thresholds MUST be sourced from configuration (YAML), not hardcoded.

---

## 6. Error Handling

**Rule 6.1** — Do NOT use `fmt.Errorf` or `errors.New` for domain or application errors. All application errors MUST be expressed using typed error structs defined within the relevant package.

**Rule 6.2** — Each package that defines domain errors MUST declare:
- A `Code` type (string-based) with named constants for each distinct error condition.
- A typed error struct that implements the `error` interface via an `Error() string` method.
- Named constructor functions (e.g., `CreateUnknownSubscriber(msg string)`) to instantiate errors — do not construct error structs inline at call sites.

**Rule 6.3** — The pattern established in `internal/chargeengine/ocserrors/errors.go` is the reference implementation. New packages MUST follow this pattern when defining their own error types.

**Rule 6.4** — Error codes MUST be meaningful, stable identifiers (e.g., `"UNKNOWN_SUBSCRIBER"`, `"OUT_OF_FUNDS"`). Do not use numeric codes or vague labels such as `"ERROR"`.

**Rule 6.5** — When a specialised error carries additional structured data (e.g., a response payload), define a separate struct for it with typed accessor methods, as demonstrated by `RetransmitError` in `ocserrors`.

**Rule 6.6** — Error type assertions (`errors.As`) MUST be used when inspecting error types at call sites. Do not inspect errors by string comparison.

**Rule 6.7** — `fmt.Errorf` MAY be used only for wrapping infrastructure-level errors (e.g., database, network, I/O) where no domain meaning is attached. All domain-meaningful errors MUST use typed error structs.

---

## 7. Concurrency and Goroutine Safety

**Rule 7.1** — All shared mutable state MUST be protected against concurrent access using an appropriate mechanism: `sync.Mutex`, `sync.RWMutex`, atomic operations, or channel-based ownership. Unprotected shared state is prohibited.

**Rule 7.2** — Goroutines MUST NOT be spawned without a clearly documented lifecycle: how and when the goroutine terminates, and how errors or panics are surfaced to the caller or supervisor.

**Rule 7.3** — Goroutines MUST NOT be leaked. Every goroutine MUST have a defined shutdown path, typically via context cancellation or a dedicated stop channel.

**Rule 7.4** — The Go race detector MUST be used when running tests for any code that involves concurrent access: `go test -race ./...`. A test suite that passes without `-race` but fails with it is not considered passing.

**Rule 7.5** — Do not use `init()` functions to initialise shared state. Package-level shared state MUST be initialised explicitly and passed as dependencies.

---

## 8. Time Handling

**Rule 8.1** — `time.Now()` MUST NOT be called directly inside business logic, domain functions, or service methods. Doing so makes behaviour non-deterministic and untestable, violating Rule 10.1 (charging determinism).

**Rule 8.2** — Current time MUST be injected as an explicit parameter or via a clock interface (e.g., `func(now func() time.Time)`) so that tests can control time without side effects.

**Rule 8.3** — All timestamps stored in the database or included in events MUST be in UTC. Use `time.Now().UTC()` at system entry points (handlers, consumers) and propagate the UTC value inward — never convert to local time inside business logic.

**Rule 8.4** — Do not compare `time.Time` values using `==`. Use `.Equal()` for equality and `.Before()` / `.After()` for ordering.

---

## 9. Architectural Rules

All architectural boundaries are defined in `ARCHITECTURE.md`. The following rules are absolute.

**Rule 9.1** — Transport layer handlers MUST be thin and delegate all business logic to service-layer components. No business logic is permitted in HTTP or Diameter handlers.

**Rule 9.2** — Business logic MUST reside in service-layer or domain-layer packages under `internal/`.

**Rule 9.3** — Domain logic MUST NOT be placed in infrastructure, transport, or persistence packages.

**Rule 9.4** — All database access MUST go through repository interfaces defined in `internal/store/`. Direct SQL in other packages is prohibited.

**Rule 9.5** — Kafka consumers MUST delegate business logic to services and MUST NOT implement it directly.

**Rule 9.6** — Event publishing MUST be isolated from core charging logic.

**Rule 9.7** — New applications MUST follow the same structural patterns used in `charging-dra` and `charging-engine`.

**Rule 9.8** — Configuration MUST be loaded from YAML. Reading configuration from environment variables in application code is prohibited.

---

## 10. Domain Safety Rules

This system processes real money. These rules are non-negotiable and override any other consideration.

**Rule 10.1** — Charging behaviour MUST be deterministic: the same inputs MUST always produce the same outputs.

**Rule 10.2** — Quota counters MUST NEVER become negative under any code path.

**Rule 10.3** — All quota operations MUST be wrapped in database transactions to ensure atomicity.

**Rule 10.4** — Duplicate or replayed events MUST NOT cause double charging. Idempotency MUST be enforced at every entry point.

**Rule 10.5** — Billing semantics MUST NEVER be invented or inferred. If requirements are unclear, stop and ask.

**Rule 10.6** — Any change that could affect charging, quota, or rating behaviour MUST include a written risk explanation before implementation begins.

**Rule 10.7 — Decimal representation of financial values**
All financial values (prices, tariffs, balances, amounts) MUST be represented as `decimal.Decimal` from the `github.com/shopspring/decimal` package. Native Go numeric types (`float32`, `float64`, `int`, `int64`) MUST NOT be used to store or compute financial values. Floating-point types are prohibited due to inherent binary precision loss that produces incorrect monetary results.

**Rule 10.8 — Decimal precision must come from configuration**
All rounding and division operations on financial values MUST use the `DecimalDigits` precision value read from the application configuration. The precision MUST NEVER be hardcoded as a literal (e.g., `.Round(22)`) anywhere in business logic. Always read and pass the configured value explicitly, as demonstrated in `internal/chargeengine/engine/steps/rating-step.go`:
```go
decimalDigits := dc.AppContext.Config.Engine.DecimalDigits
unitPrice = rateLine.BaseTariff.DivRound(unitOfMeasure, decimalDigits)
```

**Rule 10.9 — DecimalDigits configuration field**
Every application that handles financial values MUST define a `DecimalDigits int32` field in its configuration struct. The default value MUST be set to `22` in the `NewConfig` constructor when the YAML configuration does not supply a value. The reference implementation is `EngineConfig.DecimalDigits` in `internal/chargeengine/appcontext/config.go`.

**Rule 10.10 — Decimal precision scope**
The `DecimalDigits` value MUST be propagated from the application config down through the call chain to every function that performs financial arithmetic. Functions that perform rounding or division on financial values MUST accept `decimalDigits int32` as an explicit parameter rather than accessing config directly, keeping domain functions testable and config-independent.

---

## 11. Sensitive Data Handling

This system processes subscriber identity and financial transaction data. Accidental exposure in logs or error messages constitutes a compliance and security risk.

**Rule 11.1** — Subscriber identifiers, account balances, transaction amounts, and rate plan details MUST NOT be written to log output at `DEBUG` level or above in production configuration.

**Rule 11.2** — Internal state, raw database errors, stack traces, and query details MUST NOT be included in error responses returned to external callers (HTTP responses, Diameter answers). Return a sanitised error code and message only.

**Rule 11.3** — Credentials, API keys, and connection strings MUST NOT appear anywhere in source code, log output, or error messages. These values MUST be sourced exclusively from the YAML configuration files referenced in Rule 9.8.

**Rule 11.4** — When constructing typed error structs (see Section 6), ensure the `Message` field contains only safe, non-sensitive context. Do not include raw data values in error messages that may be propagated to logs or external responses.

---

## 12. Go Build Verification

These steps are mandatory after any change to Go source files, imports, or dependencies. The assistant MUST execute these commands and report results.

**Rule 12.1** — Run `go mod tidy` to resolve and verify module consistency.

**Rule 12.2** — Run `go build ./...` to verify the entire repository compiles cleanly.

**Rule 12.3** — Run `go test ./...` (or the targeted package) to verify tests pass.

**Rule 12.4** — NEVER claim an implementation is complete unless all three verification steps above have executed successfully.

**Rule 12.5** — If any verification command fails, diagnose the root cause before retrying. Do not repeat a failing command without analysis.

**Rule 12.6** — Report the exact command and full error output whenever a verification step fails.

**Rule 12.7** — If verification steps cannot be executed (e.g., no build environment), state this explicitly and explain why.

---

## 13. Go Testing Requirements

**Rule 13.1** — Every Go source file containing functions or methods MUST have an accompanying `_test.go` file.

**Rule 13.2** — When modifying an existing Go file, add or update tests so that all changed behaviour is covered.

**Rule 13.3** — Tests MUST cover: success cases, failure cases, edge cases, and significant branching paths.

**Rule 13.4** — Tests MUST be executed and MUST pass. Writing tests without running them does not satisfy this rule.

**Rule 13.5** — If a test fails, fix the issue before moving to the next file or claiming the task is complete.

**Rule 13.6** — Do not use deprecated functions or methods. If a deprecated call is genuinely unavoidable, document the reason and seek human approval before proceeding.

**Rule 13.7** — Files that only declare structs, constants, types, or interfaces with no executable logic are exempt from Rule 13.1.

**Rule 13.8** — Unit tests MUST NOT require running external services (PostgreSQL, Kafka). Tests that require external services MUST be isolated using build tags or a naming convention agreed with the team.

**Rule 13.9 — Table-driven tests**
Functions with multiple input/output combinations MUST use table-driven tests. Define a slice of anonymous structs, each with a `name` field, the relevant inputs, and the expected outputs. Use `t.Run(tc.name, ...)` for each case so that failures identify the specific scenario by name.

```go
tests := []struct {
    name     string
    input    SomeType
    expected SomeResult
}{
    {name: "zero value returns default", input: SomeType{}, expected: defaultResult},
    {name: "negative amount returns error", input: SomeType{Amount: -1}, expected: errResult},
}
for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
        result := FunctionUnderTest(tc.input)
        // assert result == tc.expected
    })
}
```

**Rule 13.10 — Test naming**
Test function names MUST follow the pattern `TestFunctionName_Scenario_ExpectedBehaviour` (e.g., `TestDebitQuota_InsufficientBalance_ReturnsOutOfFunds`). This makes failure output self-explanatory without needing to read the test body.

**Rule 13.11** — Run the race detector when testing any code involving concurrency: `go test -race ./...`. A passing test suite without `-race` that fails with it is not considered passing.

---

## 14. Dependency Management

**Rule 14.1** — Prefer libraries already used in the project over introducing new ones.

**Rule 14.2** — Introduce a new dependency only when there is a clear and justified reason that cannot be satisfied by existing dependencies.

**Rule 14.3** — Before adding any new import, verify the module path exists and is correct on `pkg.go.dev` or `proxy.golang.org`. Do not assume import paths or module versions without verification.

**Rule 14.4** — If internet access is unavailable during dependency verification, explicitly state that verification was not performed.

**Rule 14.5** — Files generated by tools such as `sqlc` (marked `// Code generated ... DO NOT EDIT`) MUST NOT be manually modified. Re-run the generator to update them.

---

## 15. Database Migration Safety

**Rule 15.1** — Every migration MUST be reversible. Each `up` migration MUST have a corresponding `down` migration that fully restores the previous schema state.

**Rule 15.2** — Migrations MUST be backward compatible with the currently deployed application code. A migration that would break the running application before a new deployment is complete is prohibited.

**Rule 15.3** — Columns, tables, or indexes that are still referenced in active application code MUST NOT be dropped. Follow a two-phase approach: first deprecate (stop writing to the column), then remove in a subsequent migration after the code change is deployed.

**Rule 15.4** — Migrations MUST be applied and verified against a local copy of the current schema using `make migrate-up` before being committed. Never commit an untested migration.

**Rule 15.5** — Data migrations (transforming existing rows) MUST be proposed to a human developer before implementation. The risk to existing live data must be explicitly stated.

---

## 16. Documentation Requirements

**Rule 16.1** — All public functions and methods MUST have a Go doc comment explaining their purpose.

**Rule 16.2** — Comments MUST describe what the function does and any non-obvious behaviour. Comments MUST NOT merely restate the code.

---

## 17. File and Repository Safety

AI assistants MAY:
- Read any repository file
- Propose design changes
- Implement small, scoped changes when requested

AI assistants MUST ask a human developer before:
- Deleting any file
- Performing broad refactors
- Changing public APIs
- Modifying core charging, quota, or rating logic
- Introducing new dependencies

---

## 18. Git and Branching Workflow

**Rule 18.1** — NEVER make any code changes while on the `master` or `main` branch. This is unconditional — no exceptions.

**Rule 18.2** — Before making any changes, the assistant MUST check the current branch using `git branch --show-current`.

- If on `master` or `main`: stop immediately and inform the user. Do not proceed until the user explicitly names a branch to work on.
- If already on a feature branch: ask the user — *"You are currently on branch `<branch-name>`. Do you want me to continue making changes on this branch?"* — and wait for explicit confirmation before proceeding.

**Rule 18.3** — If a new branch is required and the current branch has uncommitted changes, the assistant MUST NOT create the new branch over a dirty working tree. The assistant MUST:
1. Inform the user that uncommitted changes exist and list them.
2. Ask the user whether to **commit** or **stash** the changes before proceeding.
3. Execute the chosen action (`git commit` or `git stash`) to clean the working tree.
4. Only then create the new branch from the clean state.

**Rule 18.4** — All code generation for a task MUST occur in the same workspace as the branch being worked on. Do not switch worktrees or directories mid-task.

**Rule 18.5** — Branch names MUST clearly reflect the purpose of the change (e.g., `feature/add-quota-tax`, `fix/double-charge-idempotency`).

**Rule 18.6** — When staging files for a commit, the assistant MUST actively identify and stage new files introduced as part of the change. Files produced by code generators (e.g., `sqlc`, `protoc`, `mockgen`) MUST be excluded from staging unless they qualify as one-off generated code under Rule 18.7.

**Rule 18.7** — Before staging, verify that all continuously-generated output directories and files are listed in `.gitignore`. The exception is one-off generated code — a bootstrap or scaffold that the developer will hand-maintain from that point forward. Such files are no longer "generated" in the ongoing sense, are not expected to be in `.gitignore`, and MUST be staged and committed as regular source files.

**Rule 18.8** — If a file is explicitly excluded by `.gitignore`, the assistant MUST NOT stage it, force-add it (`git add -f`), or silently skip it. Instead, the assistant MUST:
1. Inform the human that the file is gitignored and explain why it appears relevant to the current change.
2. Ask whether the file should be included in the repository.
3. If the human confirms it should be committed, update `.gitignore` with a specific negation exception (e.g., `!path/to/file`) rather than force-adding the file. This ensures the intent is explicit and prevents future changes to that file from being silently omitted from commits and lost.

**Rule 18.9** — Stage and commit all files that are part of the change before opening a pull request.

**Rule 18.10** — When opening a pull request the description MUST include:
- A concise title that identifies the nature of the change (feature, fix, refactor, etc.)
- What changed and which packages or files are affected
- Why the change was made
- Any risks or trade-offs, especially if charging, quota, or rating logic is involved
- A brief testing summary confirming what was run and that it passed

**Rule 18.11** — NEVER merge a pull request — leave that for human review.

---

## 19. Communication Standards

**Rule 19.1** — Clearly explain what changed, referencing specific files or packages.

**Rule 19.2** — Explain the reasoning behind design decisions.

**Rule 19.3** — Explicitly highlight potential risks, especially for changes that touch charging, quota, or rating logic.

**Rule 19.4** — If any verification step could not be performed, state this explicitly and explain why.

**Rule 19.5** — Prefer clarity over brevity when describing risks or trade-offs.
