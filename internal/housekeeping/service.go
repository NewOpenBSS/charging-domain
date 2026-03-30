package housekeeping

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"go-ocs/internal/logging"
	"go-ocs/internal/store"
)

// HousekeepingService implements the three non-quota housekeeping operations:
// deleting stale charging sessions, purging old trace records, and removing
// superseded ACTIVE rate plan versions.
type HousekeepingService struct {
	store *store.Store
}

// NewHousekeepingService creates a new HousekeepingService backed by the given store.
func NewHousekeepingService(s *store.Store) *HousekeepingService {
	return &HousekeepingService{store: s}
}

// CleanStaleSessions deletes charging_data rows with modified_on older than the threshold
// (i.e. before now - threshold). Returns the number of rows deleted.
func (s *HousekeepingService) CleanStaleSessions(ctx context.Context, now time.Time, threshold time.Duration) (int64, error) {
	before := pgtype.Timestamptz{Time: now.Add(-threshold), Valid: true}
	n, err := s.store.Q.DeleteStaleChargingData(ctx, before)
	if err != nil {
		return 0, fmt.Errorf("delete stale charging data: %w", err)
	}
	return n, nil
}

// PurgeOldTraces deletes charging_trace rows with created_at older than the threshold
// (i.e. before now - threshold). Returns the number of rows deleted.
func (s *HousekeepingService) PurgeOldTraces(ctx context.Context, now time.Time, threshold time.Duration) (int64, error) {
	before := pgtype.Timestamptz{Time: now.Add(-threshold), Valid: true}
	n, err := s.store.Q.DeleteOldChargingTrace(ctx, before)
	if err != nil {
		return 0, fmt.Errorf("delete old charging trace: %w", err)
	}
	return n, nil
}

// CleanupSupersededRatePlans deletes superseded ACTIVE rate plan versions whose effective_at
// is older than the threshold and which have been replaced by a newer ACTIVE version for the
// same plan_id. DRAFT and PENDING versions are never touched. Returns the number of rows deleted.
func (s *HousekeepingService) CleanupSupersededRatePlans(ctx context.Context, now time.Time, threshold time.Duration) (int64, error) {
	before := pgtype.Timestamptz{Time: now.Add(-threshold), Valid: true}
	versions, err := s.store.Q.ListSupersededRatePlanVersions(ctx, before)
	if err != nil {
		return 0, fmt.Errorf("list superseded rate plan versions: %w", err)
	}
	var deleted int64
	for _, v := range versions {
		logging.Info("Deleting superseded rate plan version", "id", v.ID, "plan_id", v.PlanID, "effective_at", v.EffectiveAt)
		if err := s.store.Q.DeleteRatePlanVersionById(ctx, v.ID); err != nil {
			return deleted, fmt.Errorf("delete rate plan version %d: %w", v.ID, err)
		}
		deleted++
	}
	return deleted, nil
}
