package housekeeping

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"go-ocs/internal/logging"
)

// ExpireQuotas finds all subscribers with expired quotas and processes each one via the
// configured QuotaExpirer. Returns the number of subscribers successfully processed.
// Processing continues even if individual subscribers fail — errors are logged and the
// last error is returned alongside the successful count.
func (s *HousekeepingService) ExpireQuotas(ctx context.Context, now time.Time) (int, error) {
	if s.quotaExpirer == nil {
		return 0, fmt.Errorf("quota expirer not configured")
	}

	subscribers, err := s.store.Q.FindExpiredQuotaSubscribers(ctx, pgtype.Timestamptz{Time: now, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("find expired quota subscribers: %w", err)
	}

	var (
		count  int
		lastErr error
	)

	for _, pgID := range subscribers {
		subscriberID := uuid.UUID(pgID.Bytes)
		if err := s.quotaExpirer.ProcessExpiredQuota(ctx, now, subscriberID); err != nil {
			logging.Error("Quota expiry: failed to process subscriber", "err", err)
			lastErr = err
			// continue — process remaining subscribers
		} else {
			count++
		}
	}

	return count, lastErr
}
