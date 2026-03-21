package consumer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"go-ocs/internal/events"
	"go-ocs/internal/logging"
	"go-ocs/internal/quota"
)

// QuotaProvisioner provisions a counter onto a subscriber's quota. It is the
// narrow interface the QuotaProvisioningConsumer requires — satisfied by *quota.QuotaManager.
type QuotaProvisioner interface {
	// ProvisionCounter provisions a new counter from the supplied request.
	// Returns nil on success or idempotent skip. Returns a non-nil error on
	// failure, which causes the Kafka offset to not be committed.
	ProvisionCounter(ctx context.Context, req quota.ProvisionCounterRequest) error
}

// QuotaProvisioningConsumer consumes QuotaProvisioningEvent messages from a Kafka
// topic and provisions counters onto subscriber quotas.
//
// Unlike the existing consumers in this package, offsets are committed manually
// (kgo.DisableAutoCommit) to honour at-least-once delivery semantics: the offset
// is committed only after ProvisionCounter returns nil. On any error the offset is
// not committed and the event is redelivered on the next restart.
//
// When Kafka is disabled (cfg.Enabled = false) the consumer starts as a no-op —
// Start returns immediately and Stop is safe to call.
type QuotaProvisioningConsumer struct {
	client      *kgo.Client // nil when Kafka is disabled
	provisioner QuotaProvisioner
	topic       string
}

// NewQuotaProvisioningConsumer constructs a QuotaProvisioningConsumer.
// When cfg.Enabled is false the returned consumer is a non-blocking no-op.
func NewQuotaProvisioningConsumer(cfg *events.KafkaConfig, provisioner QuotaProvisioner, topic string) *QuotaProvisioningConsumer {
	c := &QuotaProvisioningConsumer{
		provisioner: provisioner,
		topic:       topic,
	}

	if cfg == nil || !cfg.Enabled {
		return c
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ClientID(cfg.ClientID+"-quota-provisioning-consumer"),
		kgo.ConsumerGroup("charging-backend-quota-provisioning"),
		kgo.ConsumeTopics(topic),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		logging.Fatal("Failed to create QuotaProvisioningConsumer Kafka client", "err", err)
	}

	c.client = client
	return c
}

// Start launches the consumer background goroutine. The goroutine runs until
// ctx is cancelled or Stop is called. When Kafka is disabled, Start is a no-op.
func (c *QuotaProvisioningConsumer) Start(ctx context.Context) {
	if c.client == nil {
		logging.Info("QuotaProvisioningConsumer disabled — Kafka not configured")
		return
	}
	go c.run(ctx)
}

// Stop closes the underlying Kafka client. Safe to call when Kafka is disabled.
func (c *QuotaProvisioningConsumer) Stop() {
	if c.client != nil {
		c.client.Close()
	}
}

// run is the consumer loop. It polls for fetches and processes each record.
// On success the record's offset is committed. On failure the offset is skipped
// (not committed) so the event is redelivered on the next consumer startup.
func (c *QuotaProvisioningConsumer) run(ctx context.Context) {
	logging.Info("QuotaProvisioningConsumer started", "topic", c.topic)
	for {
		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			logging.Info("QuotaProvisioningConsumer stopped — client closed")
			return
		}

		if err := ctx.Err(); err != nil {
			logging.Info("QuotaProvisioningConsumer stopped — context cancelled")
			return
		}

		fetches.EachError(func(t string, p int32, err error) {
			logging.Error("QuotaProvisioningConsumer fetch error", "topic", t, "partition", p, "err", err)
		})

		fetches.EachRecord(func(r *kgo.Record) {
			if err := c.handleRecord(ctx, r); err != nil {
				// Log at ERROR and do NOT commit the offset — the event will be
				// redelivered on the next consumer restart.
				logging.Error("QuotaProvisioningConsumer handler error — offset not committed",
					"topic", r.Topic,
					"partition", r.Partition,
					"offset", r.Offset,
					"err", err,
				)
				return
			}

			// Commit the offset only after successful processing.
			if err := c.client.CommitRecords(ctx, r); err != nil {
				logging.Error("QuotaProvisioningConsumer failed to commit offset",
					"topic", r.Topic,
					"partition", r.Partition,
					"offset", r.Offset,
					"err", err,
				)
			}
		})
	}
}

// handleRecord deserialises and processes a single Kafka record.
// Malformed JSON is logged and skipped (returns nil so the offset is committed,
// preventing poison-pill messages from blocking the consumer).
// Unrecognised reason codes are substituted with QUOTA_PROVISIONED with a warning.
func (c *QuotaProvisioningConsumer) handleRecord(ctx context.Context, r *kgo.Record) error {
	var event events.QuotaProvisioningEvent
	if err := json.Unmarshal(r.Value, &event); err != nil {
		logging.Warn("QuotaProvisioningConsumer: malformed event skipped",
			"topic", r.Topic,
			"partition", r.Partition,
			"offset", r.Offset,
			"err", err,
		)
		return nil // skip — malformed messages must not block the consumer
	}

	reasonCode, ok := toQuotaReasonCode(event.ReasonCode)
	if !ok {
		logging.Warn("QuotaProvisioningConsumer: unrecognised reason code substituted",
			"receivedReasonCode", event.ReasonCode,
			"substituted", quota.ReasonQuotaProvisioned,
		)
	}

	req := quota.ProvisionCounterRequest{
		SubscriberID:         event.SubscriberID,
		CounterID:            event.EventID, // derived from EventID — not on the wire
		ProductID:            event.ProductID,
		ProductName:          event.ProductName,
		Description:          event.Description,
		UnitType:             event.UnitType,
		Priority:             event.Priority,
		InitialBalance:       event.InitialBalance,
		ExpiryDate:           event.ExpiryDate,
		CanRepayLoan:         event.CanRepayLoan,
		CanTransfer:          event.CanTransfer,
		CanConvert:           event.CanConvert,
		UnitPrice:            event.UnitPrice,
		TaxRate:              event.TaxRate,
		CounterSelectionKeys: event.CounterSelectionKeys,
		ExternalReference:    event.ExternalReference,
		ReasonCode:           reasonCode,
		Now:                  time.Now().UTC(),
		TransactionID:        event.EventID.String(),
	}

	if event.LoanInfo != nil {
		req.LoanInfo = &quota.LoanProvisionInfo{
			TransactionFee:     event.LoanInfo.TransactionFee,
			MinRepayment:       event.LoanInfo.MinRepayment,
			ClawbackPercentage: event.LoanInfo.ClawbackPercentage,
		}
	}

	return c.provisioner.ProvisionCounter(ctx, req)
}

// toQuotaReasonCode maps a ProvisioningReasonCode from the event payload to a
// quota.ReasonCode. Only QUOTA_PROVISIONED and TRANSFER_IN are valid provisioning
// reasons. Returns (code, true) on a known mapping and
// (ReasonQuotaProvisioned, false) for any unrecognised value.
func toQuotaReasonCode(rc events.ProvisioningReasonCode) (quota.ReasonCode, bool) {
	switch rc {
	case events.ProvisioningReasonQuotaProvisioned:
		return quota.ReasonQuotaProvisioned, true
	case events.ProvisioningReasonTransferIn:
		return quota.ReasonTransferIn, true
	default:
		return quota.ReasonQuotaProvisioned, false
	}
}
