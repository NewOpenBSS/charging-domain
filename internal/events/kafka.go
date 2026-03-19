package events

import (
	"context"
	"encoding/json"
	"fmt"
	"go-ocs/internal/logging"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

type KafkaConfig struct {
	Enabled          bool              `yaml:"enabled"`
	Brokers          []string          `yaml:"brokers"`
	ClientID         string            `yaml:"clientId"`
	RequiredAcks     string            `yaml:"requiredAcks"`
	DialTimeout      time.Duration     `yaml:"dialTimeout"`
	WriteTimeout     time.Duration     `yaml:"writeTimeout"`
	BatchTimeout     time.Duration     `yaml:"batchTimeout"`
	CompressionCodec string            `yaml:"compressionCodec"`
	Topics           map[string]string `yaml:"topics"`
}

type KafkaManager struct {
	KafkaClient *kgo.Client
	KafkaConfig KafkaConfig
}

func NewKafkaConfig() *KafkaConfig {
	return &KafkaConfig{
		Enabled:          false,
		Brokers:          []string{"localhost:9092"},
		ClientID:         "go-ocs",
		RequiredAcks:     "all",
		DialTimeout:      10 * time.Second,
		WriteTimeout:     10 * time.Second,
		BatchTimeout:     100 * time.Millisecond,
		CompressionCodec: "snappy",
		Topics:           make(map[string]string),
	}
}

func (c *KafkaManager) StopKafka() {
	if c.KafkaConfig.Enabled == true {
		c.KafkaClient.Close()
	}
}

func ConnectKafka(c *KafkaConfig) *KafkaManager {

	if c.Enabled == true {
		client, err := kgo.NewClient(
			kgo.SeedBrokers(c.Brokers...),
			kgo.ClientID(c.ClientID),
			kgo.RequiredAcks(kgo.AllISRAcks()),
		)

		if err != nil {
			logging.Fatal("Failed to create kafka client", "err", err)
		}

		admin := kadm.NewClient(client)
		topics, err := admin.ListTopics(context.Background())
		if err != nil {
			logging.Error("Failed to list kafka topics", "err", err)
		}

		// Iterate over topics and create if not present
		for _, t := range c.Topics {
			if _, ok := topics[t]; ok {
				continue
			}

			logging.Info(fmt.Sprintf("Creating Kafka topic %s", t))
			_, err := admin.CreateTopics(context.Background(), 3, 1, nil, t)
			if err != nil {
				logging.Error("Failed to create kafka topic", "err", err)
			}
		}
		logging.Info(fmt.Sprintf("Connected to Kafka cluster at %s", c.Brokers[0]))

		return &KafkaManager{
			KafkaClient: client,
			KafkaConfig: *c,
		}
	}

	return &KafkaManager{
		KafkaClient: nil,
		KafkaConfig: *c,
	}
}

func (m *KafkaManager) Produce(ctx context.Context, r *kgo.Record, promise func(*kgo.Record, error)) {
	if m.KafkaClient != nil {
		m.KafkaClient.Produce(ctx, r, promise)
	}
}

func (m *KafkaManager) PublishEvent(topicName string, key string, event any) {
	payload, err := json.Marshal(event)
	if err != nil {
		logging.Error("Failed to marshal quota journal", "err", err)
	}

	topic, ok := m.KafkaConfig.Topics[topicName]
	if ok == false {
		topic = topicName
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: payload,
	}

	m.Produce(context.Background(), record, func(r *kgo.Record, err error) {
		if err != nil {
			logging.Error("Failed to publish event", "err", err, "topic", r.Topic)
			return
		}

		logging.Debug("Record Published",
			"topic", r.Topic,
			"partition", r.Partition,
			"offset", r.Offset,
		)
	})
}
