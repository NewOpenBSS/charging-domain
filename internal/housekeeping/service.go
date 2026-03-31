package housekeeping

import (
	"context"
	"time"

	"github.com/google/uuid"

	"go-ocs/internal/store"
)

// QuotaExpirer processes expired quota for a single subscriber.
type QuotaExpirer interface {
	ProcessExpiredQuota(ctx context.Context, now time.Time, subscriberID uuid.UUID) error
}

// HousekeepingService implements the four housekeeping operations:
// quota expiry, deleting stale charging sessions, purging old trace records,
// and removing superseded ACTIVE rate plan versions.
type HousekeepingService struct {
	store        *store.Store
	quotaExpirer QuotaExpirer
}

// NewHousekeepingService creates a new HousekeepingService backed by the given store
// and quota expirer. The quotaExpirer may be nil if quota expiry is not needed.
func NewHousekeepingService(s *store.Store, qe QuotaExpirer) *HousekeepingService {
	return &HousekeepingService{store: s, quotaExpirer: qe}
}
