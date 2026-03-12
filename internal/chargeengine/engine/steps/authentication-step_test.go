package steps

import (
	"context"
	"errors"
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/engine/providers/carriers"
	"go-ocs/internal/chargeengine/model"
	"go-ocs/internal/chargeengine/ocserrors"
	"go-ocs/internal/nchf"
	"go-ocs/internal/store/sqlc"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/twmb/franz-go/pkg/kgo"
)

type mockKafkaForAuthStep struct {
	mock.Mock
}

func (m *mockKafkaForAuthStep) Produce(ctx context.Context, r *kgo.Record, promise func(*kgo.Record, error)) {
	m.Called(ctx, r, promise)
}

func (m *mockKafkaForAuthStep) PublishEvent(topicName string, key string, event any) {
	m.Called(topicName, key, event)
}

type mockAuthInfra struct {
	mock.Mock
}

func (m *mockAuthInfra) FetchClassificationPlan() (*model.Plan, error) {
	args := m.Called()
	return args.Get(0).(*model.Plan), args.Error(1)
}

func (m *mockAuthInfra) FetchCarrierContainer() (*carriers.CarrierContainer, error) {
	args := m.Called()
	return args.Get(0).(*carriers.CarrierContainer), args.Error(1)
}

func (m *mockAuthInfra) FindCarrierByMccMnc(mcc string, mnc string) (*sqlc.Carrier, error) {
	args := m.Called(mcc, mnc)
	return args.Get(0).(*sqlc.Carrier), args.Error(1)
}

func (m *mockAuthInfra) FindCarrierBySource(mcc string, mnc string) string {
	args := m.Called(mcc, mnc)
	return args.String(0)
}

func (m *mockAuthInfra) FindNumberPlan(number string) (*sqlc.AllNumbersRow, error) {
	args := m.Called(number)
	return args.Get(0).(*sqlc.AllNumbersRow), args.Error(1)
}

func (m *mockAuthInfra) FindCarrierByDestination(number string) string {
	args := m.Called(number)
	return args.String(0)
}

func (m *mockAuthInfra) FindRatingPlan(id uuid.UUID) (*model.RatePlan, error) {
	args := m.Called(id)
	return args.Get(0).(*model.RatePlan), args.Error(1)
}

func (m *mockAuthInfra) FindSubscriber(msisdn string) (*model.Subscriber, error) {
	args := m.Called(msisdn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Subscriber), args.Error(1)
}

func TestAuthenticate(t *testing.T) {
	nationalDialCode := "64"
	appCtx := &appcontext.AppContext{
		Config: &appcontext.Config{
			Engine: appcontext.EngineConfig{
				NationalDialCode: nationalDialCode,
			},
		},
		KafkaManager: new(mockKafkaForAuthStep),
	}

	tests := []struct {
		name                 string
		subscriberIdentifier *string
		mockSetup            func(m *mockAuthInfra)
		expectedError        ocserrors.Code
		expectedMsisdn       string
		verifySubscriber     bool
	}{
		{
			name:                 "Nil Subscriber Identifier",
			subscriberIdentifier: nil,
			mockSetup:            func(m *mockAuthInfra) {},
			expectedError:        ocserrors.CodeUnknownSubscriber,
		},
		{
			name: "MSISDN already starts with 0",
			subscriberIdentifier: func() *string {
				s := "021123456"
				return &s
			}(),
			mockSetup: func(m *mockAuthInfra) {
				m.On("FindSubscriber", "021123456").Return(&model.Subscriber{Msisdn: "021123456"}, nil)
			},
			expectedMsisdn:   "021123456",
			verifySubscriber: true,
		},
		{
			name: "MSISDN starts with NationalDialCode",
			subscriberIdentifier: func() *string {
				s := nationalDialCode + "21123456"
				return &s
			}(),
			mockSetup: func(m *mockAuthInfra) {
				m.On("FindSubscriber", "021123456").Return(&model.Subscriber{Msisdn: "021123456"}, nil)
			},
			expectedMsisdn:   "021123456",
			verifySubscriber: true,
		},
		{
			name: "MSISDN is international, not starting with NationalDialCode",
			subscriberIdentifier: func() *string {
				s := "121123456"
				return &s
			}(),
			mockSetup: func(m *mockAuthInfra) {
				m.On("FindSubscriber", "0121123456").Return(&model.Subscriber{Msisdn: "0121123456"}, nil)
			},
			expectedMsisdn:   "0121123456",
			verifySubscriber: true,
		},
		{
			name: "Subscriber Not Found",
			subscriberIdentifier: func() *string {
				s := "021111111"
				return &s
			}(),
			mockSetup: func(m *mockAuthInfra) {
				m.On("FindSubscriber", "021111111").Return(nil, errors.New("not found"))
			},
			expectedError: ocserrors.CodeUnknownSubscriber,
		},
		{
			name: "MSISDN is empty",
			subscriberIdentifier: func() *string {
				s := ""
				return &s
			}(),
			mockSetup: func(m *mockAuthInfra) {
				m.On("FindSubscriber", "").Return(&model.Subscriber{Msisdn: ""}, nil)
			},
			expectedMsisdn:   "",
			verifySubscriber: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			infra := new(mockAuthInfra)
			tt.mockSetup(infra)

			req := nchf.NewChargingDataRequest()
			req.SubscriberIdentifier = tt.subscriberIdentifier

			dc := engine.NewChargingContext(appCtx, infra, "test-session", req)

			err := Authenticate(dc)

			if tt.expectedError != "" {
				assert.Error(t, err)
				var ocsErr *ocserrors.OcsError
				if assert.True(t, errors.As(err, &ocsErr)) {
					assert.Equal(t, tt.expectedError, ocsErr.Code)
				}
			} else {
				assert.NoError(t, err)
				if tt.verifySubscriber {
					assert.NotNil(t, dc.ChargingData.Subscriber)
					assert.Equal(t, tt.expectedMsisdn, dc.ChargingData.Subscriber.Msisdn)
				}
			}
			infra.AssertExpectations(t)
		})
	}
}
