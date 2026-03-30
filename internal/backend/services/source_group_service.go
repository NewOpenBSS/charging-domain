package services

import (
	"context"
	"fmt"

	"go-ocs/internal/backend/filter"
	"go-ocs/internal/backend/graphql/model"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// sourceGroupColumns maps GraphQL field names (camelCase) to SQL column names (snake_case)
// for the carrier_source_group table. Only fields present in this map are accepted as filter
// or sort keys — any other value is rejected by BuildWhere / BuildOrderBy to prevent SQL injection.
var sourceGroupColumns = map[string]string{
	"groupName": "group_name",
	"region":    "region",
}

// sourceGroupWildcardCols are the SQL column names searched when a wildcard term is provided.
var sourceGroupWildcardCols = []string{
	"group_name", "region",
}

// SourceGroupService handles all business logic for the source group resource.
type SourceGroupService struct {
	store *store.Store
}

// NewSourceGroupService creates a new SourceGroupService backed by the supplied store.
func NewSourceGroupService(s *store.Store) *SourceGroupService {
	return &SourceGroupService{store: s}
}

// ListSourceGroups returns a filtered, sorted, paginated list of source groups.
func (s *SourceGroupService) ListSourceGroups(
	ctx context.Context,
	page *model.PageRequest,
	filterReq *model.FilterRequest,
) ([]*model.SourceGroup, error) {
	where, err := filter.BuildWhere(filterReq, sourceGroupColumns, sourceGroupWildcardCols)
	if err != nil {
		return nil, err
	}
	orderBy, err := filter.BuildOrderBy(page, "group_name", sourceGroupColumns)
	if err != nil {
		return nil, err
	}
	limit, offset := filter.PageOffset(page)

	rows, err := s.store.ListSourceGroups(ctx, store.ListSourceGroupsParams{
		WhereSQL: where.SQL,
		Args:     where.Args,
		OrderSQL: orderBy,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*model.SourceGroup, 0, len(rows))
	for _, g := range rows {
		result = append(result, sourceGroupToModel(g))
	}
	return result, nil
}

// CountSourceGroups returns the total count of source groups matching the supplied filter.
func (s *SourceGroupService) CountSourceGroups(
	ctx context.Context,
	filterReq *model.FilterRequest,
) (int, error) {
	where, err := filter.BuildWhere(filterReq, sourceGroupColumns, sourceGroupWildcardCols)
	if err != nil {
		return 0, err
	}
	n, err := s.store.CountSourceGroups(ctx, where.SQL, where.Args)
	return int(n), err
}

// SourceGroupByGroupName returns a single source group by group name, or nil if not found.
func (s *SourceGroupService) SourceGroupByGroupName(
	ctx context.Context,
	groupName string,
) (*model.SourceGroup, error) {
	g, err := s.store.Q.SourceGroupByGroupName(ctx, groupName)
	if err != nil {
		return nil, err
	}
	return sourceGroupToModel(g), nil
}

// CreateSourceGroup persists a new source group and returns the created record.
func (s *SourceGroupService) CreateSourceGroup(
	ctx context.Context,
	input model.SourceGroupInput,
) (*model.SourceGroup, error) {
	g, err := s.store.Q.CreateSourceGroup(ctx, input.GroupName, input.Region)
	if err != nil {
		return nil, err
	}
	return sourceGroupToModel(g), nil
}

// UpdateSourceGroup modifies an existing source group by group name and returns the updated record.
// The groupName argument is authoritative; input.Region is the value to update.
func (s *SourceGroupService) UpdateSourceGroup(
	ctx context.Context,
	groupName string,
	input model.SourceGroupInput,
) (*model.SourceGroup, error) {
	g, err := s.store.Q.UpdateSourceGroup(ctx, groupName, input.Region)
	if err != nil {
		return nil, fmt.Errorf("update source group %s: %w", groupName, err)
	}
	return sourceGroupToModel(g), nil
}

// DeleteSourceGroup removes a source group by group name. Returns true on success.
func (s *SourceGroupService) DeleteSourceGroup(ctx context.Context, groupName string) (bool, error) {
	if err := s.store.Q.DeleteSourceGroup(ctx, groupName); err != nil {
		return false, fmt.Errorf("delete source group %s: %w", groupName, err)
	}
	return true, nil
}

// sourceGroupToModel maps a sqlc.CarrierSourceGroup row to the GraphQL model type.
func sourceGroupToModel(g sqlc.CarrierSourceGroup) *model.SourceGroup {
	return &model.SourceGroup{
		GroupName: g.GroupName,
		Region:    g.Region,
	}
}
