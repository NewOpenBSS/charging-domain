package appcontext

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go-ocs/internal/events"
)

// ---------------------------------------------------------------------------
// subscriberEventTopic
// ---------------------------------------------------------------------------

func TestSubscriberEventTopic_NilConfig_ReturnsDefault(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "public.subscriber-event", subscriberEventTopic(nil))
}

func TestSubscriberEventTopic_NilTopicsMap_ReturnsDefault(t *testing.T) {
	t.Parallel()
	cfg := &events.KafkaConfig{Topics: nil}
	assert.Equal(t, "public.subscriber-event", subscriberEventTopic(cfg))
}

func TestSubscriberEventTopic_TopicNotInMap_ReturnsDefault(t *testing.T) {
	t.Parallel()
	cfg := &events.KafkaConfig{
		Topics: map[string]string{
			"quota-journal": "public.quota-journal",
		},
	}
	assert.Equal(t, "public.subscriber-event", subscriberEventTopic(cfg))
}

func TestSubscriberEventTopic_TopicInMap_ReturnsConfiguredTopic(t *testing.T) {
	t.Parallel()
	cfg := &events.KafkaConfig{
		Topics: map[string]string{
			"subscriber-event": "custom.subscriber-event",
		},
	}
	assert.Equal(t, "custom.subscriber-event", subscriberEventTopic(cfg))
}

func TestSubscriberEventTopic_DefaultTopicName_RoundTrips(t *testing.T) {
	t.Parallel()
	// When the canonical topic name is used directly in the map the function
	// should still return it unchanged.
	cfg := &events.KafkaConfig{
		Topics: map[string]string{
			"subscriber-event": "public.subscriber-event",
		},
	}
	assert.Equal(t, "public.subscriber-event", subscriberEventTopic(cfg))
}
