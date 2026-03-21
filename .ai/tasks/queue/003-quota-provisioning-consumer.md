# Task 003 — QuotaProvisioningConsumer

**Feature:** F-007 — QuotaProvisioningEventConsumer
**Sequence:** 3 of 4
**Date:** 2026-03-21

---

## Objective

Implement the `QuotaProvisioningConsumer` in `internal/backend/consumer/`. This consumer
listens on the `public.quota-provisioning` Kafka topic, deserialises each message into a
`QuotaProvisioningEvent`, and delegates to `QuotaProvisioner.ProvisionCounter`. It differs
from the existing consumers in one critical way: **offsets are committed manually** (using
`kgo.DisableAutoCommit()`) to honour at-least-once delivery — the offset is committed only
after successful processing.

---

## Scope

### `internal/backend/consumer/quota_provisioning_consumer.go`

**QuotaProvisioner interface** (defined at point of consumption):
```go
// QuotaProvisioner provisions a counter onto a subscriber's quota. It is the
// narrow interface the consumer requires — satisfied by *quota.QuotaManager.
type QuotaProvisioner interface {
    ProvisionCounter(ctx context.Context, req quota.ProvisionCounterRequest) error
}
```

**QuotaProvisioningConsumer struct:**
```go
type QuotaProvisioningConsumer struct {
    client     *kgo.Client      // nil when Kafka disabled
    provisioner QuotaProvisioner
    topic      string
}
```

**Constructor** `NewQuotaProvisioningConsumer(cfg *events.KafkaConfig, provisioner QuotaProvisioner, topic string) *QuotaProvisioningConsumer`:
- When `cfg == nil || !cfg.Enabled`: return consumer with nil client (no-op)
- When enabled: create `kgo.Client` with options:
  - `kgo.SeedBrokers(cfg.Brokers...)`
  - `kgo.ClientID(cfg.ClientID + "-quota-provisioning-consumer")`
  - `kgo.ConsumerGroup("charging-backend-quota-provisioning")`
  - `kgo.ConsumeTopics(topic)`
  - `kgo.DisableAutoCommit()` ← **manual commit only**
- On error: `logging.Fatal(...)`

**Start(ctx context.Context):**
- If client nil: log "QuotaProvisioningConsumer disabled" and return
- Otherwise: `go c.run(ctx)`

**Stop():**
- If client non-nil: `c.client.Close()`

**run(ctx context.Context):**
```
for {
    fetches := c.client.PollFetches(ctx)
    if fetches.IsClientClosed() { return }
    if ctx.Err() != nil { return }
    fetches.EachError(func(t string, p int32, err error) { logging.Error(...) })
    fetches.EachRecord(func(r *kgo.Record) {
        if err := c.handleRecord(ctx, r); err != nil {
            logging.Error("QuotaProvisioningConsumer handler error — offset not committed", ...)
            // do NOT commit — event will be redelivered on restart
            return
        }
        // commit only on success
        if err := c.client.CommitRecords(ctx, r); err != nil {
            logging.Error("QuotaProvisioningConsumer failed to commit offset", ...)
        }
    })
}
```

**handleRecord(ctx context.Context, r *kgo.Record) error:**
1. Unmarshal `r.Value` into `events.QuotaProvisioningEvent`. On JSON error: log Warn, return nil (skip malformed)
2. Resolve reason code:
   - Map `ProvisioningReasonCode` → `quota.ReasonCode`
   - If unrecognised: log Warn, substitute `quota.ReasonQuotaProvisioned`
3. Build `quota.ProvisionCounterRequest` from event fields:
   - `Now = time.Now().UTC()`
   - `TransactionID = event.EventID.String()`
   - All counter fields from event
4. Call `c.provisioner.ProvisionCounter(ctx, req)`
5. Return any error from ProvisionCounter (triggers no-commit path)

**Reason code mapping function** (private):
```go
func toQuotaReasonCode(rc events.ProvisioningReasonCode) (quota.ReasonCode, bool) {
    switch rc {
    case events.ProvisioningReasonQuotaProvisioned:
        return quota.ReasonQuotaProvisioned, true
    case events.ProvisioningReasonLoanRepayment:
        return quota.ReasonLoanRepayment, true
    case events.ProvisioningReasonRefund:
        return quota.ReasonRefund, true  // if it exists; if not, map to QuotaProvisioned
    case events.ProvisioningReasonTransferIn:
        return quota.ReasonTransferIn, true
    case events.ProvisioningReasonConversion:
        return quota.ReasonConversion, true
    default:
        return quota.ReasonQuotaProvisioned, false
    }
}
```
Note: check which ReasonCode constants exist in `internal/quota/reasoncode.go` and map only those. For any event reason code that has no quota equivalent, substitute QUOTA_PROVISIONED with a warning.

---

### `internal/backend/consumer/quota_provisioning_consumer_test.go`

Mock `QuotaProvisioner` for unit tests. Tests:

```
TestQuotaProvisioningConsumer_HandleRecord_ValidEvent_CallsProvisionCounter
TestQuotaProvisioningConsumer_HandleRecord_MalformedJSON_SkipsRecord
TestQuotaProvisioningConsumer_HandleRecord_UnrecognisedReasonCode_SubstitutesQuotaProvisioned
TestQuotaProvisioningConsumer_HandleRecord_ProvisionerError_ReturnsError
TestQuotaProvisioningConsumer_Start_DisabledKafka_ReturnsImmediately
TestQuotaProvisioningConsumer_Stop_NilClient_NoPanic
```

---

## Acceptance Criteria

- [ ] `QuotaProvisioner` interface exists and `*quota.QuotaManager` satisfies it
- [ ] Constructor creates no-op consumer when Kafka disabled
- [ ] `kgo.DisableAutoCommit()` is set on the Kafka client
- [ ] Successful record processing commits the offset via `CommitRecords`
- [ ] Failed record processing does NOT commit the offset
- [ ] Malformed JSON events are skipped (logged + nil returned)
- [ ] Unrecognised reason codes are substituted with `QUOTA_PROVISIONED` + Warn log
- [ ] All unit tests pass with `go test -race ./internal/backend/consumer/...`
- [ ] `go build ./...` passes
