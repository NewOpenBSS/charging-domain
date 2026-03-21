# Task 004 — Wire QuotaProvisioningConsumer into AppContext and main.go

**Feature:** F-007 — QuotaProvisioningEventConsumer
**Sequence:** 4 of 4
**Date:** 2026-03-21

---

## Objective

Wire the `QuotaProvisioningConsumer` (built in Task 003) into the charging-backend application:
add it to `AppContext`, register the topic name in configuration, start and stop it in `main.go`.
This is the final integration task that makes the consumer live in the running application.
It follows the `WholesaleContractConsumer` wiring pattern exactly.

---

## Scope

### `internal/backend/appcontext/context.go`

1. Add field to `AppContext`:
   ```go
   QuotaProvisioningConsumer *consumer.QuotaProvisioningConsumer
   ```

2. In `NewAppContext`: construct the consumer using `quotaManager` as the provisioner:
   ```go
   QuotaProvisioningConsumer: consumer.NewQuotaProvisioningConsumer(
       cfg.Kafkaconfig,
       quotaManager,
       quotaProvisioningTopic(cfg.Kafkaconfig),
   ),
   ```

3. Add topic helper function:
   ```go
   // quotaProvisioningTopic resolves the quota-provisioning topic name from the Kafka
   // topics map, falling back to the canonical topic name if not configured.
   func quotaProvisioningTopic(cfg *events.KafkaConfig) string {
       const defaultTopic = "public.quota-provisioning"
       if cfg == nil || cfg.Topics == nil {
           return defaultTopic
       }
       if t, ok := cfg.Topics["quota-provisioning"]; ok {
           return t
       }
       return defaultTopic
   }
   ```

### `cmd/charging-backend/backend-config.yaml`

Add one entry to `kafka.topics`:
```yaml
quota-provisioning: "public.quota-provisioning"
```

### `cmd/charging-backend/main.go`

Add consumer lifecycle immediately after the wholesale consumer block, following
the same naming pattern:
```go
// Start the quota provisioning consumer so subscriber balances are populated on provisioning events.
quotaProvisioningCtx, quotaProvisioningCancel := context.WithCancel(context.Background())
defer quotaProvisioningCancel()
defer appCtx.QuotaProvisioningConsumer.Stop()
appCtx.QuotaProvisioningConsumer.Start(quotaProvisioningCtx)
```

**Out of scope:** any changes to consumer logic, provisioner, or event types.

---

## Acceptance Criteria

- [ ] `AppContext.QuotaProvisioningConsumer` field exists and is populated in `NewAppContext`
- [ ] `quotaProvisioningTopic` helper function exists in `context.go`
- [ ] `backend-config.yaml` contains `quota-provisioning: "public.quota-provisioning"` under `kafka.topics`
- [ ] `main.go` starts and stops `QuotaProvisioningConsumer` with its own context
- [ ] `go build ./...` passes
- [ ] `go test -race ./...` passes

---

## Risk Assessment

Low. Pure wiring — no new logic introduced. The consumer is a no-op when
`kafka.enabled = false` (the default in config), so starting it unconditionally is
safe for all environments including local development and CI.
