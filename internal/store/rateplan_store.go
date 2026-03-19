package store

import (
	"context"
	"fmt"

	"go-ocs/internal/store/sqlc"
)

// ListRatePlansParams holds runtime-constructed SQL fragments for a dynamic rate plan query.
// WhereSQL and Args come from filter.BuildWhere; OrderSQL from filter.BuildOrderBy.
type ListRatePlansParams struct {
	WhereSQL string // e.g. "WHERE plan_status = $1"  (empty = no filter)
	Args     []any  // positional args bound to the WHERE clause
	OrderSQL string // e.g. "ORDER BY plan_name ASC"
	Limit    int
	Offset   int
}

// ListRatePlans executes a dynamic rate plan query with optional filtering, sorting, and
// pagination. All versions of all plans are included — no DISTINCT ON plan_id.
// LIMIT and OFFSET are appended as the final positional arguments.
func (s *Store) ListRatePlans(ctx context.Context, p ListRatePlansParams) ([]sqlc.Rateplan, error) {
	limitIdx := len(p.Args) + 1
	offsetIdx := limitIdx + 1

	q := fmt.Sprintf(
		`SELECT id, plan_id, modified_at, plan_type, wholesale_id, plan_name,
		        rateplan, plan_status, created_by, approved_by, effective_at
		 FROM rateplan %s %s LIMIT $%d OFFSET $%d`,
		p.WhereSQL, p.OrderSQL, limitIdx, offsetIdx,
	)
	args := append(p.Args, p.Limit, p.Offset) //nolint:gocritic // intentional append to slice copy

	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []sqlc.Rateplan
	for rows.Next() {
		var r sqlc.Rateplan
		if err := rows.Scan(
			&r.ID, &r.PlanID, &r.ModifiedAt, &r.PlanType, &r.WholesaleID,
			&r.PlanName, &r.Rateplan, &r.PlanStatus, &r.CreatedBy,
			&r.ApprovedBy, &r.EffectiveAt,
		); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

// CountRatePlans executes a dynamic count query with optional filtering.
// whereSQL and args are produced by filter.BuildWhere.
func (s *Store) CountRatePlans(ctx context.Context, whereSQL string, args []any) (int64, error) {
	q := fmt.Sprintf("SELECT COUNT(*) FROM rateplan %s", whereSQL)
	var n int64
	if err := s.DB.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
