package quota

import (
	"context"

	"github.com/google/uuid"
)

func (m *QuotaManager) Release(ctx context.Context, subscriberId uuid.UUID, reservationId uuid.UUID) error {

	return m.executeWithQuota(ctx, subscriberId, func(q *Quota) error {
		q.ReleaseReservations(reservationId)
		return nil
	})
}
