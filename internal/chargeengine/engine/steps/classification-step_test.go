package steps

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/engine/providers/carriers"
	"go-ocs/internal/model"
	"go-ocs/internal/charging"
	"go-ocs/internal/nchf"
	"go-ocs/internal/store/sqlc"
)

// mockClassifyInfra is a testify-backed mock for the Infrastructure interface,
// used specifically by the classification-step tests.
type mockClassifyInfra struct {
	mock.Mock
}

func (m *mockClassifyInfra) FetchClassificationPlan() (*model.ClassificationPlan, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ClassificationPlan), args.Error(1)
}

func (m *mockClassifyInfra) FetchCarrierContainer() (*carriers.CarrierContainer, error) {
	args := m.Called()
	return args.Get(0).(*carriers.CarrierContainer), args.Error(1)
}

func (m *mockClassifyInfra) FindCarrierByMccMnc(mcc string, mnc string) (*sqlc.Carrier, error) {
	args := m.Called(mcc, mnc)
	return args.Get(0).(*sqlc.Carrier), args.Error(1)
}

func (m *mockClassifyInfra) FindCarrierBySource(mcc string, mnc string) string {
	return m.Called(mcc, mnc).String(0)
}

func (m *mockClassifyInfra) FindNumberPlan(number string) (*sqlc.AllNumbersRow, error) {
	args := m.Called(number)
	return args.Get(0).(*sqlc.AllNumbersRow), args.Error(1)
}

func (m *mockClassifyInfra) FindCarrierByDestination(number string) string {
	return m.Called(number).String(0)
}

func (m *mockClassifyInfra) FindRatingPlan(id uuid.UUID) (*model.RatePlan, error) {
	args := m.Called(id)
	return args.Get(0).(*model.RatePlan), args.Error(1)
}

func (m *mockClassifyInfra) FindSubscriber(msisdn string) (*model.Subscriber, error) {
	args := m.Called(msisdn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Subscriber), args.Error(1)
}

// buildClassifyDC constructs a ChargingContext for Classify tests.
func buildClassifyDC(infra *mockClassifyInfra, units []nchf.MultipleUnitUsage) *engine.ChargingContext {
	rg := int64(1)
	if len(units) == 0 {
		units = []nchf.MultipleUnitUsage{{RatingGroup: &rg}}
	}

	req := nchf.NewChargingDataRequest()
	req.ImsChargingInformation = &nchf.IMSChargingInformation{}
	req.MultipleUnitUsage = units

	cd := model.NewChargingData()
	cd.Subscriber = &model.Subscriber{
		SubscriberId:         uuid.New(),
		RatePlanId:           uuid.New(),
		WholesalerRatePlanId: uuid.New(),
	}

	appCtx := &appcontext.AppContext{
		Config: &appcontext.Config{},
	}

	return &engine.ChargingContext{
		StartTime:    time.Now(),
		AppContext:   appCtx,
		Infra:        infra,
		Request:      req,
		Response:     nchf.NewChargingDataResponse(),
		ChargingData: cd,
	}
}

func TestClassify_FetchClassificationPlanError(t *testing.T) {
	infra := &mockClassifyInfra{}
	infra.On("FetchClassificationPlan").Return(nil, errors.New("plan not found"))

	dc := buildClassifyDC(infra, nil)

	err := Classify(dc)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "plan not found")
	infra.AssertExpectations(t)
}

func TestClassify_Success(t *testing.T) {
	rg := int64(1)
	plan := &model.ClassificationPlan{
		ServiceTypes: []model.ServiceType{
			{
				ServiceIdentifier:   "voice",
				UnitType:            charging.UNITS,
				ServiceCategory:     "local",
				ChargingInformation: "IMS",
				SourceType:          `"*"`,
				ServiceDirection:    `"MO"`,
			},
		},
	}

	infra := &mockClassifyInfra{}
	infra.On("FetchClassificationPlan").Return(plan, nil)

	dc := buildClassifyDC(infra, []nchf.MultipleUnitUsage{{RatingGroup: &rg}})

	err := Classify(dc)

	require.NoError(t, err)
	assert.NotEmpty(t, dc.ChargingData.Classifications)
	infra.AssertExpectations(t)
}

func TestClassify_ClassificationsStoredOnContext(t *testing.T) {
	rg := int64(42)
	plan := &model.ClassificationPlan{
		ServiceTypes: []model.ServiceType{
			{
				ServiceIdentifier:   "sms",
				UnitType:            charging.UNITS,
				ServiceCategory:     "standard",
				ChargingInformation: "IMS",
				SourceType:          `"Home"`,
				ServiceDirection:    `"MO"`,
			},
		},
	}

	infra := &mockClassifyInfra{}
	infra.On("FetchClassificationPlan").Return(plan, nil)

	dc := buildClassifyDC(infra, []nchf.MultipleUnitUsage{{RatingGroup: &rg}})

	err := Classify(dc)

	require.NoError(t, err)
	_, ok := dc.ChargingData.Classifications[rg]
	assert.True(t, ok, "classification for rating group %d should be set", rg)
	infra.AssertExpectations(t)
}
