package service

import (
	"context"
	"fmt"
	"sync"

	sharedkafka "funnyoption/internal/shared/kafka"
)

type PositionStore interface {
	ApplyDelta(ctx context.Context, marketID, userID int64, outcome, positionAsset string, delta int64) error
	ResolveMarket(ctx context.Context, input ResolveMarketInput) (bool, error)
	CancelActiveOrders(ctx context.Context, marketID int64, reason string) ([]cancelledOrder, error)
	WinningPositions(ctx context.Context, marketID int64, outcome string) ([]winningPosition, error)
	MarkSettled(ctx context.Context, event sharedkafka.SettlementCompletedEvent) error
	RollupFrozen(ctx context.Context) (bool, error)
}

type ResolveMarketInput struct {
	MarketID         int64
	ResolvedOutcome  string
	OccurredAtMillis int64
}

type positionKey struct {
	MarketID int64
	UserID   int64
	Outcome  string
}

type positionStore struct {
	mu        sync.RWMutex
	positions map[positionKey]int64
	resolved  map[int64]string
	settled   map[positionKey]struct{}
}

func newPositionStore() *positionStore {
	return &positionStore{
		positions: make(map[positionKey]int64),
		resolved:  make(map[int64]string),
		settled:   make(map[positionKey]struct{}),
	}
}

func (s *positionStore) ApplyDelta(_ context.Context, marketID, userID int64, outcome, _ string, delta int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := positionKey{MarketID: marketID, UserID: userID, Outcome: outcome}
	s.positions[key] += delta
	return nil
}

func (s *positionStore) ResolveMarket(_ context.Context, input ResolveMarketInput) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.resolved[input.MarketID]; ok {
		if existing == input.ResolvedOutcome {
			return false, nil
		}
		return false, fmt.Errorf("market %d already resolved with outcome %s", input.MarketID, existing)
	}
	s.resolved[input.MarketID] = input.ResolvedOutcome
	return true, nil
}

type winningPosition struct {
	MarketID int64
	UserID   int64
	Outcome  string
	Quantity int64
}

type cancelledOrder struct {
	OrderID           string
	CommandID         string
	ClientOrderID     string
	UserID            int64
	MarketID          int64
	Outcome           string
	Side              string
	OrderType         string
	TimeInForce       string
	CollateralAsset   string
	FreezeID          string
	FreezeAsset       string
	FreezeAmount      int64
	Price             int64
	Quantity          int64
	FilledQuantity    int64
	RemainingQuantity int64
	Status            string
	CancelReason      string
	UpdatedAtMillis   int64
}

func (s *positionStore) WinningPositions(_ context.Context, marketID int64, outcome string) ([]winningPosition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]winningPosition, 0)
	for key, qty := range s.positions {
		if key.MarketID != marketID || key.Outcome != outcome || qty <= 0 {
			continue
		}
		if _, exists := s.settled[key]; exists {
			continue
		}
		result = append(result, winningPosition{
			MarketID: marketID,
			UserID:   key.UserID,
			Outcome:  outcome,
			Quantity: qty,
		})
	}
	return result, nil
}

func (s *positionStore) CancelActiveOrders(_ context.Context, _ int64, _ string) ([]cancelledOrder, error) {
	return nil, nil
}

func (s *positionStore) MarkSettled(_ context.Context, event sharedkafka.SettlementCompletedEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settled[positionKey{MarketID: event.MarketID, UserID: event.UserID, Outcome: event.WinningOutcome}] = struct{}{}
	return nil
}

func (s *positionStore) RollupFrozen(_ context.Context) (bool, error) {
	return false, nil
}
