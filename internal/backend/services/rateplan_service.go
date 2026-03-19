package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"go-ocs/internal/auth/tenant"
	"go-ocs/internal/backend/filter"
	"go-ocs/internal/backend/graphql/model"
	"go-ocs/internal/charging"
	gomodel "go-ocs/internal/model"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// ratePlanColumns maps GraphQL field names (camelCase) to SQL column names (snake_case)
// for the rateplan table. Only fields present in this map are accepted as filter or
// sort keys — any other value is rejected to prevent SQL injection.
var ratePlanColumns = map[string]string{
	"planId":      "plan_id",
	"planName":    "plan_name",
	"planType":    "plan_type",
	"planStatus":  "plan_status",
	"createdBy":   "created_by",
	"approvedBy":  "approved_by",
	"effectiveAt": "effective_at",
	"modifiedAt":  "modified_at",
	"wholesaleId": "wholesale_id",
}

// ratePlanWildcardCols are the SQL column names searched when a wildcard term is provided.
// Mirrors Java RatePlanEntity.WILDCARD_FIELDS.
var ratePlanWildcardCols = []string{
	"plan_id", "plan_name", "plan_type", "plan_status",
}

// RatePlanService handles all business logic for the rate plan resource.
type RatePlanService struct {
	store *store.Store
}

// NewRatePlanService creates a new RatePlanService backed by the supplied store.
func NewRatePlanService(s *store.Store) *RatePlanService {
	return &RatePlanService{store: s}
}

// ListRatePlans returns a filtered, sorted, paginated list of rate plan rows (all versions).
func (s *RatePlanService) ListRatePlans(
	ctx context.Context,
	page *model.PageRequest,
	filterReq *model.FilterRequest,
) ([]*model.RatePlan, error) {
	where, err := filter.BuildWhere(filterReq, ratePlanColumns, ratePlanWildcardCols)
	if err != nil {
		return nil, err
	}
	orderBy, err := filter.BuildOrderBy(page, "plan_name", ratePlanColumns)
	if err != nil {
		return nil, err
	}
	limit, offset := filter.PageOffset(page)

	rows, err := s.store.ListRatePlans(ctx, store.ListRatePlansParams{
		WhereSQL: where.SQL,
		Args:     where.Args,
		OrderSQL: orderBy,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*model.RatePlan, 0, len(rows))
	for _, r := range rows {
		m, err := ratePlanToModel(r)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}

// CountRatePlans returns the total count of rate plan rows matching the supplied filter.
func (s *RatePlanService) CountRatePlans(
	ctx context.Context,
	filterReq *model.FilterRequest,
) (int, error) {
	where, err := filter.BuildWhere(filterReq, ratePlanColumns, ratePlanWildcardCols)
	if err != nil {
		return 0, err
	}
	n, err := s.store.CountRatePlans(ctx, where.SQL, where.Args)
	return int(n), err
}

// GetRatePlan returns the latest version of a rate plan by planId, or nil if not found.
func (s *RatePlanService) GetRatePlan(ctx context.Context, planID string) (*model.RatePlan, error) {
	uid, err := parseRatePlanUUID(planID)
	if err != nil {
		return nil, err
	}
	r, err := s.store.Q.FindLatestRatePlanByPlanId(ctx, uid)
	if err != nil {
		return nil, err
	}
	return ratePlanToModel(r)
}

// LatestRatePlanList returns the most recent version of each logical plan for the given planType.
// For RETAIL plans the wholesaleId is resolved from the tenant context; returns an error if absent.
func (s *RatePlanService) LatestRatePlanList(
	ctx context.Context,
	planType model.RatePlanType,
) ([]*model.RatePlan, error) {
	wholesaleID := pgtype.UUID{} // null by default (SETTLEMENT / WHOLESALE)

	if planType == model.RatePlanTypeRetail {
		uid, ok := tenant.WholesaleIDFromContext(ctx)
		if !ok {
			return nil, fmt.Errorf("latestRatePlanList: RETAIL plans require a tenant context (wholesale ID not resolved from Host header)")
		}
		wholesaleID = uid
	}

	rows, err := s.store.Q.LatestRatePlanByType(ctx, string(planType), wholesaleID)
	if err != nil {
		return nil, fmt.Errorf("latestRatePlanList: %w", err)
	}

	result := make([]*model.RatePlan, 0, len(rows))
	for _, r := range rows {
		m, err := ratePlanToModel(r)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}

// CreateRatePlan inserts a new rate plan in DRAFT status with a new planId UUID.
// For RETAIL plans the wholesaleId is resolved from the tenant context.
func (s *RatePlanService) CreateRatePlan(
	ctx context.Context,
	input model.RatePlanInput,
) (*model.RatePlan, error) {
	newPlanID, err := newPgUUID()
	if err != nil {
		return nil, fmt.Errorf("create rate plan: generate planId: %w", err)
	}
	effectiveAt, err := parseDateTime(input.EffectiveAt)
	if err != nil {
		return nil, fmt.Errorf("create rate plan: parse effectiveAt: %w", err)
	}

	wholesaleID := pgtype.UUID{} // null for SETTLEMENT / WHOLESALE
	if input.PlanType == model.RatePlanTypeRetail {
		uid, ok := tenant.WholesaleIDFromContext(ctx)
		if !ok {
			return nil, fmt.Errorf("create rate plan: RETAIL plans require a tenant context (wholesale ID not resolved)")
		}
		wholesaleID = uid
	}

	planJSON, err := buildRatePlanJSON(newPlanID, input, effectiveAt)
	if err != nil {
		return nil, fmt.Errorf("create rate plan: marshal rateplan: %w", err)
	}

	r, err := s.store.Q.CreateRatePlan(ctx, sqlc.CreateRatePlanParams{
		PlanID:      newPlanID,
		PlanType:    string(input.PlanType),
		WholesaleID: wholesaleID,
		PlanName:    input.PlanName,
		Rateplan:    planJSON,
		CreatedBy:   emailFromContext(ctx),
		EffectiveAt: effectiveAt,
	})
	if err != nil {
		return nil, fmt.Errorf("create rate plan: %w", err)
	}
	return ratePlanToModel(r)
}

// UpdateRatePlan updates the name, type, effectiveAt, and full rateplan JSONB of the DRAFT version.
// Returns an error if no DRAFT version exists for the given planId.
// TODO: confirm whether UpdateRatePlan or UpdateRatePlanRules is the one to keep in production.
func (s *RatePlanService) UpdateRatePlan(
	ctx context.Context,
	planID string,
	input model.RatePlanInput,
) (*model.RatePlan, error) {
	uid, err := parseRatePlanUUID(planID)
	if err != nil {
		return nil, err
	}
	effectiveAt, err := parseDateTime(input.EffectiveAt)
	if err != nil {
		return nil, fmt.Errorf("update rate plan: parse effectiveAt: %w", err)
	}
	planJSON, err := buildRatePlanJSON(uid, input, effectiveAt)
	if err != nil {
		return nil, fmt.Errorf("update rate plan: marshal rateplan: %w", err)
	}

	r, err := s.store.Q.UpdateRatePlan(ctx, sqlc.UpdateRatePlanParams{
		PlanID:      uid,
		PlanName:    input.PlanName,
		PlanType:    string(input.PlanType),
		EffectiveAt: effectiveAt,
		Rateplan:    planJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("update rate plan %s: %w", planID, err)
	}
	return ratePlanToModel(r)
}

// UpdateRatePlanRules updates only the rateplan JSONB of the DRAFT version.
// Metadata (planName, planType, effectiveAt) is preserved unchanged.
// TODO: confirm whether UpdateRatePlan or UpdateRatePlanRules is the one to keep in production.
func (s *RatePlanService) UpdateRatePlanRules(
	ctx context.Context,
	planID string,
	rateLines []*model.RateLineInput,
) (*model.RatePlan, error) {
	uid, err := parseRatePlanUUID(planID)
	if err != nil {
		return nil, err
	}

	// Fetch the current DRAFT to preserve its metadata for the JSONB sync.
	current, err := s.store.Q.FindLatestRatePlanByPlanId(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("update rate plan rules: fetch current: %w", err)
	}
	if current.PlanStatus != "DRAFT" {
		return nil, fmt.Errorf("update rate plan rules: plan %s is not in DRAFT status", planID)
	}

	lines, err := inputLinesToDomain(rateLines)
	if err != nil {
		return nil, fmt.Errorf("update rate plan rules: %w", err)
	}

	domainPlan := gomodel.RatePlan{
		RatePlanID:    pgUUIDToString(current.PlanID),
		RatePlanName:  current.PlanName,
		RatePlanType:  gomodel.RatePlanType(current.PlanType),
		EffectiveFrom: current.EffectiveAt.Time,
		RateLines:     lines,
	}
	planJSON, err := json.Marshal(domainPlan)
	if err != nil {
		return nil, fmt.Errorf("update rate plan rules: marshal: %w", err)
	}

	r, err := s.store.Q.UpdateRatePlanRules(ctx, uid, planJSON)
	if err != nil {
		return nil, fmt.Errorf("update rate plan rules %s: %w", planID, err)
	}
	return ratePlanToModel(r)
}

// CloneRatePlan creates a new DRAFT version of an existing rate plan with the same planId.
// createdBy is the current authenticated user; approvedBy is cleared.
func (s *RatePlanService) CloneRatePlan(ctx context.Context, planID string) (*model.RatePlan, error) {
	uid, err := parseRatePlanUUID(planID)
	if err != nil {
		return nil, err
	}
	src, err := s.store.Q.FindLatestRatePlanByPlanId(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("clone rate plan: find source %s: %w", planID, err)
	}

	r, err := s.store.Q.CreateRatePlan(ctx, sqlc.CreateRatePlanParams{
		PlanID:      src.PlanID, // same planId — new version of same logical plan
		PlanType:    src.PlanType,
		WholesaleID: src.WholesaleID,
		PlanName:    src.PlanName,
		Rateplan:    src.Rateplan,
		CreatedBy:   emailFromContext(ctx),
		EffectiveAt: src.EffectiveAt,
	})
	if err != nil {
		return nil, fmt.Errorf("clone rate plan: %w", err)
	}
	return ratePlanToModel(r)
}

// SubmitRatePlanForApproval transitions the DRAFT version to PENDING.
func (s *RatePlanService) SubmitRatePlanForApproval(
	ctx context.Context,
	planID string,
) (*model.RatePlan, error) {
	uid, err := parseRatePlanUUID(planID)
	if err != nil {
		return nil, err
	}
	r, err := s.store.Q.SubmitRatePlan(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("submit rate plan %s: %w", planID, err)
	}
	return ratePlanToModel(r)
}

// ApproveRatePlan transitions the PENDING version to ACTIVE.
// approvedBy is extracted from the authenticated JWT in ctx.
func (s *RatePlanService) ApproveRatePlan(ctx context.Context, planID string) (*model.RatePlan, error) {
	uid, err := parseRatePlanUUID(planID)
	if err != nil {
		return nil, err
	}
	approvedBy := pgtype.Text{String: emailFromContext(ctx), Valid: true}
	r, err := s.store.Q.ApproveRatePlan(ctx, uid, approvedBy)
	if err != nil {
		return nil, fmt.Errorf("approve rate plan %s: %w", planID, err)
	}
	return ratePlanToModel(r)
}

// DeclineRatePlan transitions the PENDING version back to DRAFT, clearing approvedBy.
func (s *RatePlanService) DeclineRatePlan(ctx context.Context, planID string) (*model.RatePlan, error) {
	uid, err := parseRatePlanUUID(planID)
	if err != nil {
		return nil, err
	}
	r, err := s.store.Q.DeclineRatePlan(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("decline rate plan %s: %w", planID, err)
	}
	return ratePlanToModel(r)
}

// DeleteRatePlan permanently deletes the DRAFT version of a rate plan. Returns true on success.
func (s *RatePlanService) DeleteRatePlan(ctx context.Context, planID string) (bool, error) {
	uid, err := parseRatePlanUUID(planID)
	if err != nil {
		return false, err
	}
	if err := s.store.Q.DeleteRatePlan(ctx, uid); err != nil {
		return false, fmt.Errorf("delete rate plan %s: %w", planID, err)
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// parseRatePlanUUID converts a GraphQL ID string to a pgtype.UUID.
func parseRatePlanUUID(id string) (pgtype.UUID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid planId %q: %w", id, err)
	}
	var pgUID pgtype.UUID
	copy(pgUID.Bytes[:], uid[:])
	pgUID.Valid = true
	return pgUID, nil
}

// ratePlanToModel maps a sqlc.Rateplan row to the GraphQL model type.
// The embedded rateplan JSONB is unmarshalled to extract the rate lines.
func ratePlanToModel(r sqlc.Rateplan) (*model.RatePlan, error) {
	var domainPlan gomodel.RatePlan
	if err := json.Unmarshal(r.Rateplan, &domainPlan); err != nil {
		return nil, fmt.Errorf("unmarshal rateplan for %v: %w", r.PlanID, err)
	}

	m := &model.RatePlan{
		PlanID:      pgUUIDToString(r.PlanID),
		PlanName:    r.PlanName,
		PlanType:    model.RatePlanType(r.PlanType),
		PlanStatus:  model.RatePlanStatus(r.PlanStatus),
		CreatedBy:   r.CreatedBy,
		EffectiveAt: r.EffectiveAt.Time.Format(time.RFC3339),
		RateLines:   domainLinesToGQL(domainPlan.RateLines),
	}
	if r.WholesaleID.Valid {
		wid := pgUUIDToString(r.WholesaleID)
		m.WholesaleID = &wid
	}
	if r.ApprovedBy.Valid {
		m.ApprovedBy = &r.ApprovedBy.String
	}
	if r.ModifiedAt.Valid {
		s := r.ModifiedAt.Time.Format(time.RFC3339)
		m.ModifiedAt = &s
	}
	return m, nil
}

// domainLinesToGQL converts a slice of domain RateLine to GraphQL RateLine pointers.
func domainLinesToGQL(lines []gomodel.RateLine) []*model.RateLine {
	result := make([]*model.RateLine, 0, len(lines))
	for _, l := range lines {
		gl := &model.RateLine{
			ClassificationKey: l.ClassificationKey.String(),
			TariffType:        string(l.TariffType),
			UnitType:          string(l.UnitType),
			BaseTariff:        l.BaseTariff.String(),
			UnitOfMeasure:     int(l.UnitOfMeasure),
			Multiplier:        l.Multiplier.String(),
			MinimumUnits:      int(l.MinimumUnits),
			RoundingIncrement: int(l.RoundingIncrement),
			Barred:            l.Barred,
			MonetaryOnly:      l.MonetaryOnly,
		}
		if l.GroupKey != "" {
			gl.GroupKey = &l.GroupKey
		}
		if l.Description != "" {
			gl.Description = &l.Description
		}
		if l.QosProfile != "" {
			gl.QosProfile = &l.QosProfile
		}
		result = append(result, gl)
	}
	return result
}

// inputLinesToDomain converts GraphQL RateLineInput pointers to domain RateLine values.
func inputLinesToDomain(inputs []*model.RateLineInput) ([]gomodel.RateLine, error) {
	lines := make([]gomodel.RateLine, 0, len(inputs))
	for _, in := range inputs {
		rk, err := charging.ParseRateKey(in.ClassificationKey)
		if err != nil {
			return nil, fmt.Errorf("invalid classificationKey %q: %w", in.ClassificationKey, err)
		}
		baseTariff, err := decimal.NewFromString(in.BaseTariff)
		if err != nil {
			return nil, fmt.Errorf("invalid baseTariff %q: %w", in.BaseTariff, err)
		}
		multiplier, err := decimal.NewFromString(in.Multiplier)
		if err != nil {
			return nil, fmt.Errorf("invalid multiplier %q: %w", in.Multiplier, err)
		}

		l := gomodel.RateLine{
			ClassificationKey: *rk,
			TariffType:        gomodel.TariffType(in.TariffType),
			UnitType:          charging.UnitType(in.UnitType),
			BaseTariff:        baseTariff,
			UnitOfMeasure:     gomodel.Quantity(in.UnitOfMeasure),
			Multiplier:        multiplier,
			MinimumUnits:      gomodel.Quantity(in.MinimumUnits),
			RoundingIncrement: gomodel.Quantity(in.RoundingIncrement),
			Barred:            in.Barred,
			MonetaryOnly:      in.MonetaryOnly,
		}
		if in.GroupKey != nil {
			l.GroupKey = *in.GroupKey
		}
		if in.Description != nil {
			l.Description = *in.Description
		}
		if in.QosProfile != nil {
			l.QosProfile = *in.QosProfile
		}
		lines = append(lines, l)
	}
	return lines, nil
}

// buildRatePlanJSON constructs the model.RatePlan struct (keeping table columns and JSONB
// in sync) and marshals it to JSON for storage. The planId UUID is embedded so the
// charging engine has a self-consistent document.
func buildRatePlanJSON(
	planID pgtype.UUID,
	input model.RatePlanInput,
	effectiveAt pgtype.Timestamptz,
) ([]byte, error) {
	lines, err := inputLinesToDomain(input.RateLines)
	if err != nil {
		return nil, err
	}
	domainPlan := gomodel.RatePlan{
		RatePlanID:    pgUUIDToString(planID),
		RatePlanName:  input.PlanName,
		RatePlanType:  gomodel.RatePlanType(string(input.PlanType)),
		EffectiveFrom: effectiveAt.Time,
		RateLines:     lines,
	}
	return json.Marshal(domainPlan)
}

// emailFromContext, parseDateTime, pgUUIDToString, and newPgUUID are shared helpers
// defined in classification_service.go (same package).
