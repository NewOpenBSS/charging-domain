package housekeeping

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"go-ocs/internal/logging"
)

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
