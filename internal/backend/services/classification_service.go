package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"go-ocs/internal/auth/keycloak"
	"go-ocs/internal/backend/filter"
	"go-ocs/internal/backend/graphql/model"
	"go-ocs/internal/charging"
	"go-ocs/internal/common"
	gomodel "go-ocs/internal/model"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// classificationColumns maps GraphQL field names (camelCase) to SQL column names
// (snake_case) for the classification table. Only fields present in this map are
// accepted as filter or sort keys — any other value is rejected to prevent SQL injection.
var classificationColumns = map[string]string{
	"classificationId": "classification_id",
	"name":             "name",
	"status":           "status",
	"createdBy":        "created_by",
	"approvedBy":       "approved_by",
	"effectiveTime":    "effective_time",
	"createdOn":        "created_on",
}

// classificationWildcardCols are the SQL column names searched when a wildcard term
// is provided. Mirrors Java ClassificationEntity.WILDCARD_FIELDS.
var classificationWildcardCols = []string{
	"classification_id", "name", "status",
}

// ClassificationService handles all business logic for the classification resource.
type ClassificationService struct {
	store *store.Store
}

// NewClassificationService creates a new ClassificationService backed by the supplied store.
func NewClassificationService(s *store.Store) *ClassificationService {
	return &ClassificationService{store: s}
}

// ListClassifications returns a filtered, sorted, paginated list of classifications.
func (s *ClassificationService) ListClassifications(
	ctx context.Context,
	page *model.PageRequest,
	filterReq *model.FilterRequest,
) ([]*model.Classification, error) {
	where, err := filter.BuildWhere(filterReq, classificationColumns, classificationWildcardCols)
	if err != nil {
		return nil, err
	}
	orderBy, err := filter.BuildOrderBy(page, "name", classificationColumns)
	if err != nil {
		return nil, err
	}
	limit, offset := filter.PageOffset(page)

	rows, err := s.store.ListClassifications(ctx, store.ListClassificationsParams{
		WhereSQL: where.SQL,
		Args:     where.Args,
		OrderSQL: orderBy,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*model.Classification, 0, len(rows))
	for _, c := range rows {
		m, err := classificationToModel(c)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}

// CountClassifications returns the total count of classifications matching the supplied filter.
func (s *ClassificationService) CountClassifications(
	ctx context.Context,
	filterReq *model.FilterRequest,
) (int, error) {
	where, err := filter.BuildWhere(filterReq, classificationColumns, classificationWildcardCols)
	if err != nil {
		return 0, err
	}
	n, err := s.store.CountClassifications(ctx, where.SQL, where.Args)
	return int(n), err
}

// GetClassification returns a single classification by its UUID string, or nil if not found.
func (s *ClassificationService) GetClassification(
	ctx context.Context,
	classificationID string,
) (*model.Classification, error) {
	uid, err := parseClassificationUUID(classificationID)
	if err != nil {
		return nil, err
	}
	c, err := s.store.Q.FindClassificationByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	return classificationToModel(c)
}

// RateKeyInput derives lookup data from the currently active classification plan.
// It is used by the frontend to populate rate-plan configuration dropdowns.
func (s *ClassificationService) RateKeyInput(ctx context.Context) (*model.RateKeyInput, error) {
	rec, err := s.store.Q.FindActiveClassification(ctx)
	if err != nil {
		return nil, fmt.Errorf("rateKeyInput: no active classification plan: %w", err)
	}

	var plan gomodel.ClassificationPlan
	if err := json.Unmarshal(rec.Plan, &plan); err != nil {
		return nil, fmt.Errorf("rateKeyInput: unmarshal plan: %w", err)
	}

	seenServiceTypes := map[string]bool{}
	seenSourceTypes := map[string]bool{}
	seenDirections := map[string]bool{}

	result := &model.RateKeyInput{
		ServiceTypes:      []*model.LookupData{},
		SourceTypes:       []*model.LookupData{},
		ServiceDirections: []*model.LookupData{},
		ServiceCategories: []*model.ServiceCategoryLookup{},
		ServiceWindows:    []*model.ServiceCategoryLookup{},
	}

	for _, st := range plan.ServiceTypes {
		if !seenServiceTypes[st.ServiceType] {
			seenServiceTypes[st.ServiceType] = true
			result.ServiceTypes = append(result.ServiceTypes, &model.LookupData{
				Code: st.ServiceType,
				Name: st.ServiceType,
			})
		}
		if !seenSourceTypes[st.SourceType] {
			seenSourceTypes[st.SourceType] = true
			result.SourceTypes = append(result.SourceTypes, &model.LookupData{
				Code: st.SourceType,
				Name: st.SourceType,
			})
		}
		dir := st.ServiceDirection
		if !seenDirections[dir] {
			seenDirections[dir] = true
			result.ServiceDirections = append(result.ServiceDirections, &model.LookupData{
				Code: dir,
				Name: dir,
			})
		}
		for code, name := range st.ServiceCategoryMap {
			result.ServiceCategories = append(result.ServiceCategories, &model.ServiceCategoryLookup{
				Code:            code,
				Name:            name,
				ServiceTypeCode: st.ServiceType,
			})
		}
		for _, windowName := range st.ServiceWindows {
			result.ServiceWindows = append(result.ServiceWindows, &model.ServiceCategoryLookup{
				Code:            windowName,
				Name:            windowName,
				ServiceTypeCode: st.ServiceType,
			})
		}
	}

	return result, nil
}

// CreateClassification inserts a new classification in DRAFT status.
// createdBy is extracted from the authenticated JWT in ctx.
func (s *ClassificationService) CreateClassification(
	ctx context.Context,
	input model.ClassificationInput,
) (*model.Classification, error) {
	planBytes, err := marshalPlanInput(input.Plan)
	if err != nil {
		return nil, err
	}
	effectiveTime, err := parseDateTime(input.EffectiveTime)
	if err != nil {
		return nil, fmt.Errorf("create classification: parse effectiveTime: %w", err)
	}
	newID, err := newPgUUID()
	if err != nil {
		return nil, fmt.Errorf("create classification: generate uuid: %w", err)
	}

	c, err := s.store.Q.CreateClassification(ctx, sqlc.CreateClassificationParams{
		ClassificationID: newID,
		Name:             input.Name,
		EffectiveTime:    effectiveTime,
		CreatedBy:        emailFromContext(ctx),
		Plan:             planBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("create classification: %w", err)
	}
	return classificationToModel(c)
}

// CloneClassification creates a DRAFT copy of an existing classification.
// The clone gets a new UUID; createdBy is the current authenticated user.
func (s *ClassificationService) CloneClassification(
	ctx context.Context,
	classificationID string,
) (*model.Classification, error) {
	uid, err := parseClassificationUUID(classificationID)
	if err != nil {
		return nil, err
	}
	src, err := s.store.Q.FindClassificationByID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("clone classification: find source %s: %w", classificationID, err)
	}
	newID, err := newPgUUID()
	if err != nil {
		return nil, fmt.Errorf("clone classification: generate uuid: %w", err)
	}

	c, err := s.store.Q.CreateClassification(ctx, sqlc.CreateClassificationParams{
		ClassificationID: newID,
		Name:             src.Name,
		EffectiveTime:    src.EffectiveTime,
		CreatedBy:        emailFromContext(ctx),
		Plan:             src.Plan,
	})
	if err != nil {
		return nil, fmt.Errorf("clone classification: %w", err)
	}
	return classificationToModel(c)
}

// UpdateClassificationPlan updates the name, effectiveTime, and plan of a DRAFT classification.
// Returns an error if the record is not in DRAFT status.
func (s *ClassificationService) UpdateClassificationPlan(
	ctx context.Context,
	classificationID string,
	input model.ClassificationInput,
) (*model.Classification, error) {
	uid, err := parseClassificationUUID(classificationID)
	if err != nil {
		return nil, err
	}
	planBytes, err := marshalPlanInput(input.Plan)
	if err != nil {
		return nil, err
	}
	effectiveTime, err := parseDateTime(input.EffectiveTime)
	if err != nil {
		return nil, fmt.Errorf("update classification plan: parse effectiveTime: %w", err)
	}

	c, err := s.store.Q.UpdateClassificationPlan(ctx, uid, input.Name, effectiveTime, planBytes)
	if err != nil {
		return nil, fmt.Errorf("update classification plan %s: %w", classificationID, err)
	}
	return classificationToModel(c)
}

// SubmitClassificationForApproval transitions a DRAFT classification to PENDING.
func (s *ClassificationService) SubmitClassificationForApproval(
	ctx context.Context,
	classificationID string,
) (*model.Classification, error) {
	uid, err := parseClassificationUUID(classificationID)
	if err != nil {
		return nil, err
	}
	c, err := s.store.Q.SubmitClassification(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("submit classification %s: %w", classificationID, err)
	}
	return classificationToModel(c)
}

// ApproveClassificationPlan transitions a PENDING classification to ACTIVE.
// approvedBy is extracted from the authenticated JWT in ctx.
func (s *ClassificationService) ApproveClassificationPlan(
	ctx context.Context,
	classificationID string,
) (*model.Classification, error) {
	uid, err := parseClassificationUUID(classificationID)
	if err != nil {
		return nil, err
	}
	approvedBy := pgtype.Text{String: emailFromContext(ctx), Valid: true}
	c, err := s.store.Q.ApproveClassification(ctx, uid, approvedBy)
	if err != nil {
		return nil, fmt.Errorf("approve classification %s: %w", classificationID, err)
	}
	return classificationToModel(c)
}

// DeclineClassificationPlan transitions a PENDING classification back to DRAFT.
func (s *ClassificationService) DeclineClassificationPlan(
	ctx context.Context,
	classificationID string,
) (*model.Classification, error) {
	uid, err := parseClassificationUUID(classificationID)
	if err != nil {
		return nil, err
	}
	c, err := s.store.Q.DeclineClassification(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("decline classification %s: %w", classificationID, err)
	}
	return classificationToModel(c)
}

// DeleteClassification permanently deletes a DRAFT classification. Returns true on success.
func (s *ClassificationService) DeleteClassification(
	ctx context.Context,
	classificationID string,
) (bool, error) {
	uid, err := parseClassificationUUID(classificationID)
	if err != nil {
		return false, err
	}
	if err := s.store.Q.DeleteClassification(ctx, uid); err != nil {
		return false, fmt.Errorf("delete classification %s: %w", classificationID, err)
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// emailFromContext extracts the authenticated user's email address from the request context.
// Returns "unknown" if auth is disabled or no claims are present in the context.
func emailFromContext(ctx context.Context) string {
	claims, ok := keycloak.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return "unknown"
	}
	return claims.Email
}

// parseClassificationUUID converts a string UUID (from the GraphQL ID scalar) to
// the pgtype.UUID used by sqlc.
func parseClassificationUUID(id string) (pgtype.UUID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid uuid %q: %w", id, err)
	}
	var pgUID pgtype.UUID
	copy(pgUID.Bytes[:], uid[:])
	pgUID.Valid = true
	return pgUID, nil
}

// newPgUUID generates a new random UUID as a pgtype.UUID.
func newPgUUID() (pgtype.UUID, error) {
	uid := uuid.New()
	var pgUID pgtype.UUID
	copy(pgUID.Bytes[:], uid[:])
	pgUID.Valid = true
	return pgUID, nil
}

// parseDateTime parses an RFC3339 DateTime scalar string into a pgtype.Timestamptz.
func parseDateTime(s string) (pgtype.Timestamptz, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return pgtype.Timestamptz{}, fmt.Errorf("invalid DateTime %q (want RFC3339): %w", s, err)
	}
	return pgtype.Timestamptz{Time: t, Valid: true}, nil
}

// classificationToModel maps a sqlc.Classification row to the GraphQL model type.
// The embedded plan JSONB is unmarshalled into gomodel.ClassificationPlan then
// converted field-by-field including map flattening.
func classificationToModel(c sqlc.Classification) (*model.Classification, error) {
	var plan gomodel.ClassificationPlan
	if err := json.Unmarshal(c.Plan, &plan); err != nil {
		return nil, fmt.Errorf("unmarshal plan for classification %v: %w", c.ClassificationID, err)
	}

	m := &model.Classification{
		ClassificationID: pgUUIDToString(c.ClassificationID),
		Name:             c.Name,
		EffectiveTime:    c.EffectiveTime.Time.Format(time.RFC3339),
		CreatedBy:        c.CreatedBy,
		Status:           model.ClassificationStatus(c.Status),
		Plan:             domainPlanToGQL(&plan),
	}
	if c.CreatedOn.Valid {
		s := c.CreatedOn.Time.Format(time.RFC3339)
		m.CreatedOn = &s
	}
	if c.ApprovedBy.Valid {
		m.ApprovedBy = &c.ApprovedBy.String
	}
	return m, nil
}

// pgUUIDToString formats a pgtype.UUID as a standard hyphenated UUID string.
func pgUUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	id, _ := uuid.FromBytes(u.Bytes[:])
	return id.String()
}

// domainPlanToGQL converts a gomodel.ClassificationPlan (domain struct) to the
// gqlgen-generated GraphQL type, flattening maps to slices.
func domainPlanToGQL(p *gomodel.ClassificationPlan) *model.ClassificationPlan {
	gql := &model.ClassificationPlan{
		UseServiceWindows:    p.UseServiceWindows,
		DefaultServiceWindow: p.DefaultServiceWindow,
		DefaultSourceType:    p.DefaultSourceType,
	}
	if p.RuleSetId != "" {
		gql.RuleSetID = &p.RuleSetId
	}
	if p.RuleSetName != "" {
		gql.RuleSetName = &p.RuleSetName
	}

	// Flatten map[string]ServiceWindow → []*ServiceWindowEntry
	for name, sw := range p.ServiceWindows {
		gql.ServiceWindows = append(gql.ServiceWindows, &model.ServiceWindowEntry{
			Name:      name,
			StartTime: sw.StartTime.Format("15:04"),
			EndTime:   sw.EndTime.Format("15:04"),
		})
	}

	// Convert each ServiceType
	for _, st := range p.ServiceTypes {
		gst := &model.ClassificationServiceType{
			Type:                st.ServiceType,
			ChargingInformation: string(st.ChargingInformation),
			SourceType:          st.SourceType,
			ServiceDirection:    st.ServiceDirection,
			ServiceCategory:     st.ServiceCategory,
			UnitType:            string(st.UnitType),
			ServiceWindows:      st.ServiceWindows,
		}
		if st.ServiceTypeRule != "" {
			gst.ServiceTypeRule = &st.ServiceTypeRule
		}
		if st.Description != "" {
			gst.Description = &st.Description
		}
		if st.ServiceIdentifier != "" {
			gst.ServiceIdentifier = &st.ServiceIdentifier
		}
		if st.DefaultServiceCategory != "" {
			gst.DefaultServiceCategory = &st.DefaultServiceCategory
		}
		// Flatten map[string]string → []*ServiceCategoryMapEntry
		for k, v := range st.ServiceCategoryMap {
			gst.ServiceCategoryMap = append(gst.ServiceCategoryMap, &model.ServiceCategoryMapEntry{
				Key:   k,
				Value: v,
			})
		}
		gql.ServiceTypes = append(gql.ServiceTypes, gst)
	}

	return gql
}

// marshalPlanInput converts a GraphQL ClassificationPlanInput to the domain
// gomodel.ClassificationPlan and marshals it to JSON for database storage.
func marshalPlanInput(input *model.ClassificationPlanInput) ([]byte, error) {
	if input == nil {
		return nil, fmt.Errorf("plan must not be nil")
	}

	plan := gomodel.ClassificationPlan{
		UseServiceWindows:    input.UseServiceWindows,
		DefaultServiceWindow: input.DefaultServiceWindow,
		DefaultSourceType:    input.DefaultSourceType,
	}
	if input.RuleSetID != nil {
		plan.RuleSetId = *input.RuleSetID
	}
	if input.RuleSetName != nil {
		plan.RuleSetName = *input.RuleSetName
	}

	// Build map[string]ServiceWindow from input list
	if len(input.ServiceWindows) > 0 {
		plan.ServiceWindows = make(map[string]gomodel.ServiceWindow, len(input.ServiceWindows))
		for _, sw := range input.ServiceWindows {
			startT, err := time.Parse("15:04", sw.StartTime)
			if err != nil {
				return nil, fmt.Errorf("invalid startTime %q for window %q: %w", sw.StartTime, sw.Name, err)
			}
			endT, err := time.Parse("15:04", sw.EndTime)
			if err != nil {
				return nil, fmt.Errorf("invalid endTime %q for window %q: %w", sw.EndTime, sw.Name, err)
			}
			plan.ServiceWindows[sw.Name] = gomodel.ServiceWindow{
				StartTime: common.LocalTime{Time: startT},
				EndTime:   common.LocalTime{Time: endT},
			}
		}
	}

	// Convert each service type input
	for _, st := range input.ServiceTypes {
		dst := gomodel.ServiceType{
			ServiceType:         st.Type,
			ChargingInformation: gomodel.ChargingInformation(st.ChargingInformation),
			SourceType:          st.SourceType,
			ServiceDirection:    st.ServiceDirection,
			ServiceCategory:     st.ServiceCategory,
			UnitType:            charging.UnitType(st.UnitType),
			ServiceWindows:      st.ServiceWindows,
		}
		if st.ServiceTypeRule != nil {
			dst.ServiceTypeRule = *st.ServiceTypeRule
		}
		if st.Description != nil {
			dst.Description = *st.Description
		}
		if st.ServiceIdentifier != nil {
			dst.ServiceIdentifier = *st.ServiceIdentifier
		}
		if st.DefaultServiceCategory != nil {
			dst.DefaultServiceCategory = *st.DefaultServiceCategory
		}
		// Build map[string]string from input list
		if len(st.ServiceCategoryMap) > 0 {
			dst.ServiceCategoryMap = make(map[string]string, len(st.ServiceCategoryMap))
			for _, e := range st.ServiceCategoryMap {
				dst.ServiceCategoryMap[e.Key] = e.Value
			}
		}
		plan.ServiceTypes = append(plan.ServiceTypes, dst)
	}

	return json.Marshal(plan)
}
