package business

import (
	"context"
	"errors"
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/engine/providers/carriers"
	"go-ocs/internal/chargeengine/model"
	"go-ocs/internal/charging"
	"go-ocs/internal/store/sqlc"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/twmb/franz-go/pkg/kgo"
)

type mockKafkaForRating struct {
	mock.Mock
}

func (m *mockKafkaForRating) Produce(ctx context.Context, r *kgo.Record, promise func(*kgo.Record, error)) {
	m.Called(ctx, r, promise)
}

func (m *mockKafkaForRating) PublishEvent(topicName string, key string, event any) {
	m.Called(topicName, key, event)
}

// MockInfrastructure for testing RateService
type MockInfrastructure struct {
	mock.Mock
}

func (m *MockInfrastructure) FindSubscriber(msisdn string) (*model.Subscriber, error) {
	return nil, nil
}

func (m *MockInfrastructure) FindRatingPlan(id uuid.UUID) (*model.RatePlan, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.RatePlan), args.Error(1)
}

func (m *MockInfrastructure) FetchClassificationPlan() (*model.Plan, error) { return nil, nil }
func (m *MockInfrastructure) FetchCarrierContainer() (*carriers.CarrierContainer, error) {
	return nil, nil
}
func (m *MockInfrastructure) FindCarrierByMccMnc(mcc string, mnc string) (*sqlc.Carrier, error) {
	return nil, nil
}
func (m *MockInfrastructure) FindCarrierBySource(mcc string, mnc string) string { return "" }
func (m *MockInfrastructure) FindNumberPlan(number string) (*sqlc.AllNumbersRow, error) {
	return nil, nil
}
func (m *MockInfrastructure) FindCarrierByDestination(number string) string { return "" }

func TestFindBestRateLine(t *testing.T) {
	ratePlan := &model.RatePlan{
		RatePlanID: "test-plan",
		RateLines: []model.RateLine{
			{
				ClassificationKey: charging.RateKey{ServiceType: "VOICE", SourceType: "*", ServiceDirection: charging.MO, ServiceCategory: "*"},
				Description:       "General Voice",
			},
			{
				ClassificationKey: charging.RateKey{ServiceType: "VOICE", SourceType: "LTE", ServiceDirection: charging.MO, ServiceCategory: "PREMIUM"},
				Description:       "Premium Voice LTE",
			},
		},
	}

	tests := []struct {
		name          string
		key           charging.RateKey
		expectedDesc  string
		expectedError bool
	}{
		{
			name: "Exact match wins",
			key: charging.RateKey{
				ServiceType:      "VOICE",
				SourceType:       "LTE",
				ServiceDirection: charging.MO,
				ServiceCategory:  "PREMIUM",
			},
			expectedDesc: "Premium Voice LTE",
		},
		{
			name: "Wildcard match works",
			key: charging.RateKey{
				ServiceType:      "VOICE",
				SourceType:       "3G",
				ServiceDirection: charging.MO,
				ServiceCategory:  "NORMAL",
			},
			expectedDesc: "General Voice",
		},
		{
			name: "No match returns error",
			key: charging.RateKey{
				ServiceType:      "DATA",
				SourceType:       "LTE",
				ServiceDirection: charging.MO,
				ServiceCategory:  "NORMAL",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, err := findBestRateLine(ratePlan, tt.key)
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, line)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, line)
				assert.Equal(t, tt.expectedDesc, line.Description)
			}
		})
	}
}

func TestRateService(t *testing.T) {
	settlementId := uuid.New()
	wholesaleId := uuid.New()
	retailId := uuid.New()

	settlementPlan := &model.RatePlan{
		RatePlanID: settlementId.String(),
		RateLines: []model.RateLine{
			{ClassificationKey: charging.RateKey{ServiceType: "*", SourceType: "*", ServiceDirection: charging.ANY, ServiceCategory: "*"}, Description: "Settlement"},
		},
	}
	wholesalePlan := &model.RatePlan{
		RatePlanID: wholesaleId.String(),
		RateLines: []model.RateLine{
			{ClassificationKey: charging.RateKey{ServiceType: "*", SourceType: "*", ServiceDirection: charging.ANY, ServiceCategory: "*"}, Description: "Wholesale"},
		},
	}
	retailPlan := &model.RatePlan{
		RatePlanID: retailId.String(),
		RateLines: []model.RateLine{
			{ClassificationKey: charging.RateKey{ServiceType: "*", SourceType: "*", ServiceDirection: charging.ANY, ServiceCategory: "*"}, Description: "Retail"},
		},
	}

	infra := &MockInfrastructure{}
	infra.On("FindRatingPlan", settlementId).Return(settlementPlan, nil)
	infra.On("FindRatingPlan", wholesaleId).Return(wholesalePlan, nil)
	infra.On("FindRatingPlan", retailId).Return(retailPlan, nil)

	config := &appcontext.Config{
		Engine: appcontext.EngineConfig{
			SettlementPlanId: settlementId,
		},
	}
	appCtx := &appcontext.AppContext{
		Config:       config,
		KafkaManager: new(mockKafkaForRating),
	}

	dc := &engine.ChargingContext{
		AppContext: appCtx,
		Infra:      infra,
		ChargingData: &model.ChargingData{
			Subscriber: &model.Subscriber{
				WholesalerRatePlanId: wholesaleId,
				RatePlanId:           retailId,
			},
		},
	}

	classification := model.Classification{
		Ratekey: charging.RateKey{
			ServiceType:      "VOICE",
			SourceType:       "LTE",
			ServiceDirection: charging.MO,
			ServiceCategory:  "NORMAL",
		},
	}

	sLine, wLine, rLine, err := RateService(dc, classification)

	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, "Settlement", sLine.Description)
	assert.Equal(t, "Wholesale", wLine.Description)
	assert.Equal(t, "Retail", rLine.Description)

	infra.AssertExpectations(t)
}

func TestRateService_PlanNotFound(t *testing.T) {
	settlementId := uuid.New()

	infra := &MockInfrastructure{}
	infra.On("FindRatingPlan", settlementId).Return(nil, errors.New("not found"))

	config := &appcontext.Config{
		Engine: appcontext.EngineConfig{
			SettlementPlanId: settlementId,
		},
	}
	appCtx := &appcontext.AppContext{
		Config:       config,
		KafkaManager: new(mockKafkaForRating),
	}

	dc := &engine.ChargingContext{
		AppContext: appCtx,
		Infra:      infra,
	}

	classification := model.Classification{}

	sLine, wLine, rLine, err := RateService(dc, classification)

	assert.Error(t, err)
	assert.Nil(t, sLine)
	assert.Nil(t, wLine)
	assert.Nil(t, rLine)
}
