package appcontext

import (
	"context"
	"go-ocs/internal/quota"
	"go-ocs/internal/store"

	"github.com/twmb/franz-go/pkg/kgo"
)

type KafkaProducer interface {
	Produce(ctx context.Context, r *kgo.Record, promise func(*kgo.Record, error))
	PublishEvent(topicName string, key string, event any)
}

type AppContext struct {
	Config       *Config
	Metrics      *AppMetrics
	Store        *store.Store
	QuotaManager quota.QuotaManagerInterface
	KafkaManager KafkaProducer
}
