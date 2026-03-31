package housekeeping

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

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
