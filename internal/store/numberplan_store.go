package store

import (
	"context"
	"fmt"

	"go-ocs/internal/store/sqlc"
)

// ListNumberPlansParams holds runtime-constructed SQL fragments for a dynamic number plan query.
// WhereSQL and Args come from filter.BuildWhere; OrderSQL from filter.BuildOrderBy.
type ListNumberPlansParams struct {
	WhereSQL string // e.g. "WHERE name = $1"  (empty = no filter)
	Args     []any  // positional args bound to the WHERE clause
	OrderSQL string // e.g. "ORDER BY number_range ASC"
	Limit    int
	Offset   int
}

// ListNumberPlans executes a dynamic number plan query with optional filtering, sorting, and
// pagination against the number_plan table.
func (s *Store) ListNumberPlans(ctx context.Context, p ListNumberPlansParams) ([]sqlc.NumberPlan, error) {
	limitIdx := len(p.Args) + 1
	offsetIdx := limitIdx + 1

	q := fmt.Sprintf(
		`SELECT number_id, modified_on, name, plmn, number_range, number_length
		 FROM number_plan %s %s LIMIT $%d OFFSET $%d`,
		p.WhereSQL, p.OrderSQL, limitIdx, offsetIdx,
	)
	args := append(p.Args, p.Limit, p.Offset) //nolint:gocritic // intentional append to slice copy

	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []sqlc.NumberPlan
	for rows.Next() {
		var r sqlc.NumberPlan
		if err := rows.Scan(
			&r.NumberID, &r.ModifiedOn, &r.Name, &r.Plmn, &r.NumberRange, &r.NumberLength,
		); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

// CountNumberPlans executes a dynamic count query against the number_plan table with optional filtering.
// whereSQL and args are produced by filter.BuildWhere.
func (s *Store) CountNumberPlans(ctx context.Context, whereSQL string, args []any) (int64, error) {
	q := fmt.Sprintf("SELECT COUNT(*) FROM number_plan %s", whereSQL)
	var n int64
	if err := s.DB.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
