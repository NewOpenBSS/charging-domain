package consumer

import (
	"context"
	"encoding/json"

	"github.com/twmb/franz-go/pkg/kgo"

	"go-ocs/internal/events"
	"go-ocs/internal/logging"
)

// WholesaleContractConsumer consumes WholesaleContractEvent messages from a Kafka
// topic and keeps the shadow wholesaler table in sync.
//
// When Kafka is disabled (cfg.Enabled = false) the consumer starts as a no-op —
// Start returns immediately and Stop is safe to call.
type WholesaleContractConsumer struct {
	client *kgo.Client // nil when Kafka is disabled
	storer WholesaleStorer
	topic  string
}

// NewWholesaleContractConsumer constructs a WholesaleContractConsumer.
// When cfg.Enabled is false the returned consumer is a non-blocking no-op.
func NewWholesaleContractConsumer(cfg *events.KafkaConfig, storer WholesaleStorer, topic string) *WholesaleContractConsumer {
	c := &WholesaleContractConsumer{
		storer: storer,
		topic:  topic,
	}

	if cfg == nil || !cfg.Enabled {
		return c
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ClientID(cfg.ClientID+"-wholesale-consumer"),
		kgo.ConsumerGroup("charging-backend-wholesale"),
		kgo.ConsumeTopics(topic),
	)
	if err != nil {
		logging.Fatal("Failed to create WholesaleContractConsumer Kafka client", "err", err)
	}

	c.client = client
	return c
}

// Start launches the consumer background goroutine. The goroutine runs until
// ctx is cancelled or Stop is called. When Kafka is disabled, Start is a no-op.
func (c *WholesaleContractConsumer) Start(ctx context.Context) {
	if c.client == nil {
		logging.Info("WholesaleContractConsumer disabled — Kafka not configured")
		return
	}
	go c.run(ctx)
}

// Stop closes the underlying Kafka client. Safe to call when Kafka is disabled.
func (c *WholesaleContractConsumer) Stop() {
	if c.client != nil {
		c.client.Close()
	}
}

// run is the consumer loop. It polls for fetches and processes each record.
func (c *WholesaleContractConsumer) run(ctx context.Context) {
	logging.Info("WholesaleContractConsumer started", "topic", c.topic)
	for {
		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			logging.Info("WholesaleContractConsumer stopped — client closed")
			return
		}

		if err := ctx.Err(); err != nil {
			logging.Info("WholesaleContractConsumer stopped — context cancelled")
			return
		}

		fetches.EachError(func(t string, p int32, err error) {
			logging.Error("WholesaleContractConsumer fetch error", "topic", t, "partition", p, "err", err)
		})

		fetches.EachRecord(func(r *kgo.Record) {
			if err := c.handleRecord(ctx, r); err != nil {
				logging.Error("WholesaleContractConsumer handler error", "err", err)
			}
		})
	}
}

// handleRecord unmarshals and processes a single Kafka record.
// The event type is decoded first to select the correct full struct for unmarshalling.
// Malformed or unrecognised events are logged and skipped — the consumer continues.
func (c *WholesaleContractConsumer) handleRecord(ctx context.Context, r *kgo.Record) error {
	// Decode only the event type first to dispatch to the correct struct.
	var envelope struct {
		EventType events.WholesaleContractEventType `json:"eventType"`
	}
	if err := json.Unmarshal(r.Value, &envelope); err != nil {
		logging.Warn("WholesaleContractConsumer: malformed event skipped",
			"topic", r.Topic,
			"partition", r.Partition,
			"offset", r.Offset,
			"err", err,
		)
		return nil
	}

	switch envelope.EventType {
	case events.WholesaleContractProvisioned:
		var event events.WholesaleContractProvisionedEvent
		if err := json.Unmarshal(r.Value, &event); err != nil {
			logging.Warn("WholesaleContractConsumer: malformed provisioned event skipped",
				"topic", r.Topic,
				"partition", r.Partition,
				"offset", r.Offset,
				"err", err,
			)
			return nil
		}
		return c.storer.UpsertWholesaler(ctx, &event)

	case events.WholesaleContractDeregistering:
		var event events.WholesaleContractDeregisteringEvent
		if err := json.Unmarshal(r.Value, &event); err != nil {
			logging.Warn("WholesaleContractConsumer: malformed deregistering event skipped",
				"topic", r.Topic,
				"partition", r.Partition,
				"offset", r.Offset,
				"err", err,
			)
			return nil
		}
		return c.storer.DeregisterWholesaler(ctx, event.WholesalerID)

	case events.WholesaleContractSuspend:
		var event events.WholesaleContractSuspendEvent
		if err := json.Unmarshal(r.Value, &event); err != nil {
			logging.Warn("WholesaleContractConsumer: malformed suspend event skipped",
				"topic", r.Topic,
				"partition", r.Partition,
				"offset", r.Offset,
				"err", err,
			)
			return nil
		}
		return c.storer.SuspendWholesaler(ctx, event.WholesalerID, event.Suspend)

	default:
		logging.Warn("WholesaleContractConsumer: unknown event type skipped",
			"eventType", envelope.EventType,
			"topic", r.Topic,
			"partition", r.Partition,
			"offset", r.Offset,
		)
		return nil
	}
}
