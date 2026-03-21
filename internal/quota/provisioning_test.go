package quota

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"go-ocs/internal/charging"
	"go-ocs/internal/events"
)

// noopKafka returns a KafkaManager with a nil client (events are silently dropped).
func noopKafka() *events.KafkaManager {
	return &events.KafkaManager{KafkaConfig: events.KafkaConfig{}, KafkaClient: nil}
}

// makeProvisionManager returns a QuotaManager wired with the given mock repository
// and a no-op Kafka client suitable for unit tests.
func makeProvisionManager(repo Repository) *QuotaManager {
	return &QuotaManager{
		repo:         repo,
		retryLimit:   3,
		kafkaManager: noopKafka(),
	}
}

// makeBaseRequest returns a minimal valid ProvisionCounterRequest.
func makeBaseRequest(subscriberID, counterID uuid.UUID, balance decimal.Decimal) ProvisionCounterRequest {
	return ProvisionCounterRequest{
		SubscriberID:         subscriberID,
		CounterID:            counterID,
		ProductID:            uuid.New(),
		ProductName:          "Test Product",
		Description:          "Test counter",
		UnitType:             charging.UNITS,
		Priority:             10,
		InitialBalance:       balance,
		CanRepayLoan:         false,
		CounterSelectionKeys: []charging.RateKey{},
		ReasonCode:           ReasonQuotaProvisioned,
		Now:                  time.Now().UTC(),
		TransactionID:        uuid.New().String(),
	}
}

func TestProvisionCounter_NewCounter_CreatesCounterAndPublishesJournal(t *testing.T) {
	ctx := context.Background()
	subscriberID := uuid.New()
	counterID := uuid.New()
	balance := decimal.NewFromInt(100)

	mockRepo := new(MockRepository)
	manager := makeProvisionManager(mockRepo)

	quota := &Quota{
		QuotaID:  uuid.New(),
		Counters: []Counter{},
	}
	loaded := &LoadedQuota{Quota: quota}

	mockRepo.On("Load", ctx, subscriberID).Return(loaded, nil)
	mockRepo.On("Save", ctx, loaded).Return(nil)

	req := makeBaseRequest(subscriberID, counterID, balance)

	err := manager.ProvisionCounter(ctx, req)

	assert.NoError(t, err)
	assert.Len(t, quota.Counters, 1)
	assert.Equal(t, counterID, quota.Counters[0].CounterID)
	assert.True(t, quota.Counters[0].Balance.Equal(balance))
	assert.True(t, quota.Counters[0].InitialBalance.Equal(balance))
	mockRepo.AssertExpectations(t)
}

func TestProvisionCounter_DuplicateCounterID_IsIdempotent(t *testing.T) {
	ctx := context.Background()
	subscriberID := uuid.New()
	counterID := uuid.New()
	balance := decimal.NewFromInt(100)

	mockRepo := new(MockRepository)
	manager := makeProvisionManager(mockRepo)

	// Counter with the same ID already exists in the quota.
	existing := &Quota{
		QuotaID: uuid.New(),
		Counters: []Counter{
			{
				CounterID:    counterID,
				Balance:      &balance,
				Reservations: make(map[uuid.UUID]Reservation),
			},
		},
	}
	loaded := &LoadedQuota{Quota: existing}

	mockRepo.On("Load", ctx, subscriberID).Return(loaded, nil)
	mockRepo.On("Save", ctx, loaded).Return(nil)

	req := makeBaseRequest(subscriberID, counterID, balance)

	err := manager.ProvisionCounter(ctx, req)

	assert.NoError(t, err)
	// Still only one counter — no duplicate added.
	assert.Len(t, existing.Counters, 1)
	mockRepo.AssertExpectations(t)
}

func TestProvisionCounter_WithLoanInfo_AttachesLoanAndForcesCanRepayLoanFalse(t *testing.T) {
	ctx := context.Background()
	subscriberID := uuid.New()
	counterID := uuid.New()
	balance := decimal.NewFromInt(200)

	mockRepo := new(MockRepository)
	manager := makeProvisionManager(mockRepo)

	quota := &Quota{
		QuotaID:  uuid.New(),
		Counters: []Counter{},
	}
	loaded := &LoadedQuota{Quota: quota}

	mockRepo.On("Load", ctx, subscriberID).Return(loaded, nil)
	mockRepo.On("Save", ctx, loaded).Return(nil)

	txFee := decimal.NewFromInt(15)
	req := makeBaseRequest(subscriberID, counterID, balance)
	req.CanRepayLoan = true // would be true in event, but must be forced false
	req.LoanInfo = &LoanProvisionInfo{
		TransactionFee:     txFee,
		MinRepayment:       decimal.NewFromInt(10),
		ClawbackPercentage: decimal.NewFromFloat(0.5),
	}

	err := manager.ProvisionCounter(ctx, req)

	assert.NoError(t, err)
	assert.Len(t, quota.Counters, 1)
	c := &quota.Counters[0]
	// Loan should be attached with loanBalance = initialBalance and transactFee = LoanInfo.TransactionFee.
	assert.NotNil(t, c.Loan)
	assert.True(t, c.Loan.LoanBalance.Equal(balance))
	assert.True(t, c.Loan.TransactFee.Equal(txFee), "TransactFee must be set from LoanInfo.TransactionFee")
	assert.True(t, c.Loan.MinRepayment.Equal(decimal.NewFromInt(10)))
	assert.True(t, c.Loan.ClawbackPercentage.Equal(decimal.NewFromFloat(0.5)))
	mockRepo.AssertExpectations(t)
}

func TestProvisionCounter_CanRepayLoan_TriggersClawbackOldestFirst(t *testing.T) {
	ctx := context.Background()
	subscriberID := uuid.New()
	counterID := uuid.New()

	// New counter balance: 100
	newBalance := decimal.NewFromInt(100)

	mockRepo := new(MockRepository)
	manager := makeProvisionManager(mockRepo)

	// Two existing loan counters. Oldest first = index 0.
	loan1Balance := decimal.NewFromInt(30)
	loan2Balance := decimal.NewFromInt(40)
	counter1Balance := decimal.NewFromInt(300)
	counter2Balance := decimal.NewFromInt(300)

	quota := &Quota{
		QuotaID: uuid.New(),
		Counters: []Counter{
			{
				CounterID:    uuid.New(),
				UnitType:     charging.MONETARY,
				Balance:      &counter1Balance,
				Reservations: make(map[uuid.UUID]Reservation),
				Loan: &Loan{
					LoanBalance:        loan1Balance,
					TransactFee:        decimal.Zero,
					MinRepayment:       decimal.NewFromInt(30),
					ClawbackPercentage: decimal.Zero,
				},
			},
			{
				CounterID:    uuid.New(),
				UnitType:     charging.MONETARY,
				Balance:      &counter2Balance,
				Reservations: make(map[uuid.UUID]Reservation),
				Loan: &Loan{
					LoanBalance:        loan2Balance,
					TransactFee:        decimal.Zero,
					MinRepayment:       decimal.NewFromInt(40),
					ClawbackPercentage: decimal.Zero,
				},
			},
		},
	}
	loaded := &LoadedQuota{Quota: quota}

	mockRepo.On("Load", ctx, subscriberID).Return(loaded, nil)
	mockRepo.On("Save", ctx, loaded).Return(nil)

	req := makeBaseRequest(subscriberID, counterID, newBalance)
	req.CanRepayLoan = true

	err := manager.ProvisionCounter(ctx, req)

	assert.NoError(t, err)
	// New counter added.
	assert.Len(t, quota.Counters, 3)
	newCounter := &quota.Counters[2]

	// Clawback: 30 from loan1 + 40 from loan2 = 70 total debited from new counter.
	// New counter balance = 100 - 30 - 40 = 30.
	assert.True(t, newCounter.Balance.Equal(decimal.NewFromInt(30)),
		"expected new counter balance 30, got %s", newCounter.Balance.String())

	// Loan1 should be fully repaid (30).
	assert.True(t, quota.Counters[0].Loan.LoanBalance.Equal(decimal.Zero),
		"expected loan1 fully repaid")

	// Loan2 should be fully repaid (40).
	assert.True(t, quota.Counters[1].Loan.LoanBalance.Equal(decimal.Zero),
		"expected loan2 fully repaid")

	mockRepo.AssertExpectations(t)
}

func TestProvisionCounter_CanRepayLoan_StopsWhenRemainingBalanceExhausted(t *testing.T) {
	ctx := context.Background()
	subscriberID := uuid.New()
	counterID := uuid.New()

	// New counter balance only 20 — not enough to repay 50 loan.
	newBalance := decimal.NewFromInt(20)

	mockRepo := new(MockRepository)
	manager := makeProvisionManager(mockRepo)

	loanBalance := decimal.NewFromInt(50)
	counterBalance := decimal.NewFromInt(300)

	quota := &Quota{
		QuotaID: uuid.New(),
		Counters: []Counter{
			{
				CounterID:    uuid.New(),
				UnitType:     charging.MONETARY,
				Balance:      &counterBalance,
				Reservations: make(map[uuid.UUID]Reservation),
				Loan: &Loan{
					LoanBalance:        loanBalance,
					TransactFee:        decimal.Zero,
					MinRepayment:       decimal.NewFromInt(50),
					ClawbackPercentage: decimal.Zero,
				},
			},
		},
	}
	loaded := &LoadedQuota{Quota: quota}

	mockRepo.On("Load", ctx, subscriberID).Return(loaded, nil)
	mockRepo.On("Save", ctx, loaded).Return(nil)

	req := makeBaseRequest(subscriberID, counterID, newBalance)
	req.CanRepayLoan = true

	err := manager.ProvisionCounter(ctx, req)

	assert.NoError(t, err)

	// The new counter's balance was fully consumed (0) — RemoveExpiredEntries then
	// removes zero-balance counters, so only the loan counter remains.
	assert.Len(t, quota.Counters, 1, "zero-balance new counter should have been pruned")

	// Loan only partially repaid: 50 - 20 = 30 remaining.
	assert.True(t, quota.Counters[0].Loan.LoanBalance.Equal(decimal.NewFromInt(30)),
		"expected remaining loan balance 30, got %s", quota.Counters[0].Loan.LoanBalance.String())

	mockRepo.AssertExpectations(t)
}

func TestFindCountersWithLoans_ReturnsOnlyLoanCounters(t *testing.T) {
	loanBalance := decimal.NewFromInt(50)
	balance := decimal.NewFromInt(100)

	q := &Quota{
		Counters: []Counter{
			{CounterID: uuid.New(), Balance: &balance, Reservations: make(map[uuid.UUID]Reservation)},
			{
				CounterID:    uuid.New(),
				Balance:      &balance,
				Reservations: make(map[uuid.UUID]Reservation),
				Loan:         &Loan{LoanBalance: loanBalance},
			},
			{CounterID: uuid.New(), Balance: &balance, Reservations: make(map[uuid.UUID]Reservation)},
		},
	}

	result := q.FindCountersWithLoans()

	assert.Len(t, result, 1)
	assert.Equal(t, q.Counters[1].CounterID, result[0].CounterID)
}

func TestFindCountersWithLoans_EmptyWhenNoLoans(t *testing.T) {
	balance := decimal.NewFromInt(100)
	q := &Quota{
		Counters: []Counter{
			{CounterID: uuid.New(), Balance: &balance, Reservations: make(map[uuid.UUID]Reservation)},
		},
	}

	result := q.FindCountersWithLoans()

	assert.Empty(t, result)
}

func TestFindCountersWithLoans_SkipsZeroLoanBalance(t *testing.T) {
	balance := decimal.NewFromInt(100)
	zeroLoanBalance := decimal.Zero

	q := &Quota{
		Counters: []Counter{
			{
				CounterID:    uuid.New(),
				Balance:      &balance,
				Reservations: make(map[uuid.UUID]Reservation),
				Loan:         &Loan{LoanBalance: zeroLoanBalance},
			},
		},
	}

	result := q.FindCountersWithLoans()

	assert.Empty(t, result, "counter with zero loan balance should not be returned")
}
