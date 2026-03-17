package store

import (
	"context"
	"fmt"

	"go-ocs/internal/store/sqlc"
)

// ListCarriersParams holds the runtime-constructed SQL fragments for a dynamic carrier query.
// WhereSQL and Args come from filter.BuildWhere; OrderSQL from filter.BuildOrderBy.
type ListCarriersParams struct {
	WhereSQL string // e.g. "WHERE carrier_name ILIKE $1"  (empty = no filter)
	Args     []any  // positional args bound to the WHERE clause
	OrderSQL string // e.g. "ORDER BY plmn ASC"
	Limit    int
	Offset   int
}

// ListCarriers executes a dynamic carrier query with optional filtering, sorting,
// and pagination. LIMIT and OFFSET are appended as the final positional arguments
// so their indices always follow those of the WHERE clause args.
func (s *Store) ListCarriers(ctx context.Context, p ListCarriersParams) ([]sqlc.Carrier, error) {
	limitIdx := len(p.Args) + 1
	offsetIdx := limitIdx + 1

	q := fmt.Sprintf(
		`SELECT plmn, modified_on, mcc, mnc, carrier_name, source_group,
		        destination_group, country_name, iso
		 FROM carrier %s %s LIMIT $%d OFFSET $%d`,
		p.WhereSQL, p.OrderSQL, limitIdx, offsetIdx,
	)
	args := append(p.Args, p.Limit, p.Offset) //nolint:gocritic // intentional append to slice copy

	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var carriers []sqlc.Carrier
	for rows.Next() {
		var c sqlc.Carrier
		if err := rows.Scan(
			&c.Plmn, &c.ModifiedOn, &c.Mcc, &c.Mnc,
			&c.CarrierName, &c.SourceGroup, &c.DestinationGroup,
			&c.CountryName, &c.Iso,
		); err != nil {
			return nil, err
		}
		carriers = append(carriers, c)
	}
	return carriers, rows.Err()
}

// CountCarriers executes a dynamic count query with optional filtering.
// whereSQL and args are produced by filter.BuildWhere.
func (s *Store) CountCarriers(ctx context.Context, whereSQL string, args []any) (int64, error) {
	q := fmt.Sprintf("SELECT COUNT(*) FROM carrier %s", whereSQL)
	var n int64
	if err := s.DB.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
