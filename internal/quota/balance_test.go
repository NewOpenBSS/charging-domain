package quota

import (
	"context"
	"errors"
	"go-ocs/internal/charging"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func boolPtr(b bool) *bool       { return &b }
func unitPtr(u charging.UnitType) *charging.UnitType { return &u }

func decPtr(f float64) *decimal.Decimal {
	d := decimal.NewFromFloat(f)
	return &d
}

func timePtr(t time.Time) *time.Time { return &t }

// makeCounter builds a Counter with sensible defaults for use in tests.
func makeCounter(opts ...func(*Counter)) Counter {
	bal := decimal.NewFromInt(100)
	c := Counter{
		CounterID:    uuid.New(),
		ProductID:    uuid.New(),
		ProductName:  "test-product",
		UnitType:     charging.UNITS,
		Balance:      &bal,
		Reservations: make(map[uuid.UUID]Reservation),
		CanTransfer:  false,
		CanConvert:   false,
	}
	for _, o := range opts {
		o(&c)
	}
	return c
}

func withUnitType(u charging.UnitType) func(*Counter) {
	return func(c *Counter) { c.UnitType = u }
}

func withBalance(f float64) func(*Counter) {
	return func(c *Counter) {
		d := decimal.NewFromFloat(f)
		c.Balance = &d
	}
}

func withExpiry(t time.Time) func(*Counter) {
	return func(c *Counter) { c.Expiry = &t }
}

func withCanTransfer(v bool) func(*Counter) {
	return func(c *Counter) { c.CanTransfer = v }
}

func withCanConvert(v bool) func(*Counter) {
	return func(c *Counter) { c.CanConvert = v }
}

func withReservation(id uuid.UUID, units int64) func(*Counter) {
	return func(c *Counter) {
		u := units
		v := decimal.Zero
		c.Reservations[id] = Reservation{Units: &u, Value: &v}
	}
}

func withMonetaryReservation(id uuid.UUID, value float64) func(*Counter) {
	return func(c *Counter) {
		u := int64(0)
		v := decimal.NewFromFloat(value)
		c.Reservations[id] = Reservation{Units: &u, Value: &v}
	}
}

func loadedWith(counters ...Counter) *LoadedQuota {
	return &LoadedQuota{
		Quota:   &Quota{QuotaID: uuid.New(), Counters: counters},
		Version: time.Now(),
	}
}

// ---------------------------------------------------------------------------
// BalanceQuery.matches tests
// ---------------------------------------------------------------------------

func TestBalanceQuery_Matches(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)

	cases := []struct {
		name    string
		query   BalanceQuery
		counter Counter
		want    bool
	}{
		{
			name:    "empty query matches any counter",
			query:   BalanceQuery{},
			counter: makeCounter(withExpiry(future)),
			want:    true,
		},
		{
			name:    "UnitType matches",
			query:   BalanceQuery{UnitType: unitPtr(charging.MONETARY)},
			counter: makeCounter(withUnitType(charging.MONETARY)),
			want:    true,
		},
		{
			name:    "UnitType does not match",
			query:   BalanceQuery{UnitType: unitPtr(charging.MONETARY)},
			counter: makeCounter(withUnitType(charging.OCTETS)),
			want:    false,
		},
		{
			name:    "Transferable=true matches CanTransfer counter",
			query:   BalanceQuery{Transferable: boolPtr(true)},
			counter: makeCounter(withCanTransfer(true)),
			want:    true,
		},
		{
			name:    "Transferable=true does not match non-transferable counter",
			query:   BalanceQuery{Transferable: boolPtr(true)},
			counter: makeCounter(withCanTransfer(false)),
			want:    false,
		},
		{
			name:    "Transferable=false matches non-transferable counter",
			query:   BalanceQuery{Transferable: boolPtr(false)},
			counter: makeCounter(withCanTransfer(false)),
			want:    true,
		},
		{
			name:    "Convertible=true matches CanConvert counter",
			query:   BalanceQuery{Convertible: boolPtr(true)},
			counter: makeCounter(withCanConvert(true)),
			want:    true,
		},
		{
			name:    "Convertible=true does not match non-convertible counter",
			query:   BalanceQuery{Convertible: boolPtr(true)},
			counter: makeCounter(withCanConvert(false)),
			want:    false,
		},
		{
			name:  "combined UnitType and Transferable both match",
			query: BalanceQuery{UnitType: unitPtr(charging.MONETARY), Transferable: boolPtr(true)},
			counter: makeCounter(
				withUnitType(charging.MONETARY),
				withCanTransfer(true),
			),
			want: true,
		},
		{
			name:  "combined UnitType matches but Transferable does not",
			query: BalanceQuery{UnitType: unitPtr(charging.MONETARY), Transferable: boolPtr(true)},
			counter: makeCounter(
				withUnitType(charging.MONETARY),
				withCanTransfer(false),
			),
			want: false,
		},
		{
			name:  "all three filters match",
			query: BalanceQuery{UnitType: unitPtr(charging.UNITS), Transferable: boolPtr(true), Convertible: boolPtr(true)},
			counter: makeCounter(
				withUnitType(charging.UNITS),
				withCanTransfer(true),
				withCanConvert(true),
			),
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.query.matches(&tc.counter)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// GetBalance tests
// ---------------------------------------------------------------------------

func TestQuotaManager_GetBalance(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	t.Run("nil quota record returns empty slice", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{repo: mockRepo, retryLimit: 1}

		mockRepo.On("Load", ctx, uuid.Nil).Return(nil, nil)

		result, err := manager.GetBalance(ctx, now, uuid.Nil, BalanceQuery{})

		require.NoError(t, err)
		assert.Empty(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("load error is returned", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{repo: mockRepo, retryLimit: 1}

		subID := uuid.New()
		loadErr := errors.New("db unavailable")
		mockRepo.On("Load", ctx, subID).Return(nil, loadErr)

		result, err := manager.GetBalance(ctx, now, subID, BalanceQuery{})

		assert.ErrorIs(t, err, loadErr)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty query returns all non-expired non-zero counters", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{repo: mockRepo, retryLimit: 1}

		subID := uuid.New()
		loaded := loadedWith(
			makeCounter(withExpiry(future), withUnitType(charging.UNITS)),
			makeCounter(withExpiry(future), withUnitType(charging.MONETARY)),
		)
		mockRepo.On("Load", ctx, subID).Return(loaded, nil)

		result, err := manager.GetBalance(ctx, now, subID, BalanceQuery{})

		require.NoError(t, err)
		assert.Len(t, result, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("expired counters are excluded", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{repo: mockRepo, retryLimit: 1}

		subID := uuid.New()
		loaded := loadedWith(
			makeCounter(withExpiry(future), withUnitType(charging.UNITS)),
			makeCounter(withExpiry(past), withUnitType(charging.OCTETS)),
		)
		mockRepo.On("Load", ctx, subID).Return(loaded, nil)

		result, err := manager.GetBalance(ctx, now, subID, BalanceQuery{})

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, charging.UNITS, result[0].UnitType)
		mockRepo.AssertExpectations(t)
	})

	t.Run("zero balance counters are excluded", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{repo: mockRepo, retryLimit: 1}

		subID := uuid.New()
		loaded := loadedWith(
			makeCounter(withExpiry(future), withBalance(100)),
			makeCounter(withExpiry(future), withBalance(0)),
		)
		mockRepo.On("Load", ctx, subID).Return(loaded, nil)

		result, err := manager.GetBalance(ctx, now, subID, BalanceQuery{})

		require.NoError(t, err)
		assert.Len(t, result, 1)
		mockRepo.AssertExpectations(t)
	})

	t.Run("filter by UnitType", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{repo: mockRepo, retryLimit: 1}

		subID := uuid.New()
		loaded := loadedWith(
			makeCounter(withExpiry(future), withUnitType(charging.MONETARY)),
			makeCounter(withExpiry(future), withUnitType(charging.OCTETS)),
			makeCounter(withExpiry(future), withUnitType(charging.MONETARY)),
		)
		mockRepo.On("Load", ctx, subID).Return(loaded, nil)

		result, err := manager.GetBalance(ctx, now, subID, BalanceQuery{UnitType: unitPtr(charging.MONETARY)})

		require.NoError(t, err)
		assert.Len(t, result, 2)
		for _, r := range result {
			assert.Equal(t, charging.MONETARY, r.UnitType)
		}
		mockRepo.AssertExpectations(t)
	})

	t.Run("filter by Transferable=true", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{repo: mockRepo, retryLimit: 1}

		subID := uuid.New()
		loaded := loadedWith(
			makeCounter(withExpiry(future), withCanTransfer(true)),
			makeCounter(withExpiry(future), withCanTransfer(false)),
		)
		mockRepo.On("Load", ctx, subID).Return(loaded, nil)

		result, err := manager.GetBalance(ctx, now, subID, BalanceQuery{Transferable: boolPtr(true)})

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.True(t, result[0].CanTransfer)
		mockRepo.AssertExpectations(t)
	})

	t.Run("filter by Convertible=true", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{repo: mockRepo, retryLimit: 1}

		subID := uuid.New()
		loaded := loadedWith(
			makeCounter(withExpiry(future), withCanConvert(true)),
			makeCounter(withExpiry(future), withCanConvert(false)),
			makeCounter(withExpiry(future), withCanConvert(true)),
		)
		mockRepo.On("Load", ctx, subID).Return(loaded, nil)

		result, err := manager.GetBalance(ctx, now, subID, BalanceQuery{Convertible: boolPtr(true)})

		require.NoError(t, err)
		assert.Len(t, result, 2)
		for _, r := range result {
			assert.True(t, r.CanConvert)
		}
		mockRepo.AssertExpectations(t)
	})

	t.Run("combined UnitType and Transferable filter", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{repo: mockRepo, retryLimit: 1}

		subID := uuid.New()
		loaded := loadedWith(
			makeCounter(withExpiry(future), withUnitType(charging.MONETARY), withCanTransfer(true)),
			makeCounter(withExpiry(future), withUnitType(charging.MONETARY), withCanTransfer(false)),
			makeCounter(withExpiry(future), withUnitType(charging.UNITS), withCanTransfer(true)),
		)
		mockRepo.On("Load", ctx, subID).Return(loaded, nil)

		result, err := manager.GetBalance(ctx, now, subID, BalanceQuery{
			UnitType:     unitPtr(charging.MONETARY),
			Transferable: boolPtr(true),
		})

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, charging.MONETARY, result[0].UnitType)
		assert.True(t, result[0].CanTransfer)
		mockRepo.AssertExpectations(t)
	})

	t.Run("no query matches returns empty slice", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{repo: mockRepo, retryLimit: 1}

		subID := uuid.New()
		loaded := loadedWith(
			makeCounter(withExpiry(future), withCanTransfer(false)),
		)
		mockRepo.On("Load", ctx, subID).Return(loaded, nil)

		result, err := manager.GetBalance(ctx, now, subID, BalanceQuery{Transferable: boolPtr(true)})

		require.NoError(t, err)
		assert.Empty(t, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("service unit available balance deducts reservations", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{repo: mockRepo, retryLimit: 1}

		subID := uuid.New()
		resID := uuid.New()
		loaded := loadedWith(
			makeCounter(
				withExpiry(future),
				withUnitType(charging.UNITS),
				withBalance(100),
				withReservation(resID, 30),
			),
		)
		mockRepo.On("Load", ctx, subID).Return(loaded, nil)

		result, err := manager.GetBalance(ctx, now, subID, BalanceQuery{})

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.True(t, decimal.NewFromInt(100).Equal(result[0].TotalBalance), "TotalBalance: expected 100, got %s", result[0].TotalBalance)
		assert.True(t, decimal.NewFromInt(70).Equal(result[0].AvailableBalance), "AvailableBalance: expected 70, got %s", result[0].AvailableBalance)
		mockRepo.AssertExpectations(t)
	})

	t.Run("monetary available balance deducts reservations", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{repo: mockRepo, retryLimit: 1}

		subID := uuid.New()
		resID := uuid.New()
		loaded := loadedWith(
			makeCounter(
				withExpiry(future),
				withUnitType(charging.MONETARY),
				withBalance(200),
				withMonetaryReservation(resID, 50),
			),
		)
		mockRepo.On("Load", ctx, subID).Return(loaded, nil)

		result, err := manager.GetBalance(ctx, now, subID, BalanceQuery{})

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.True(t, decimal.NewFromInt(200).Equal(result[0].TotalBalance), "TotalBalance: expected 200, got %s", result[0].TotalBalance)
		assert.True(t, decimal.NewFromInt(150).Equal(result[0].AvailableBalance), "AvailableBalance: expected 150, got %s", result[0].AvailableBalance)
		mockRepo.AssertExpectations(t)
	})

	t.Run("counter with nil expiry is included", func(t *testing.T) {
		mockRepo := new(MockRepository)
		manager := &QuotaManager{repo: mockRepo, retryLimit: 1}

		subID := uuid.New()
		// makeCounter does not set Expiry by default — nil expiry means no expiry.
		loaded := loadedWith(makeCounter())
		mockRepo.On("Load", ctx, subID).Return(loaded, nil)

		result, err := manager.GetBalance(ctx, now, subID, BalanceQuery{})

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Nil(t, result[0].Expiry)
		mockRepo.AssertExpectations(t)
	})
}
