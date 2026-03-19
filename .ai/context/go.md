# Go — Language and Framework Standards

Apply these rules when working in any Go package in this repository.

---

## Build Verification

After any change to Go source, imports, or dependencies — run in this order:

```bash
go mod tidy
go build ./...
go test ./...
```

Never claim an implementation is complete without all three passing.

---

## Go Coding Standards

**Context propagation**
Every function performing I/O (database, HTTP, Kafka, or any downstream call) must
accept `context.Context` as its first parameter and propagate it to all downstream
calls. Never use `context.Background()` inside business logic — only at entry points
(HTTP handlers, Kafka consumers). Never store context in structs.

**Nil safety**
Check all pointer, interface, slice, and map returns before use.
Do not assume a return value is non-nil without explicit documentation.

**Panics**
Never use `panic` in business logic. Only permitted in `main()` for startup
validation. HTTP handlers and Kafka consumers must recover from unexpected panics
and return a safe, controlled error response.

**Interface design**
Define interfaces at the point of consumption, not implementation. Keep them small
and focused — prefer single-method or narrow interfaces. Accept interfaces, return
concrete types.

**Struct initialisation**
Always use named fields in struct literals. Positional initialisation is prohibited
in all code including tests.
```go
// Correct
foo := MyStruct{Field1: "value", Field2: 42}
// Prohibited
foo := MyStruct{"value", 42}
```

**Constants**
Numeric literals and strings with business or operational meaning must be defined
as named constants. Timeouts, retry counts, buffer sizes, and thresholds must come
from configuration (YAML), not hardcoded.


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

- All shared mutable state must be protected: `sync.Mutex`, `sync.RWMutex`,
  atomics, or channels
- Every goroutine must have a documented lifecycle and shutdown path
- No goroutine leaks — every goroutine must terminate via context or stop channel
- Never use `init()` to initialise shared state
- Run `go test -race ./...` for any code involving concurrent access
- A test suite that passes without `-race` but fails with it is not passing

---

## Time

- Never call `time.Now()` inside business logic — inject time as a parameter
- All timestamps stored or published must be UTC — use `time.Now().UTC()` at entry
  points only
- Never compare `time.Time` with `==` — use `.Equal()`, `.Before()`, `.After()`

---

## Financial Values

- All financial values use `github.com/shopspring/decimal` — no float types permitted
- `float32`, `float64`, `int`, `int64` must not be used for financial arithmetic
- Precision comes from `DecimalDigits int32` config field (default: 22)
- Never hardcode precision — always read from config and propagate explicitly
- Reference: `internal/chargeengine/engine/steps/rating-step.go`

---

## Testing

**Commands:**
```bash
go test ./...                    # all tests
go test -race ./...              # with race detector (required for concurrent code)
go test ./internal/quota/...     # specific package
go test -run TestName ./...      # specific test
```

**Requirements:**
- Every Go source file with functions must have an accompanying `_test.go` file
- Files that only declare structs, constants, types, or interfaces are exempt
- Tests must run and pass — writing without running does not satisfy this rule
- Unit tests must NOT require external services (PostgreSQL, Kafka)
- Isolate integration tests using build tags or naming conventions

**Table-driven tests:**
Functions with multiple input/output combinations must use table-driven tests:
```go
tests := []struct {
    name     string
    input    SomeType
    expected SomeResult
}{
    {name: "zero value returns default", ...},
    {name: "negative amount returns error", ...},
}
for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) { ... })
}
```

**Test naming:**
Follow `TestFunctionName_Scenario_ExpectedBehaviour`:
e.g. `TestDebitQuota_InsufficientBalance_ReturnsOutOfFunds`


---

## Dependency Management

- Prefer libraries already used in the project over introducing new ones
- Verify new module paths on `pkg.go.dev` before adding — do not assume import paths
- If internet access is unavailable, state explicitly that verification was not performed
- Never modify files marked `// Code generated ... DO NOT EDIT` — re-run the generator

---

## Database Migrations

- Every migration must be reversible — up migration requires a down migration
- Migrations must be backward compatible with currently deployed application code
- Never drop columns or tables still referenced by application code — use a
  two-phase approach: deprecate first, remove in a subsequent migration
- Test migrations locally with `make migrate-up` before committing
- Data migrations must be proposed to a human with explicit risk statement

---

## Architecture Boundaries

- Transport handlers must be thin — delegate all logic to services
- No business logic in HTTP or Diameter handlers
- All database access through repository interfaces in `internal/store/`
- Kafka consumers must delegate to services — no business logic in consumers
- Event publishing isolated from core business logic
- New applications must follow structural patterns of existing applications
- Configuration from YAML only — no environment variables in application code

---

## Sensitive Data

- Subscriber identifiers, account balances, and transaction amounts must not
  appear in log output
- Internal state, raw database errors, and stack traces must not appear in
  API responses returned to external callers
- Credentials, API keys, and connection strings in YAML config only —
  never in source code, log output, or error messages

---

## Documentation

- All public functions and methods must have a Go doc comment
- Comments must describe what and why — not restate the code
- Do not add comments that merely repeat what the code already says

