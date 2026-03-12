package classificationplan

import (
	"context"
	"encoding/json"
	"fmt"
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/model"
	"go-ocs/internal/logging"
	"log"
	"sync"
	"time"
)

type ClassificationLoader func(ctx context.Context) (*model.Plan, error)

type ClassificationContainer struct {
	ClassificationPlan *model.Plan
	loader             ClassificationLoader
	shutdown           func()
	mu                 sync.RWMutex
}

func (c *ClassificationContainer) Shutdown() {
	if c.shutdown != nil {
		c.shutdown()
	}
}

func (c *ClassificationContainer) reloadClassificationPlan(ctx context.Context) {
	classificationPlan, err := c.loader(ctx)
	if err != nil {
		logging.Error("Failed to load active classification", "err", err)
		return
	}

	for i := range classificationPlan.ServiceTypes {
		st := &classificationPlan.ServiceTypes[i]
		st.ServiceWindowMap = make(map[string]struct{})
		if st.ServiceWindows != nil {
			for _, n := range st.ServiceWindows {
				st.ServiceWindowMap[n] = struct{}{}
			}
		}
	}

	c.mu.Lock()
	c.ClassificationPlan = classificationPlan
	c.mu.Unlock()

	logging.Info("Classification plan loaded")
}

func loadClassificationPlan(container *ClassificationContainer, appctx *appcontext.AppContext) {
	logging.Info("Loading the active classification plan")

	ctx, cancel := context.WithCancel(context.Background())
	container.shutdown = cancel

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		container.reloadClassificationPlan(ctx)

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			log.Println("reload loop stopped")
			return
		}
	}
}

func NewClassificationContainer(appctx *appcontext.AppContext) *ClassificationContainer {
	logging.Info("Creating new classification container")

	loader := func(ctx context.Context) (*model.Plan, error) {
		rec, err := appctx.Store.Q.FindActiveClassification(ctx)
		if err != nil {
			return nil, err
		}

		classificationPlan := &model.Plan{}
		if err = json.Unmarshal(rec.Plan, classificationPlan); err != nil {
			return nil, err
		}

		return classificationPlan, nil
	}

	container := ClassificationContainer{
		loader: loader,
	}

	go loadClassificationPlan(&container, appctx)
	return &container
}

func (c *ClassificationContainer) FetchClassificationPlan() (*model.Plan, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.ClassificationPlan == nil {
		return nil, fmt.Errorf("classification plan not loaded")
	}

	return c.ClassificationPlan, nil
}
