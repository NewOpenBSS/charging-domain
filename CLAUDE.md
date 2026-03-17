# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Session Initialization

Before performing any task, read these files in order:
1. `PROJECT_BRIEF.md` — project purpose and domain context
2. `ARCHITECTURE.md` — system structure and architectural boundaries
3. `AI_GUIDANCE.md` — mandatory development rules for this repository

## Commands

### Build
```bash
make build           # Build all applications (charging-dra, charging-engine)
make build-dra       # Build DRA server only
make build-engine    # Build charging engine only
make clean           # Remove built binaries
```

### Test & Verify
```bash
go mod tidy          # Resolve and tidy dependencies
go build ./...       # Verify entire repository compiles
go test ./...        # Run all tests
go test ./internal/quota/...         # Run tests for a specific package
go test -run TestName ./internal/... # Run a single test
```

### Database
```bash
make migrate-up      # Apply all migrations
make migrate-down    # Rollback last migration
make migrate-clean   # Drop and clean database
make seed            # Seed test data
```
Default DB: `postgres://gobss:gobss@localhost:5432/gobss?sslmode=disable` (schema: `charging`)

## Applications

Three independent applications share a database and Kafka infrastructure:

| Application | Entry Point | Port | Purpose |
|---|---|---|---|
| `charging-engine` | `cmd/charging-engine/` | `:8080` | NCHF HTTP charging API |
| `charging-dra` | `cmd/charging-dra/` | — | Diameter Ro interface for wholesale carriers |
| `charging-backend` | `cmd/charging-backend/` | `:8081` | Admin REST + GraphQL API |

New applications must follow the same structure and patterns as `charging-engine` and `charging-dra`.

## Architecture

### Layering

```
Transport Layer  → HTTP (chi), Diameter protocol handlers
Service Layer    → Charging service orchestration, quota management
Domain Layer     → Charging models, quota logic, rule evaluation, rating
Persistence Layer→ PostgreSQL via sqlc-generated queries (internal/store/)
Messaging Layer  → Kafka producers/consumers (internal/events/)
```

**Rules enforced by architecture:**
- Transport handlers must be thin — delegate all logic to services
- No business logic in transport handlers
- No domain queries in persistence layer
- All database access through repository interfaces (`internal/store/`)
- Event publishing isolated from core charging logic

### Charging Engine Pipeline

Requests flow through ordered steps in `internal/chargeengine/engine/steps/`:
`authentication → classification → rating → accounting → chargedata → response`

Data providers (`internal/chargeengine/engine/providers/`) supply business data (subscribers, rate plans, carriers, number plans, classification plans) to the pipeline steps.

### Key Packages

| Package | Purpose |
|---|---|
| `internal/chargeengine` | HTTP handlers and pipeline orchestration |
| `internal/quota` | Quota reservation, debit, release, and tax |
| `internal/ruleevaluator` | Policy-based pricing rule evaluation |
| `internal/store` | Repository pattern with sqlc-generated queries |
| `internal/nchf` | 3GPP NCHF protocol models and mapping |
| `internal/diameter` | Diameter protocol for DRA server |
| `internal/auth` | OAuth2/Keycloak authentication middleware |
| `internal/backend` | Charging backend REST/GraphQL handlers |
| `internal/events` | Kafka producer for event streaming |
| `internal/baseconfig` | Shared YAML configuration loading |

### Configuration

All applications use YAML configuration (no environment variables in code). Shared base config:
```yaml
base:
  appName: string
  metrics: { enabled, addr, path }
  database: { url }
  logging: { level, format }
```

## Domain Safety Rules

This system processes real money. These rules are non-negotiable:
- **Charging must be deterministic** — same inputs must always produce same outputs
- **Quota counters must never go negative** — no overdraft permitted
- **All quota operations must be transactional** — use database transactions
- **Duplicate/replayed events must not cause double charging** — enforce idempotency
- **Never invent or infer billing semantics** — if requirements are unclear, ask

If a change could affect charging, quota, or rating behaviour, clearly explain the risk before proceeding.

## Development Rules

- Run `go mod tidy` then `go build ./...` after any Go code or dependency changes
- All public functions must have a short comment explaining their purpose
- All Go files with functions/methods must have accompanying unit tests; run them before claiming success
- Never claim code works without running tests and verifying the build passes
- Ask before: deleting files, broad refactors, changing public APIs, modifying core charging logic, introducing new dependencies
- Prefer libraries already used in the project
- Verify new dependency module paths on pkg.go.dev before adding them

## Git Workflow

- Never commit directly to `master`
- Create a feature branch for each task
- Open a pull request when the feature is complete
- Never merge pull requests — leave that for human review