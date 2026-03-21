package consumer

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"go-ocs/internal/events"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// StoreSubscriberAdapter adapts *store.Store to the SubscriberStorer interface
// by delegating to the sqlc-generated Insert, Update, and Delete queries.
type StoreSubscriberAdapter struct {
	s *store.Store
}

// NewStoreSubscriberAdapter wraps a store.Store to satisfy SubscriberStorer.
func NewStoreSubscriberAdapter(s *store.Store) *StoreSubscriberAdapter {
	return &StoreSubscriberAdapter{s: s}
}

// InsertSubscriber inserts a new subscriber row from a CREATED event.
func (a *StoreSubscriberAdapter) InsertSubscriber(ctx context.Context, event *events.SubscriberEvent) error {
	return a.s.Q.InsertSubscriber(ctx, sqlc.InsertSubscriberParams{
		SubscriberID:     uuidToPgtype(event.SubscriberID),
		RateplanID:       uuidToPgtype(event.RatePlanID),
		CustomerID:       uuidToPgtype(event.CustomerID),
		WholesaleID:      uuidToPgtype(event.WholesaleID),
		Msisdn:           event.Msisdn,
		Iccid:            event.Iccid,
		ContractID:       uuidToPgtype(event.ContractID),
		Status:           event.Status,
		AllowOobCharging: event.AllowOobCharging,
	})
}

// UpdateSubscriber updates all mutable subscriber fields from an UPDATED,
// MSISDN_SWAP, or SIM_SWAP event.
func (a *StoreSubscriberAdapter) UpdateSubscriber(ctx context.Context, event *events.SubscriberEvent) error {
	return a.s.Q.UpdateSubscriber(ctx, sqlc.UpdateSubscriberParams{
		SubscriberID:     uuidToPgtype(event.SubscriberID),
		RateplanID:       uuidToPgtype(event.RatePlanID),
		CustomerID:       uuidToPgtype(event.CustomerID),
		WholesaleID:      uuidToPgtype(event.WholesaleID),
		Msisdn:           event.Msisdn,
		Iccid:            event.Iccid,
		ContractID:       uuidToPgtype(event.ContractID),
		Status:           event.Status,
		AllowOobCharging: event.AllowOobCharging,
	})
}

// DeleteSubscriber hard-deletes a subscriber by subscriber_id, then attempts to
// cascade-delete the associated wholesaler if it is inactive and has no remaining
// subscribers. The cascade is atomic at the SQL level — it is a no-op when the
// wholesaler is still active or still has other subscribers.
func (a *StoreSubscriberAdapter) DeleteSubscriber(ctx context.Context, subscriberID uuid.UUID, wholesaleID uuid.UUID) error {
	if err := a.s.Q.DeleteSubscriber(ctx, uuidToPgtype(subscriberID)); err != nil {
		return err
	}
	return a.s.Q.DeleteInactiveWholesalerIfEmpty(ctx, uuidToPgtype(wholesaleID))
}

// uuidToPgtype converts a google/uuid.UUID to its pgtype.UUID equivalent.
func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}
