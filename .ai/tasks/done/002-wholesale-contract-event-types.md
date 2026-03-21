# Task 002 — Wholesale contract event types

**Feature:** F-006 — WholesaleContractConsumer
**Sequence:** 2 of 5
**Date:** 2026-03-21
**Status:** Active

---

## Objective

Create `internal/events/wholesale_contract_event.go` containing the three event-type constants and
three corresponding event structs for the Wholesale CRM domain. These types are the wire format for
messages published by the Wholesale CRM onto the Kafka topic consumed in Task 003. This task is
additive only — no existing files are changed.

---

## Scope

**In scope:**
- Create `internal/events/wholesale_contract_event.go` with:
  - Type alias `WholesaleContractEventType string`
  - Three constants:
    - `WholesaleContractProvisioned WholesaleContractEventType = "WholesaleContractProvisionedEvent"`
    - `WholesaleContractDeregistering WholesaleContractEventType = "WholesaleContractDeregisteringEvent"`
    - `WholesaleContractSuspend WholesaleContractEventType = "WholesaleContractSuspendEvent"`
  - `WholesaleContractProvisionedEvent` struct with JSON tags:
    - `EventType WholesaleContractEventType`
    - `WholesalerID uuid.UUID` — maps to `wholesalerId`
    - `ContractID uuid.UUID` — maps to `contractId`
    - `RatePlanID uuid.UUID` — maps to `ratePlanId`
    - `LegalName string` — maps to `legalName`
    - `DisplayName string` — maps to `displayName`
    - `Realm string`
    - `Hosts []string`
    - `NchfUrl string` — maps to `nchfUrl`
    - `RateLimit float64` — maps to `rateLimit` (DB stores as numeric; float64 sufficient for rate limit)
    - `Active bool`
  - `WholesaleContractDeregisteringEvent` struct:
    - `EventType WholesaleContractEventType`
    - `WholesalerID uuid.UUID` — maps to `wholesalerId`
  - `WholesaleContractSuspendEvent` struct:
    - `EventType WholesaleContractEventType`
    - `WholesalerID uuid.UUID` — maps to `wholesalerId`
    - `Suspend bool`
- Verify `go build ./...` passes

**Out of scope:**
- Tests — this file declares only structs and constants; no logic to test
- Consumer implementation — that is in Task 003
- Any modification to existing event files

---

## Context

- Pattern reference: `internal/events/subscriber_event.go` — follow the same package, doc-comment,
  type alias, and struct conventions exactly
- All public types and constants require Go doc comments (project standards in `.ai/context/go.md`)
- The `WholesaleContractProvisionedEvent` carries extra fields from the Wholesale CRM
  (`registrationNumber`, `taxNumber`, `addressInfo`, `contactInfo`, `invoiceMessage`) that are
  intentionally excluded — they are out of scope per F-006 constraints
- `github.com/google/uuid` is already a project dependency — import it directly
- `RateLimit` is stored in the DB as `numeric` — using `float64` in the event struct is acceptable
  because rate limits are whole numbers or simple decimals, not financial values

---

## Decisions Made During Design

| Decision | Rationale |
|---|---|
| Separate structs per event type | The three event types carry different payloads; a single struct with optional fields would obscure intent |
| Exclude non-DB fields from provisioned struct | Fields not present in the wholesaler table schema are explicitly out of scope per F-006 constraints |
| `RateLimit float64` not `decimal.Decimal` | Rate limits are operational, not financial; float64 matches JSON unmarshalling naturally |

---

## Acceptance Criteria

- [ ] `internal/events/wholesale_contract_event.go` exists with all three event types and constants
- [ ] All public symbols have Go doc comments
- [ ] `go build ./...` passes

---

## Risk Assessment

None. Pure type definitions with no logic. No existing files modified.
