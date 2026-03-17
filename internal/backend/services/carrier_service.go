package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"go-ocs/internal/backend/filter"
	"go-ocs/internal/backend/graphql/model"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// carrierColumns maps GraphQL field names (camelCase) to SQL column names (snake_case)
// for the carrier table. Only fields present in this map are accepted as filter or sort
// keys — any other value is rejected by BuildWhere / BuildOrderBy to prevent SQL injection.
var carrierColumns = map[string]string{
	"plmn":             "plmn",
	"mcc":              "mcc",
	"mnc":              "mnc",
	"carrierName":      "carrier_name",
	"sourceGroup":      "source_group",
	"destinationGroup": "destination_group",
	"countryName":      "country_name",
	"iso":              "iso",
	"modifiedOn":       "modified_on",
}

// carrierWildcardCols are the SQL column names searched when a wildcard term is provided.
// Mirrors Java CarrierEntity.WILDCARD_FIELDS.
var carrierWildcardCols = []string{
	"plmn", "carrier_name", "country_name", "iso", "source_group", "destination_group",
}

// CarrierService handles all business logic for the carrier resource.
type CarrierService struct {
	store *store.Store
}

// NewCarrierService creates a new CarrierService backed by the supplied store.
func NewCarrierService(s *store.Store) *CarrierService {
	return &CarrierService{store: s}
}

// ListCarriers returns a filtered, sorted, paginated list of carriers.
func (s *CarrierService) ListCarriers(
	ctx context.Context,
	page *model.PageRequest,
	filterReq *model.FilterRequest,
) ([]*model.Carrier, error) {
	where, err := filter.BuildWhere(filterReq, carrierColumns, carrierWildcardCols)
	if err != nil {
		return nil, err
	}
	orderBy, err := filter.BuildOrderBy(page, "plmn", carrierColumns)
	if err != nil {
		return nil, err
	}
	limit, offset := filter.PageOffset(page)

	rows, err := s.store.ListCarriers(ctx, store.ListCarriersParams{
		WhereSQL: where.SQL,
		Args:     where.Args,
		OrderSQL: orderBy,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*model.Carrier, 0, len(rows))
	for _, c := range rows {
		result = append(result, carrierToModel(c))
	}
	return result, nil
}

// CarrierByPlmn returns a single carrier by PLMN, or nil if not found.
func (s *CarrierService) CarrierByPlmn(ctx context.Context, plmn string) (*model.Carrier, error) {
	c, err := s.store.Q.CarrierByPlmn(ctx, plmn)
	if err != nil {
		return nil, err
	}
	return carrierToModel(c), nil
}

// CountCarriers returns the total count of carriers matching the supplied filter.
func (s *CarrierService) CountCarriers(
	ctx context.Context,
	filterReq *model.FilterRequest,
) (int, error) {
	where, err := filter.BuildWhere(filterReq, carrierColumns, carrierWildcardCols)
	if err != nil {
		return 0, err
	}
	n, err := s.store.CountCarriers(ctx, where.SQL, where.Args)
	return int(n), err
}

// CreateCarrier persists a new carrier and returns the created record.
func (s *CarrierService) CreateCarrier(
	ctx context.Context,
	input model.CarrierInput,
) (*model.Carrier, error) {
	c, err := s.store.Q.CreateCarrier(ctx, sqlc.CreateCarrierParams{
		Plmn:             input.Plmn,
		Mcc:              input.Mcc,
		Mnc:              textFromPtr(input.Mnc),
		CarrierName:      input.CarrierName,
		SourceGroup:      input.SourceGroup,
		DestinationGroup: input.DestinationGroup,
		CountryName:      input.CountryName,
		Iso:              input.Iso,
	})
	if err != nil {
		return nil, err
	}
	return carrierToModel(c), nil
}

// UpdateCarrier modifies an existing carrier by PLMN and returns the updated record.
// The plmn argument is authoritative; input.Plmn is ignored for the WHERE predicate.
func (s *CarrierService) UpdateCarrier(
	ctx context.Context,
	plmn string,
	input model.CarrierInput,
) (*model.Carrier, error) {
	c, err := s.store.Q.UpdateCarrier(ctx, sqlc.UpdateCarrierParams{
		Plmn:             plmn,
		Mcc:              input.Mcc,
		Mnc:              textFromPtr(input.Mnc),
		CarrierName:      input.CarrierName,
		SourceGroup:      input.SourceGroup,
		DestinationGroup: input.DestinationGroup,
		CountryName:      input.CountryName,
		Iso:              input.Iso,
	})
	if err != nil {
		return nil, fmt.Errorf("update carrier %s: %w", plmn, err)
	}
	return carrierToModel(c), nil
}

// DeleteCarrier removes a carrier by PLMN. Returns true on success.
func (s *CarrierService) DeleteCarrier(ctx context.Context, plmn string) (bool, error) {
	if err := s.store.Q.DeleteCarrier(ctx, plmn); err != nil {
		return false, fmt.Errorf("delete carrier %s: %w", plmn, err)
	}
	return true, nil
}

// carrierToModel maps a sqlc.Carrier row to the GraphQL model type.
// ModifiedOn is formatted as RFC3339 since the DateTime scalar resolves to string.
func carrierToModel(c sqlc.Carrier) *model.Carrier {
	m := &model.Carrier{
		Plmn:             c.Plmn,
		Mcc:              c.Mcc,
		CarrierName:      c.CarrierName,
		SourceGroup:      c.SourceGroup,
		DestinationGroup: c.DestinationGroup,
		CountryName:      c.CountryName,
		Iso:              c.Iso,
	}
	if c.Mnc.Valid {
		m.Mnc = &c.Mnc.String
	}
	if c.ModifiedOn.Valid {
		s := c.ModifiedOn.Time.Format(time.RFC3339)
		m.ModifiedOn = &s
	}
	return m
}

// textFromPtr converts a nullable *string to a pgtype.Text for sqlc parameters.
func textFromPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}
