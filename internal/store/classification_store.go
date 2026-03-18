package store

import (
	"context"
	"fmt"

	"go-ocs/internal/store/sqlc"
)

// ListClassificationsParams holds runtime-constructed SQL fragments for a dynamic
// classification query. WhereSQL and Args come from filter.BuildWhere;
// OrderSQL from filter.BuildOrderBy.
type ListClassificationsParams struct {
	WhereSQL string // e.g. "WHERE status = $1"  (empty = no filter)
	Args     []any  // positional args bound to the WHERE clause
	OrderSQL string // e.g. "ORDER BY name ASC"
	Limit    int
	Offset   int
}

// ListClassifications executes a dynamic classification query with optional filtering,
// sorting, and pagination. LIMIT and OFFSET are appended as the final positional
// arguments so their indices always follow those of the WHERE clause args.
func (s *Store) ListClassifications(
	ctx context.Context,
	p ListClassificationsParams,
) ([]sqlc.Classification, error) {
	limitIdx := len(p.Args) + 1
	offsetIdx := limitIdx + 1

	q := fmt.Sprintf(
		`SELECT classification_id, name, created_on, effective_time,
		        created_by, approved_by, status, plan
		 FROM classification %s %s LIMIT $%d OFFSET $%d`,
		p.WhereSQL, p.OrderSQL, limitIdx, offsetIdx,
	)
	args := append(p.Args, p.Limit, p.Offset) //nolint:gocritic // intentional append to slice copy

	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []sqlc.Classification
	for rows.Next() {
		var c sqlc.Classification
		if err := rows.Scan(
			&c.ClassificationID, &c.Name, &c.CreatedOn, &c.EffectiveTime,
			&c.CreatedBy, &c.ApprovedBy, &c.Status, &c.Plan,
		); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

// CountClassifications executes a dynamic count query with optional filtering.
// whereSQL and args are produced by filter.BuildWhere.
func (s *Store) CountClassifications(ctx context.Context, whereSQL string, args []any) (int64, error) {
	q := fmt.Sprintf("SELECT COUNT(*) FROM classification %s", whereSQL)
	var n int64
	if err := s.DB.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
