package main

import (
	"sync"

	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

type WholesalerLimiterStore struct {
	mu sync.Mutex
	m  map[uuid.UUID]*rate.Limiter
}

func NewWholesalerLimiterStore() *WholesalerLimiterStore {
	return &WholesalerLimiterStore{
		m: make(map[uuid.UUID]*rate.Limiter),
	}
}

// Get returns the limiter for this wholesaler, creating it if needed.
// rps is requests-per-second. burst controls short spikes.
func (s *WholesalerLimiterStore) Get(id uuid.UUID, rps float64, burst int) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	if lim, ok := s.m[id]; ok {
		// If rps can change at runtime, you might want to update it here:
		// lim.SetLimit(rate.Limit(rps))
		return lim
	}

	lim := rate.NewLimiter(rate.Limit(rps), burst)
	s.m[id] = lim
	return lim
}
