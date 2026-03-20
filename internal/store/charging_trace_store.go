package store

import (
	"context"
	"fmt"

	"go-ocs/internal/store/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

// ListChargingTracesParams holds the runtime-constructed SQL fragments for a dynamic
// charging trace query. WhereSQL and Args come from filter.BuildWhere;
// OrderSQL from filter.BuildOrderBy.
type ListChargingTracesParams struct {
	WhereSQL string // e.g. "WHERE charging_id ILIKE $1"  (empty = no filter)
	Args     []any  // positional args bound to the WHERE clause
	OrderSQL string // e.g. "ORDER BY created_at DESC"
	Limit    int
	Offset   int
}

// ListChargingTraces executes a dynamic charging trace query with optional filtering,
// sorting, and pagination. LIMIT and OFFSET are appended as the final positional
// arguments so their indices always follow those of the WHERE clause args.
func (s *Store) ListChargingTraces(ctx context.Context, p ListChargingTracesParams) ([]sqlc.ChargingTrace, error) {
	limitIdx := len(p.Args) + 1
	offsetIdx := limitIdx + 1

	q := fmt.Sprintf(
		`SELECT trace_id, created_at, request, response, execution_time,
		        charging_id, sequence_nr, msisdn
		 FROM charging_trace %s %s LIMIT $%d OFFSET $%d`,
		p.WhereSQL, p.OrderSQL, limitIdx, offsetIdx,
	)
	args := append(p.Args, p.Limit, p.Offset) //nolint:gocritic // intentional append to slice copy

	rows, err := s.querier.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var traces []sqlc.ChargingTrace
	for rows.Next() {
		var t sqlc.ChargingTrace
		if err := rows.Scan(
			&t.TraceID,
			&t.CreatedAt,
			&t.Request,
			&t.Response,
			&t.ExecutionTime,
			&t.ChargingID,
			&t.SequenceNr,
			&t.Msisdn,
		); err != nil {
			return nil, err
		}
		traces = append(traces, t)
	}
	return traces, rows.Err()
}

// CountChargingTraces executes a dynamic count query with optional filtering.
// whereSQL and args are produced by filter.BuildWhere.
func (s *Store) CountChargingTraces(ctx context.Context, whereSQL string, args []any) (int64, error) {
	q := fmt.Sprintf("SELECT COUNT(*) FROM charging_trace %s", whereSQL)
	var n int64
	if err := s.querier.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// FindChargingTraceByTraceId fetches a single charging trace record by its UUID primary key.
// Returns the full ChargingTrace including request and response JSONB payloads.
func (s *Store) FindChargingTraceByTraceId(ctx context.Context, traceID pgtype.UUID) (sqlc.ChargingTrace, error) {
	return s.Q.FindChargingTraceByTraceId(ctx, traceID)
}
