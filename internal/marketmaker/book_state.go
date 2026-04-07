package marketmaker

import "sync"

// MarketState tracks the bot's view of a single market.
type MarketState struct {
	MarketID     int64
	Title        string
	Seeded       bool
	YesInventory int64
	NoInventory  int64
	MidPrice     int64
	ActiveOrders map[string]activeOrder // orderID → order
}

type activeOrder struct {
	OrderID  string
	Outcome  string
	Price    int64
	Quantity int64
}

// StateBook manages all per-market state the bot tracks, goroutine-safe.
type StateBook struct {
	mu      sync.Mutex
	markets map[int64]*MarketState
}

func NewStateBook() *StateBook {
	return &StateBook{markets: make(map[int64]*MarketState)}
}

func (b *StateBook) GetOrCreate(marketID int64) *MarketState {
	b.mu.Lock()
	defer b.mu.Unlock()
	if state, ok := b.markets[marketID]; ok {
		return state
	}
	state := &MarketState{
		MarketID:     marketID,
		ActiveOrders: make(map[string]activeOrder),
	}
	b.markets[marketID] = state
	return state
}

func (b *StateBook) Get(marketID int64) (*MarketState, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	state, ok := b.markets[marketID]
	return state, ok
}

func (b *StateBook) AllMarketIDs() []int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	ids := make([]int64, 0, len(b.markets))
	for id := range b.markets {
		ids = append(ids, id)
	}
	return ids
}

func (b *StateBook) Remove(marketID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.markets, marketID)
}

// ApplyTradeFill adjusts inventory when one of the bot's orders is filled.
func (s *MarketState) ApplyTradeFill(outcome string, quantity int64) {
	switch outcome {
	case "YES":
		s.YesInventory -= quantity
		if s.YesInventory < 0 {
			s.YesInventory = 0
		}
	case "NO":
		s.NoInventory -= quantity
		if s.NoInventory < 0 {
			s.NoInventory = 0
		}
	}
}

// AddInventory increases holdings after a complete-set mint.
func (s *MarketState) AddInventory(quantity int64) {
	s.YesInventory += quantity
	s.NoInventory += quantity
}

func (s *MarketState) TrackOrder(orderID, outcome string, price, quantity int64) {
	if s.ActiveOrders == nil {
		s.ActiveOrders = make(map[string]activeOrder)
	}
	s.ActiveOrders[orderID] = activeOrder{
		OrderID:  orderID,
		Outcome:  outcome,
		Price:    price,
		Quantity: quantity,
	}
}

func (s *MarketState) RemoveOrder(orderID string) {
	delete(s.ActiveOrders, orderID)
}
