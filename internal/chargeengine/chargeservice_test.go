package chargeengine

import (
	"encoding/json"
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/engine/providers/carriers"
	"go-ocs/internal/model"
	"go-ocs/internal/charging"
	"go-ocs/internal/nchf"
	"go-ocs/internal/quota"
	"go-ocs/internal/store"
	"go-ocs/internal/store/sqlc"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockInfrastructure is a mock of interfaces.Infrastructure
type MockInfrastructure struct {
	mock.Mock
}

func (m *MockInfrastructure) FetchClassificationPlan() (*model.ClassificationPlan, error) {
	args := m.Called()
	return args.Get(0).(*model.ClassificationPlan), args.Error(1)
}

func (m *MockInfrastructure) FetchCarrierContainer() (*carriers.CarrierContainer, error) {
	args := m.Called()
	return args.Get(0).(*carriers.CarrierContainer), args.Error(1)
}

func (m *MockInfrastructure) FindCarrierByMccMnc(mcc string, mnc string) (*sqlc.Carrier, error) {
	args := m.Called(mcc, mnc)
	return args.Get(0).(*sqlc.Carrier), args.Error(1)
}

func (m *MockInfrastructure) FindCarrierBySource(mcc string, mnc string) string {
	args := m.Called(mcc, mnc)
	return args.String(0)
}

func (m *MockInfrastructure) FindNumberPlan(number string) (*sqlc.AllNumbersRow, error) {
	args := m.Called(number)
	return args.Get(0).(*sqlc.AllNumbersRow), args.Error(1)
}

func (m *MockInfrastructure) FindCarrierByDestination(number string) string {
	args := m.Called(number)
	return args.String(0)
}

func (m *MockInfrastructure) FindRatingPlan(id uuid.UUID) (*model.RatePlan, error) {
	args := m.Called(id)
	return args.Get(0).(*model.RatePlan), args.Error(1)
}

func (m *MockInfrastructure) FindSubscriber(msisdn string) (*model.Subscriber, error) {
	args := m.Called(msisdn)
	return args.Get(0).(*model.Subscriber), args.Error(1)
}

func setupTest(t *testing.T) (*appcontext.AppContext, *MockInfrastructure, *MockQuotaManager, *MockDBTX, *MockKafkaManager) {
	mockInfra := new(MockInfrastructure)
	mockQM := new(MockQuotaManager)
	mockDB := new(MockDBTX)
	mockKafka := new(MockKafkaManager)

	config := &appcontext.Config{}
	config.Engine.NationalDialCode = "27"
	config.Engine.DecimalDigits = 2
	config.Engine.DefaultValidityWindow = 3600 * time.Second
	config.Engine.SettlementPlanId = uuid.New()

	appCtx := &appcontext.AppContext{
		Config:       config,
		QuotaManager: mockQM,
		Store:        &store.Store{Q: sqlc.New(mockDB)},
		KafkaManager: mockKafka,
	}

	return appCtx, mockInfra, mockQM, mockDB, mockKafka
}

func TestProcessCharging(t *testing.T) {
	appCtx, mockInfra, mockQM, mockDB, mockKafka := setupTest(t)
	sessionId := "test-session"
	msisdn := "27123456789"
	subscriberID := uuid.New()
	ratingGroup := int64(1)
	chargingID := "test-charging-id"

	request := nchf.NewChargingDataRequest()
	request.ChargingId = &chargingID
	request.SubscriberIdentifier = &msisdn
	request.InvocationSequenceNumber = new(int64)
	*request.InvocationSequenceNumber = 0
	now := time.Now()
	request.InvocationTimeStamp = (*nchf.LocalDateTime)(&[]time.Time{now}[0])
	request.ImsChargingInformation = &nchf.IMSChargingInformation{}
	request.MultipleUnitUsage = []nchf.MultipleUnitUsage{
		{
			RatingGroup: &ratingGroup,
			RequestedUnit: &nchf.RequestedUnit{
				ServiceSpecificUnits: new(int64),
			},
		},
	}
	*request.MultipleUnitUsage[0].RequestedUnit.ServiceSpecificUnits = 100

	subscriber := &model.Subscriber{
		SubscriberId:         subscriberID,
		Msisdn:               msisdn,
		RatePlanId:           uuid.New(),
		WholesalerRatePlanId: uuid.New(),
	}

	mockInfra.On("FindSubscriber", "0123456789").Return(subscriber, nil)

	classificationPlan := &model.ClassificationPlan{
		ServiceTypes: []model.ServiceType{
			{
				ServiceIdentifier:   "test-service",
				UnitType:            charging.UNITS,
				ServiceCategory:     "test-category",
				ChargingInformation: "IMS",
				SourceType:          "\"*\"",
				ServiceDirection:    "\"MO\"",
			},
		},
	}
	mockInfra.On("FetchClassificationPlan").Return(classificationPlan, nil)

	ratePlan := &model.RatePlan{
		RatePlanID: subscriber.RatePlanId.String(),
		RateLines: []model.RateLine{
			{
				ClassificationKey: charging.RateKey{
					ServiceType:      "*",
					SourceType:       "*",
					ServiceDirection: charging.ANY,
					ServiceCategory:  "*",
				},
				BaseTariff:    decimal.NewFromInt(10),
				UnitOfMeasure: model.Quantity(1),
				TariffType:    model.ACTUAL,
				Multiplier:    decimal.NewFromInt(1),
			},
		},
	}
	mockInfra.On("FindRatingPlan", subscriber.RatePlanId).Return(ratePlan, nil)
	mockInfra.On("FindRatingPlan", appCtx.Config.Engine.SettlementPlanId).Return(ratePlan, nil)
	mockInfra.On("FindRatingPlan", subscriber.WholesalerRatePlanId).Return(ratePlan, nil)

	mockQM.On("ReserveQuota", mock.Anything, mock.Anything, mock.Anything, subscriberID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(int64(100), nil)

	// Mock DB calls for CreateChargeDataStep
	mockRow := new(MockRow)
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow).Once()
	mockRow.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(pgx.ErrNoRows).Once()
	// Mock DB call for CreateTrace
	mockRowTrace := new(MockRow)
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockRowTrace).Once()
	mockRowTrace.On("Scan", mock.Anything).Return(nil).Once()

	// Mock DB calls for SaveChargeDataStep
	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, nil)

	mockKafka.On("PublishEvent", mock.Anything, mock.Anything, mock.Anything).Return()

	resp, err := ProcessCharging(appCtx, mockInfra, sessionId, request)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, nchf.ResultCodeSuccess, *resp.MultipleUnitInformation[0].ResultCode)
	assert.Equal(t, int64(100), *resp.MultipleUnitInformation[0].GrantedUnit.ServiceSpecificUnits)

	mockInfra.AssertExpectations(t)
	mockQM.AssertExpectations(t)
	mockDB.AssertExpectations(t)
}

func TestProcessOneTimeCharging(t *testing.T) {
	appCtx, mockInfra, mockQM, mockDB, mockKafka := setupTest(t)
	sessionId := "test-session-ot"
	msisdn := "27123456789"
	subscriberID := uuid.New()
	ratingGroup := int64(1)
	chargingID := "test-charging-id-ot"

	request := nchf.NewChargingDataRequest()
	request.ChargingId = &chargingID
	request.SubscriberIdentifier = &msisdn
	request.InvocationSequenceNumber = new(int64)
	*request.InvocationSequenceNumber = 0
	now := time.Now()
	request.InvocationTimeStamp = (*nchf.LocalDateTime)(&[]time.Time{now}[0])
	request.ImsChargingInformation = &nchf.IMSChargingInformation{}
	request.MultipleUnitUsage = []nchf.MultipleUnitUsage{
		{
			RatingGroup: &ratingGroup,
			RequestedUnit: &nchf.RequestedUnit{
				ServiceSpecificUnits: new(int64),
			},
			UsedUnitContainer: []nchf.UsedUnitContainer{
				{
					ServiceSpecificUnits: new(int64),
				},
			},
		},
	}
	*request.MultipleUnitUsage[0].RequestedUnit.ServiceSpecificUnits = 100
	*request.MultipleUnitUsage[0].UsedUnitContainer[0].ServiceSpecificUnits = 50

	subscriber := &model.Subscriber{
		SubscriberId:         subscriberID,
		Msisdn:               msisdn,
		RatePlanId:           uuid.New(),
		WholesalerRatePlanId: uuid.New(),
	}

	mockInfra.On("FindSubscriber", "0123456789").Return(subscriber, nil)

	classificationPlan := &model.ClassificationPlan{
		ServiceTypes: []model.ServiceType{
			{
				ServiceIdentifier:   "test-service",
				UnitType:            charging.UNITS,
				ServiceCategory:     "test-category",
				ChargingInformation: "IMS",
				SourceType:          "\"*\"",
				ServiceDirection:    "\"MO\"",
			},
		},
	}
	mockInfra.On("FetchClassificationPlan").Return(classificationPlan, nil)

	ratePlan := &model.RatePlan{
		RatePlanID: subscriber.RatePlanId.String(),
		RateLines: []model.RateLine{
			{
				ClassificationKey: charging.RateKey{
					ServiceType:      "*",
					SourceType:       "*",
					ServiceDirection: charging.ANY,
					ServiceCategory:  "*",
				},
				BaseTariff:    decimal.NewFromInt(10),
				UnitOfMeasure: model.Quantity(1),
				TariffType:    model.ACTUAL,
				Multiplier:    decimal.NewFromInt(1),
			},
		},
	}
	mockInfra.On("FindRatingPlan", subscriber.RatePlanId).Return(ratePlan, nil)
	mockInfra.On("FindRatingPlan", appCtx.Config.Engine.SettlementPlanId).Return(ratePlan, nil)
	mockInfra.On("FindRatingPlan", subscriber.WholesalerRatePlanId).Return(ratePlan, nil)

	mockQM.On("ReserveQuota", mock.Anything, mock.Anything, mock.Anything, subscriberID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(int64(100), nil)

	mockQM.On("Debit", mock.Anything, mock.Anything, subscriberID, mock.Anything, mock.Anything, int64(50), mock.Anything, mock.Anything).Return(&quota.DebitResponse{
		UnitsDebited: 50,
		UnitsValue:   decimal.NewFromInt(500),
	}, nil)

	// Mock DB calls for CreateChargeDataStep
	mockRow := new(MockRow)
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow).Once()
	mockRow.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(pgx.ErrNoRows).Once()
	// Mock DB call for CreateTrace
	mockRowTrace := new(MockRow)
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockRowTrace).Once()
	mockRowTrace.On("Scan", mock.Anything).Return(nil).Once()

	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, nil)

	mockKafka.On("PublishEvent", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

	resp, err := ProcessOneTimeCharging(appCtx, mockInfra, sessionId, request)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, nchf.ResultCodeSuccess, *resp.MultipleUnitInformation[0].ResultCode)

	mockInfra.AssertExpectations(t)
	mockQM.AssertExpectations(t)
}

func TestUpdateChargingData(t *testing.T) {
	appCtx, mockInfra, mockQM, mockDB, _ := setupTest(t)
	sessionId := "test-session-update"
	msisdn := "27123456789"
	subscriberID := uuid.New()
	ratingGroup := int64(1)
	chargingID := "test-charging-id-update"

	request := nchf.NewChargingDataRequest()
	request.ChargingId = &chargingID
	request.SubscriberIdentifier = &msisdn
	request.InvocationSequenceNumber = new(int64)
	*request.InvocationSequenceNumber = 1
	now := time.Now()
	request.InvocationTimeStamp = (*nchf.LocalDateTime)(&[]time.Time{now}[0])
	request.ImsChargingInformation = &nchf.IMSChargingInformation{}
	request.MultipleUnitUsage = []nchf.MultipleUnitUsage{
		{
			RatingGroup: &ratingGroup,
			RequestedUnit: &nchf.RequestedUnit{
				ServiceSpecificUnits: new(int64),
			},
			UsedUnitContainer: []nchf.UsedUnitContainer{
				{
					ServiceSpecificUnits: new(int64),
				},
			},
		},
	}
	*request.MultipleUnitUsage[0].RequestedUnit.ServiceSpecificUnits = 100
	*request.MultipleUnitUsage[0].UsedUnitContainer[0].ServiceSpecificUnits = 50

	subscriber := &model.Subscriber{
		SubscriberId:         subscriberID,
		Msisdn:               msisdn,
		RatePlanId:           uuid.New(),
		WholesalerRatePlanId: uuid.New(),
	}

	mockInfra.On("FindSubscriber", "0123456789").Return(subscriber, nil)

	classificationPlan := &model.ClassificationPlan{
		ServiceTypes: []model.ServiceType{
			{
				ServiceIdentifier:   "test-service",
				UnitType:            charging.UNITS,
				ServiceCategory:     "test-category",
				ChargingInformation: "IMS",
				SourceType:          "\"*\"",
				ServiceDirection:    "\"MO\"",
			},
		},
	}
	mockInfra.On("FetchClassificationPlan").Return(classificationPlan, nil)

	ratePlan := &model.RatePlan{
		RatePlanID: subscriber.RatePlanId.String(),
		RateLines: []model.RateLine{
			{
				ClassificationKey: charging.RateKey{
					ServiceType:      "*",
					SourceType:       "*",
					ServiceDirection: charging.ANY,
					ServiceCategory:  "*",
				},
				BaseTariff:    decimal.NewFromInt(10),
				UnitOfMeasure: model.Quantity(1),
				TariffType:    model.ACTUAL,
				Multiplier:    decimal.NewFromInt(1),
			},
		},
	}
	mockInfra.On("FindRatingPlan", subscriber.RatePlanId).Return(ratePlan, nil)
	mockInfra.On("FindRatingPlan", appCtx.Config.Engine.SettlementPlanId).Return(ratePlan, nil)
	mockInfra.On("FindRatingPlan", subscriber.WholesalerRatePlanId).Return(ratePlan, nil)

	mockQM.On("ReserveQuota", mock.Anything, mock.Anything, mock.Anything, subscriberID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(int64(100), nil)

	mockQM.On("Debit", mock.Anything, mock.Anything, subscriberID, mock.Anything, mock.Anything, int64(50), mock.Anything, mock.Anything).Return(&quota.DebitResponse{
		UnitsDebited: 50,
		UnitsValue:   decimal.NewFromInt(500),
	}, nil).Once()

	// Mock DB calls for LoadChargeDataStep
	initialCD := model.NewChargingData()
	initialCD.Subscriber = subscriber
	initialCD.NewRecord = false
	initialCD.Classifications = make(map[int64]model.Classification)
	initialCD.Classifications[ratingGroup] = model.Classification{
		Ratekey: charging.RateKey{
			ServiceType:      "test-service",
			SourceType:       "ANY",
			ServiceDirection: "MO",
			ServiceCategory:  "test-category",
			ServiceWindow:    "ANY",
		},
		UnitType: charging.UNITS,
	}
	initialCD.Grants = make(map[int64][]model.Grants)
	initialCD.Grants[ratingGroup] = []model.Grants{
		{
			GrantId:     uuid.New(),
			RatingGroup: ratingGroup,
			RateKey: charging.RateKey{
				ServiceType:      "test-service",
				SourceType:       "ANY",
				ServiceDirection: "MO",
				ServiceCategory:  "test-category",
				ServiceWindow:    "ANY",
			},
		},
	}
	cdBytes, _ := json.Marshal(initialCD)

	mockRowLoad := new(MockRow)
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRowLoad).Once()
	mockRowLoad.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		*(args[0].(*string)) = chargingID
		*(args[1].(*int64)) = 0
		*(args[3].(*[]byte)) = cdBytes
	}).Return(nil).Once()

	// Mock DB call for CreateTrace
	mockRowTrace := new(MockRow)
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockRowTrace).Once()
	mockRowTrace.On("Scan", mock.Anything).Return(nil).Once()

	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, nil)

	resp, err := UpdateChargingData(appCtx, mockInfra, sessionId, request)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, nchf.ResultCodeSuccess, *resp.MultipleUnitInformation[0].ResultCode)
}

func TestReleaseChargingData(t *testing.T) {
	appCtx, mockInfra, mockQM, mockDB, _ := setupTest(t)
	sessionId := "test-session-release"
	msisdn := "27123456789"
	subscriberID := uuid.New()
	ratingGroup := int64(1)
	chargingID := "test-charging-id-release"

	request := nchf.NewChargingDataRequest()
	request.ChargingId = &chargingID
	request.SubscriberIdentifier = &msisdn
	request.InvocationSequenceNumber = new(int64)
	*request.InvocationSequenceNumber = 2
	request.ImsChargingInformation = &nchf.IMSChargingInformation{}
	request.MultipleUnitUsage = []nchf.MultipleUnitUsage{
		{
			RatingGroup: &ratingGroup,
			UsedUnitContainer: []nchf.UsedUnitContainer{
				{
					ServiceSpecificUnits: new(int64),
				},
			},
		},
	}
	*request.MultipleUnitUsage[0].UsedUnitContainer[0].ServiceSpecificUnits = 50

	subscriber := &model.Subscriber{
		SubscriberId:         subscriberID,
		Msisdn:               msisdn,
		RatePlanId:           uuid.New(),
		WholesalerRatePlanId: uuid.New(),
	}

	// Mock DB calls for LoadChargeDataStep
	initialCD := model.NewChargingData()
	initialCD.Subscriber = subscriber
	initialCD.NewRecord = false
	grantID := uuid.New()
	initialCD.Classifications = make(map[int64]model.Classification)
	initialCD.Classifications[ratingGroup] = model.Classification{
		Ratekey: charging.RateKey{
			ServiceType:      "test-service",
			SourceType:       "ANY",
			ServiceDirection: "MO",
			ServiceCategory:  "test-category",
			ServiceWindow:    "ANY",
		},
		UnitType: charging.UNITS,
	}
	initialCD.Grants = make(map[int64][]model.Grants)
	initialCD.Grants[ratingGroup] = []model.Grants{
		{
			GrantId:     grantID,
			RatingGroup: ratingGroup,
			RateKey: charging.RateKey{
				ServiceType:      "test-service",
				SourceType:       "ANY",
				ServiceDirection: "MO",
				ServiceCategory:  "test-category",
				ServiceWindow:    "ANY",
			},
		},
	}
	cdBytes, _ := json.Marshal(initialCD)

	mockRowLoad := new(MockRow)
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRowLoad).Once()
	mockRowLoad.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		*(args[0].(*string)) = chargingID
		*(args[1].(*int64)) = 1
		*(args[3].(*[]byte)) = cdBytes
	}).Return(nil).Once()

	mockQM.On("Debit", mock.Anything, mock.Anything, subscriberID, mock.Anything, grantID, int64(50), mock.Anything, mock.Anything).Return(&quota.DebitResponse{
		UnitsDebited: 50,
		UnitsValue:   decimal.NewFromInt(500),
	}, nil).Once()

	// Mock DB call for CreateTrace
	mockRowTrace := new(MockRow)
	mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockRowTrace).Once()
	mockRowTrace.On("Scan", mock.Anything).Return(nil).Once()

	mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.CommandTag{}, nil)

	resp, err := ReleaseChargingData(appCtx, mockInfra, sessionId, request)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, nchf.ResultCodeSuccess, *resp.MultipleUnitInformation[0].ResultCode)
}
