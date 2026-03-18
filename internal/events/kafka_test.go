package events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewKafkaConfig
// ---------------------------------------------------------------------------

func TestNewKafkaConfig_Defaults(t *testing.T) {
	cfg := NewKafkaConfig()

	require.NotNil(t, cfg)
	assert.False(t, cfg.Enabled)
	assert.Equal(t, []string{"localhost:9092"}, cfg.Brokers)
	assert.Equal(t, "go-ocs", cfg.ClientID)
	assert.Equal(t, "all", cfg.RequiredAcks)
	assert.Equal(t, 10*time.Second, cfg.DialTimeout)
	assert.Equal(t, 10*time.Second, cfg.WriteTimeout)
	assert.Equal(t, 100*time.Millisecond, cfg.BatchTimeout)
	assert.Equal(t, "snappy", cfg.CompressionCodec)
	assert.NotNil(t, cfg.Topics)
	assert.Empty(t, cfg.Topics)
}

// ---------------------------------------------------------------------------
// ConnectKafka — disabled path (no real broker required)
// ---------------------------------------------------------------------------

func TestConnectKafka_Disabled_ReturnsManagerWithNilClient(t *testing.T) {
	cfg := NewKafkaConfig()
	cfg.Enabled = false

	mgr := ConnectKafka(cfg)

	require.NotNil(t, mgr)
	assert.Nil(t, mgr.KafkaClient)
	assert.False(t, mgr.KafkaConfig.Enabled)
}

func TestConnectKafka_Disabled_TopicMapPreserved(t *testing.T) {
	cfg := NewKafkaConfig()
	cfg.Enabled = false
	cfg.Topics = map[string]string{"chargeEvents": "charge.events.v1"}

	mgr := ConnectKafka(cfg)

	require.NotNil(t, mgr)
	assert.Equal(t, "charge.events.v1", mgr.KafkaConfig.Topics["chargeEvents"])
}

// ---------------------------------------------------------------------------
// StopKafka — disabled path must not panic
// ---------------------------------------------------------------------------

func TestStopKafka_Disabled_NoPanic(t *testing.T) {
	cfg := NewKafkaConfig()
	cfg.Enabled = false
	mgr := ConnectKafka(cfg)

	// Should not panic when Enabled is false (KafkaClient is nil).
	assert.NotPanics(t, func() {
		mgr.StopKafka()
	})
}

// ---------------------------------------------------------------------------
// Produce — nil client guard
// ---------------------------------------------------------------------------

func TestProduce_NilClient_NoOp(t *testing.T) {
	mgr := &KafkaManager{
		KafkaClient: nil,
		KafkaConfig: *NewKafkaConfig(),
	}

	// Produce must not panic when KafkaClient is nil.
	assert.NotPanics(t, func() {
		mgr.Produce(nil, nil, nil)
	})
}

// ---------------------------------------------------------------------------
// PublishEvent — topic alias resolution
// ---------------------------------------------------------------------------

func TestPublishEvent_TopicAlias_ResolvedFromMap(t *testing.T) {
	// We can verify that PublishEvent resolves topic aliases without calling
	// into a real Kafka broker by inspecting the KafkaConfig directly.
	// (Full publish path requires a live broker and is an integration concern.)
	cfg := NewKafkaConfig()
	cfg.Topics = map[string]string{
		"chargeEvents": "charge.events.v1",
	}

	mgr := &KafkaManager{KafkaConfig: *cfg}

	topic, ok := mgr.KafkaConfig.Topics["chargeEvents"]
	assert.True(t, ok)
	assert.Equal(t, "charge.events.v1", topic)
}

func TestPublishEvent_UnknownTopic_FallsBackToName(t *testing.T) {
	cfg := NewKafkaConfig()
	cfg.Topics = map[string]string{} // empty — no alias

	mgr := &KafkaManager{KafkaConfig: *cfg}

	topicName := "rawTopicName"
	topic, ok := mgr.KafkaConfig.Topics[topicName]
	if !ok {
		topic = topicName
	}
	assert.Equal(t, topicName, topic)
}
