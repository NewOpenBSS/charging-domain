# Project Structure

This document describes the top-level directory layout of go-ocs and the purpose
of each folder. It is intended for developers new to the project.

---

## Overview

```
go-ocs/
├── CLAUDE.md
├── Makefile
├── go.mod / go.sum
├── sqlc.yaml
│
├── .ai/
├── .claude/
│
├── api-tests/
├── cmd/
├── db/
├── deploy/
├── docs/
├── gql/
├── internal/
└── out/
```

---

## Root Files

| File | Purpose |
|---|---|
| `CLAUDE.md` | Agent protocol for Claude Code — session rules, workflow, coding standards |
| `Makefile` | Common development tasks: build, migrate, seed |
| `go.mod` / `go.sum` | Go module definition and dependency lock |
| `sqlc.yaml` | Configuration for sqlc — generates type-safe Go from SQL queries |

---

## Source Directories

### `cmd/`
Application entry points. One subdirectory per deployable application.

| Application | Port | Purpose |
|---|---|---|
| `cmd/charging-engine/` | `:8080` | Real-time NCHF HTTP charging API |
| `cmd/charging-dra/` | — | Diameter Ro interface for wholesale carriers |
| `cmd/charging-backend/` | `:8081` | Admin REST and GraphQL API |

Each application directory contains a `main.go` and a `*-config.yaml` for local development.

### `internal/`
All Go source code. Packages here are private to the module. Organised by concern:

| Package | Purpose |
|---|---|
| `internal/chargeengine/` | Charging pipeline, HTTP handlers, step-based processing |
| `internal/quota/` | Quota reservation, debit, release, and tax calculation |
| `internal/ruleevaluator/` | Policy-based pricing rule evaluation |
| `internal/model/` | Shared domain types (ClassificationPlan, RatePlan, etc.) |
| `internal/store/` | Repository pattern — pgxpool, sqlc queries, dynamic query methods |
| `internal/backend/` | charging-backend business logic, GraphQL resolvers, services |
| `internal/auth/` | Keycloak OAuth2 client, JWT middleware, claims extraction |
| `internal/events/` | Kafka producer for charge records and quota journal events |
| `internal/nchf/` | 3GPP NCHF protocol models and AVP mapping |
| `internal/diameter/` | Diameter protocol support for DRA server |
| `internal/charging/` | Core charging domain types (RateKey, UnitType, errors) |
| `internal/logging/` | Structured logging with Chi middleware |
| `internal/baseconfig/` | Shared YAML configuration loading |
| `internal/appl/` | Shared application lifecycle (metrics server, signal handling) |
| `internal/common/` | Shared utilities (masking, local datetime) |

### `gql/`
GraphQL schema definitions and gqlgen configuration.

| Path | Purpose |
|---|---|
| `gql/schema/` | Source-of-truth `.graphql` schema files |
| `gql/gqlgen.yml` | gqlgen code generation configuration |

Schema changes here require running `gqlgen generate` to regenerate
`internal/backend/graphql/generated/`.

### `db/`
Database migration scripts and seed data.

| Path | Purpose |
|---|---|
| `db/migrations/` | Numbered SQL migration files (applied via `make migrate-up`) |
| `db/seeds/` | Test data for local development (applied via `make seed`) |
| `db/init/` | Initial schema for Docker-based local setup |

### `deploy/`
Deployment configuration.

| Path | Purpose |
|---|---|
| `deploy/docker/` | Docker Compose and Dockerfile definitions |
| `deploy/k8s/` | Kubernetes manifests |

---

## Tooling Directories

These directories are prefixed with `.` following the standard convention for
tooling and infrastructure — not application source code.

### `.ai/`
AI agent process files. Managed by Claude Code during development sessions.
Not part of the application — do not edit manually unless restructuring the
AI workflow itself.

| Path | Purpose |
|---|---|
| `.ai/memory/STATUS.md` | Current implementation state — updated after every task |
| `.ai/memory/DECISIONS.md` | Append-only architecture decision log (ADRs) |
| `.ai/tasks/TASK_TEMPLATE.md` | Template for new task specifications |
| `.ai/tasks/CURRENT.md` | Active task spec — created at design time, consumed at implementation |
| `.ai/tasks/done/` | Completed task specs archived here |

### `.claude/`
Claude Code runtime settings.

| Path | Purpose |
|---|---|
| `.claude/settings.local.json` | Bash command permissions for Claude Code |

---

## Other Directories

### `api-tests/`
IntelliJ HTTP client files for manually testing the GraphQL and NCHF APIs.
These are not automated tests — they are used interactively during development
via the IntelliJ HTTP client tool.

| File | Purpose |
|---|---|
| `CarrierGraphQL.http` | Carrier resource GraphQL queries and mutations |
| `ClassficationGraphQL.http` | Classification resource GraphQL queries and mutations |
| `NumberPlanGraphQL.http` | Number plan resource GraphQL queries and mutations |
| `RatePlanGraphQL.http` | Rate plan resource GraphQL queries and mutations |
| `nchfChargeRequest.http` | NCHF charging engine request examples |
| `http-client.private.env.json` | IntelliJ HTTP client environment variables (not committed with secrets) |

### `docs/`
Project documentation for humans.

| Path | Purpose |
|---|---|
| `docs/PROJECT_BRIEF.md` | Project purpose, domain context, key components |
| `docs/ARCHITECTURE.md` | System architecture, layering rules, key flows |
| `docs/PROJECT_STRUCTURE.md` | This file |
| `docs/archive/` | Historical design documents — preserved for reference |

### `out/`
Generated output from tooling. Not manually edited.

| Path | Purpose |
|---|---|
| `out/GraphQLEndpoints.http` | Generated HTTP client file for GraphQL endpoints |
| `out/production/` | Compiled production binaries |

---

## Automated Tests

Go unit tests live alongside the source files they test, following Go conventions.
Every package in `internal/` has `_test.go` files in the same directory.

Unit tests do not require external services. Tests that require PostgreSQL or Kafka
are isolated using build tags.

Run all tests:
```bash
go test ./...
```

Run with race detector:
```bash
go test -race ./...
```
