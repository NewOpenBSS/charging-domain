package events

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-ocs/internal/charging"
)

// ---------------------------------------------------------------------------
// NewChargeEvent
// ---------------------------------------------------------------------------

func TestNewChargeEvent_FieldsSet(t *testing.T) {
	reqID := "req-001"
	eventTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	ratedAt := time.Date(2024, 1, 15, 10, 30, 1, 0, time.UTC)

	ev := NewChargeEvent(reqID, eventTime, ratedAt)

	require.NotNil(t, ev)
	assert.Equal(t, reqID, ev.RequestID)
	assert.Equal(t, eventTime, ev.EventDateTime)
	require.NotNil(t, ev.RatedAt)
	assert.Equal(t, ratedAt, *ev.RatedAt)
}

// ---------------------------------------------------------------------------
// NewChargeSubscriber
// ---------------------------------------------------------------------------

func TestNewChargeSubscriber_FieldsSet(t *testing.T) {
	wholesaleID := uuid.New()
	contractID := uuid.New()
	subscriberID := uuid.New()
	ratePlanID := uuid.New()
	mvnoRatePlanID := uuid.New()
	msisdn := "0211234567"

	sub := NewChargeSubscriber(wholesaleID, contractID, subscriberID, msisdn, ratePlanID, mvnoRatePlanID, true)

	require.NotNil(t, sub)
	assert.Equal(t, wholesaleID, sub.WholesaleID)
	assert.Equal(t, contractID, sub.ContractID)
	assert.Equal(t, subscriberID, sub.SubscriberID)
	assert.Equal(t, msisdn, sub.Msisdn)
	assert.Equal(t, ratePlanID, sub.RatePlanID)
	assert.Equal(t, mvnoRatePlanID, sub.MvnoRatePlanID)
	assert.True(t, sub.AllowOOBCharging)
}

func TestNewChargeSubscriber_OOBChargingFalse(t *testing.T) {
	sub := NewChargeSubscriber(uuid.Nil, uuid.Nil, uuid.Nil, "021000000", uuid.Nil, uuid.Nil, false)
	assert.False(t, sub.AllowOOBCharging)
}

// ---------------------------------------------------------------------------
// NewChargeService
// ---------------------------------------------------------------------------

func TestNewChargeService_FieldsSet(t *testing.T) {
	rk := charging.RateKey{
		ServiceType:      "voice",
		SourceType:       "Home",
		ServiceDirection: charging.MO,
		ServiceCategory:  "local",
	}

	svc := NewChargeService(rk, "voice", 120, charging.SECONDS)

	require.NotNil(t, svc)
	assert.Equal(t, rk, svc.RateKey)
	assert.Equal(t, "voice", svc.ServiceIdentifier)
	assert.Equal(t, int64(120), svc.UnitsUsed)
	assert.Equal(t, charging.SECONDS, svc.UnitType)
}

// ---------------------------------------------------------------------------
// NewChargeInfo
// ---------------------------------------------------------------------------

func TestNewChargeInfo_CoreFieldsSet(t *testing.T) {
	units := int64(500)
	multiplier := decimal.NewFromFloat(1.5)
	minUnits := int64(60)
	roundIncrement := int64(1)
	unitPrice := decimal.NewFromFloat(0.05)
	value := decimal.NewFromFloat(25.0)

	ci := NewChargeInfo(units, multiplier, minUnits, roundIncrement, unitPrice, value,
		false, "qos-gold", 10, decimal.NewFromFloat(0.5), "group-a")

	require.NotNil(t, ci)
	assert.Equal(t, uuid.Nil, ci.ContractID)
	assert.Equal(t, units, ci.Units)
	assert.True(t, multiplier.Equal(ci.Multiplier))
	assert.Equal(t, minUnits, ci.MinimumUnits)
	assert.Equal(t, roundIncrement, ci.RoundingIncrement)
	assert.True(t, unitPrice.Equal(ci.UnitPrice))
	assert.True(t, value.Equal(ci.Value))
	assert.False(t, ci.ZeroRated)
}

func TestNewChargeInfo_ZeroRated(t *testing.T) {
	ci := NewChargeInfo(0, decimal.Zero, 0, 0, decimal.Zero, decimal.Zero,
		true, "", 0, decimal.Zero, "")

	require.NotNil(t, ci)
	assert.True(t, ci.ZeroRated)
	assert.Equal(t, int64(0), ci.Units)
}

// NOTE: NewChargeInfo accepts qosProfile, unaccountedUnits, unaccountedValue, and
// groupKey parameters but does not currently assign them to the returned struct.
// The fields QosProfile, UnaccountedUnits, UnaccountedValue, and GroupKey will
// always be zero-valued regardless of what is passed to the constructor.
func TestNewChargeInfo_UnassignedParams_ZeroValued(t *testing.T) {
	ci := NewChargeInfo(100, decimal.NewFromFloat(1.0), 0, 1,
		decimal.NewFromFloat(0.1), decimal.NewFromFloat(10.0),
		false, "qos-silver", 5, decimal.NewFromFloat(0.5), "group-b")

	// These parameters are accepted but not yet assigned by the constructor.
	assert.Empty(t, ci.QosProfile)
	assert.Equal(t, int64(0), ci.UnaccountedUnits)
	assert.True(t, ci.UnaccountedValue.IsZero())
	assert.Empty(t, ci.GroupKey)
}

// ---------------------------------------------------------------------------
// NewChargeRecord
// ---------------------------------------------------------------------------

func TestNewChargeRecord_FieldsSet(t *testing.T) {
	chargeID := "charge-xyz"
	event := &ChargeEvent{RequestID: "req-1", EventDateTime: time.Now()}
	subscriber := &ChargeSubscriber{Msisdn: "0211111111"}
	service := &ChargeService{ServiceIdentifier: "voice"}
	settlement := &ChargeInfo{Units: 100}
	wholesale := &ChargeInfo{Units: 200}
	retail := &ChargeInfo{Units: 300}

	rec := NewChargeRecord(chargeID, event, subscriber, service, settlement, wholesale, retail)

	require.NotNil(t, rec)
	assert.Equal(t, chargeID, rec.ChargeRecordID)
	assert.Equal(t, *event, rec.Event)
	assert.Equal(t, *subscriber, rec.Subscriber)
	assert.Equal(t, *service, rec.Service)
	assert.Equal(t, *settlement, rec.Settlement)
	assert.Equal(t, *wholesale, rec.Wholesale)
	assert.Equal(t, *retail, rec.Retail)
}
