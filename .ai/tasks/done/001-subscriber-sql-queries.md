# Task 001 — Subscriber SQL Queries (sqlc)

## Feature
F-005 — SubscriberEventConsumer

## What to Build
Add INSERT, UPDATE, and DELETE SQL queries for the `subscriber` table so the
consumer can persist `SubscriberEvent` changes. Run sqlc to regenerate the
Go code.

## Files to Create / Modify
- **Create** `internal/store/queries/subscriber.sql` — three queries
- **Generate** `internal/store/sqlc/subscriber.sql.go` via `sqlc generate`

## Queries Required

### InsertSubscriber (:exec)
Insert a new row into `subscriber`. Params: subscriber_id, rateplan_id,
customer_id, wholesale_id, msisdn, iccid, contract_id, status,
allow_oob_charging. (modified_on uses DB DEFAULT)

### UpdateSubscriber (:exec)
Update all non-key fields by subscriber_id. Set modified_on = NOW().
Params: subscriber_id (WHERE key), rateplan_id, customer_id, wholesale_id,
msisdn, iccid, contract_id, status, allow_oob_charging.

### DeleteSubscriber (:exec)
Hard-delete by subscriber_id.

## Acceptance Criteria
- [ ] `internal/store/queries/subscriber.sql` exists with all three queries
- [ ] `~/go/bin/sqlc generate` produces `internal/store/sqlc/subscriber.sql.go`
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
