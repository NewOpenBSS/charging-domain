package steps

import (
	"context"
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/model"
	"go-ocs/internal/charging"
	"go-ocs/internal/events"
	"go-ocs/internal/nchf"
	"go-ocs/internal/quota"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/twmb/franz-go/pkg/kgo"
)

type mockKafkaProducer struct {
	mock.Mock
}

func (m *mockKafkaProducer) PublishEvent(topicName string, key string, event any) {
	m.Called(topicName, key, event)
}

func (m *mockKafkaProducer) Produce(ctx context.Context, r *kgo.Record, promise func(*kgo.Record, error)) {
	m.Called(ctx, r, promise)
	if promise != nil {
		promise(r, nil)
	}
}

type mockAccountingQuotaManager struct {
	mock.Mock
}

func (m *mockAccountingQuotaManager) ReserveQuota(ctx context.Context, reservationId uuid.UUID, subscriberId uuid.UUID, reason quota.ReasonCode, rateKey charging.RateKey, unitType charging.UnitType, requestedUnits int64, unitPrice decimal.Decimal, multiplier decimal.Decimal, validityTime time.Duration, allowOOBCharging bool) (int64, error) {
	args := m.Called(ctx, reservationId, subscriberId, reason, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, allowOOBCharging)
	return int64(args.Int(0)), args.Error(1)
}

func (m *mockAccountingQuotaManager) Debit(ctx context.Context, subscriberID uuid.UUID, requestId string, reservationId uuid.UUID, usedUnits int64, unitType charging.UnitType, reclaimUnusedUnits bool) (*quota.DebitResponse, error) {
	args := m.Called(ctx, subscriberID, requestId, reservationId, usedUnits, unitType, reclaimUnusedUnits)
	return args.Get(0).(*quota.DebitResponse), args.Error(1)
}

func (m *mockAccountingQuotaManager) Release(ctx context.Context, subscriberId uuid.UUID, reservationId uuid.UUID) error {
	args := m.Called(ctx, subscriberId, reservationId)
	return args.Error(0)
}

func TestAccounting_Success(t *testing.T) {
	mockQM := new(mockAccountingQuotaManager)
	mockKafka := new(mockKafkaProducer)

	subscriberId := uuid.New()
	contractId := uuid.New()
	wholesaleId := uuid.New()
	ratePlanId := uuid.New()
	wholesalerRatePlanId := uuid.New()

	chargingId := "test-charging-id"
	invocationSeq := int64(1)
	ratingGroup := int64(10)
	usedSeconds := int(30)

	dc := &engine.ChargingContext{
		Request: &nchf.ChargingDataRequest{
			ChargingId:               &chargingId,
			InvocationSequenceNumber: &invocationSeq,
			InvocationTimeStamp:      (*nchf.LocalDateTime)(&[]time.Time{time.Now()}[0]),
			MultipleUnitUsage: []nchf.MultipleUnitUsage{
				{
					RatingGroup: &ratingGroup,
					UsedUnitContainer: []nchf.UsedUnitContainer{
						{
							Time: &usedSeconds,
						},
					},
				},
			},
		},
		ChargingData: &model.ChargingData{
			Subscriber: &model.Subscriber{
				SubscriberId:         subscriberId,
				ContractId:           contractId,
				WholesaleId:          wholesaleId,
				RatePlanId:           ratePlanId,
				WholesalerRatePlanId: wholesalerRatePlanId,
				Msisdn:               "123456789",
			},
			Classifications: map[int64]model.Classification{
				ratingGroup: {
					UnitType: charging.SECONDS,
				},
			},
			Grants: map[int64][]model.Grants{
				ratingGroup: {
					{
						GrantId:                  uuid.New(),
						InvocationSequenceNumber: 0,
						UnitsGranted:             60,
						UnitType:                 charging.SECONDS,
						GrantedTime:              time.Now(),
						ValidityTime:             3600,
						SettlementTariff: model.Tariff{
							UnitPrice: decimal.NewFromFloat(0.01),
							RateLine: &model.RateLine{
								MinimumUnits:      1,
								RoundingIncrement: 1,
							},
						},
						WholesaleTariff: model.Tariff{
							UnitPrice: decimal.NewFromFloat(0.02),
							RateLine: &model.RateLine{
								MinimumUnits:      1,
								RoundingIncrement: 1,
							},
						},
						RetailTariff: model.Tariff{
							UnitPrice:  decimal.NewFromFloat(0.05),
							Multiplier: decimal.NewFromInt(1),
							RateLine: &model.RateLine{
								MinimumUnits:      1,
								RoundingIncrement: 1,
							},
						},
					},
				},
			},
		},
		AppContext: &appcontext.AppContext{
			QuotaManager: mockQM,
			KafkaManager: mockKafka,
			Config: &appcontext.Config{
				Kafkaconfig: &events.KafkaConfig{
					Topics: map[string]string{
						"charge-record": "test-topic",
					},
				},
			},
		},
	}

	grant := dc.ChargingData.Grants[ratingGroup][0]
	chargeRecordId := chargingId + ";1;" + grant.GrantId.String()

	mockQM.On("Debit", mock.Anything, subscriberId, chargeRecordId, grant.GrantId, int64(usedSeconds), grant.UnitType, false).
		Return(&quota.DebitResponse{
			UnitsDebited: int64(usedSeconds),
			UnitsValue:   decimal.NewFromFloat(1.5), // 30 * 0.05
		}, nil)

	mockKafka.On("PublishEvent", "charge-record", chargeRecordId, mock.Anything).Return()

	err := Accounting(dc)

	assert.NoError(t, err)
	assert.Equal(t, int64(30), dc.ChargingData.Grants[ratingGroup][0].UnitsGranted)
	mockQM.AssertExpectations(t)
	mockKafka.AssertExpectations(t)
}

func TestAccounting_ExpiredGrant(t *testing.T) {
	ratingGroup := int64(10)
	chargingId := "test-charging-id"

	dc := &engine.ChargingContext{
		Request: &nchf.ChargingDataRequest{
			ChargingId: &chargingId,
			MultipleUnitUsage: []nchf.MultipleUnitUsage{
				{
					RatingGroup: &ratingGroup,
				},
			},
		},
		ChargingData: &model.ChargingData{
			Grants: map[int64][]model.Grants{
				ratingGroup: {
					{
						GrantId:      uuid.New(),
						GrantedTime:  time.Now().Add(-2 * time.Hour),
						ValidityTime: 3600, // 1 hour
					},
				},
			},
		},
	}

	err := Accounting(dc)

	assert.NoError(t, err)
	assert.Empty(t, dc.ChargingData.Grants[ratingGroup])
}

func TestAccounting_UsedMoreThanGranted(t *testing.T) {
	ratingGroup := int64(10)
	chargingId := "test-charging-id"
	invocationSeq := int64(1)
	usedSecondsVal := int(100)

	dc := &engine.ChargingContext{
		Request: &nchf.ChargingDataRequest{
			ChargingId:               &chargingId,
			InvocationSequenceNumber: &invocationSeq,
			MultipleUnitUsage: []nchf.MultipleUnitUsage{
				{
					RatingGroup: &ratingGroup,
					UsedUnitContainer: []nchf.UsedUnitContainer{
						{
							Time: &usedSecondsVal,
						},
					},
				},
			},
		},
		ChargingData: &model.ChargingData{
			Subscriber: &model.Subscriber{},
			Classifications: map[int64]model.Classification{
				ratingGroup: {
					UnitType: charging.SECONDS,
				},
			},
			Grants: map[int64][]model.Grants{
				ratingGroup: {
					{
						GrantId:      uuid.New(),
						UnitsGranted: 60,
						GrantedTime:  time.Now(),
						ValidityTime: 3600,
						RetailTariff: model.Tariff{
							Multiplier: decimal.NewFromInt(1),
							RateLine: &model.RateLine{
								MinimumUnits:      1,
								RoundingIncrement: 1,
							},
						},
					},
				},
			},
		},
	}

	err := Accounting(dc)

	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "more units debited than granted")
	}
}
