package store

import (
	"context"
	"fmt"

	"go-ocs/internal/store/sqlc"
)

// ListSourceGroupsParams holds the runtime-constructed SQL fragments for a dynamic
// source group query. WhereSQL and Args come from filter.BuildWhere; OrderSQL from
// filter.BuildOrderBy.
type ListSourceGroupsParams struct {
	WhereSQL string // e.g. "WHERE group_name ILIKE $1"  (empty = no filter)
	Args     []any  // positional args bound to the WHERE clause
	OrderSQL string // e.g. "ORDER BY group_name ASC"
	Limit    int
	Offset   int
}

// ListSourceGroups executes a dynamic source group query with optional filtering,
// sorting, and pagination. LIMIT and OFFSET are appended as the final positional arguments
// so their indices always follow those of the WHERE clause args.
func (s *Store) ListSourceGroups(ctx context.Context, p ListSourceGroupsParams) ([]sqlc.CarrierSourceGroup, error) {
	limitIdx := len(p.Args) + 1
	offsetIdx := limitIdx + 1

	q := fmt.Sprintf(
		`SELECT group_name, region
		 FROM carrier_source_group %s %s LIMIT $%d OFFSET $%d`,
		p.WhereSQL, p.OrderSQL, limitIdx, offsetIdx,
	)
	args := append(p.Args, p.Limit, p.Offset) //nolint:gocritic // intentional append to slice copy

	rows, err := s.querier.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []sqlc.CarrierSourceGroup
	for rows.Next() {
		var g sqlc.CarrierSourceGroup
		if err := rows.Scan(&g.GroupName, &g.Region); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// CountSourceGroups executes a dynamic count query with optional filtering.
// whereSQL and args are produced by filter.BuildWhere.
func (s *Store) CountSourceGroups(ctx context.Context, whereSQL string, args []any) (int64, error) {
	q := fmt.Sprintf("SELECT COUNT(*) FROM carrier_source_group %s", whereSQL)
	var n int64
	if err := s.querier.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
