package housekeeping

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

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
