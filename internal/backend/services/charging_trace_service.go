package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"go-ocs/internal/backend/filter"
	"go-ocs/internal/backend/graphql/model"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// chargingTraceColumns maps GraphQL field names (camelCase) to SQL column names
// (snake_case) for the charging_trace table. Only fields listed here are accepted
// as filter or sort keys — any other value is rejected by BuildWhere / BuildOrderBy.
var chargingTraceColumns = map[string]string{
	"traceId":       "trace_id",
	"createdAt":     "created_at",
	"chargingId":    "charging_id",
	"sequenceNr":    "sequence_nr",
	"msisdn":        "msisdn",
	"executionTime": "execution_time",
}

// chargingTraceWildcardCols are the SQL column names searched when a wildcard term
// is provided. Mirrors the Java ChargingTraceEntity.WILDCARD_FIELDS.
var chargingTraceWildcardCols = []string{
	"charging_id", "msisdn",
}

// ChargingTraceService handles all read-only business logic for the charging trace
// resource. It bridges the GraphQL layer and the store layer.
type ChargingTraceService struct {
	store *store.Store
}

// NewChargingTraceService creates a new ChargingTraceService backed by the supplied store.
func NewChargingTraceService(s *store.Store) *ChargingTraceService {
	return &ChargingTraceService{store: s}
}

// ListChargingTraces returns a filtered, sorted, paginated list of charging traces.
// Results are ordered by created_at descending by default (most recent first).
func (s *ChargingTraceService) ListChargingTraces(
	ctx context.Context,
	page *model.PageRequest,
	filterReq *model.FilterRequest,
) ([]*model.ChargingTrace, error) {
	where, err := filter.BuildWhere(filterReq, chargingTraceColumns, chargingTraceWildcardCols)
	if err != nil {
		return nil, err
	}
	orderBy, err := filter.BuildOrderBy(page, "created_at", chargingTraceColumns)
	if err != nil {
		return nil, err
	}
	limit, offset := filter.PageOffset(page)

	rows, err := s.store.ListChargingTraces(ctx, store.ListChargingTracesParams{
		WhereSQL: where.SQL,
		Args:     where.Args,
		OrderSQL: orderBy,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*model.ChargingTrace, 0, len(rows))
	for _, t := range rows {
		result = append(result, chargingTraceToModel(t))
	}
	return result, nil
}

// CountChargingTrace returns the total count of charging traces matching the supplied filter.
func (s *ChargingTraceService) CountChargingTrace(
	ctx context.Context,
	filterReq *model.FilterRequest,
) (int, error) {
	where, err := filter.BuildWhere(filterReq, chargingTraceColumns, chargingTraceWildcardCols)
	if err != nil {
		return 0, err
	}
	n, err := s.store.CountChargingTraces(ctx, where.SQL, where.Args)
	return int(n), err
}

// ChargingTraceById returns a single charging trace by its UUID string.
// Returns an error if traceId is not a valid UUID or if no record is found.
func (s *ChargingTraceService) ChargingTraceById(
	ctx context.Context,
	traceId string,
) (*model.ChargingTrace, error) {
	parsed, err := uuid.Parse(traceId)
	if err != nil {
		return nil, fmt.Errorf("invalid traceId %q: %w", traceId, err)
	}

	pgUUID := pgtype.UUID{
		Bytes: parsed,
		Valid: true,
	}

	t, err := s.store.FindChargingTraceByTraceId(ctx, pgUUID)
	if err != nil {
		return nil, err
	}
	return chargingTraceToModel(t), nil
}

// chargingTraceToModel maps a sqlc.ChargingTrace row to the GraphQL model type.
// createdAt is formatted as RFC3339 since the DateTime scalar resolves to string.
// request and response JSONB payloads are mapped directly to their string representations.
func chargingTraceToModel(t sqlc.ChargingTrace) *model.ChargingTrace {
	m := &model.ChargingTrace{
		TraceID:       formatUUID(t.TraceID),
		Request:       string(t.Request),
		Response:      string(t.Response),
		ExecutionTime: int(t.ExecutionTime),
		ChargingID:    t.ChargingID,
		SequenceNr:    int(t.SequenceNr),
		Msisdn:        t.Msisdn,
	}
	if t.CreatedAt.Valid {
		s := t.CreatedAt.Time.UTC().Format(time.RFC3339)
		m.CreatedAt = &s
	}
	return m
}

// formatUUID converts a pgtype.UUID to its canonical hyphenated string representation.
func formatUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}
