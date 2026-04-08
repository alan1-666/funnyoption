package model

import "sort"

type OrderBook struct {
	Key      string
	Bids     []*DepthLevel
	Asks     []*DepthLevel
	OrderMap map[string]*Order
}

func NewOrderBook(key string) *OrderBook {
	return &OrderBook{
		Key:      key,
		OrderMap: make(map[string]*Order),
	}
}

func (b *OrderBook) HasOrder(orderID string) bool {
	_, ok := b.OrderMap[orderID]
	return ok
}

func (b *OrderBook) AddOrder(order *Order) {
	book := b.bookFor(order)
	idx := b.findOrInsertLevel(order.Price, order.IsBuy())
	level := (*book)[idx]
	level.Append(order)
	b.OrderMap[order.OrderID] = order
}

func (b *OrderBook) RemoveOrder(order *Order) {
	book := b.bookFor(order)
	for i, level := range *book {
		if level.Price != order.Price {
			continue
		}
		remaining := level.Orders[:0]
		for _, candidate := range level.Orders {
			if candidate.OrderID == order.OrderID {
				continue
			}
			remaining = append(remaining, candidate)
		}
		level.Orders = remaining
		if level.IsEmpty() {
			*book = append((*book)[:i], (*book)[i+1:]...)
		}
		break
	}
	delete(b.OrderMap, order.OrderID)
}

func (b *OrderBook) RemoveFromMap(orderID string) {
	delete(b.OrderMap, orderID)
}

func (b *OrderBook) BestBidPrice() (int64, bool) {
	if len(b.Bids) == 0 {
		return 0, false
	}
	return b.Bids[0].Price, true
}

func (b *OrderBook) BestAskPrice() (int64, bool) {
	if len(b.Asks) == 0 {
		return 0, false
	}
	return b.Asks[0].Price, true
}

func (b *OrderBook) IsCross(order *Order) bool {
	if order.IsBuy() {
		bestAsk, ok := b.BestAskPrice()
		return ok && order.Price >= bestAsk
	}
	bestBid, ok := b.BestBidPrice()
	return ok && order.Price <= bestBid
}

func (b *OrderBook) IsCrossWithPrice(order *Order, price int64) bool {
	if order.IsBuy() {
		return order.Price >= price
	}
	return order.Price <= price
}

func (b *OrderBook) OppositeLevels(order *Order) []*DepthLevel {
	if order.IsBuy() {
		return b.Asks
	}
	return b.Bids
}

func (b *OrderBook) SetOppositeLevels(order *Order, levels []*DepthLevel) {
	if order.IsBuy() {
		b.Asks = levels
		return
	}
	b.Bids = levels
}

func (b *OrderBook) bookFor(order *Order) *[]*DepthLevel {
	if order.IsBuy() {
		return &b.Bids
	}
	return &b.Asks
}

func (b *OrderBook) findOrInsertLevel(price int64, isBuy bool) int {
	book := &b.Asks
	if isBuy {
		book = &b.Bids
	}
	idx := b.findLevelIndex(price, isBuy)
	if idx >= 0 {
		return idx
	}

	insertAt := sort.Search(len(*book), func(i int) bool {
		if isBuy {
			return (*book)[i].Price <= price
		}
		return (*book)[i].Price >= price
	})

	level := &DepthLevel{Price: price}
	*book = append(*book, nil)
	copy((*book)[insertAt+1:], (*book)[insertAt:])
	(*book)[insertAt] = level
	return insertAt
}

func (b *OrderBook) findLevelIndex(price int64, isBuy bool) int {
	book := b.Asks
	if isBuy {
		book = b.Bids
	}
	for i, level := range book {
		if level.Price == price {
			return i
		}
	}
	return -1
}

func (b *OrderBook) OrderCount() int {
	return len(b.OrderMap)
}

func (b *OrderBook) Snapshot(limit int) BookSnapshot {
	if limit <= 0 {
		limit = 5
	}
	snapshot := BookSnapshot{
		Key:  b.Key,
		Bids: aggregateLevels(b.Bids, limit),
		Asks: aggregateLevels(b.Asks, limit),
	}
	if best, ok := b.BestBidPrice(); ok {
		snapshot.BestBid = best
	}
	if best, ok := b.BestAskPrice(); ok {
		snapshot.BestAsk = best
	}
	if marketID, outcome, ok := parseBookKey(b.Key); ok {
		snapshot.MarketID = marketID
		snapshot.Outcome = outcome
	}
	return snapshot
}
