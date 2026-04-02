package service

import (
	"context"
	"sync"
)

type PositionStore interface {
	ApplyDelta(ctx context.Context, marketID, userID int64, outcome, positionAsset string, delta int64) error
	ResolveMarket(ctx context.Context, marketID int64, outcome string) error
	CancelActiveOrders(ctx context.Context, marketID int64, reason string) ([]cancelledOrder, error)
	WinningPositions(ctx context.Context, marketID int64, outcome string) ([]winningPosition, error)
	MarkSettled(ctx context.Context, eventID string, marketID, userID int64, outcome string, quantity int64, payoutAsset string, payoutAmount int64) error
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

func (s *positionStore) ResolveMarket(_ context.Context, marketID int64, outcome string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resolved[marketID] = outcome
	return nil
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

func (s *positionStore) MarkSettled(_ context.Context, _ string, marketID, userID int64, outcome string, _ int64, _ string, _ int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settled[positionKey{MarketID: marketID, UserID: userID, Outcome: outcome}] = struct{}{}
	return nil
}
