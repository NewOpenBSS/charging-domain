package store

import (
	"context"
	"fmt"

	"go-ocs/internal/store/sqlc"
)

// ListDestinationGroupsParams holds the runtime-constructed SQL fragments for a dynamic
// destination group query. WhereSQL and Args come from filter.BuildWhere; OrderSQL from
// filter.BuildOrderBy.
type ListDestinationGroupsParams struct {
	WhereSQL string // e.g. "WHERE group_name ILIKE $1"  (empty = no filter)
	Args     []any  // positional args bound to the WHERE clause
	OrderSQL string // e.g. "ORDER BY group_name ASC"
	Limit    int
	Offset   int
}

// ListDestinationGroups executes a dynamic destination group query with optional filtering,
// sorting, and pagination. LIMIT and OFFSET are appended as the final positional arguments
// so their indices always follow those of the WHERE clause args.
func (s *Store) ListDestinationGroups(ctx context.Context, p ListDestinationGroupsParams) ([]sqlc.CarrierDestinationGroup, error) {
	limitIdx := len(p.Args) + 1
	offsetIdx := limitIdx + 1

	q := fmt.Sprintf(
		`SELECT group_name, region
		 FROM carrier_destination_group %s %s LIMIT $%d OFFSET $%d`,
		p.WhereSQL, p.OrderSQL, limitIdx, offsetIdx,
	)
	args := append(p.Args, p.Limit, p.Offset) //nolint:gocritic // intentional append to slice copy

	rows, err := s.querier.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []sqlc.CarrierDestinationGroup
	for rows.Next() {
		var g sqlc.CarrierDestinationGroup
		if err := rows.Scan(&g.GroupName, &g.Region); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// CountDestinationGroups executes a dynamic count query with optional filtering.
// whereSQL and args are produced by filter.BuildWhere.
func (s *Store) CountDestinationGroups(ctx context.Context, whereSQL string, args []any) (int64, error) {
	q := fmt.Sprintf("SELECT COUNT(*) FROM carrier_destination_group %s", whereSQL)
	var n int64
	if err := s.querier.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
