package carriers

import (
	"context"
	"fmt"
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/ocserrors"
	"go-ocs/internal/logging"
	"go-ocs/internal/store/sqlc"
	"log"
	"sync"
	"time"
)

type CarrierLoader func(ctx context.Context) ([]sqlc.Carrier, error)

type CarrierContainer struct {
	carriers map[string]sqlc.Carrier
	loader   CarrierLoader
	shutdown func()
	mu       sync.RWMutex
}

func (c *CarrierContainer) Shutdown() {
	if c.shutdown != nil {
		c.shutdown()
	}
}

func (c *CarrierContainer) reloadCarriers(ctx context.Context) {
	carriers, err := c.loader(ctx)
	if err != nil {
		logging.Error("Failed to load carrier list", "err", err)
		return
	}

	carrierList := make(map[string]sqlc.Carrier)
	for _, carrier := range carriers {
		carrierList[carrier.Plmn] = carrier
	}

	c.mu.Lock()
	c.carriers = carrierList
	c.mu.Unlock()

	logging.Info("Carrier list loaded")
}

func loadCarriers(container *CarrierContainer, appctx *appcontext.AppContext) {
	logging.Info("Loading the all carriers")

	ctx, cancel := context.WithCancel(context.Background())
	container.shutdown = cancel

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		container.reloadCarriers(ctx)

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			log.Println("reload loop stopped")
			return
		}
	}
}

func NewCarrierContainer(appctx *appcontext.AppContext) *CarrierContainer {
	logging.Info("Creating new classification container")

	loader := func(ctx context.Context) ([]sqlc.Carrier, error) {
		return appctx.Store.Q.AllCarriers(ctx)
	}

	container := CarrierContainer{
		loader: loader,
	}

	go loadCarriers(&container, appctx)
	return &container
}

func (c *CarrierContainer) FindCarrierByMccMnc(mcc string, mnc string) (*sqlc.Carrier, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.carriers == nil {
		return nil, fmt.Errorf("carrier list not loaded")
	}

	if carrier, ok := c.carriers[mcc+mnc]; ok {
		return &carrier, nil
	}

	if carrier, ok := c.carriers[mcc]; ok {
		return &carrier, nil
	}

	return nil, ocserrors.CreateUnknownCarrier(fmt.Sprintf("carrier for %s not found", mcc+mnc))
}

func (c *CarrierContainer) FindCarrierBySource(mcc string, mnc string) string {
	if mcc == "" {
		return "?"
	}

	carrier, err := c.FindCarrierByMccMnc(mcc, mnc)
	if err != nil {
		return "?"
	}

	return carrier.SourceGroup
}
