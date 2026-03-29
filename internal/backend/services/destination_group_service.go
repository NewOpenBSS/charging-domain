package services

import (
	"context"
	"fmt"

	"go-ocs/internal/backend/filter"
	"go-ocs/internal/backend/graphql/model"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// destinationGroupColumns maps GraphQL field names (camelCase) to SQL column names (snake_case)
// for the carrier_destination_group table. Only fields present in this map are accepted as filter
// or sort keys — any other value is rejected by BuildWhere / BuildOrderBy to prevent SQL injection.
var destinationGroupColumns = map[string]string{
	"groupName": "group_name",
	"region":    "region",
}

// destinationGroupWildcardCols are the SQL column names searched when a wildcard term is provided.
var destinationGroupWildcardCols = []string{
	"group_name", "region",
}

// DestinationGroupService handles all business logic for the destination group resource.
type DestinationGroupService struct {
	store *store.Store
}

// NewDestinationGroupService creates a new DestinationGroupService backed by the supplied store.
func NewDestinationGroupService(s *store.Store) *DestinationGroupService {
	return &DestinationGroupService{store: s}
}

// ListDestinationGroups returns a filtered, sorted, paginated list of destination groups.
func (s *DestinationGroupService) ListDestinationGroups(
	ctx context.Context,
	page *model.PageRequest,
	filterReq *model.FilterRequest,
) ([]*model.DestinationGroup, error) {
	where, err := filter.BuildWhere(filterReq, destinationGroupColumns, destinationGroupWildcardCols)
	if err != nil {
		return nil, err
	}
	orderBy, err := filter.BuildOrderBy(page, "group_name", destinationGroupColumns)
	if err != nil {
		return nil, err
	}
	limit, offset := filter.PageOffset(page)

	rows, err := s.store.ListDestinationGroups(ctx, store.ListDestinationGroupsParams{
		WhereSQL: where.SQL,
		Args:     where.Args,
		OrderSQL: orderBy,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*model.DestinationGroup, 0, len(rows))
	for _, g := range rows {
		result = append(result, destinationGroupToModel(g))
	}
	return result, nil
}

// CountDestinationGroups returns the total count of destination groups matching the supplied filter.
func (s *DestinationGroupService) CountDestinationGroups(
	ctx context.Context,
	filterReq *model.FilterRequest,
) (int, error) {
	where, err := filter.BuildWhere(filterReq, destinationGroupColumns, destinationGroupWildcardCols)
	if err != nil {
		return 0, err
	}
	n, err := s.store.CountDestinationGroups(ctx, where.SQL, where.Args)
	return int(n), err
}

// DestinationGroupByGroupName returns a single destination group by group name, or nil if not found.
func (s *DestinationGroupService) DestinationGroupByGroupName(
	ctx context.Context,
	groupName string,
) (*model.DestinationGroup, error) {
	g, err := s.store.Q.DestinationGroupByGroupName(ctx, groupName)
	if err != nil {
		return nil, err
	}
	return destinationGroupToModel(g), nil
}

// CreateDestinationGroup persists a new destination group and returns the created record.
func (s *DestinationGroupService) CreateDestinationGroup(
	ctx context.Context,
	input model.DestinationGroupInput,
) (*model.DestinationGroup, error) {
	g, err := s.store.Q.CreateDestinationGroup(ctx, input.GroupName, input.Region)
	if err != nil {
		return nil, err
	}
	return destinationGroupToModel(g), nil
}

// UpdateDestinationGroup modifies an existing destination group by group name and returns the updated record.
// The groupName argument is authoritative; input.GroupName is used as the new region value only.
func (s *DestinationGroupService) UpdateDestinationGroup(
	ctx context.Context,
	groupName string,
	input model.DestinationGroupInput,
) (*model.DestinationGroup, error) {
	g, err := s.store.Q.UpdateDestinationGroup(ctx, groupName, input.Region)
	if err != nil {
		return nil, fmt.Errorf("update destination group %s: %w", groupName, err)
	}
	return destinationGroupToModel(g), nil
}

// DeleteDestinationGroup removes a destination group by group name. Returns true on success.
func (s *DestinationGroupService) DeleteDestinationGroup(ctx context.Context, groupName string) (bool, error) {
	if err := s.store.Q.DeleteDestinationGroup(ctx, groupName); err != nil {
		return false, fmt.Errorf("delete destination group %s: %w", groupName, err)
	}
	return true, nil
}

// destinationGroupToModel maps a sqlc.CarrierDestinationGroup row to the GraphQL model type.
func destinationGroupToModel(g sqlc.CarrierDestinationGroup) *model.DestinationGroup {
	return &model.DestinationGroup{
		GroupName: g.GroupName,
		Region:    g.Region,
	}
}
