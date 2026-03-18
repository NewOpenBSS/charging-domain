package quota

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"go-ocs/internal/store"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository interface {
	Load(ctx context.Context, subscriberID uuid.UUID) (*LoadedQuota, error)
	Create(ctx context.Context, subscriberID uuid.UUID) (*LoadedQuota, error)
	// Save persists the loaded quota. now is the version timestamp used for optimistic locking
	// and must be provided by the caller to ensure deterministic versioning.
	Save(ctx context.Context, loaded *LoadedQuota, now time.Time) error
}

type QuotaRepository struct {
	store store.Store
}

type PessimisticLockError struct {
	Message string
}

func (e *PessimisticLockError) Error() string {
	return e.Message
}

func CreatePessimisticLockError(msg string) *PessimisticLockError {
	return &PessimisticLockError{
		Message: msg,
	}
}

func NewQuotaRepository(store store.Store) *QuotaRepository {
	return &QuotaRepository{store: store}
}

func (r *QuotaRepository) Load(ctx context.Context, subscriberId uuid.UUID) (*LoadedQuota, error) {

	rec, err := r.store.Q.FindQuota(ctx, toUUID(subscriberId))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // quota does not exist yet
		}

		return nil, err
	}

	quota := Quota{}
	err = json.Unmarshal(rec.Quota, &quota)
	if err != nil {
		return nil, err
	}
	quota.QuotaID = rec.QuotaID.Bytes

	l := &LoadedQuota{
		Quota:   &quota,
		Version: rec.LastModified.Time,
	}

	return l, nil
}

func (r *QuotaRepository) Create(ctx context.Context, subscriberId uuid.UUID) (*LoadedQuota, error) {

	quota := NewEmptyQuota()

	q, err := json.Marshal(quota)
	if err != nil {
		return nil, err
	}

	rec, err := r.store.Q.CreateQuota(ctx,
		toUUID(quota.QuotaID),
		toUUID(subscriberId),
		q,
	)
	if err != nil {
		return nil, err
	}

	l := &LoadedQuota{
		Quota:   quota,
		Version: rec.LastModified.Time,
	}

	return l, nil
}

// Save persists the loaded quota to the database using now as the new version timestamp.
// Returns a PessimisticLockError if another writer has modified the quota since it was loaded.
func (r *QuotaRepository) Save(ctx context.Context, loaded *LoadedQuota, now time.Time) error {

	q, err := json.Marshal(loaded.Quota)
	if err != nil {
		return err
	}

	rows, err := r.store.Q.UpdateQuota(ctx, toUUID(loaded.Quota.QuotaID), pgtype.Timestamptz{Time: loaded.Version, Valid: true}, q, pgtype.Timestamptz{Time: now, Valid: true})
	if err != nil {
		return err
	}
	if rows == 0 {
		return CreatePessimisticLockError("failed to update quota")
	}

	loaded.Version = now

	return nil
}

func toUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}
