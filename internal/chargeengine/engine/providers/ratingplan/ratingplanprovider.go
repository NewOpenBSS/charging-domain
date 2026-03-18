package ratingplan

import (
	"context"
	"encoding/json"
	"fmt"
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/model"
	"go-ocs/internal/logging"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type RatePlanLoader func(ctx context.Context) (map[uuid.UUID]model.RatePlan, error)

type RatePlanContainer struct {
	ratePlans map[uuid.UUID]model.RatePlan
	loader    RatePlanLoader
	shutdown  func()
	mu        sync.RWMutex
}

func (c *RatePlanContainer) Shutdown() {
	if c.shutdown != nil {
		c.shutdown()
	}
}

func (sc *RatePlanContainer) reloadRatePlans(ctx context.Context) {
	plans, err := sc.loader(ctx)
	if err != nil {
		logging.Error("Failed to load rate plan list", "err", err)
		return
	}

	sc.mu.Lock()
	sc.ratePlans = plans
	sc.mu.Unlock()

	logging.Info("Rate Plan list loaded")
}

func loadRatePlans(container *RatePlanContainer, appctx *appcontext.AppContext) {
	logging.Info("Loading the all rate plans")

	ctx, cancel := context.WithCancel(context.Background())
	container.shutdown = cancel

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		container.reloadRatePlans(ctx)

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			log.Println("reload loop stopped")
			return
		}
	}
}

func NewRatePlanContainer(appctx *appcontext.AppContext) *RatePlanContainer {
	logging.Info("Creating new rate plan container")

	loader := func(ctx context.Context) (map[uuid.UUID]model.RatePlan, error) {
		plans, err := appctx.Store.Q.FindActiveRatePlans(ctx)
		if err != nil {
			return nil, err
		}

		list := make(map[uuid.UUID]model.RatePlan)
		for _, plan := range plans {
			id := uuid.UUID(plan.PlanID.Bytes)

			if _, ok := list[id]; ok {
				continue
			}

			p := model.RatePlan{}
			if err := json.Unmarshal(plan.Rateplan, &p); err != nil {
				return nil, err
			}

			list[id] = p
		}

		return list, nil
	}

	container := RatePlanContainer{
		loader: loader,
	}

	go loadRatePlans(&container, appctx)
	return &container
}

func (sc *RatePlanContainer) FindRatingPlan(uuid uuid.UUID) (*model.RatePlan, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if sc.ratePlans == nil {
		return nil, fmt.Errorf("rate plan list not loaded")
	}

	if p, ok := sc.ratePlans[uuid]; ok {
		return &p, nil
	}

	return nil, fmt.Errorf("rate plan not found")
}
