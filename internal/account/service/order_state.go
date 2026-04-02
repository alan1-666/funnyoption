package service

import (
	"context"
	"sync"
)

type OrderState struct {
	OrderID           string
	UserID            int64
	Side              string
	Price             int64
	FreezeID          string
	FreezeAsset       string
	FreezeApplied     bool
	Status            string
	RemainingQuantity int64
}

type OrderRegistry struct {
	mu     sync.RWMutex
	orders map[string]*OrderState
	store  interface {
		LoadOrderState(ctx context.Context, orderID string) (*OrderState, error)
		MirrorOrderState(ctx context.Context, state OrderState) error
	}
}

func NewOrderRegistry() *OrderRegistry {
	return &OrderRegistry{
		orders: make(map[string]*OrderState),
	}
}

func NewPersistentOrderRegistry(store interface {
	LoadOrderState(ctx context.Context, orderID string) (*OrderState, error)
	MirrorOrderState(ctx context.Context, state OrderState) error
}) *OrderRegistry {
	return &OrderRegistry{
		orders: make(map[string]*OrderState),
		store:  store,
	}
}

func (r *OrderRegistry) Get(orderID string) (*OrderState, bool) {
	r.mu.RLock()
	state, ok := r.orders[orderID]
	if ok {
		r.mu.RUnlock()
		cloned := *state
		return &cloned, true
	}
	r.mu.RUnlock()
	if r.store != nil {
		state, _ := r.store.LoadOrderState(context.Background(), orderID)
		if state != nil {
			r.mu.Lock()
			r.orders[orderID] = state
			r.mu.Unlock()
			cloned := *state
			return &cloned, true
		}
	}
	return nil, false
}

func (r *OrderRegistry) Upsert(state OrderState) *OrderState {
	var persisted *OrderState
	if r.store != nil && state.OrderID != "" {
		persisted, _ = r.store.LoadOrderState(context.Background(), state.OrderID)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.orders[state.OrderID]
	if !ok {
		if persisted != nil {
			cloned := *persisted
			existing = &cloned
		} else {
			existing = &OrderState{OrderID: state.OrderID}
		}
		r.orders[state.OrderID] = existing
	}
	if state.UserID != 0 {
		existing.UserID = state.UserID
	}
	if state.Side != "" {
		existing.Side = state.Side
	}
	if state.Price != 0 {
		existing.Price = state.Price
	}
	if state.FreezeID != "" {
		existing.FreezeID = state.FreezeID
	}
	if state.FreezeAsset != "" {
		existing.FreezeAsset = state.FreezeAsset
	}
	if state.Status != "" {
		existing.Status = state.Status
	}
	existing.RemainingQuantity = state.RemainingQuantity
	if state.FreezeApplied {
		existing.FreezeApplied = true
	}

	cloned := *existing
	if r.store != nil {
		_ = r.store.MirrorOrderState(context.Background(), cloned)
	}
	return &cloned
}
