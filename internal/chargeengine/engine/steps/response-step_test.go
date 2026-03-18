package steps

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/model"
	"go-ocs/internal/charging"
	"go-ocs/internal/nchf"
)

// buildResponseDC constructs a minimal ChargingContext sufficient for BuildResponse tests.
func buildResponseDC(seqNr int64, units []nchf.MultipleUnitUsage, grants map[int64][]model.Grants) *engine.ChargingContext {
	req := nchf.NewChargingDataRequest()
	req.InvocationSequenceNumber = &seqNr
	req.MultipleUnitUsage = units

	cd := model.NewChargingData()
	if grants != nil {
		cd.Grants = grants
	}

	return &engine.ChargingContext{
		StartTime:    time.Now(),
		Request:      req,
		Response:     nchf.NewChargingDataResponse(),
		ChargingData: cd,
	}
}

func TestBuildResponse_NoUnits(t *testing.T) {
	dc := buildResponseDC(1, nil, nil)
	err := BuildResponse(dc)
	require.NoError(t, err)
	assert.Nil(t, dc.Response.MultipleUnitInformation)
}

func TestBuildResponse_UnitWithNoGrant(t *testing.T) {
	rg := int64(10)
	dc := buildResponseDC(1, []nchf.MultipleUnitUsage{{RatingGroup: &rg}}, nil)
	err := BuildResponse(dc)
	require.NoError(t, err)
	require.Len(t, dc.Response.MultipleUnitInformation, 1)
	info := dc.Response.MultipleUnitInformation[0]
	assert.Equal(t, nchf.ResultCodeSuccess, *info.ResultCode)
	assert.Nil(t, info.GrantedUnit)
}

func TestBuildResponse_GrantedOctets(t *testing.T) {
	rg := int64(10)
	seqNr := int64(1)
	grants := map[int64][]model.Grants{
		rg: {{
			InvocationSequenceNumber: seqNr,
			RatingGroup:              rg,
			UnitType:                 charging.OCTETS,
			UnitsGranted:             2048,
			ValidityTime:             300,
			RetailTariff:             model.Tariff{QosProfileId: "gold"},
		}},
	}
	dc := buildResponseDC(seqNr, []nchf.MultipleUnitUsage{{RatingGroup: &rg}}, grants)

	err := BuildResponse(dc)
	require.NoError(t, err)
	require.Len(t, dc.Response.MultipleUnitInformation, 1)
	info := dc.Response.MultipleUnitInformation[0]
	assert.Equal(t, nchf.ResultCodeSuccess, *info.ResultCode)
	require.NotNil(t, info.GrantedUnit)
	assert.Equal(t, int64(2048), *info.GrantedUnit.TotalVolume)
	assert.Nil(t, info.GrantedUnit.Time)
	assert.Equal(t, int32(300), *info.ValidityTime)
	assert.Equal(t, "gold", *info.QosProfile)
}

func TestBuildResponse_GrantedSeconds(t *testing.T) {
	rg := int64(20)
	seqNr := int64(1)
	grants := map[int64][]model.Grants{
		rg: {{
			InvocationSequenceNumber: seqNr,
			RatingGroup:              rg,
			UnitType:                 charging.SECONDS,
			UnitsGranted:             60,
			ValidityTime:             600,
		}},
	}
	dc := buildResponseDC(seqNr, []nchf.MultipleUnitUsage{{RatingGroup: &rg}}, grants)

	err := BuildResponse(dc)
	require.NoError(t, err)
	info := dc.Response.MultipleUnitInformation[0]
	assert.Equal(t, nchf.ResultCodeSuccess, *info.ResultCode)
	require.NotNil(t, info.GrantedUnit)
	assert.Equal(t, int64(60), *info.GrantedUnit.Time)
	assert.Nil(t, info.GrantedUnit.TotalVolume)
}

func TestBuildResponse_GrantedServiceSpecificUnits(t *testing.T) {
	rg := int64(30)
	seqNr := int64(1)
	grants := map[int64][]model.Grants{
		rg: {{
			InvocationSequenceNumber: seqNr,
			RatingGroup:              rg,
			UnitType:                 charging.UNITS,
			UnitsGranted:             100,
		}},
	}
	dc := buildResponseDC(seqNr, []nchf.MultipleUnitUsage{{RatingGroup: &rg}}, grants)

	err := BuildResponse(dc)
	require.NoError(t, err)
	info := dc.Response.MultipleUnitInformation[0]
	assert.Equal(t, nchf.ResultCodeSuccess, *info.ResultCode)
	require.NotNil(t, info.GrantedUnit)
	assert.Equal(t, int64(100), *info.GrantedUnit.ServiceSpecificUnits)
}

func TestBuildResponse_ZeroUnitsGranted_QuotaLimitReached(t *testing.T) {
	rg := int64(10)
	seqNr := int64(1)
	grants := map[int64][]model.Grants{
		rg: {{
			InvocationSequenceNumber: seqNr,
			RatingGroup:              rg,
			UnitType:                 charging.OCTETS,
			UnitsGranted:             0,
		}},
	}
	dc := buildResponseDC(seqNr, []nchf.MultipleUnitUsage{{RatingGroup: &rg}}, grants)

	err := BuildResponse(dc)
	require.NoError(t, err)
	info := dc.Response.MultipleUnitInformation[0]
	assert.Equal(t, nchf.ResultCodeQuotaLimitReached, *info.ResultCode)
}

func TestBuildResponse_FinalUnitIndication(t *testing.T) {
	rg := int64(10)
	seqNr := int64(1)
	grants := map[int64][]model.Grants{
		rg: {{
			InvocationSequenceNumber: seqNr,
			RatingGroup:              rg,
			UnitType:                 charging.SECONDS,
			UnitsGranted:             30,
			FinalUnitIndication:      true,
		}},
	}
	dc := buildResponseDC(seqNr, []nchf.MultipleUnitUsage{{RatingGroup: &rg}}, grants)

	err := BuildResponse(dc)
	require.NoError(t, err)
	info := dc.Response.MultipleUnitInformation[0]
	require.NotNil(t, info.FinalUnitIndication)
	assert.Equal(t, nchf.FinalUnitActionTerminate, *info.FinalUnitIndication.FinalUnitAction)
}

func TestBuildResponse_GrantForDifferentSequenceNumber(t *testing.T) {
	// A grant exists but for a different invocation sequence number — treated as no grant.
	rg := int64(10)
	seqNr := int64(2)
	grants := map[int64][]model.Grants{
		rg: {{
			InvocationSequenceNumber: 1, // different sequence
			RatingGroup:              rg,
			UnitType:                 charging.SECONDS,
			UnitsGranted:             60,
		}},
	}
	dc := buildResponseDC(seqNr, []nchf.MultipleUnitUsage{{RatingGroup: &rg}}, grants)

	err := BuildResponse(dc)
	require.NoError(t, err)
	info := dc.Response.MultipleUnitInformation[0]
	// No matching grant → fallback success with no granted unit
	assert.Equal(t, nchf.ResultCodeSuccess, *info.ResultCode)
	assert.Nil(t, info.GrantedUnit)
}

func TestBuildResponse_SessionFailoverSet(t *testing.T) {
	dc := buildResponseDC(1, nil, nil)
	err := BuildResponse(dc)
	require.NoError(t, err)
	require.NotNil(t, dc.Response.SessionFailover)
	assert.Equal(t, "FAILOVER_NOT_SUPPORTED", *dc.Response.SessionFailover)
}

func TestBuildResponse_InvocationSequenceNumberCopied(t *testing.T) {
	seqNr := int64(5)
	dc := buildResponseDC(seqNr, nil, nil)
	err := BuildResponse(dc)
	require.NoError(t, err)
	require.NotNil(t, dc.Response.InvocationSequenceNumber)
	assert.Equal(t, seqNr, *dc.Response.InvocationSequenceNumber)
}

func TestBuildResponse_MultipleRatingGroups(t *testing.T) {
	rg1, rg2 := int64(10), int64(20)
	seqNr := int64(1)
	grantId := uuid.New()
	grants := map[int64][]model.Grants{
		rg1: {{InvocationSequenceNumber: seqNr, RatingGroup: rg1, UnitType: charging.SECONDS, UnitsGranted: 60, GrantId: grantId}},
		// rg2 has no grant
	}
	dc := buildResponseDC(seqNr, []nchf.MultipleUnitUsage{
		{RatingGroup: &rg1},
		{RatingGroup: &rg2},
	}, grants)

	err := BuildResponse(dc)
	require.NoError(t, err)
	require.Len(t, dc.Response.MultipleUnitInformation, 2)
}
