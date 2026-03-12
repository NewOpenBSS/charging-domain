package subscribers

import (
	"context"
	"fmt"
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/model"
	"go-ocs/internal/logging"
	"go-ocs/internal/store/sqlc"
	"log"
	"sync"
	"time"
)

type FindSubscriberLoader func(ctx context.Context, msisdn string) (sqlc.FindSubscriberWithWholesalerByMSISDNRow, error)

type SubscriberContainer struct {
	subscribers map[string]*model.Subscriber
	loader      FindSubscriberLoader
	shutdown    func()
	mu          sync.RWMutex
}

func (c *SubscriberContainer) Shutdown() {
	if c.shutdown != nil {
		c.shutdown()
	}
}

func clearCache(sc *SubscriberContainer) {
	logging.Info("Clearing the cache subscribers")

	ctx, cancel := context.WithCancel(context.Background())
	sc.shutdown = cancel

	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		sc.mu.Lock()
		sc.subscribers = make(map[string]*model.Subscriber)
		sc.mu.Unlock()

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			log.Println("reload loop stopped")
			return
		}
	}
}

func NewSubscriberContainer(appctx *appcontext.AppContext) *SubscriberContainer {
	loader := func(ctx context.Context, msisdn string) (sqlc.FindSubscriberWithWholesalerByMSISDNRow, error) {
		return appctx.Store.Q.FindSubscriberWithWholesalerByMSISDN(ctx, msisdn)
	}

	container := &SubscriberContainer{
		subscribers: make(map[string]*model.Subscriber),
		loader:      loader,
	}

	go clearCache(container)
	return container
}

func (c *SubscriberContainer) FindSubscriber(msisdn string) (*model.Subscriber, error) {
	c.mu.RLock()
	if subscriber, ok := c.subscribers[msisdn]; ok {
		c.mu.RUnlock()
		return subscriber, nil
	}
	c.mu.RUnlock()

	subRec, err := c.loader(context.Background(), msisdn)
	if err != nil {
		return nil, fmt.Errorf("failed to load subscriber: %w", err)
	}

	if !subRec.WholesalerActive {
		return nil, fmt.Errorf("subscriber is not active")
	}

	if subRec.Status != "ACTIVE" {
		return nil, fmt.Errorf("subscriber is not active")
	}

	subscriber := &model.Subscriber{
		Msisdn:               msisdn,
		SubscriberId:         subRec.SubscriberID.Bytes,
		ContractId:           subRec.ContractID.Bytes,
		RatePlanId:           subRec.RateplanID.Bytes,
		WholesaleId:          subRec.WholesalerID.Bytes,
		WholesalerRatePlanId: subRec.WholesalerRateplanID.Bytes,
		AllowOOBCharging:     subRec.AllowOobCharging,
	}

	c.mu.Lock()
	c.subscribers[msisdn] = subscriber
	c.mu.Unlock()

	return subscriber, nil
}
