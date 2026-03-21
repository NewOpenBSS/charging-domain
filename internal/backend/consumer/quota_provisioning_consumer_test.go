package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"

	"go-ocs/internal/charging"
	"go-ocs/internal/events"
	"go-ocs/internal/quota"
)

// ---------------------------------------------------------------------------
// Mock QuotaProvisioner
// ---------------------------------------------------------------------------

type mockQuotaProvisioner struct {
	mock.Mock
}

func (m *mockQuotaProvisioner) ProvisionCounter(ctx context.Context, req quota.ProvisionCounterRequest) error {
	return m.Called(ctx, req).Error(0)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var (
	testProvEventID      = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	testProvSubscriberID = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	testProvProductID    = uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
)

// newQuotaProvisioningRecord encodes v as JSON and wraps it in a *kgo.Record.
func newQuotaProvisioningRecord(t *testing.T, v any) *kgo.Record {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return &kgo.Record{
		Topic:     "public.quota-provisioning",
		Partition: 0,
		Offset:    0,
		Value:     data,
	}
}

// newQuotaProvisioningConsumer builds a consumer with a nil kgo.Client (disabled-Kafka mode).
func newQuotaProvisioningConsumer(p QuotaProvisioner) *QuotaProvisioningConsumer {
	return &QuotaProvisioningConsumer{
		client:      nil,
		provisioner: p,
		topic:       "public.quota-provisioning",
	}
}

// baseProvisioningEvent returns a minimal valid QuotaProvisioningEvent.
func baseProvisioningEvent() events.QuotaProvisioningEvent {
	return events.QuotaProvisioningEvent{
		EventID:              testProvEventID,
		SubscriberID:         testProvSubscriberID,
		ProductID:            testProvProductID,
		ProductName:          "Test Product",
		Description:          "Test counter",
		UnitType:             charging.UNITS,
		Priority:             10,
		InitialBalance:       decimal.NewFromInt(100),
		CanRepayLoan:         false,
		CounterSelectionKeys: []charging.RateKey{},
		ReasonCode:           events.ProvisioningReasonQuotaProvisioned,
	}
}

// ---------------------------------------------------------------------------
// handleRecord — valid event dispatches to provisioner
// ---------------------------------------------------------------------------

func TestQuotaProvisioningConsumer_HandleRecord_ValidEvent_CallsProvisionCounter(t *testing.T) {
	t.Parallel()
	p := &mockQuotaProvisioner{}
	event := baseProvisioningEvent()

	// CounterID is derived from EventID in the consumer — verify the derivation.
	p.On("ProvisionCounter", mock.Anything, mock.MatchedBy(func(req quota.ProvisionCounterRequest) bool {
		return req.CounterID == testProvEventID &&
			req.SubscriberID == testProvSubscriberID &&
			req.ReasonCode == quota.ReasonQuotaProvisioned
	})).Return(nil)

	c := newQuotaProvisioningConsumer(p)
	err := c.handleRecord(context.Background(), newQuotaProvisioningRecord(t, event))

	require.NoError(t, err)
	p.AssertExpectations(t)
}

func TestQuotaProvisioningConsumer_HandleRecord_WithLoanInfo_PassesLoanInfoToRequest(t *testing.T) {
	t.Parallel()
	p := &mockQuotaProvisioner{}
	event := baseProvisioningEvent()
	event.LoanInfo = &events.LoanInfo{
		TransactionFee:     decimal.NewFromInt(5),
		MinRepayment:       decimal.NewFromInt(10),
		ClawbackPercentage: decimal.NewFromFloat(0.5),
	}

	p.On("ProvisionCounter", mock.Anything, mock.MatchedBy(func(req quota.ProvisionCounterRequest) bool {
		return req.LoanInfo != nil &&
			req.LoanInfo.TransactionFee.Equal(decimal.NewFromInt(5)) &&
			req.LoanInfo.MinRepayment.Equal(decimal.NewFromInt(10))
	})).Return(nil)

	c := newQuotaProvisioningConsumer(p)
	err := c.handleRecord(context.Background(), newQuotaProvisioningRecord(t, event))

	require.NoError(t, err)
	p.AssertExpectations(t)
}

func TestQuotaProvisioningConsumer_HandleRecord_WithExpiryDate_PassesExpiryToRequest(t *testing.T) {
	t.Parallel()
	p := &mockQuotaProvisioner{}
	event := baseProvisioningEvent()
	expiry := time.Now().Add(30 * 24 * time.Hour).UTC().Truncate(time.Second)
	event.ExpiryDate = &expiry

	p.On("ProvisionCounter", mock.Anything, mock.MatchedBy(func(req quota.ProvisionCounterRequest) bool {
		return req.ExpiryDate != nil
	})).Return(nil)

	c := newQuotaProvisioningConsumer(p)
	err := c.handleRecord(context.Background(), newQuotaProvisioningRecord(t, event))

	require.NoError(t, err)
	p.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// handleRecord — reason code mapping
// ---------------------------------------------------------------------------

func TestQuotaProvisioningConsumer_HandleRecord_TransferInReasonCode_MapsToTransferIn(t *testing.T) {
	t.Parallel()
	p := &mockQuotaProvisioner{}
	event := baseProvisioningEvent()
	event.ReasonCode = events.ProvisioningReasonTransferIn

	p.On("ProvisionCounter", mock.Anything, mock.MatchedBy(func(req quota.ProvisionCounterRequest) bool {
		return req.ReasonCode == quota.ReasonTransferIn
	})).Return(nil)

	c := newQuotaProvisioningConsumer(p)
	err := c.handleRecord(context.Background(), newQuotaProvisioningRecord(t, event))

	require.NoError(t, err)
	p.AssertExpectations(t)
}

func TestQuotaProvisioningConsumer_HandleRecord_UnrecognisedReasonCode_SubstitutesQuotaProvisioned(t *testing.T) {
	t.Parallel()
	p := &mockQuotaProvisioner{}
	event := baseProvisioningEvent()
	event.ReasonCode = "SOME_UNKNOWN_CODE"

	// The consumer substitutes QUOTA_PROVISIONED for unknown reason codes.
	p.On("ProvisionCounter", mock.Anything, mock.MatchedBy(func(req quota.ProvisionCounterRequest) bool {
		return req.ReasonCode == quota.ReasonQuotaProvisioned
	})).Return(nil)

	c := newQuotaProvisioningConsumer(p)
	err := c.handleRecord(context.Background(), newQuotaProvisioningRecord(t, event))

	require.NoError(t, err)
	p.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// handleRecord — error paths
// ---------------------------------------------------------------------------

func TestQuotaProvisioningConsumer_HandleRecord_MalformedJSON_SkipsRecord(t *testing.T) {
	t.Parallel()
	p := &mockQuotaProvisioner{}
	c := newQuotaProvisioningConsumer(p)

	r := &kgo.Record{
		Topic: "public.quota-provisioning",
		Value: []byte(`{not valid json`),
	}
	err := c.handleRecord(context.Background(), r)

	// Malformed JSON is skipped (nil returned) — offset is committed to avoid blocking.
	assert.NoError(t, err, "malformed JSON must be skipped without returning an error")
	p.AssertNotCalled(t, "ProvisionCounter")
}

func TestQuotaProvisioningConsumer_HandleRecord_ProvisionerError_ReturnsError(t *testing.T) {
	t.Parallel()
	p := &mockQuotaProvisioner{}
	event := baseProvisioningEvent()
	provErr := errors.New("quota save failed")

	p.On("ProvisionCounter", mock.Anything, mock.Anything).Return(provErr)

	c := newQuotaProvisioningConsumer(p)
	err := c.handleRecord(context.Background(), newQuotaProvisioningRecord(t, event))

	// Provisioner error is returned — the run loop will NOT commit the offset.
	assert.Equal(t, provErr, err)
	p.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Start / Stop lifecycle
// ---------------------------------------------------------------------------

func TestQuotaProvisioningConsumer_Start_DisabledKafka_ReturnsImmediately(t *testing.T) {
	t.Parallel()
	cfg := &events.KafkaConfig{Enabled: false}
	p := &mockQuotaProvisioner{}
	c := NewQuotaProvisioningConsumer(cfg, p, "public.quota-provisioning")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Start(ctx) // must not block
}

func TestQuotaProvisioningConsumer_Stop_NilClient_NoPanic(t *testing.T) {
	t.Parallel()
	p := &mockQuotaProvisioner{}
	c := newQuotaProvisioningConsumer(p)

	assert.NotPanics(t, func() { c.Stop() })
}

// ---------------------------------------------------------------------------
// toQuotaReasonCode
// ---------------------------------------------------------------------------

func TestToQuotaReasonCode_KnownCodes_ReturnsTrueAndMappedCode(t *testing.T) {
	tests := []struct {
		name     string
		input    events.ProvisioningReasonCode
		expected quota.ReasonCode
	}{
		{
			name:     "QUOTA_PROVISIONED maps to ReasonQuotaProvisioned",
			input:    events.ProvisioningReasonQuotaProvisioned,
			expected: quota.ReasonQuotaProvisioned,
		},
		{
			name:     "TRANSFER_IN maps to ReasonTransferIn",
			input:    events.ProvisioningReasonTransferIn,
			expected: quota.ReasonTransferIn,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			code, ok := toQuotaReasonCode(tc.input)
			assert.True(t, ok)
			assert.Equal(t, tc.expected, code)
		})
	}
}

func TestToQuotaReasonCode_UnknownCode_ReturnsFalseAndQuotaProvisioned(t *testing.T) {
	t.Parallel()
	code, ok := toQuotaReasonCode("SOME_UNKNOWN_REASON")
	assert.False(t, ok)
	assert.Equal(t, quota.ReasonQuotaProvisioned, code)
}
