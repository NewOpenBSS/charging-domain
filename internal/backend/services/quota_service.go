package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"go-ocs/internal/backend/graphql/model"
	"go-ocs/internal/charging"
	"go-ocs/internal/quota"
)

// QuotaService handles all business logic for the quota GraphQL resource.
// It wraps the domain QuotaManagerInterface and translates between GraphQL model
// types and the quota domain types.
type QuotaService struct {
	manager quota.QuotaManagerInterface
}

// NewQuotaService creates a new QuotaService backed by the supplied QuotaManagerInterface.
func NewQuotaService(manager quota.QuotaManagerInterface) *QuotaService {
	return &QuotaService{manager: manager}
}

// GetBalance returns the aggregated balance for a specific subscriber and unit type.
// Both subscriberId and unitType are required. Returns nil if no matching counters exist.
func (s *QuotaService) GetBalance(
	ctx context.Context,
	now time.Time,
	req model.QuotaBalanceRequestInput,
) (*model.QuotaBalanceResponse, error) {
	if req.UnitType == nil {
		return nil, fmt.Errorf("unitType is required for quotaBalance")
	}

	subID, err := uuid.Parse(req.SubscriberID)
	if err != nil {
		return nil, fmt.Errorf("invalid subscriberId: %w", err)
	}

	q := balanceQueryFromRequest(req)
	counters, err := s.manager.GetBalance(ctx, now, subID, q)
	if err != nil {
		return nil, err
	}

	unitType := charging.UnitType(req.UnitType.String())
	return aggregateByUnitType(counters, unitType), nil
}

// GetBalances returns aggregated balances for all unit types held by the subscriber.
// unitType is optional — when set it restricts results to that unit type.
func (s *QuotaService) GetBalances(
	ctx context.Context,
	now time.Time,
	req model.QuotaBalanceRequestInput,
) ([]*model.QuotaBalanceResponse, error) {
	subID, err := uuid.Parse(req.SubscriberID)
	if err != nil {
		return nil, fmt.Errorf("invalid subscriberId: %w", err)
	}

	q := balanceQueryFromRequest(req)
	counters, err := s.manager.GetBalance(ctx, now, subID, q)
	if err != nil {
		return nil, err
	}

	// Group and aggregate by UnitType.
	totals := make(map[charging.UnitType]*decimal.Decimal)
	avail := make(map[charging.UnitType]*decimal.Decimal)
	order := make([]charging.UnitType, 0)

	for _, c := range counters {
		ut := c.UnitType
		if _, seen := totals[ut]; !seen {
			z := decimal.Zero
			totals[ut] = &z
			a := decimal.Zero
			avail[ut] = &a
			order = append(order, ut)
		}
		t := totals[ut].Add(c.TotalBalance)
		totals[ut] = &t
		a := avail[ut].Add(c.AvailableBalance)
		avail[ut] = &a
	}

	result := make([]*model.QuotaBalanceResponse, 0, len(order))
	for _, ut := range order {
		result = append(result, &model.QuotaBalanceResponse{
			UnitType:         model.UnitType(ut),
			TotalValue:       totals[ut].String(),
			AvailableBalance: avail[ut].String(),
		})
	}
	return result, nil
}

// CancelReservations releases all quota reservations for the given reservation ID.
func (s *QuotaService) CancelReservations(
	ctx context.Context,
	reservationID string,
	subscriberID string,
) (*model.QuotaOperationResponse, error) {
	resID, err := uuid.Parse(reservationID)
	if err != nil {
		return nil, fmt.Errorf("invalid reservationId: %w", err)
	}
	subID, err := uuid.Parse(subscriberID)
	if err != nil {
		return nil, fmt.Errorf("invalid subscriberId: %w", err)
	}

	if err := s.manager.Release(ctx, subID, resID); err != nil {
		return nil, err
	}
	return &model.QuotaOperationResponse{Success: true}, nil
}

// ReserveQuota reserves units from a subscriber's quota.
func (s *QuotaService) ReserveQuota(
	ctx context.Context,
	now time.Time,
	reservationID string,
	subscriberID string,
	reasonCode model.ReasonCode,
	rateKey model.QuotaRateKeyInput,
	unitType model.UnitType,
	requestedUnits int,
	unitPrice string,
	validitySeconds int,
	allowOOBCharging bool,
) (*model.QuotaReserveResponse, error) {
	if requestedUnits <= 0 {
		return nil, fmt.Errorf("requestedUnits must be greater than zero")
	}

	resID, err := uuid.Parse(reservationID)
	if err != nil {
		return nil, fmt.Errorf("invalid reservationId: %w", err)
	}
	subID, err := uuid.Parse(subscriberID)
	if err != nil {
		return nil, fmt.Errorf("invalid subscriberId: %w", err)
	}
	price, err := decimal.NewFromString(unitPrice)
	if err != nil {
		return nil, fmt.Errorf("invalid unitPrice %q: %w", unitPrice, err)
	}

	domainRateKey, err := buildRateKey(rateKey)
	if err != nil {
		return nil, err
	}

	granted, err := s.manager.ReserveQuota(
		ctx,
		now,
		resID,
		subID,
		quota.ReasonCode(reasonCode),
		domainRateKey,
		charging.UnitType(unitType),
		int64(requestedUnits),
		price,
		decimal.NewFromInt(1), // multiplier hardcoded to 1 — matches Java source
		time.Duration(validitySeconds)*time.Second,
		allowOOBCharging,
	)
	if err != nil {
		return nil, err
	}
	return &model.QuotaReserveResponse{GrantedUnits: int(granted)}, nil
}

// DebitQuota debits reserved units from a subscriber's quota.
func (s *QuotaService) DebitQuota(
	ctx context.Context,
	now time.Time,
	subscriberID string,
	reservationID string,
	usedUnits int,
	unitType model.UnitType,
	reclaimUnusedUnits bool,
) (*model.QuotaDebitResponse, error) {
	if usedUnits <= 0 {
		return nil, fmt.Errorf("usedUnits must be greater than zero")
	}

	subID, err := uuid.Parse(subscriberID)
	if err != nil {
		return nil, fmt.Errorf("invalid subscriberId: %w", err)
	}
	resID, err := uuid.Parse(reservationID)
	if err != nil {
		return nil, fmt.Errorf("invalid reservationId: %w", err)
	}

	resp, err := s.manager.Debit(
		ctx,
		now,
		subID,
		resID.String(),
		resID,
		int64(usedUnits),
		charging.UnitType(unitType),
		reclaimUnusedUnits,
	)
	if err != nil {
		return nil, err
	}
	return &model.QuotaDebitResponse{
		UnitsDebited:     int(resp.UnitsDebited),
		UnitsValue:       resp.UnitsValue.String(),
		ValueUnits:       int(resp.ValueUnits),
		UnaccountedUnits: int(resp.UnaccountedUnits),
	}, nil
}

// balanceQueryFromRequest converts a GraphQL request into a domain BalanceQuery.
func balanceQueryFromRequest(req model.QuotaBalanceRequestInput) quota.BalanceQuery {
	q := quota.BalanceQuery{}

	if req.UnitType != nil {
		ut := charging.UnitType(req.UnitType.String())
		q.UnitType = &ut
	}

	switch req.BalanceType {
	case model.BalanceTypeTransferableBalance:
		t := true
		q.Transferable = &t
	case model.BalanceTypeConvertableBalance:
		t := true
		q.Convertible = &t
	// BalanceTypeAvailableBalance: no filter — all counters included.
	}

	return q
}

// aggregateByUnitType sums TotalBalance and AvailableBalance across all counters
// of the given unit type. Returns nil if there are no matching counters.
func aggregateByUnitType(counters []*quota.CounterBalance, unitType charging.UnitType) *model.QuotaBalanceResponse {
	total := decimal.Zero
	avail := decimal.Zero
	found := false

	for _, c := range counters {
		if c.UnitType != unitType {
			continue
		}
		total = total.Add(c.TotalBalance)
		avail = avail.Add(c.AvailableBalance)
		found = true
	}

	if !found {
		return nil
	}
	return &model.QuotaBalanceResponse{
		UnitType:         model.UnitType(unitType),
		TotalValue:       total.String(),
		AvailableBalance: avail.String(),
	}
}

// buildRateKey converts a GraphQL QuotaRateKeyInput into a charging.RateKey.
func buildRateKey(input model.QuotaRateKeyInput) (charging.RateKey, error) {
	cd, err := charging.ParseCallDirection(input.ServiceDirection)
	if err != nil {
		return charging.RateKey{}, fmt.Errorf("invalid serviceDirection %q: %w", input.ServiceDirection, err)
	}
	rk := charging.RateKey{
		ServiceType:      input.ServiceType,
		SourceType:       input.SourceType,
		ServiceDirection: cd,
		ServiceCategory:  input.ServiceCategory,
	}
	if input.ServiceWindow != nil {
		rk.ServiceWindow = *input.ServiceWindow
	}
	return rk, nil
}
