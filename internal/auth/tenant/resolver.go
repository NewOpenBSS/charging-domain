// Package tenant provides multi-tenant resolution for the charging-backend.
// It maps the HTTP Host header to a wholesaler UUID by maintaining an in-memory
// lookup table built from the wholesaler.hosts database column. The table is
// refreshed on a configurable interval so changes propagate without a restart.
package tenant

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"go-ocs/internal/logging"
	"go-ocs/internal/store"
)

// Resolver maintains the hostname → wholesaler UUID mapping and provides the
// HTTP middleware that resolves each incoming request to a tenant.
type Resolver struct {
	store    *store.Store
	interval time.Duration

	mu       sync.RWMutex
	hostsMap map[string]pgtype.UUID // hostname (lower-case, no port) → wholesaler UUID
}

// NewResolver creates a Resolver backed by the supplied store. The lookup table
// is populated immediately on construction; Start must be called to enable
// periodic refresh.
func NewResolver(s *store.Store, interval time.Duration) *Resolver {
	r := &Resolver{
		store:    s,
		interval: interval,
		hostsMap: make(map[string]pgtype.UUID),
	}
	// Initial load — ignore error; an empty map is safe (middleware becomes a no-op).
	r.refresh(context.Background())
	return r
}

// Start launches the background goroutine that periodically reloads the
// hostname lookup table. It runs until ctx is cancelled.
func (r *Resolver) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.refresh(ctx)
			}
		}
	}()
}

// ResolveHost returns the wholesaler UUID for the given hostname.
// The port suffix is stripped before lookup. Returns false if no match.
func (r *Resolver) ResolveHost(host string) (pgtype.UUID, bool) {
	bare := stripPort(host)
	r.mu.RLock()
	defer r.mu.RUnlock()
	uid, ok := r.hostsMap[strings.ToLower(bare)]
	return uid, ok
}

// refresh reloads the wholesaler table from the database and rebuilds the
// hostname → UUID map. It acquires the write lock only during the final swap
// so read throughput is not affected during the reload.
func (r *Resolver) refresh(ctx context.Context) {
	rows, err := r.store.Q.AllWholesalers(ctx)
	if err != nil {
		logging.Warn("tenant resolver: failed to reload wholesalers", "err", err)
		return
	}

	newMap := make(map[string]pgtype.UUID, len(rows)*2)
	for _, row := range rows {
		for _, h := range row.Hosts {
			h = strings.ToLower(strings.TrimSpace(h))
			if h != "" {
				newMap[h] = row.ID
			}
		}
	}

	r.mu.Lock()
	r.hostsMap = newMap
	r.mu.Unlock()

	logging.Debug("tenant resolver: reloaded wholesaler host map", "entries", len(newMap))
}

// stripPort removes the port portion from a host string (e.g. "example.com:8080" → "example.com").
func stripPort(host string) string {
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}
