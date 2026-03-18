package classificationplan

import (
	"context"
	"errors"
	"go-ocs/internal/model"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClassificationContainer_reloadClassificationPlan(t *testing.T) {
	expectedPlan := &model.ClassificationPlan{
		RuleSetId: "test-ruleset",
		ServiceTypes: []model.ServiceType{
			{
				ServiceType:    "VOICE",
				ServiceWindows: []string{"morning", "evening"},
			},
			{
				ServiceType: "DATA",
			},
		},
	}

	loader := func(ctx context.Context) (*model.ClassificationPlan, error) {
		return expectedPlan, nil
	}

	container := &ClassificationContainer{
		loader: loader,
	}

	container.reloadClassificationPlan(context.Background())

	assert.NotNil(t, container.ClassificationPlan)
	assert.Equal(t, "test-ruleset", container.ClassificationPlan.RuleSetId)
	assert.Len(t, container.ClassificationPlan.ServiceTypes, 2)

	// Check if ServiceWindowMap is populated
	voiceST := container.ClassificationPlan.ServiceTypes[0]
	assert.Equal(t, "VOICE", voiceST.ServiceType)
	assert.NotNil(t, voiceST.ServiceWindowMap)
	assert.Contains(t, voiceST.ServiceWindowMap, "morning")
	assert.Contains(t, voiceST.ServiceWindowMap, "evening")

	dataST := container.ClassificationPlan.ServiceTypes[1]
	assert.Equal(t, "DATA", dataST.ServiceType)
	assert.NotNil(t, dataST.ServiceWindowMap)
	assert.Empty(t, dataST.ServiceWindowMap)
}

func TestClassificationContainer_reloadClassificationPlan_Error(t *testing.T) {
	loader := func(ctx context.Context) (*model.ClassificationPlan, error) {
		return nil, errors.New("load error")
	}

	container := &ClassificationContainer{
		ClassificationPlan: &model.ClassificationPlan{RuleSetId: "initial"},
		loader:             loader,
	}

	container.reloadClassificationPlan(context.Background())

	// Plan should remain unchanged on error
	assert.NotNil(t, container.ClassificationPlan)
	assert.Equal(t, "initial", container.ClassificationPlan.RuleSetId)
}

func TestClassificationContainer_FetchClassificationPlan(t *testing.T) {
	t.Run("PlanLoaded", func(t *testing.T) {
		expectedPlan := &model.ClassificationPlan{RuleSetId: "test-ruleset"}
		container := &ClassificationContainer{
			ClassificationPlan: expectedPlan,
		}

		plan, err := container.FetchClassificationPlan()
		assert.NoError(t, err)
		assert.Equal(t, expectedPlan, plan)
	})

	t.Run("PlanNotLoaded", func(t *testing.T) {
		container := &ClassificationContainer{}
		plan, err := container.FetchClassificationPlan()
		assert.Error(t, err)
		assert.Nil(t, plan)
		assert.Equal(t, "classification plan not loaded", err.Error())
	})
}

func TestClassificationContainer_Shutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func() {
		shutdownCalled = true
	}

	container := &ClassificationContainer{
		shutdown: shutdown,
	}

	container.Shutdown()
	assert.True(t, shutdownCalled)
}

func TestClassificationContainer_Concurrency(t *testing.T) {
	plan := &model.ClassificationPlan{RuleSetId: "test-ruleset"}
	container := &ClassificationContainer{
		ClassificationPlan: plan,
		loader: func(ctx context.Context) (*model.ClassificationPlan, error) {
			time.Sleep(10 * time.Millisecond)
			return plan, nil
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = container.FetchClassificationPlan()
		}()
		go func() {
			defer wg.Done()
			container.reloadClassificationPlan(context.Background())
		}()
	}
	wg.Wait()
}
