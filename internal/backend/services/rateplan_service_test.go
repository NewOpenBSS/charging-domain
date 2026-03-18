package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	graphqlmodel "go-ocs/internal/backend/graphql/model"
	"go-ocs/internal/charging"
	gomodel "go-ocs/internal/model"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// newRatePlanService wires a RatePlanService backed by a mock DBTX.
// servicesMockDBTX is defined in carrier_service_test.go (same package).
func newRatePlanService(mockDB *servicesMockDBTX) *RatePlanService {
	return NewRatePlanService(&store.Store{Q: sqlc.New(mockDB)})
}

// buildPlanID returns a deterministic pgtype.UUID and its string form for tests.
func buildPlanID(t *testing.T) (pgtype.UUID, string) {
	t.Helper()
	raw := uuid.New()
	var pgUID pgtype.UUID
	copy(pgUID.Bytes[:], raw[:])
	pgUID.Valid = true
	return pgUID, raw.String()
}

// minimalRatePlanJSON returns serialised domain gomodel.RatePlan JSON for use in test rows.
func minimalRatePlanJSON(t *testing.T, pgUID pgtype.UUID) []byte {
	t.Helper()
	rk := charging.RateKey{
		ServiceType:      "voice",
		SourceType:       "Home",
		ServiceDirection: "MO",
		ServiceCategory:  "local",
	}
	plan := gomodel.RatePlan{
		RatePlanID:    pgUUIDToString(pgUID),
		RatePlanName:  "TestPlan",
		RatePlanType:  gomodel.RatePlanType("SETTLEMENT"),
		EffectiveFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		RateLines: []gomodel.RateLine{
			{
				ClassificationKey: rk,
				TariffType:        gomodel.ACTUAL,
				UnitType:          charging.UnitType("SECONDS"),
				BaseTariff:        decimal.NewFromFloat(0.10),
				UnitOfMeasure:     60,
				Multiplier:        decimal.NewFromInt(1),
				MinimumUnits:      60,
				RoundingIncrement: 60,
				Barred:            false,
				MonetaryOnly:      false,
			},
		},
	}
	b, err := json.Marshal(plan)
	require.NoError(t, err)
	return b
}

// populateRatePlanScan fills the 11 Scan destinations for a rateplan row.
func populateRatePlanScan(pgUID pgtype.UUID, name, status string, planJSON []byte) func(mock.Arguments) {
	return func(args mock.Arguments) {
		*(args[0].(*int64)) = 1                                                                      // id (bigserial)
		*(args[1].(*pgtype.UUID)) = pgUID                                                            // plan_id
		*(args[2].(*pgtype.Timestamptz)) = pgtype.Timestamptz{Time: time.Now(), Valid: true}        // modified_at
		*(args[3].(*string)) = "SETTLEMENT"                                                          // plan_type
		*(args[4].(*pgtype.UUID)) = pgtype.UUID{Valid: false}                                       // wholesale_id (null)
		*(args[5].(*string)) = name                                                                  // plan_name
		*(args[6].(*[]byte)) = planJSON                                                              // rateplan JSONB
		*(args[7].(*string)) = status                                                                // plan_status
		*(args[8].(*string)) = "creator@test.com"                                                   // created_by
		*(args[9].(*pgtype.Text)) = pgtype.Text{Valid: false}                                       // approved_by (null)
		*(args[10].(*pgtype.Timestamptz)) = pgtype.Timestamptz{Time: time.Now(), Valid: true}       // effective_at
	}
}

const ratePlanScanArgCount = 11

func ratePlanScanMatchers() []interface{} {
	m := make([]interface{}, ratePlanScanArgCount)
	for i := range m {
		m[i] = mock.Anything
	}
	return m
}

// anyQueryRow1RatePlan registers a 1-arg QueryRow expectation (FindLatestRatePlanByPlanId, etc.)
func anyQueryRow1RatePlan(mockDB *servicesMockDBTX, row *servicesMockRow) {
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(row)
}

// anyQueryRow2RatePlan registers a 2-arg QueryRow expectation (ApproveRatePlan, UpdateRatePlanRules)
func anyQueryRow2RatePlan(mockDB *servicesMockDBTX, row *servicesMockRow) {
	mockDB.On("QueryRow",
		mock.Anything, mock.Anything,
		mock.Anything, mock.Anything,
	).Return(row)
}

// anyQueryRow5RatePlan registers a 5-arg QueryRow expectation (UpdateRatePlan)
func anyQueryRow5RatePlan(mockDB *servicesMockDBTX, row *servicesMockRow) {
	mockDB.On("QueryRow",
		mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(row)
}

// anyQueryRow7RatePlan registers a 7-arg QueryRow expectation (CreateRatePlan)
func anyQueryRow7RatePlan(mockDB *servicesMockDBTX, row *servicesMockRow) {
	mockDB.On("QueryRow",
		mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(row)
}

// minimalRatePlanInput builds a valid RatePlanInput for mutation tests.
func minimalRatePlanInput() graphqlmodel.RatePlanInput {
	return graphqlmodel.RatePlanInput{
		PlanName:    "TestPlan",
		PlanType:    graphqlmodel.RatePlanTypeSettlement,
		EffectiveAt: "2024-01-01T00:00:00Z",
		RateLines: []*graphqlmodel.RateLineInput{
			{
				ClassificationKey: "voice.Home.MO.local",
				TariffType:        "ACTUAL",
				UnitType:          "SECONDS",
				BaseTariff:        "0.10",
				UnitOfMeasure:     60,
				Multiplier:        "1",
				MinimumUnits:      60,
				RoundingIncrement: 60,
				Barred:            false,
				MonetaryOnly:      false,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// parseRatePlanUUID
// ---------------------------------------------------------------------------

func TestParseRatePlanUUID_Valid(t *testing.T) {
	id := uuid.New().String()
	pgUID, err := parseRatePlanUUID(id)
	require.NoError(t, err)
	assert.True(t, pgUID.Valid)
}

func TestParseRatePlanUUID_Invalid_Error(t *testing.T) {
	_, err := parseRatePlanUUID("not-a-uuid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid planId")
}

// ---------------------------------------------------------------------------
// inputLinesToDomain
// ---------------------------------------------------------------------------

func TestInputLinesToDomain_ValidInput(t *testing.T) {
	inputs := []*graphqlmodel.RateLineInput{
		{
			ClassificationKey: "voice.Home.MO.local",
			TariffType:        "ACTUAL",
			UnitType:          "SECONDS",
			BaseTariff:        "0.15",
			UnitOfMeasure:     60,
			Multiplier:        "1.0",
			MinimumUnits:      60,
			RoundingIncrement: 60,
			Barred:            false,
			MonetaryOnly:      false,
		},
	}

	lines, err := inputLinesToDomain(inputs)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	assert.Equal(t, "voice.Home.MO.local", lines[0].ClassificationKey.String())
	assert.Equal(t, "0.15", lines[0].BaseTariff.String())
}

func TestInputLinesToDomain_InvalidClassificationKey_Error(t *testing.T) {
	inputs := []*graphqlmodel.RateLineInput{
		{
			ClassificationKey: "bad-key",
			TariffType:        "ACTUAL",
			UnitType:          "SECONDS",
			BaseTariff:        "0.10",
			UnitOfMeasure:     60,
			Multiplier:        "1",
		},
	}
	_, err := inputLinesToDomain(inputs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid classificationKey")
}

func TestInputLinesToDomain_InvalidBaseTariff_Error(t *testing.T) {
	inputs := []*graphqlmodel.RateLineInput{
		{
			ClassificationKey: "voice.Home.MO.local",
			TariffType:        "ACTUAL",
			UnitType:          "SECONDS",
			BaseTariff:        "not-a-number",
			UnitOfMeasure:     60,
			Multiplier:        "1",
		},
	}
	_, err := inputLinesToDomain(inputs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid baseTariff")
}

func TestInputLinesToDomain_InvalidMultiplier_Error(t *testing.T) {
	inputs := []*graphqlmodel.RateLineInput{
		{
			ClassificationKey: "voice.Home.MO.local",
			TariffType:        "ACTUAL",
			UnitType:          "SECONDS",
			BaseTariff:        "0.10",
			UnitOfMeasure:     60,
			Multiplier:        "not-a-number",
		},
	}
	_, err := inputLinesToDomain(inputs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid multiplier")
}

func TestInputLinesToDomain_OptionalFields(t *testing.T) {
	grp := "G1"
	desc := "Voice local"
	qos := "qos1"
	inputs := []*graphqlmodel.RateLineInput{
		{
			ClassificationKey: "voice.Home.MO.local",
			TariffType:        "ACTUAL",
			UnitType:          "SECONDS",
			BaseTariff:        "0.10",
			UnitOfMeasure:     60,
			Multiplier:        "1",
			GroupKey:          &grp,
			Description:       &desc,
			QosProfile:        &qos,
		},
	}
	lines, err := inputLinesToDomain(inputs)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	assert.Equal(t, "G1", lines[0].GroupKey)
	assert.Equal(t, "Voice local", lines[0].Description)
	assert.Equal(t, "qos1", lines[0].QosProfile)
}

// ---------------------------------------------------------------------------
// domainLinesToGQL
// ---------------------------------------------------------------------------

func TestDomainLinesToGQL_BasicMapping(t *testing.T) {
	rk := charging.RateKey{
		ServiceType:      "voice",
		SourceType:       "Home",
		ServiceDirection: "MO",
		ServiceCategory:  "local",
	}
	lines := []gomodel.RateLine{
		{
			ClassificationKey: rk,
			TariffType:        gomodel.ACTUAL,
			UnitType:          charging.UnitType("SECONDS"),
			BaseTariff:        decimal.NewFromFloat(0.12),
			UnitOfMeasure:     60,
			Multiplier:        decimal.NewFromInt(1),
			MinimumUnits:      60,
			RoundingIncrement: 60,
			Barred:            false,
			MonetaryOnly:      false,
		},
	}

	gql := domainLinesToGQL(lines)
	require.Len(t, gql, 1)
	assert.Equal(t, "voice.Home.MO.local", gql[0].ClassificationKey)
	assert.Equal(t, "ACTUAL", gql[0].TariffType)
	assert.Equal(t, "SECONDS", gql[0].UnitType)
	assert.Equal(t, "0.12", gql[0].BaseTariff)
	assert.Equal(t, 60, gql[0].UnitOfMeasure)
	assert.Nil(t, gql[0].GroupKey)
}

func TestDomainLinesToGQL_OptionalFieldsPopulated(t *testing.T) {
	rk := charging.RateKey{ServiceType: "data", SourceType: "Home", ServiceDirection: "MO", ServiceCategory: "internet"}
	lines := []gomodel.RateLine{
		{
			ClassificationKey: rk,
			TariffType:        gomodel.ACTUAL,
			UnitType:          charging.UnitType("OCTETS"),
			BaseTariff:        decimal.NewFromFloat(0.01),
			UnitOfMeasure:     1024,
			Multiplier:        decimal.NewFromInt(1),
			GroupKey:          "grp1",
			Description:       "Data plan",
			QosProfile:        "default",
		},
	}

	gql := domainLinesToGQL(lines)
	require.Len(t, gql, 1)
	require.NotNil(t, gql[0].GroupKey)
	assert.Equal(t, "grp1", *gql[0].GroupKey)
	require.NotNil(t, gql[0].Description)
	assert.Equal(t, "Data plan", *gql[0].Description)
	require.NotNil(t, gql[0].QosProfile)
	assert.Equal(t, "default", *gql[0].QosProfile)
}

// ---------------------------------------------------------------------------
// ratePlanToModel
// ---------------------------------------------------------------------------

func TestRatePlanToModel_Success(t *testing.T) {
	pgUID, idStr := buildPlanID(t)
	planJSON := minimalRatePlanJSON(t, pgUID)

	row := sqlc.Rateplan{
		ID:          1,
		PlanID:      pgUID,
		ModifiedAt:  pgtype.Timestamptz{Time: time.Now(), Valid: true},
		PlanType:    "SETTLEMENT",
		WholesaleID: pgtype.UUID{Valid: false},
		PlanName:    "TestPlan",
		Rateplan:    planJSON,
		PlanStatus:  "DRAFT",
		CreatedBy:   "creator@test.com",
		ApprovedBy:  pgtype.Text{Valid: false},
		EffectiveAt: pgtype.Timestamptz{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
	}

	m, err := ratePlanToModel(row)
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, idStr, m.PlanID)
	assert.Equal(t, "TestPlan", m.PlanName)
	assert.Equal(t, graphqlmodel.RatePlanTypeSettlement, m.PlanType)
	assert.Equal(t, graphqlmodel.RatePlanStatusDraft, m.PlanStatus)
	assert.Equal(t, "creator@test.com", m.CreatedBy)
	assert.Nil(t, m.WholesaleID)
	assert.Nil(t, m.ApprovedBy)
	require.Len(t, m.RateLines, 1)
	assert.Equal(t, "voice.Home.MO.local", m.RateLines[0].ClassificationKey)
}

func TestRatePlanToModel_WithWholesaleIDAndApprovedBy(t *testing.T) {
	pgUID, _ := buildPlanID(t)
	wUID, _ := buildPlanID(t)
	planJSON := minimalRatePlanJSON(t, pgUID)

	row := sqlc.Rateplan{
		ID:          2,
		PlanID:      pgUID,
		ModifiedAt:  pgtype.Timestamptz{Valid: false},
		PlanType:    "RETAIL",
		WholesaleID: wUID,
		PlanName:    "RetailPlan",
		Rateplan:    planJSON,
		PlanStatus:  "ACTIVE",
		CreatedBy:   "a@test.com",
		ApprovedBy:  pgtype.Text{String: "b@test.com", Valid: true},
		EffectiveAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}

	m, err := ratePlanToModel(row)
	require.NoError(t, err)
	require.NotNil(t, m.WholesaleID)
	require.NotNil(t, m.ApprovedBy)
	assert.Equal(t, "b@test.com", *m.ApprovedBy)
	assert.Equal(t, graphqlmodel.RatePlanStatusActive, m.PlanStatus)
	assert.Nil(t, m.ModifiedAt)
}

func TestRatePlanToModel_InvalidJSON_Error(t *testing.T) {
	pgUID, _ := buildPlanID(t)
	row := sqlc.Rateplan{
		ID:          3,
		PlanID:      pgUID,
		Rateplan:    []byte(`{invalid}`),
		PlanStatus:  "DRAFT",
		EffectiveAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	_, err := ratePlanToModel(row)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal rateplan")
}

// ---------------------------------------------------------------------------
// GetRatePlan
// ---------------------------------------------------------------------------

func TestGetRatePlan_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	pgUID, idStr := buildPlanID(t)
	planJSON := minimalRatePlanJSON(t, pgUID)

	anyQueryRow1RatePlan(mockDB, mockRow)
	mockRow.On("Scan", ratePlanScanMatchers()...).
		Run(populateRatePlanScan(pgUID, "TestPlan", "DRAFT", planJSON)).
		Return(nil)

	svc := newRatePlanService(mockDB)
	m, err := svc.GetRatePlan(context.Background(), idStr)

	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, idStr, m.PlanID)
	assert.Equal(t, graphqlmodel.RatePlanStatusDraft, m.PlanStatus)
	mockDB.AssertExpectations(t)
}

func TestGetRatePlan_InvalidUUID_Error(t *testing.T) {
	svc := newRatePlanService(&servicesMockDBTX{})
	_, err := svc.GetRatePlan(context.Background(), "bad-id")
	require.Error(t, err)
}

func TestGetRatePlan_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow1RatePlan(mockDB, mockRow)
	mockRow.On("Scan", ratePlanScanMatchers()...).Return(errors.New("connection refused"))

	svc := newRatePlanService(mockDB)
	_, err := svc.GetRatePlan(context.Background(), uuid.New().String())
	require.Error(t, err)
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CreateRatePlan
// ---------------------------------------------------------------------------

func TestCreateRatePlan_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	pgUID, idStr := buildPlanID(t)
	planJSON := minimalRatePlanJSON(t, pgUID)

	anyQueryRow7RatePlan(mockDB, mockRow)
	mockRow.On("Scan", ratePlanScanMatchers()...).
		Run(populateRatePlanScan(pgUID, "TestPlan", "DRAFT", planJSON)).
		Return(nil)

	svc := newRatePlanService(mockDB)
	m, err := svc.CreateRatePlan(context.Background(), minimalRatePlanInput())

	require.NoError(t, err)
	require.NotNil(t, m)
	// The planId in the response is the DB-generated one (mocked to pgUID).
	assert.Equal(t, idStr, m.PlanID)
	assert.Equal(t, graphqlmodel.RatePlanStatusDraft, m.PlanStatus)
	mockDB.AssertExpectations(t)
}

func TestCreateRatePlan_InvalidEffectiveAt_Error(t *testing.T) {
	svc := newRatePlanService(&servicesMockDBTX{})
	input := minimalRatePlanInput()
	input.EffectiveAt = "not-a-date"
	_, err := svc.CreateRatePlan(context.Background(), input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse effectiveAt")
}

func TestCreateRatePlan_InvalidRateLine_Error(t *testing.T) {
	svc := newRatePlanService(&servicesMockDBTX{})
	input := minimalRatePlanInput()
	input.RateLines[0].BaseTariff = "not-a-decimal"
	_, err := svc.CreateRatePlan(context.Background(), input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal rateplan")
}

// ---------------------------------------------------------------------------
// UpdateRatePlan
// ---------------------------------------------------------------------------

func TestUpdateRatePlan_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	pgUID, idStr := buildPlanID(t)
	planJSON := minimalRatePlanJSON(t, pgUID)

	anyQueryRow5RatePlan(mockDB, mockRow)
	mockRow.On("Scan", ratePlanScanMatchers()...).
		Run(populateRatePlanScan(pgUID, "UpdatedPlan", "DRAFT", planJSON)).
		Return(nil)

	svc := newRatePlanService(mockDB)
	m, err := svc.UpdateRatePlan(context.Background(), idStr, minimalRatePlanInput())

	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, idStr, m.PlanID)
	mockDB.AssertExpectations(t)
}

func TestUpdateRatePlan_InvalidUUID_Error(t *testing.T) {
	svc := newRatePlanService(&servicesMockDBTX{})
	_, err := svc.UpdateRatePlan(context.Background(), "bad-id", minimalRatePlanInput())
	require.Error(t, err)
}

func TestUpdateRatePlan_InvalidEffectiveAt_Error(t *testing.T) {
	svc := newRatePlanService(&servicesMockDBTX{})
	input := minimalRatePlanInput()
	input.EffectiveAt = "bad-date"
	_, err := svc.UpdateRatePlan(context.Background(), uuid.New().String(), input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse effectiveAt")
}

// ---------------------------------------------------------------------------
// UpdateRatePlanRules
// ---------------------------------------------------------------------------

func TestUpdateRatePlanRules_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	pgUID, idStr := buildPlanID(t)
	planJSON := minimalRatePlanJSON(t, pgUID)

	// First call: FindLatestRatePlanByPlanId (1 arg).
	anyQueryRow1RatePlan(mockDB, mockRow)
	// Second call: UpdateRatePlanRules (2 args).
	anyQueryRow2RatePlan(mockDB, mockRow)

	mockRow.On("Scan", ratePlanScanMatchers()...).
		Run(populateRatePlanScan(pgUID, "TestPlan", "DRAFT", planJSON)).
		Return(nil).Times(2)

	svc := newRatePlanService(mockDB)
	m, err := svc.UpdateRatePlanRules(context.Background(), idStr, minimalRatePlanInput().RateLines)

	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, idStr, m.PlanID)
	mockDB.AssertExpectations(t)
}

func TestUpdateRatePlanRules_NotDraft_Error(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	pgUID, idStr := buildPlanID(t)
	planJSON := minimalRatePlanJSON(t, pgUID)

	anyQueryRow1RatePlan(mockDB, mockRow)
	mockRow.On("Scan", ratePlanScanMatchers()...).
		Run(populateRatePlanScan(pgUID, "ActivePlan", "ACTIVE", planJSON)).
		Return(nil)

	svc := newRatePlanService(mockDB)
	_, err := svc.UpdateRatePlanRules(context.Background(), idStr, minimalRatePlanInput().RateLines)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in DRAFT status")
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CloneRatePlan
// ---------------------------------------------------------------------------

func TestCloneRatePlan_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	pgUID, idStr := buildPlanID(t)
	planJSON := minimalRatePlanJSON(t, pgUID)

	// First call: FindLatestRatePlanByPlanId.
	anyQueryRow1RatePlan(mockDB, mockRow)
	// Second call: CreateRatePlan (7 args).
	anyQueryRow7RatePlan(mockDB, mockRow)

	mockRow.On("Scan", ratePlanScanMatchers()...).
		Run(populateRatePlanScan(pgUID, "TestPlan", "DRAFT", planJSON)).
		Return(nil).Times(2)

	svc := newRatePlanService(mockDB)
	m, err := svc.CloneRatePlan(context.Background(), idStr)

	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, graphqlmodel.RatePlanStatusDraft, m.PlanStatus)
	mockDB.AssertExpectations(t)
}

func TestCloneRatePlan_InvalidUUID_Error(t *testing.T) {
	svc := newRatePlanService(&servicesMockDBTX{})
	_, err := svc.CloneRatePlan(context.Background(), "not-a-uuid")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// SubmitRatePlanForApproval
// ---------------------------------------------------------------------------

func TestSubmitRatePlanForApproval_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	pgUID, idStr := buildPlanID(t)
	planJSON := minimalRatePlanJSON(t, pgUID)

	anyQueryRow1RatePlan(mockDB, mockRow)
	mockRow.On("Scan", ratePlanScanMatchers()...).
		Run(func(args mock.Arguments) {
			populateRatePlanScan(pgUID, "TestPlan", "PENDING", planJSON)(args)
		}).
		Return(nil)

	svc := newRatePlanService(mockDB)
	m, err := svc.SubmitRatePlanForApproval(context.Background(), idStr)

	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, graphqlmodel.RatePlanStatusPending, m.PlanStatus)
	mockDB.AssertExpectations(t)
}

func TestSubmitRatePlanForApproval_InvalidUUID_Error(t *testing.T) {
	svc := newRatePlanService(&servicesMockDBTX{})
	_, err := svc.SubmitRatePlanForApproval(context.Background(), "bad-id")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// ApproveRatePlan
// ---------------------------------------------------------------------------

func TestApproveRatePlan_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	pgUID, idStr := buildPlanID(t)
	planJSON := minimalRatePlanJSON(t, pgUID)

	anyQueryRow2RatePlan(mockDB, mockRow)
	mockRow.On("Scan", ratePlanScanMatchers()...).
		Run(func(args mock.Arguments) {
			populateRatePlanScan(pgUID, "TestPlan", "ACTIVE", planJSON)(args)
			*(args[9].(*pgtype.Text)) = pgtype.Text{String: "approver@test.com", Valid: true}
		}).
		Return(nil)

	svc := newRatePlanService(mockDB)
	m, err := svc.ApproveRatePlan(context.Background(), idStr)

	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, graphqlmodel.RatePlanStatusActive, m.PlanStatus)
	require.NotNil(t, m.ApprovedBy)
	assert.Equal(t, "approver@test.com", *m.ApprovedBy)
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// DeclineRatePlan
// ---------------------------------------------------------------------------

func TestDeclineRatePlan_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	pgUID, idStr := buildPlanID(t)
	planJSON := minimalRatePlanJSON(t, pgUID)

	anyQueryRow1RatePlan(mockDB, mockRow)
	mockRow.On("Scan", ratePlanScanMatchers()...).
		Run(populateRatePlanScan(pgUID, "TestPlan", "DRAFT", planJSON)).
		Return(nil)

	svc := newRatePlanService(mockDB)
	m, err := svc.DeclineRatePlan(context.Background(), idStr)

	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, graphqlmodel.RatePlanStatusDraft, m.PlanStatus)
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// DeleteRatePlan
// ---------------------------------------------------------------------------

func TestDeleteRatePlan_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).
		Return(pgconn.CommandTag{}, nil)

	svc := newRatePlanService(mockDB)
	ok, err := svc.DeleteRatePlan(context.Background(), uuid.New().String())

	require.NoError(t, err)
	assert.True(t, ok)
	mockDB.AssertExpectations(t)
}

func TestDeleteRatePlan_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).
		Return(pgconn.CommandTag{}, errors.New("not found"))

	svc := newRatePlanService(mockDB)
	ok, err := svc.DeleteRatePlan(context.Background(), uuid.New().String())

	require.Error(t, err)
	assert.False(t, ok)
	assert.Contains(t, err.Error(), "delete rate plan")
	mockDB.AssertExpectations(t)
}

func TestDeleteRatePlan_InvalidUUID_Error(t *testing.T) {
	svc := newRatePlanService(&servicesMockDBTX{})
	_, err := svc.DeleteRatePlan(context.Background(), "bad-uuid")
	require.Error(t, err)
}
