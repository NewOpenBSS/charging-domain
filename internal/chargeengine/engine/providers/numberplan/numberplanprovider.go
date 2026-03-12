package numberplan

import (
	"context"
	"fmt"
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/ocserrors"
	"go-ocs/internal/logging"
	"go-ocs/internal/store/sqlc"
	"log"
	"strings"
	"sync"
	"time"
)

type NumberPlanLoader func(ctx context.Context) ([]sqlc.AllNumbersRow, error)

type NumberPlanContainer struct {
	numbers          map[string]sqlc.AllNumbersRow
	loader           NumberPlanLoader
	shutdown         func()
	nationalDialCode string
	mu               sync.RWMutex
}

func (c *NumberPlanContainer) Shutdown() {
	if c.shutdown != nil {
		c.shutdown()
	}
}

func (c *NumberPlanContainer) reloadNumbers(ctx context.Context, nationalDialCode string) {
	numbers, err := c.loader(ctx)
	if err != nil {
		logging.Error("Failed to load number plans", "err", err)
		return
	}

	numberList := make(map[string]sqlc.AllNumbersRow)
	for _, number := range numbers {
		numberList[number.Plmn] = number
	}

	c.mu.Lock()
	c.numbers = numberList
	c.nationalDialCode = nationalDialCode
	c.mu.Unlock()

	logging.Info("Number plans loaded")
}

func loadNumbers(container *NumberPlanContainer, appctx *appcontext.AppContext) {
	logging.Info("Loading the all number plans")

	ctx, cancel := context.WithCancel(context.Background())
	container.shutdown = cancel

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		container.reloadNumbers(ctx, appctx.Config.Engine.NationalDialCode)

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			log.Println("reload loop stopped")
			return
		}
	}
}

func NewNumberPlanContainer(appctx *appcontext.AppContext) *NumberPlanContainer {
	logging.Info("Creating new number plan container")

	loader := func(ctx context.Context) ([]sqlc.AllNumbersRow, error) {
		return appctx.Store.Q.AllNumbers(ctx)
	}

	container := NumberPlanContainer{
		loader: loader,
	}

	go loadNumbers(&container, appctx)
	return &container
}

func (c *NumberPlanContainer) FindNumberPlan(number string) (*sqlc.AllNumbersRow, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.numbers == nil {
		return nil, fmt.Errorf("number plan list not loaded")
	}

	if number == "" {
		return nil, ocserrors.CreateUnknownCarrier("number is empty")
	}

	if number[0:1] == "0" {
		number = c.nationalDialCode + number[1:]
	}

	var match *sqlc.AllNumbersRow = nil
	matchLen := 0
	for _, row := range c.numbers {
		if strings.HasPrefix(number, row.NumberRange) {
			if len(row.NumberRange) > matchLen {
				matchLen = len(row.NumberRange)
				match = &row
			}
		}
	}

	if match != nil {
		return match, nil
	}

	return nil, ocserrors.CreateUnknownCarrier(fmt.Sprintf("number plan for %s not found", number))
}

func (c *NumberPlanContainer) FindCarrierByDestination(number string) string {

	numberPlan, err := c.FindNumberPlan(number)
	if err != nil {
		return "?"
	}

	return numberPlan.DestinationGroup
}
