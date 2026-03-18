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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	graphqlmodel "go-ocs/internal/backend/graphql/model"
	gomodel "go-ocs/internal/model"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
)

// ---------------------------------------------------------------------------
// Helper types: mock DBTX + Row (reuse pattern from carrier_service_test.go).
// The servicesMockDBTX and servicesMockRow types are already defined in
// carrier_service_test.go in the same package, so we cannot redeclare them.
// We reuse them directly.
// ---------------------------------------------------------------------------

// newClassificationService wires a ClassificationService backed by a mock DBTX.
func newClassificationService(mockDB *servicesMockDBTX) *ClassificationService {
	return NewClassificationService(&store.Store{Q: sqlc.New(mockDB)})
}

// minimalPlan returns a minimal serialised ClassificationPlan JSON for test rows.
func minimalPlanJSON(t *testing.T) []byte {
	t.Helper()
	cp := gomodel.ClassificationPlan{
		UseServiceWindows:    false,
		DefaultServiceWindow: "peak",
		DefaultSourceType:    "Home",
	}
	b, err := json.Marshal(cp)
	require.NoError(t, err)
	return b
}

// buildClassificationID returns a deterministic pgtype.UUID for tests.
func buildClassificationID(t *testing.T) (pgtype.UUID, string) {
	t.Helper()
	raw := uuid.New()
	var pgUID pgtype.UUID
	copy(pgUID.Bytes[:], raw[:])
	pgUID.Valid = true
	return pgUID, raw.String()
}

// populateClassificationScan fills the 8 Scan destinations for a classification row.
func populateClassificationScan(pgUID pgtype.UUID, name string, planJSON []byte) func(mock.Arguments) {
	return func(args mock.Arguments) {
		*(args[0].(*pgtype.UUID)) = pgUID                                                          // ClassificationID
		*(args[1].(*string)) = name                                                                // Name
		*(args[2].(*pgtype.Timestamp)) = pgtype.Timestamp{Valid: false}                           // CreatedOn
		*(args[3].(*pgtype.Timestamptz)) = pgtype.Timestamptz{Time: time.Now(), Valid: true}      // EffectiveTime
		*(args[4].(*string)) = "creator@test.com"                                                  // CreatedBy
		*(args[5].(*pgtype.Text)) = pgtype.Text{Valid: false}                                     // ApprovedBy
		*(args[6].(*string)) = "DRAFT"                                                             // Status
		*(args[7].(*[]byte)) = planJSON                                                            // Plan
	}
}

// anyQueryRow1Classification registers a 1-arg QueryRow expectation (FindClassificationByID, etc.).
func anyQueryRow1Classification(mockDB *servicesMockDBTX, row *servicesMockRow) {
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(row)
}

// anyQueryRow5Classification registers a 5-arg QueryRow expectation (CreateClassification).
func anyQueryRow5Classification(mockDB *servicesMockDBTX, row *servicesMockRow) {
	mockDB.On("QueryRow",
		mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(row)
}

// anyQueryRow4Classification registers a 4-arg QueryRow expectation (UpdateClassificationPlan).
func anyQueryRow4Classification(mockDB *servicesMockDBTX, row *servicesMockRow) {
	mockDB.On("QueryRow",
		mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(row)
}

// anyQueryRow2Classification registers a 2-arg QueryRow expectation
// (ApproveClassification: classificationID + approvedBy).
func anyQueryRow2Classification(mockDB *servicesMockDBTX, row *servicesMockRow) {
	mockDB.On("QueryRow",
		mock.Anything, mock.Anything,
		mock.Anything, mock.Anything,
	).Return(row)
}

const scanArgCount = 8

func scanMatchers() []interface{} {
	m := make([]interface{}, scanArgCount)
	for i := range m {
		m[i] = mock.Anything
	}
	return m
}

// ---------------------------------------------------------------------------
// domainPlanToGQL
// ---------------------------------------------------------------------------

func TestDomainPlanToGQL_BasicFields(t *testing.T) {
	id := "rs-1"
	name := "Weekend"
	p := &gomodel.ClassificationPlan{
		RuleSetId:            id,
		RuleSetName:          name,
		UseServiceWindows:    true,
		DefaultServiceWindow: "peak",
		DefaultSourceType:    "Home",
	}

	gql := domainPlanToGQL(p)

	require.NotNil(t, gql)
	assert.True(t, gql.UseServiceWindows)
	assert.Equal(t, "peak", gql.DefaultServiceWindow)
	assert.Equal(t, "Home", gql.DefaultSourceType)
	require.NotNil(t, gql.RuleSetID)
	assert.Equal(t, id, *gql.RuleSetID)
	require.NotNil(t, gql.RuleSetName)
	assert.Equal(t, name, *gql.RuleSetName)
}

func TestDomainPlanToGQL_EmptyRuleSetFields_NilPointers(t *testing.T) {
	p := &gomodel.ClassificationPlan{
		UseServiceWindows:    false,
		DefaultServiceWindow: "offpeak",
		DefaultSourceType:    "Roaming",
	}

	gql := domainPlanToGQL(p)

	assert.Nil(t, gql.RuleSetID)
	assert.Nil(t, gql.RuleSetName)
}

// ---------------------------------------------------------------------------
// marshalPlanInput
// ---------------------------------------------------------------------------

func TestMarshalPlanInput_NilInput_Error(t *testing.T) {
	_, err := marshalPlanInput(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plan must not be nil")
}

func TestMarshalPlanInput_ValidInput_RoundTrip(t *testing.T) {
	input := &graphqlmodel.ClassificationPlanInput{
		UseServiceWindows:    true,
		DefaultServiceWindow: "peak",
		DefaultSourceType:    "Home",
		ServiceTypes: []*graphqlmodel.ClassificationServiceTypeInput{
			{
				Type:                "voice",
				ChargingInformation: "IMS",
				SourceType:          "Home",
				ServiceDirection:    "MO",
				ServiceCategory:     "local",
				UnitType:            "SECONDS",
			},
		},
	}

	b, err := marshalPlanInput(input)
	require.NoError(t, err)
	assert.NotEmpty(t, b)

	// Round-trip: unmarshal and verify key fields survived.
	var roundTripped gomodel.ClassificationPlan
	require.NoError(t, json.Unmarshal(b, &roundTripped))
	assert.True(t, roundTripped.UseServiceWindows)
	assert.Equal(t, "peak", roundTripped.DefaultServiceWindow)
	require.Len(t, roundTripped.ServiceTypes, 1)
	assert.Equal(t, "voice", roundTripped.ServiceTypes[0].ServiceType)
}

func TestMarshalPlanInput_ServiceWindow_InvalidTime_Error(t *testing.T) {
	input := &graphqlmodel.ClassificationPlanInput{
		UseServiceWindows:    true,
		DefaultServiceWindow: "peak",
		DefaultSourceType:    "Home",
		ServiceWindows: []*graphqlmodel.ServiceWindowEntryInput{
			{Name: "peak", StartTime: "not-a-time", EndTime: "20:00"},
		},
		ServiceTypes: []*graphqlmodel.ClassificationServiceTypeInput{},
	}

	_, err := marshalPlanInput(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid startTime")
}

// ---------------------------------------------------------------------------
// parseClassificationUUID
// ---------------------------------------------------------------------------

func TestParseClassificationUUID_Valid(t *testing.T) {
	id := uuid.New().String()
	pgUID, err := parseClassificationUUID(id)
	require.NoError(t, err)
	assert.True(t, pgUID.Valid)
}

func TestParseClassificationUUID_Invalid_Error(t *testing.T) {
	_, err := parseClassificationUUID("not-a-uuid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid uuid")
}

// ---------------------------------------------------------------------------
// parseDateTime
// ---------------------------------------------------------------------------

func TestParseDateTime_ValidRFC3339(t *testing.T) {
	s := "2024-06-01T12:00:00Z"
	ts, err := parseDateTime(s)
	require.NoError(t, err)
	assert.True(t, ts.Valid)
	assert.Equal(t, 2024, ts.Time.Year())
}

func TestParseDateTime_Invalid_Error(t *testing.T) {
	_, err := parseDateTime("01/06/2024")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid DateTime")
}

// ---------------------------------------------------------------------------
// emailFromContext — no claims present
// ---------------------------------------------------------------------------

func TestEmailFromContext_NoClaims_ReturnsUnknown(t *testing.T) {
	email := emailFromContext(context.Background())
	assert.Equal(t, "unknown", email)
}

// ---------------------------------------------------------------------------
// GetClassification
// ---------------------------------------------------------------------------

func TestGetClassification_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	pgUID, idStr := buildClassificationID(t)
	planJSON := minimalPlanJSON(t)

	anyQueryRow1Classification(mockDB, mockRow)
	mockRow.On("Scan", scanMatchers()...).
		Run(populateClassificationScan(pgUID, "MyPlan", planJSON)).
		Return(nil)

	svc := newClassificationService(mockDB)
	c, err := svc.GetClassification(context.Background(), idStr)

	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, idStr, c.ClassificationID)
	assert.Equal(t, "MyPlan", c.Name)
	assert.Equal(t, graphqlmodel.ClassificationStatusDraft, c.Status)
	mockDB.AssertExpectations(t)
}

func TestGetClassification_InvalidUUID_Error(t *testing.T) {
	svc := newClassificationService(&servicesMockDBTX{})
	_, err := svc.GetClassification(context.Background(), "bad-id")
	require.Error(t, err)
}

func TestGetClassification_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	anyQueryRow1Classification(mockDB, mockRow)
	mockRow.On("Scan", scanMatchers()...).Return(errors.New("connection refused"))

	svc := newClassificationService(mockDB)
	_, err := svc.GetClassification(context.Background(), uuid.New().String())
	require.Error(t, err)
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// CreateClassification
// ---------------------------------------------------------------------------

func TestCreateClassification_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	pgUID, idStr := buildClassificationID(t)
	planJSON := minimalPlanJSON(t)

	anyQueryRow5Classification(mockDB, mockRow)
	mockRow.On("Scan", scanMatchers()...).
		Run(populateClassificationScan(pgUID, "NewPlan", planJSON)).
		Return(nil)

	svc := newClassificationService(mockDB)
	input := graphqlmodel.ClassificationInput{
		Name:          "NewPlan",
		EffectiveTime: "2024-07-01T00:00:00Z",
		Plan: &graphqlmodel.ClassificationPlanInput{
			UseServiceWindows:    false,
			DefaultServiceWindow: "peak",
			DefaultSourceType:    "Home",
			ServiceTypes:         []*graphqlmodel.ClassificationServiceTypeInput{},
		},
	}

	c, err := svc.CreateClassification(context.Background(), input)

	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, idStr, c.ClassificationID)
	assert.Equal(t, graphqlmodel.ClassificationStatusDraft, c.Status)
	mockDB.AssertExpectations(t)
}

func TestCreateClassification_InvalidEffectiveTime_Error(t *testing.T) {
	svc := newClassificationService(&servicesMockDBTX{})
	input := graphqlmodel.ClassificationInput{
		Name:          "BadTime",
		EffectiveTime: "not-a-date",
		Plan: &graphqlmodel.ClassificationPlanInput{
			UseServiceWindows: false, DefaultServiceWindow: "p", DefaultSourceType: "H",
			ServiceTypes: []*graphqlmodel.ClassificationServiceTypeInput{},
		},
	}
	_, err := svc.CreateClassification(context.Background(), input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse effectiveTime")
}

func TestCreateClassification_NilPlan_Error(t *testing.T) {
	svc := newClassificationService(&servicesMockDBTX{})
	input := graphqlmodel.ClassificationInput{
		Name:          "NoPlan",
		EffectiveTime: "2024-07-01T00:00:00Z",
		Plan:          nil,
	}
	_, err := svc.CreateClassification(context.Background(), input)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// CloneClassification
// ---------------------------------------------------------------------------

func TestCloneClassification_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	pgUID, idStr := buildClassificationID(t)
	planJSON := minimalPlanJSON(t)

	// First call: FindClassificationByID (1 SQL arg).
	anyQueryRow1Classification(mockDB, mockRow)
	// Second call: CreateClassification (5 SQL args).
	anyQueryRow5Classification(mockDB, mockRow)

	mockRow.On("Scan", scanMatchers()...).
		Run(populateClassificationScan(pgUID, "OriginalPlan", planJSON)).
		Return(nil).Times(2)

	svc := newClassificationService(mockDB)
	c, err := svc.CloneClassification(context.Background(), idStr)

	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, graphqlmodel.ClassificationStatusDraft, c.Status)
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// SubmitClassificationForApproval
// ---------------------------------------------------------------------------

func TestSubmitClassificationForApproval_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	pgUID, idStr := buildClassificationID(t)
	planJSON := minimalPlanJSON(t)

	anyQueryRow1Classification(mockDB, mockRow)
	mockRow.On("Scan", scanMatchers()...).
		Run(func(args mock.Arguments) {
			populateClassificationScan(pgUID, "Plan", planJSON)(args)
			// Override status to PENDING
			*(args[6].(*string)) = "PENDING"
		}).
		Return(nil)

	svc := newClassificationService(mockDB)
	c, err := svc.SubmitClassificationForApproval(context.Background(), idStr)

	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, graphqlmodel.ClassificationStatusPending, c.Status)
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// ApproveClassificationPlan
// ---------------------------------------------------------------------------

func TestApproveClassificationPlan_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	pgUID, idStr := buildClassificationID(t)
	planJSON := minimalPlanJSON(t)

	anyQueryRow2Classification(mockDB, mockRow)
	mockRow.On("Scan", scanMatchers()...).
		Run(func(args mock.Arguments) {
			populateClassificationScan(pgUID, "Plan", planJSON)(args)
			*(args[5].(*pgtype.Text)) = pgtype.Text{String: "approver@test.com", Valid: true}
			*(args[6].(*string)) = "ACTIVE"
		}).
		Return(nil)

	svc := newClassificationService(mockDB)
	c, err := svc.ApproveClassificationPlan(context.Background(), idStr)

	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, graphqlmodel.ClassificationStatusActive, c.Status)
	require.NotNil(t, c.ApprovedBy)
	assert.Equal(t, "approver@test.com", *c.ApprovedBy)
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// DeclineClassificationPlan
// ---------------------------------------------------------------------------

func TestDeclineClassificationPlan_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	pgUID, idStr := buildClassificationID(t)
	planJSON := minimalPlanJSON(t)

	anyQueryRow1Classification(mockDB, mockRow)
	mockRow.On("Scan", scanMatchers()...).
		Run(populateClassificationScan(pgUID, "Plan", planJSON)).
		Return(nil)

	svc := newClassificationService(mockDB)
	c, err := svc.DeclineClassificationPlan(context.Background(), idStr)

	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, graphqlmodel.ClassificationStatusDraft, c.Status)
	mockDB.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// DeleteClassification
// ---------------------------------------------------------------------------

func TestDeleteClassification_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).
		Return(pgconn.CommandTag{}, nil)

	svc := newClassificationService(mockDB)
	ok, err := svc.DeleteClassification(context.Background(), uuid.New().String())

	require.NoError(t, err)
	assert.True(t, ok)
	mockDB.AssertExpectations(t)
}

func TestDeleteClassification_DBError(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).
		Return(pgconn.CommandTag{}, errors.New("foreign key violation"))

	svc := newClassificationService(mockDB)
	ok, err := svc.DeleteClassification(context.Background(), uuid.New().String())

	require.Error(t, err)
	assert.False(t, ok)
	assert.Contains(t, err.Error(), "delete classification")
	mockDB.AssertExpectations(t)
}

func TestDeleteClassification_InvalidUUID_Error(t *testing.T) {
	svc := newClassificationService(&servicesMockDBTX{})
	_, err := svc.DeleteClassification(context.Background(), "bad-uuid")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// UpdateClassificationPlan
// ---------------------------------------------------------------------------

func TestUpdateClassificationPlan_Success(t *testing.T) {
	mockDB := &servicesMockDBTX{}
	mockRow := &servicesMockRow{}

	pgUID, idStr := buildClassificationID(t)
	planJSON := minimalPlanJSON(t)

	anyQueryRow4Classification(mockDB, mockRow)
	mockRow.On("Scan", scanMatchers()...).
		Run(populateClassificationScan(pgUID, "UpdatedPlan", planJSON)).
		Return(nil)

	svc := newClassificationService(mockDB)
	input := graphqlmodel.ClassificationInput{
		Name:          "UpdatedPlan",
		EffectiveTime: "2024-08-01T00:00:00Z",
		Plan: &graphqlmodel.ClassificationPlanInput{
			UseServiceWindows:    false,
			DefaultServiceWindow: "offpeak",
			DefaultSourceType:    "Home",
			ServiceTypes:         []*graphqlmodel.ClassificationServiceTypeInput{},
		},
	}

	c, err := svc.UpdateClassificationPlan(context.Background(), idStr, input)

	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, idStr, c.ClassificationID)
	mockDB.AssertExpectations(t)
}
