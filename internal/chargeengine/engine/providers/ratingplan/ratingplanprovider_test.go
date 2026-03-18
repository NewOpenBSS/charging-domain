package ratingplan

import (
	"context"
	"fmt"
	"go-ocs/internal/model"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRatePlanContainer_reloadRatePlans(t *testing.T) {
	planID := uuid.New()
	mockPlan := model.RatePlan{
		RatePlanID:   planID.String(),
		RatePlanName: "Test Plan",
	}

	loaderCalled := false
	mockLoader := func(ctx context.Context) (map[uuid.UUID]model.RatePlan, error) {
		loaderCalled = true
		return map[uuid.UUID]model.RatePlan{
			planID: mockPlan,
		}, nil
	}

	container := &RatePlanContainer{
		loader: mockLoader,
	}

	container.reloadRatePlans(context.Background())

	assert.True(t, loaderCalled)
	assert.NotNil(t, container.ratePlans)
	assert.Equal(t, 1, len(container.ratePlans))
	assert.Equal(t, mockPlan, container.ratePlans[planID])
}

func TestRatePlanContainer_reloadRatePlans_Error(t *testing.T) {
	mockLoader := func(ctx context.Context) (map[uuid.UUID]model.RatePlan, error) {
		return nil, fmt.Errorf("load error")
	}

	container := &RatePlanContainer{
		loader:    mockLoader,
		ratePlans: make(map[uuid.UUID]model.RatePlan),
	}

	container.reloadRatePlans(context.Background())

	// Should not overwrite if error
	assert.NotNil(t, container.ratePlans)
	assert.Equal(t, 0, len(container.ratePlans))
}

func TestRatePlanContainer_FindRatingPlan(t *testing.T) {
	planID := uuid.New()
	mockPlan := model.RatePlan{
		RatePlanID:   planID.String(),
		RatePlanName: "Test Plan",
	}

	container := &RatePlanContainer{
		ratePlans: map[uuid.UUID]model.RatePlan{
			planID: mockPlan,
		},
	}

	t.Run("PlanFound", func(t *testing.T) {
		plan, err := container.FindRatingPlan(planID)
		assert.NoError(t, err)
		assert.NotNil(t, plan)
		assert.Equal(t, mockPlan.RatePlanName, plan.RatePlanName)
	})

	t.Run("PlanNotFound", func(t *testing.T) {
		plan, err := container.FindRatingPlan(uuid.New())
		assert.Error(t, err)
		assert.Nil(t, plan)
		assert.Contains(t, err.Error(), "rate plan not found")
	})

	t.Run("NotLoaded", func(t *testing.T) {
		emptyContainer := &RatePlanContainer{}
		plan, err := emptyContainer.FindRatingPlan(planID)
		assert.Error(t, err)
		assert.Nil(t, plan)
		assert.Contains(t, err.Error(), "rate plan list not loaded")
	})
}

func TestRatePlanContainer_Shutdown(t *testing.T) {
	shutdownCalled := false
	container := &RatePlanContainer{
		shutdown: func() {
			shutdownCalled = true
		},
	}

	container.Shutdown()
	assert.True(t, shutdownCalled)
}

func TestRatePlanContainer_Concurrency(t *testing.T) {
	planID := uuid.New()
	mockPlan := model.RatePlan{
		RatePlanID:   planID.String(),
		RatePlanName: "Test Plan",
	}

	container := &RatePlanContainer{
		ratePlans: map[uuid.UUID]model.RatePlan{
			planID: mockPlan,
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			plan, err := container.FindRatingPlan(planID)
			if err == nil && plan != nil {
				_ = plan.RatePlanName
			}
		}()
	}
	wg.Wait()
}
