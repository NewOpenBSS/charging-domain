package steps

import (
	"context"
	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/engine/providers/carriers"
	"go-ocs/internal/chargeengine/model"
	"go-ocs/internal/chargeengine/ocserrors"
	"go-ocs/internal/charging"
	"go-ocs/internal/nchf"
	"go-ocs/internal/quota"
	"go-ocs/internal/store/sqlc"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/twmb/franz-go/pkg/kgo"
)

type mockKafkaForRatingStep struct {
	mock.Mock
}

func (m *mockKafkaForRatingStep) Produce(ctx context.Context, r *kgo.Record, promise func(*kgo.Record, error)) {
	m.Called(ctx, r, promise)
}

func (m *mockKafkaForRatingStep) PublishEvent(topicName string, key string, event any) {
	m.Called(topicName, key, event)
}

// MockQuotaManager for testing
type mockQuotaManager struct {
	mock.Mock
}

func (m *mockQuotaManager) ReserveQuota(ctx context.Context, now time.Time, reservationId uuid.UUID, subscriberId uuid.UUID, reason quota.ReasonCode, rateKey charging.RateKey, unitType charging.UnitType, requestedUnits int64, unitPrice decimal.Decimal, multiplier decimal.Decimal, validityTime time.Duration, allowOOBCharging bool) (int64, error) {
	args := m.Called(ctx, now, reservationId, subscriberId, reason, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, allowOOBCharging)
	if fn, ok := args.Get(0).(func(context.Context, time.Time, uuid.UUID, uuid.UUID, quota.ReasonCode, charging.RateKey, charging.UnitType, int64, decimal.Decimal, decimal.Decimal, time.Duration, bool) int64); ok {
		return fn(ctx, now, reservationId, subscriberId, reason, rateKey, unitType, requestedUnits, unitPrice, multiplier, validityTime, allowOOBCharging), args.Error(1)
	}
	return int64(args.Int(0)), args.Error(1)
}

func (m *mockQuotaManager) Debit(ctx context.Context, now time.Time, subscriberID uuid.UUID, requestId string, reservationId uuid.UUID, usedUnits int64, unitType charging.UnitType, reclaimUnusedUnits bool) (*quota.DebitResponse, error) {
	args := m.Called(ctx, now, subscriberID, requestId, reservationId, usedUnits, unitType, reclaimUnusedUnits)
	return args.Get(0).(*quota.DebitResponse), args.Error(1)
}

func (m *mockQuotaManager) Release(ctx context.Context, subscriberId uuid.UUID, reservationId uuid.UUID) error {
	args := m.Called(ctx, subscriberId, reservationId)
	return args.Error(0)
}

func TestCalcTariff(t *testing.T) {
	decimalDigits := int32(2)

	t.Run("Barred RateLine", func(t *testing.T) {
		rl := &model.RateLine{Barred: true}
		got, err := calcTariff(rl, nil, decimalDigits)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if ocsErr, ok := err.(*ocserrors.OcsError); !ok || ocsErr.Code != ocserrors.CodeServiceBarred {
			t.Errorf("expected ServiceBarred error, got %v", err)
		}
		if got != nil {
			t.Errorf("expected nil tariff, got %v", got)
		}
	})

	t.Run("ACTUAL TariffType", func(t *testing.T) {
		rl := &model.RateLine{
			TariffType:    model.ACTUAL,
			BaseTariff:    decimal.NewFromFloat(10.0),
			UnitOfMeasure: 2,
			Multiplier:    decimal.NewFromFloat(1.5),
			QosProfile:    "gold",
		}
		got, err := calcTariff(rl, nil, decimalDigits)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedPrice := decimal.NewFromFloat(5.0) // 10 / 2
		if !got.UnitPrice.Equal(expectedPrice) {
			t.Errorf("expected UnitPrice %v, got %v", expectedPrice, got.UnitPrice)
		}
		if !got.Multiplier.Equal(rl.Multiplier) {
			t.Errorf("expected Multiplier %v, got %v", rl.Multiplier, got.Multiplier)
		}
		if got.QosProfileId != rl.QosProfile {
			t.Errorf("expected QosProfileId %v, got %v", rl.QosProfile, got.QosProfileId)
		}
		if got.RateLine != rl {
			t.Errorf("expected RateLine pointer to match")
		}
	})

	t.Run("PERCENTAGE TariffType - success", func(t *testing.T) {
		rl := &model.RateLine{
			TariffType: model.PERCENTAGE,
			BaseTariff: decimal.NewFromFloat(0.1), // 10% markup
		}
		dep := &model.Tariff{
			UnitPrice: decimal.NewFromFloat(100.0),
		}
		got, err := calcTariff(rl, dep, decimalDigits)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedPrice := decimal.NewFromFloat(110.0) // 100 * (1 + 0.1)
		if !got.UnitPrice.Equal(expectedPrice) {
			t.Errorf("expected UnitPrice %v, got %v", expectedPrice, got.UnitPrice)
		}
	})

	t.Run("PERCENTAGE TariffType - no dependant", func(t *testing.T) {
		rl := &model.RateLine{TariffType: model.PERCENTAGE}
		_, err := calcTariff(rl, nil, decimalDigits)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("MARKUP TariffType - success", func(t *testing.T) {
		rl := &model.RateLine{
			TariffType: model.MARKUP,
			BaseTariff: decimal.NewFromFloat(5.0),
		}
		dep := &model.Tariff{
			UnitPrice: decimal.NewFromFloat(100.0),
		}
		got, err := calcTariff(rl, dep, decimalDigits)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedPrice := decimal.NewFromFloat(105.0) // 100 + 5
		if !got.UnitPrice.Equal(expectedPrice) {
			t.Errorf("expected UnitPrice %v, got %v", expectedPrice, got.UnitPrice)
		}
	})

	t.Run("MARKUP TariffType - no dependant", func(t *testing.T) {
		rl := &model.RateLine{TariffType: model.MARKUP}
		_, err := calcTariff(rl, nil, decimalDigits)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})
}

// MockInfrastructure for testing
type mockInfra struct {
	ratingPlans map[uuid.UUID]*model.RatePlan
}

func (m *mockInfra) FindRatingPlan(u uuid.UUID) (*model.RatePlan, error) {
	if p, ok := m.ratingPlans[u]; ok {
		return p, nil
	}
	return nil, ocserrors.CreateGeneralError("plan not found")
}
func (m *mockInfra) FetchClassificationPlan() (*model.Plan, error)              { return nil, nil }
func (m *mockInfra) FetchCarrierContainer() (*carriers.CarrierContainer, error) { return nil, nil }
func (m *mockInfra) FindCarrierByMccMnc(mcc string, mnc string) (*sqlc.Carrier, error) {
	return nil, nil
}
func (m *mockInfra) FindCarrierBySource(mcc string, mnc string) string         { return "" }
func (m *mockInfra) FindNumberPlan(number string) (*sqlc.AllNumbersRow, error) { return nil, nil }
func (m *mockInfra) FindCarrierByDestination(number string) string             { return "" }
func (m *mockInfra) FindSubscriber(msisdn string) (*model.Subscriber, error)   { return nil, nil }

// We need to import carriers and sqlc for mockInfra
// Wait, I can't easily import internal packages of other modules if I'm in steps.
// Actually they are in the same project.

func TestRateUnit(t *testing.T) {
	// Setup AppContext and Config
	settlementId := uuid.New()
	wholesaleId := uuid.New()
	retailId := uuid.New()

	cfg := &appcontext.Config{
		Engine: appcontext.EngineConfig{
			DecimalDigits:         2,
			DefaultValidityWindow: 1 * time.Hour,
			SettlementPlanId:      settlementId,
			ScalingValidityWindow: 3 * time.Minute,
		},
	}
	mqm := &mockQuotaManager{}
	appCtx := &appcontext.AppContext{
		Config:       cfg,
		QuotaManager: mqm,
		KafkaManager: new(mockKafkaForRatingStep),
	}
	mqm.On("ReserveQuota", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, now time.Time, resID uuid.UUID, subID uuid.UUID, reason quota.ReasonCode, rk charging.RateKey, ut charging.UnitType, units int64, up decimal.Decimal, mult decimal.Decimal, vt time.Duration, oob bool) int64 {
		return units
	}, (error)(nil)).Maybe()

	// Setup Rating Plans
	rk := charging.RateKey{ServiceType: "voice"}
	rlSettlement := model.RateLine{ClassificationKey: rk, TariffType: model.ACTUAL, BaseTariff: decimal.NewFromInt(1), UnitOfMeasure: 1, Multiplier: decimal.NewFromInt(1)}
	rlWholesale := model.RateLine{ClassificationKey: rk, TariffType: model.ACTUAL, BaseTariff: decimal.NewFromInt(2), UnitOfMeasure: 1, Multiplier: decimal.NewFromInt(1)}
	rlRetail := model.RateLine{ClassificationKey: rk, TariffType: model.ACTUAL, BaseTariff: decimal.NewFromInt(3), UnitOfMeasure: 1, Multiplier: decimal.NewFromInt(1)}

	infra := &mockInfra{
		ratingPlans: map[uuid.UUID]*model.RatePlan{
			settlementId: {RatePlanID: settlementId.String(), RateLines: []model.RateLine{rlSettlement}},
			wholesaleId:  {RatePlanID: wholesaleId.String(), RateLines: []model.RateLine{rlWholesale}},
			retailId:     {RatePlanID: retailId.String(), RateLines: []model.RateLine{rlRetail}},
		},
	}

	dc := &engine.ChargingContext{
		AppContext: appCtx,
		Infra:      infra,
		ChargingData: &model.ChargingData{
			Subscriber: &model.Subscriber{
				WholesalerRatePlanId: wholesaleId,
				RatePlanId:           retailId,
				SubscriberId:         uuid.New(),
			},
			Classifications: map[int64]model.Classification{
				10: {Ratekey: rk, UnitType: charging.SECONDS},
			},
			EventTime: time.Now().Add(-1 * time.Minute),
		},
		Request: &nchf.ChargingDataRequest{
			InvocationSequenceNumber: int64Ptr(1),
		},
	}

	t.Run("Success rating", func(t *testing.T) {
		rg := int64(10)
		requestedTime := 60
		u := nchf.MultipleUnitUsage{
			RatingGroup: &rg,
			RequestedUnit: &nchf.RequestedUnit{
				Time: &requestedTime,
			},
		}

		grant, err := rateUnit(dc, u)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if grant.RatingGroup != rg {
			t.Errorf("expected RG %v, got %v", rg, grant.RatingGroup)
		}
		// Scaling factor: 3 min / 1 min = 3. 60 * 3 = 180.
		if grant.UnitsGranted != 180 {
			t.Errorf("expected units granted 180 (scaled), got %v", grant.UnitsGranted)
		}
	})

	t.Run("No Rating Group", func(t *testing.T) {
		u := nchf.MultipleUnitUsage{RatingGroup: nil}
		_, err := rateUnit(dc, u)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("OCTETS units", func(t *testing.T) {
		rg := int64(10)
		requestedVolume := int64(1024)
		u := nchf.MultipleUnitUsage{
			RatingGroup: &rg,
			RequestedUnit: &nchf.RequestedUnit{
				TotalVolume: &requestedVolume,
			},
		}
		// Update classification to OCTETS
		dc.ChargingData.Classifications[rg] = model.Classification{Ratekey: rk, UnitType: charging.OCTETS}

		grant, err := rateUnit(dc, u)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Scaling: interval 1 min, scaling window 3 min -> factor 3. 1024 * 3 = 3072.
		if grant.UnitsGranted != 3072 {
			t.Errorf("expected units granted 3072, got %v", grant.UnitsGranted)
		}
	})
}

func TestRate(t *testing.T) {
	// Setup AppContext and Config
	settlementId := uuid.New()
	wholesaleId := uuid.New()
	retailId := uuid.New()

	cfg := &appcontext.Config{
		Engine: appcontext.EngineConfig{
			DecimalDigits:         2,
			DefaultValidityWindow: 1 * time.Hour,
			SettlementPlanId:      settlementId,
			ScalingValidityWindow: 3 * time.Minute,
		},
	}
	mqm := &mockQuotaManager{}
	appCtx := &appcontext.AppContext{
		Config:       cfg,
		QuotaManager: mqm,
		KafkaManager: new(mockKafkaForRatingStep),
	}
	mqm.On("ReserveQuota", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(func(ctx context.Context, now time.Time, resID uuid.UUID, subID uuid.UUID, reason quota.ReasonCode, rk charging.RateKey, ut charging.UnitType, units int64, up decimal.Decimal, mult decimal.Decimal, vt time.Duration, oob bool) int64 {
		return units
	}, (error)(nil)).Maybe()

	// Setup Rating Plans
	rk := charging.RateKey{ServiceType: "voice"}
	rlSettlement := model.RateLine{ClassificationKey: rk, TariffType: model.ACTUAL, BaseTariff: decimal.NewFromInt(1), UnitOfMeasure: 1, Multiplier: decimal.NewFromInt(1)}
	rlWholesale := model.RateLine{ClassificationKey: rk, TariffType: model.ACTUAL, BaseTariff: decimal.NewFromInt(2), UnitOfMeasure: 1, Multiplier: decimal.NewFromInt(1)}
	rlRetail := model.RateLine{ClassificationKey: rk, TariffType: model.ACTUAL, BaseTariff: decimal.NewFromInt(3), UnitOfMeasure: 1, Multiplier: decimal.NewFromInt(1)}

	infra := &mockInfra{
		ratingPlans: map[uuid.UUID]*model.RatePlan{
			settlementId: {RatePlanID: settlementId.String(), RateLines: []model.RateLine{rlSettlement}},
			wholesaleId:  {RatePlanID: wholesaleId.String(), RateLines: []model.RateLine{rlWholesale}},
			retailId:     {RatePlanID: retailId.String(), RateLines: []model.RateLine{rlRetail}},
		},
	}

	rg1 := int64(10)
	rg2 := int64(20)
	requestedTime := 60

	dc := &engine.ChargingContext{
		AppContext: appCtx,
		Infra:      infra,
		ChargingData: &model.ChargingData{
			Subscriber: &model.Subscriber{
				WholesalerRatePlanId: wholesaleId,
				RatePlanId:           retailId,
				SubscriberId:         uuid.New(),
			},
			Classifications: map[int64]model.Classification{
				rg1: {Ratekey: rk, UnitType: charging.SECONDS},
				rg2: {Ratekey: rk, UnitType: charging.SECONDS},
			},
			EventTime: time.Now().Add(-1 * time.Hour), // Large interval to skip scaling for sequence > 0
		},
		Request: &nchf.ChargingDataRequest{
			InvocationSequenceNumber: int64Ptr(0),
			MultipleUnitUsage: []nchf.MultipleUnitUsage{
				{
					RatingGroup: &rg1,
					RequestedUnit: &nchf.RequestedUnit{
						Time: &requestedTime,
					},
				},
				{
					RatingGroup: &rg2,
					RequestedUnit: &nchf.RequestedUnit{
						Time: &requestedTime,
					},
				},
			},
		},
	}

	t.Run("Success rate all units", func(t *testing.T) {
		err := Rate(dc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(dc.ChargingData.Grants) != 2 {
			t.Errorf("expected 2 grants, got %v", len(dc.ChargingData.Grants))
		}
	})

	t.Run("One unit fails", func(t *testing.T) {
		rgFail := int64(999)
		dc.Request.MultipleUnitUsage = []nchf.MultipleUnitUsage{{
			RatingGroup: &rgFail,
		}}
		err := Rate(dc)
		if err == nil {
			t.Errorf("expected error for missing classification, got nil")
		}
	})
}

func int64Ptr(i int64) *int64 { return &i }
