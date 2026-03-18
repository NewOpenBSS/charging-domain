package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go-ocs/internal/backend/filter"
	"go-ocs/internal/backend/graphql/model"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// numberPlanColumns maps GraphQL field names (camelCase) to SQL column names (snake_case)
// for the number_plan table. Only fields present in this map are accepted as filter or
// sort keys — any other value is rejected to prevent SQL injection.
var numberPlanColumns = map[string]string{
	"numberId":     "number_id",
	"name":         "name",
	"plmn":         "plmn",
	"numberRange":  "number_range",
	"numberLength": "number_length",
	"modifiedOn":   "modified_on",
}

// numberPlanWildcardCols are the SQL column names searched when a wildcard term is provided.
// Mirrors Java NumberPlanEntity.WILDCARD_FIELDS.
var numberPlanWildcardCols = []string{"name", "number_range"}

// NumberPlanService handles all business logic for the number plan resource.
type NumberPlanService struct {
	store *store.Store
}

// NewNumberPlanService creates a new NumberPlanService backed by the supplied store.
func NewNumberPlanService(s *store.Store) *NumberPlanService {
	return &NumberPlanService{store: s}
}

// ListNumberPlans returns a filtered, sorted, paginated list of number plan rows.
func (s *NumberPlanService) ListNumberPlans(
	ctx context.Context,
	page *model.PageRequest,
	filterReq *model.FilterRequest,
) ([]*model.NumberPlan, error) {
	where, err := filter.BuildWhere(filterReq, numberPlanColumns, numberPlanWildcardCols)
	if err != nil {
		return nil, err
	}
	orderBy, err := filter.BuildOrderBy(page, "number_range", numberPlanColumns)
	if err != nil {
		return nil, err
	}
	limit, offset := filter.PageOffset(page)

	rows, err := s.store.ListNumberPlans(ctx, store.ListNumberPlansParams{
		WhereSQL: where.SQL,
		Args:     where.Args,
		OrderSQL: orderBy,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*model.NumberPlan, 0, len(rows))
	for _, r := range rows {
		result = append(result, numberPlanToModel(r))
	}
	return result, nil
}

// CountNumberPlans returns the total count of number plan rows matching the supplied filter.
func (s *NumberPlanService) CountNumberPlans(
	ctx context.Context,
	filterReq *model.FilterRequest,
) (int, error) {
	where, err := filter.BuildWhere(filterReq, numberPlanColumns, numberPlanWildcardCols)
	if err != nil {
		return 0, err
	}
	n, err := s.store.CountNumberPlans(ctx, where.SQL, where.Args)
	return int(n), err
}

// GetNumberPlan returns a single number plan by its primary-key ID, or nil if not found.
func (s *NumberPlanService) GetNumberPlan(ctx context.Context, numberID string) (*model.NumberPlan, error) {
	id, err := parseNumberPlanID(numberID)
	if err != nil {
		return nil, err
	}
	r, err := s.store.Q.FindNumberPlanByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return numberPlanToModel(r), nil
}

// CreateNumberPlan inserts a new number plan row and returns it.
func (s *NumberPlanService) CreateNumberPlan(
	ctx context.Context,
	input model.NumberPlanInput,
) (*model.NumberPlan, error) {
	name := ""
	if input.Name != nil {
		name = *input.Name
	}

	r, err := s.store.Q.CreateNumberPlan(ctx, name, input.Plmn, input.NumberRange, int32(input.NumberLength))
	if err != nil {
		return nil, fmt.Errorf("create number plan: %w", err)
	}
	return numberPlanToModel(r), nil
}

// UpdateNumberPlan updates all mutable fields of an existing number plan row.
func (s *NumberPlanService) UpdateNumberPlan(
	ctx context.Context,
	numberID string,
	input model.NumberPlanInput,
) (*model.NumberPlan, error) {
	id, err := parseNumberPlanID(numberID)
	if err != nil {
		return nil, err
	}

	name := ""
	if input.Name != nil {
		name = *input.Name
	}

	r, err := s.store.Q.UpdateNumberPlan(ctx, sqlc.UpdateNumberPlanParams{
		NumberID:     id,
		Name:         name,
		Plmn:         input.Plmn,
		NumberRange:  input.NumberRange,
		NumberLength: int32(input.NumberLength),
	})
	if err != nil {
		return nil, fmt.Errorf("update number plan %s: %w", numberID, err)
	}
	return numberPlanToModel(r), nil
}

// DeleteNumberPlan permanently deletes a number plan row. Returns true on success.
func (s *NumberPlanService) DeleteNumberPlan(ctx context.Context, numberID string) (bool, error) {
	id, err := parseNumberPlanID(numberID)
	if err != nil {
		return false, err
	}
	if err := s.store.Q.DeleteNumberPlan(ctx, id); err != nil {
		return false, fmt.Errorf("delete number plan %s: %w", numberID, err)
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// parseNumberPlanID converts a GraphQL ID string to an int64 primary key.
func parseNumberPlanID(id string) (int64, error) {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numberId %q: must be a numeric ID", id)
	}
	return n, nil
}

// numberPlanToModel maps a sqlc.NumberPlan row to the GraphQL model type.
func numberPlanToModel(r sqlc.NumberPlan) *model.NumberPlan {
	m := &model.NumberPlan{
		NumberID:     strconv.FormatInt(r.NumberID, 10),
		Name:         r.Name,
		Plmn:         r.Plmn,
		NumberRange:  r.NumberRange,
		NumberLength: int(r.NumberLength),
	}
	if r.ModifiedOn.Valid {
		s := r.ModifiedOn.Time.Format(time.RFC3339)
		m.ModifiedOn = &s
	}
	return m
}
