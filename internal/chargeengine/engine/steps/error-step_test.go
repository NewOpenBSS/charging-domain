package steps

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"go-ocs/internal/chargeengine/appcontext"
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/model"
	"go-ocs/internal/nchf"
)

var sentinelErr = errors.New("original error")

// buildErrorDC constructs a ChargingContext for HandleError tests.
func buildErrorDC(
	seqNr int64,
	subscriber *model.Subscriber,
	units []nchf.MultipleUnitUsage,
	grants map[int64][]model.Grants,
	qm *mockQuotaManager,
) *engine.ChargingContext {
	req := nchf.NewChargingDataRequest()
	req.InvocationSequenceNumber = &seqNr
	req.MultipleUnitUsage = units

	cd := model.NewChargingData()
	cd.Subscriber = subscriber
	if grants != nil {
		cd.Grants = grants
	}

	appCtx := &appcontext.AppContext{
		Config:       &appcontext.Config{},
		QuotaManager: qm,
	}

	return &engine.ChargingContext{
		StartTime:    time.Now(),
		AppContext:   appCtx,
		Request:      req,
		Response:     nchf.NewChargingDataResponse(),
		ChargingData: cd,
	}
}

func TestHandleError_ReturnsOriginalError(t *testing.T) {
	qm := &mockQuotaManager{}
	dc := buildErrorDC(1, nil, nil, nil, qm)

	result := HandleError(dc, sentinelErr)

	assert.Equal(t, sentinelErr, result)
	qm.AssertNotCalled(t, "Release")
}

func TestHandleError_NoGrantsForRatingGroup(t *testing.T) {
	rg := int64(10)
	qm := &mockQuotaManager{}
	dc := buildErrorDC(1, &model.Subscriber{SubscriberId: uuid.New()},
		[]nchf.MultipleUnitUsage{{RatingGroup: &rg}},
		nil, // no grants
		qm,
	)

	result := HandleError(dc, sentinelErr)

	assert.Equal(t, sentinelErr, result)
	qm.AssertNotCalled(t, "Release")
}

func TestHandleError_GrantMatchesSequence_ReleaseCalled(t *testing.T) {
	rg := int64(10)
	seqNr := int64(1)
	subscriberID := uuid.New()
	grantID := uuid.New()

	qm := &mockQuotaManager{}
	qm.On("Release", mock.Anything, subscriberID, grantID).Return(nil)

	grants := map[int64][]model.Grants{
		rg: {{
			InvocationSequenceNumber: seqNr,
			GrantId:                  grantID,
		}},
	}
	dc := buildErrorDC(seqNr,
		&model.Subscriber{SubscriberId: subscriberID},
		[]nchf.MultipleUnitUsage{{RatingGroup: &rg}},
		grants,
		qm,
	)

	result := HandleError(dc, sentinelErr)

	assert.Equal(t, sentinelErr, result)
	qm.AssertExpectations(t)
}

func TestHandleError_GrantDoesNotMatchSequence_ReleaseNotCalled(t *testing.T) {
	rg := int64(10)
	seqNr := int64(2) // current sequence
	subscriberID := uuid.New()

	qm := &mockQuotaManager{}

	grants := map[int64][]model.Grants{
		rg: {{
			InvocationSequenceNumber: 1, // different from current seqNr
			GrantId:                  uuid.New(),
		}},
	}
	dc := buildErrorDC(seqNr,
		&model.Subscriber{SubscriberId: subscriberID},
		[]nchf.MultipleUnitUsage{{RatingGroup: &rg}},
		grants,
		qm,
	)

	result := HandleError(dc, sentinelErr)

	assert.Equal(t, sentinelErr, result)
	qm.AssertNotCalled(t, "Release")
}

func TestHandleError_GrantMatchesSequence_NilSubscriber_ReleaseNotCalled(t *testing.T) {
	rg := int64(10)
	seqNr := int64(1)

	qm := &mockQuotaManager{}

	grants := map[int64][]model.Grants{
		rg: {{
			InvocationSequenceNumber: seqNr,
			GrantId:                  uuid.New(),
		}},
	}
	dc := buildErrorDC(seqNr,
		nil, // no subscriber
		[]nchf.MultipleUnitUsage{{RatingGroup: &rg}},
		grants,
		qm,
	)

	result := HandleError(dc, sentinelErr)

	assert.Equal(t, sentinelErr, result)
	qm.AssertNotCalled(t, "Release")
}

func TestHandleError_ReleaseFails_OriginalErrorStillReturned(t *testing.T) {
	rg := int64(10)
	seqNr := int64(1)
	subscriberID := uuid.New()
	grantID := uuid.New()

	qm := &mockQuotaManager{}
	qm.On("Release", mock.Anything, subscriberID, grantID).Return(errors.New("release failed"))

	grants := map[int64][]model.Grants{
		rg: {{
			InvocationSequenceNumber: seqNr,
			GrantId:                  grantID,
		}},
	}
	dc := buildErrorDC(seqNr,
		&model.Subscriber{SubscriberId: subscriberID},
		[]nchf.MultipleUnitUsage{{RatingGroup: &rg}},
		grants,
		qm,
	)

	result := HandleError(dc, sentinelErr)

	// Original error is returned even when Release fails.
	assert.Equal(t, sentinelErr, result)
	qm.AssertExpectations(t)
}
