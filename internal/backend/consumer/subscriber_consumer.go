// Package consumer contains Kafka consumers for the charging-backend application.
package consumer

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"

	"go-ocs/internal/events"
	"go-ocs/internal/logging"
)

// SubscriberStorer is the minimal interface the SubscriberEventConsumer requires
// to persist subscriber changes. Defined at the point of consumption so that
// tests can substitute a mock without requiring a real database connection.
type SubscriberStorer interface {
	// InsertSubscriber persists a new subscriber from a CREATED event.
	InsertSubscriber(ctx context.Context, event *events.SubscriberEvent) error

	// UpdateSubscriber refreshes all subscriber fields from an UPDATED,
	// MSISDN_SWAP, or SIM_SWAP event.
	UpdateSubscriber(ctx context.Context, event *events.SubscriberEvent) error

	// DeleteSubscriber hard-deletes a subscriber from a DELETED event and
	// cascades the delete to an inactive wholesaler if it has no remaining subscribers.
	DeleteSubscriber(ctx context.Context, subscriberID uuid.UUID, wholesaleID uuid.UUID) error
}

// SubscriberEventConsumer consumes SubscriberEvent messages from a Kafka topic
// and keeps the shadow subscriber table in sync.
//
// When Kafka is disabled (cfg.Enabled = false) the consumer starts as a no-op —
// Start returns immediately and Stop is safe to call.
type SubscriberEventConsumer struct {
	client  *kgo.Client // nil when Kafka is disabled
	storer  SubscriberStorer
	topic   string
}

// NewSubscriberEventConsumer constructs a SubscriberEventConsumer.
// When cfg.Enabled is false the returned consumer is a non-blocking no-op.
func NewSubscriberEventConsumer(cfg *events.KafkaConfig, storer SubscriberStorer, topic string) *SubscriberEventConsumer {
	c := &SubscriberEventConsumer{
		storer: storer,
		topic:  topic,
	}

	if cfg == nil || !cfg.Enabled {
		return c
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ClientID(cfg.ClientID+"-subscriber-consumer"),
		kgo.ConsumerGroup("charging-backend-subscriber"),
		kgo.ConsumeTopics(topic),
	)
	if err != nil {
		logging.Fatal("Failed to create SubscriberEventConsumer Kafka client", "err", err)
	}

	c.client = client
	return c
}

// Start launches the consumer background goroutine. The goroutine runs until
// ctx is cancelled or Stop is called. When Kafka is disabled, Start is a no-op.
func (c *SubscriberEventConsumer) Start(ctx context.Context) {
	if c.client == nil {
		logging.Info("SubscriberEventConsumer disabled — Kafka not configured")
		return
	}
	go c.run(ctx)
}

// Stop closes the underlying Kafka client. Safe to call when Kafka is disabled.
func (c *SubscriberEventConsumer) Stop() {
	if c.client != nil {
		c.client.Close()
	}
}

// run is the consumer loop. It polls for fetches and processes each record.
func (c *SubscriberEventConsumer) run(ctx context.Context) {
	logging.Info("SubscriberEventConsumer started", "topic", c.topic)
	for {
		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			logging.Info("SubscriberEventConsumer stopped — client closed")
			return
		}

		if err := ctx.Err(); err != nil {
			logging.Info("SubscriberEventConsumer stopped — context cancelled")
			return
		}

		fetches.EachError(func(t string, p int32, err error) {
			logging.Error("SubscriberEventConsumer fetch error", "topic", t, "partition", p, "err", err)
		})

		fetches.EachRecord(func(r *kgo.Record) {
			if err := c.handleRecord(ctx, r); err != nil {
				logging.Error("SubscriberEventConsumer handler error", "err", err)
			}
		})
	}
}

// handleRecord unmarshals and processes a single Kafka record.
// Malformed or unrecognised events are logged and skipped — the consumer continues.
func (c *SubscriberEventConsumer) handleRecord(ctx context.Context, r *kgo.Record) error {
	var event events.SubscriberEvent
	if err := json.Unmarshal(r.Value, &event); err != nil {
		logging.Warn("SubscriberEventConsumer: malformed event skipped",
			"topic", r.Topic,
			"partition", r.Partition,
			"offset", r.Offset,
			"err", err,
		)
		return nil
	}

	switch event.EventType {
	case events.SubscriberEventCreated:
		return c.storer.InsertSubscriber(ctx, &event)

	case events.SubscriberEventUpdated,
		events.SubscriberEventMsisdnSwap,
		events.SubscriberEventSimSwap:
		return c.storer.UpdateSubscriber(ctx, &event)

	case events.SubscriberEventDeleted:
		return c.storer.DeleteSubscriber(ctx, event.SubscriberID, event.WholesaleID)

	default:
		logging.Warn("SubscriberEventConsumer: unknown event type skipped",
			"eventType", event.EventType,
			"topic", r.Topic,
			"partition", r.Partition,
			"offset", r.Offset,
		)
		return nil
	}
}
