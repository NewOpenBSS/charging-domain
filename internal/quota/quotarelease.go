package quota

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Release releases an active quota reservation for a subscriber and persists the updated quota.
func (m *QuotaManager) Release(ctx context.Context, subscriberId uuid.UUID, reservationId uuid.UUID) error {

	return m.executeWithQuota(ctx, time.Now().UTC(), subscriberId, func(q *Quota) error {
		q.ReleaseReservations(reservationId)
		return nil
	})
}
