package consumer

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"go-ocs/internal/events"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// WholesaleStorer is the minimal interface the WholesaleContractConsumer requires
// to persist wholesaler changes. Defined at the point of consumption so that
// tests can substitute a mock without requiring a real database connection.
type WholesaleStorer interface {
	// UpsertWholesaler inserts or updates a wholesaler from a PROVISIONED event.
	UpsertWholesaler(ctx context.Context, event *events.WholesaleContractProvisionedEvent) error

	// DeregisterWholesaler handles deregistering: deletes the wholesaler if it has
	// no remaining subscribers, otherwise marks it inactive.
	DeregisterWholesaler(ctx context.Context, wholesalerID uuid.UUID) error

	// SuspendWholesaler sets the wholesaler's active flag based on the suspend state.
	// suspend=true → active=false; suspend=false → active=true.
	SuspendWholesaler(ctx context.Context, wholesalerID uuid.UUID, suspend bool) error
}

// StoreWholesaleAdapter adapts *store.Store to the WholesaleStorer interface
// by delegating to the sqlc-generated wholesaler queries.
type StoreWholesaleAdapter struct {
	s *store.Store
}

// NewStoreWholesaleAdapter wraps a store.Store to satisfy WholesaleStorer.
func NewStoreWholesaleAdapter(s *store.Store) *StoreWholesaleAdapter {
	return &StoreWholesaleAdapter{s: s}
}

// UpsertWholesaler inserts a new wholesaler row or updates all mutable fields
// when the wholesaler already exists in the shadow table.
func (a *StoreWholesaleAdapter) UpsertWholesaler(ctx context.Context, event *events.WholesaleContractProvisionedEvent) error {
	pgRateLimit, err := floatToNumeric(event.RateLimit)
	if err != nil {
		return fmt.Errorf("UpsertWholesaler: converting rateLimit: %w", err)
	}

	return a.s.Q.UpsertWholesaler(ctx, sqlc.UpsertWholesalerParams{
		ID:          uuidToPgtype(event.WholesalerID),
		Active:      event.Active,
		LegalName:   event.LegalName,
		DisplayName: event.DisplayName,
		Realm:       event.Realm,
		Hosts:       event.Hosts,
		Nchfurl:     event.NchfUrl,
		Ratelimit:   pgRateLimit,
		ContractID:  uuidToPgtype(event.ContractID),
		RateplanID:  uuidToPgtype(event.RatePlanID),
	})
}

// DeregisterWholesaler handles the deregistering lifecycle: if the wholesaler has
// no remaining subscribers it is hard-deleted; otherwise it is marked inactive.
// The two DB calls are intentionally not wrapped in a transaction — a race between
// two concurrent deregistering events is extremely low-probability and accepted.
func (a *StoreWholesaleAdapter) DeregisterWholesaler(ctx context.Context, wholesalerID uuid.UUID) error {
	count, err := a.s.Q.CountSubscribersByWholesaler(ctx, uuidToPgtype(wholesalerID))
	if err != nil {
		return fmt.Errorf("DeregisterWholesaler: counting subscribers: %w", err)
	}

	if count == 0 {
		return a.s.Q.DeleteWholesaler(ctx, uuidToPgtype(wholesalerID))
	}
	return a.s.Q.SetWholesalerActive(ctx, uuidToPgtype(wholesalerID), false)
}

// SuspendWholesaler sets the wholesaler's active flag based on the suspend state.
// A suspend=true event marks the wholesaler inactive; suspend=false re-activates it.
func (a *StoreWholesaleAdapter) SuspendWholesaler(ctx context.Context, wholesalerID uuid.UUID, suspend bool) error {
	return a.s.Q.SetWholesalerActive(ctx, uuidToPgtype(wholesalerID), !suspend)
}

// floatToNumeric converts a float64 to pgtype.Numeric via the decimal string
// representation, avoiding any floating-point arithmetic on the value.
func floatToNumeric(f float64) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.Scan(decimal.NewFromFloat(f).String()); err != nil {
		return pgtype.Numeric{}, err
	}
	return n, nil
}
