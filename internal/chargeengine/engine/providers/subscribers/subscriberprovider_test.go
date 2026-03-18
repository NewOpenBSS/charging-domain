package subscribers

import (
	"context"
	"errors"
	"go-ocs/internal/model"
	"go-ocs/internal/store/sqlc"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestSubscriberContainer_FindSubscriber(t *testing.T) {
	subscriberID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	contractID := pgtype.UUID{Bytes: [16]byte{2}, Valid: true}
	rateplanID := pgtype.UUID{Bytes: [16]byte{3}, Valid: true}
	wholesalerID := pgtype.UUID{Bytes: [16]byte{4}, Valid: true}
	wholesalerRateplanID := pgtype.UUID{Bytes: [16]byte{5}, Valid: true}

	t.Run("Cache miss - successful load", func(t *testing.T) {
		msisdn := "1234567890"
		callCount := 0
		loader := func(ctx context.Context, m string) (sqlc.FindSubscriberWithWholesalerByMSISDNRow, error) {
			callCount++
			assert.Equal(t, msisdn, m)
			return sqlc.FindSubscriberWithWholesalerByMSISDNRow{
				SubscriberID:         subscriberID,
				ContractID:           contractID,
				RateplanID:           rateplanID,
				WholesalerID:         wholesalerID,
				WholesalerRateplanID: wholesalerRateplanID,
				WholesalerActive:     true,
				Status:               "ACTIVE",
				AllowOobCharging:     true,
			}, nil
		}

		container := &SubscriberContainer{
			subscribers: make(map[string]*model.Subscriber),
			loader:      loader,
		}

		sub, err := container.FindSubscriber(msisdn)
		assert.NoError(t, err)
		assert.NotNil(t, sub)
		assert.Equal(t, msisdn, sub.Msisdn)
		assert.Equal(t, uuid.UUID(subscriberID.Bytes), sub.SubscriberId)
		assert.Equal(t, uuid.UUID(contractID.Bytes), sub.ContractId)
		assert.Equal(t, uuid.UUID(rateplanID.Bytes), sub.RatePlanId)
		assert.Equal(t, uuid.UUID(wholesalerID.Bytes), sub.WholesaleId)
		assert.Equal(t, uuid.UUID(wholesalerRateplanID.Bytes), sub.WholesalerRatePlanId)
		assert.True(t, sub.AllowOOBCharging)
		assert.Equal(t, 1, callCount)

		// Test cache hit
		sub2, err := container.FindSubscriber(msisdn)
		assert.NoError(t, err)
		assert.Equal(t, sub, sub2)
		assert.Equal(t, 1, callCount)
	})

	t.Run("Loader returns error", func(t *testing.T) {
		msisdn := "999"
		loader := func(ctx context.Context, m string) (sqlc.FindSubscriberWithWholesalerByMSISDNRow, error) {
			return sqlc.FindSubscriberWithWholesalerByMSISDNRow{}, errors.New("db error")
		}

		container := &SubscriberContainer{
			subscribers: make(map[string]*model.Subscriber),
			loader:      loader,
		}

		sub, err := container.FindSubscriber(msisdn)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load subscriber")
		assert.Nil(t, sub)
	})

	t.Run("Wholesaler inactive", func(t *testing.T) {
		msisdn := "111"
		loader := func(ctx context.Context, m string) (sqlc.FindSubscriberWithWholesalerByMSISDNRow, error) {
			return sqlc.FindSubscriberWithWholesalerByMSISDNRow{
				WholesalerActive: false,
				Status:           "ACTIVE",
			}, nil
		}

		container := &SubscriberContainer{
			subscribers: make(map[string]*model.Subscriber),
			loader:      loader,
		}

		sub, err := container.FindSubscriber(msisdn)
		assert.Error(t, err)
		assert.Equal(t, "subscriber is not active", err.Error())
		assert.Nil(t, sub)
	})

	t.Run("Subscriber status not ACTIVE", func(t *testing.T) {
		msisdn := "222"
		loader := func(ctx context.Context, m string) (sqlc.FindSubscriberWithWholesalerByMSISDNRow, error) {
			return sqlc.FindSubscriberWithWholesalerByMSISDNRow{
				WholesalerActive: true,
				Status:           "SUSPENDED",
			}, nil
		}

		container := &SubscriberContainer{
			subscribers: make(map[string]*model.Subscriber),
			loader:      loader,
		}

		sub, err := container.FindSubscriber(msisdn)
		assert.Error(t, err)
		assert.Equal(t, "subscriber is not active", err.Error())
		assert.Nil(t, sub)
	})
}

func TestSubscriberContainer_Shutdown(t *testing.T) {
	shutdownCalled := false
	container := &SubscriberContainer{
		shutdown: func() {
			shutdownCalled = true
		},
	}

	container.Shutdown()
	assert.True(t, shutdownCalled)
}

func TestSubscriberContainer_ClearCacheLoop(t *testing.T) {
	// We need to test the clearCache function which is started in a goroutine in NewSubscriberContainer
	// But clearCache is not exported. We can test it by calling it directly or via NewSubscriberContainer.

	container := &SubscriberContainer{
		subscribers: make(map[string]*model.Subscriber),
		mu:          sync.RWMutex{},
	}
	container.subscribers["test"] = &model.Subscriber{Msisdn: "test"}

	// Start clearCache manually with a shorter ticker if we could, but it's hard-coded to 30m.
	// However, it clears immediately once when started.

	// We'll test that it clears immediately and then we can stop it.
	_, cancel := context.WithCancel(context.Background())
	container.shutdown = cancel

	// Since clearCache has an infinite loop with a ticker, we can't easily wait for it to "tick"
	// but we can check the initial clear.

	// To test the loop properly, we might need to refactor clearCache to take an interval,
	// but I should avoid modifying production code if possible.

	// Wait, clearCache clears the map at the BEGINNING of the loop.
	/*
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
	*/

	go clearCache(container)

	// Give it a moment to run the first iteration
	time.Sleep(10 * time.Millisecond)

	container.mu.RLock()
	assert.Empty(t, container.subscribers)
	container.mu.RUnlock()

	container.Shutdown()
}

func TestSubscriberContainer_Concurrency(t *testing.T) {
	loader := func(ctx context.Context, m string) (sqlc.FindSubscriberWithWholesalerByMSISDNRow, error) {
		return sqlc.FindSubscriberWithWholesalerByMSISDNRow{
			WholesalerActive: true,
			Status:           "ACTIVE",
			Msisdn:           m,
		}, nil
	}

	container := &SubscriberContainer{
		subscribers: make(map[string]*model.Subscriber),
		loader:      loader,
	}

	const numGoroutines = 10
	const numRequests = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numRequests; j++ {
				msisdn := "user1" // All same to test cache contention
				if j%2 == 0 {
					msisdn = "user2"
				}
				_, _ = container.FindSubscriber(msisdn)
			}
		}(i)
	}

	wg.Wait()

	assert.Len(t, container.subscribers, 2)
}
