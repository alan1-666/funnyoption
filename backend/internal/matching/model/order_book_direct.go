package model

const maxPrice = 100

// OrderBookDirect uses fixed-size bucket arrays for O(1) price lookup,
// intrusive doubly-linked lists for FIFO order management within each
// price level, and a slab allocator (OrderPool) for zero-GC order storage.
// Prices are 1–99 (prediction market cents), stored at index [price].
type OrderBookDirect struct {
	Key        string
	askBuckets [maxPrice]Bucket
	bidBuckets [maxPrice]Bucket
	orderIndex map[string]*DirectOrder
	bestAsk    int64 // 0 means no asks
	bestBid    int64 // 0 means no bids
	pool       *OrderPool
}

func NewOrderBookDirect(key string) *OrderBookDirect {
	ob := &OrderBookDirect{
		Key:        key,
		orderIndex: make(map[string]*DirectOrder, 256),
		pool:       NewOrderPool(defaultPoolSize),
	}
	for i := int64(0); i < maxPrice; i++ {
		ob.askBuckets[i].Price = i
		ob.bidBuckets[i].Price = i
	}
	return ob
}

func (ob *OrderBookDirect) HasOrder(orderID string) bool {
	_, ok := ob.orderIndex[orderID]
	return ok
}

func (ob *OrderBookDirect) AddOrder(order *Order) {
	do := ob.pool.Get()
	do.FromOrder(order)
	price := order.Price
	if order.IsBuy() {
		ob.bidBuckets[price].Append(do)
		if ob.bestBid == 0 || price > ob.bestBid {
			ob.bestBid = price
		}
	} else {
		ob.askBuckets[price].Append(do)
		if ob.bestAsk == 0 || price < ob.bestAsk {
			ob.bestAsk = price
		}
	}
	ob.orderIndex[order.OrderID] = do
}

func (ob *OrderBookDirect) RemoveOrder(order *Order) {
	do, ok := ob.orderIndex[order.OrderID]
	if !ok {
		return
	}
	price := do.Price
	isBuy := do.IsBuy()

	if isBuy {
		ob.bidBuckets[price].Remove(do)
		if ob.bidBuckets[price].IsEmpty() && price == ob.bestBid {
			ob.bestBid = ob.scanBestBid()
		}
	} else {
		ob.askBuckets[price].Remove(do)
		if ob.askBuckets[price].IsEmpty() && price == ob.bestAsk {
			ob.bestAsk = ob.scanBestAsk()
		}
	}
	delete(ob.orderIndex, order.OrderID)
	ob.pool.Put(do)
}

func (ob *OrderBookDirect) RemoveDirectOrder(do *DirectOrder) {
	if do == nil {
		return
	}
	price := do.Price
	isBuy := do.IsBuy()

	if isBuy {
		ob.bidBuckets[price].Remove(do)
		if ob.bidBuckets[price].IsEmpty() && price == ob.bestBid {
			ob.bestBid = ob.scanBestBid()
		}
	} else {
		ob.askBuckets[price].Remove(do)
		if ob.askBuckets[price].IsEmpty() && price == ob.bestAsk {
			ob.bestAsk = ob.scanBestAsk()
		}
	}
	delete(ob.orderIndex, do.OrderID)
	ob.pool.Put(do)
}

func (ob *OrderBookDirect) RemoveFromMap(orderID string) {
	do, ok := ob.orderIndex[orderID]
	if !ok {
		return
	}
	price := do.Price
	isBuy := do.IsBuy()

	if isBuy {
		ob.bidBuckets[price].Remove(do)
		if ob.bidBuckets[price].IsEmpty() && price == ob.bestBid {
			ob.bestBid = ob.scanBestBid()
		}
	} else {
		ob.askBuckets[price].Remove(do)
		if ob.askBuckets[price].IsEmpty() && price == ob.bestAsk {
			ob.bestAsk = ob.scanBestAsk()
		}
	}
	delete(ob.orderIndex, orderID)
	ob.pool.Put(do)
}

func (ob *OrderBookDirect) BestBidPrice() (int64, bool) {
	if ob.bestBid == 0 {
		return 0, false
	}
	return ob.bestBid, true
}

func (ob *OrderBookDirect) BestAskPrice() (int64, bool) {
	if ob.bestAsk == 0 {
		return 0, false
	}
	return ob.bestAsk, true
}

func (ob *OrderBookDirect) IsCross(order *Order) bool {
	if order.IsMarket() {
		return true
	}
	if order.IsBuy() {
		if ob.bestAsk == 0 {
			return false
		}
		return order.Price >= ob.bestAsk
	}
	if ob.bestBid == 0 {
		return false
	}
	return order.Price <= ob.bestBid
}

func (ob *OrderBookDirect) IsCrossWithPrice(order *Order, price int64) bool {
	if order.IsMarket() {
		return true
	}
	if order.IsBuy() {
		return order.Price >= price
	}
	return order.Price <= price
}

func (ob *OrderBookDirect) GetDirectOrder(orderID string) (*DirectOrder, bool) {
	do, ok := ob.orderIndex[orderID]
	return do, ok
}

func (ob *OrderBookDirect) Snapshot(limit int) BookSnapshot {
	if limit <= 0 {
		limit = 5
	}
	snapshot := BookSnapshot{Key: ob.Key}

	count := 0
	for p := ob.bestBid; p >= 1 && count < limit; p-- {
		b := &ob.bidBuckets[p]
		if b.IsEmpty() {
			continue
		}
		snapshot.Bids = append(snapshot.Bids, BookLevel{Price: p, Quantity: b.Volume})
		count++
	}

	count = 0
	for p := ob.bestAsk; p < maxPrice && count < limit; p++ {
		b := &ob.askBuckets[p]
		if b.IsEmpty() {
			continue
		}
		snapshot.Asks = append(snapshot.Asks, BookLevel{Price: p, Quantity: b.Volume})
		count++
	}

	if ob.bestBid > 0 {
		snapshot.BestBid = ob.bestBid
	}
	if ob.bestAsk > 0 {
		snapshot.BestAsk = ob.bestAsk
	}
	if marketID, outcome, ok := parseBookKey(ob.Key); ok {
		snapshot.MarketID = marketID
		snapshot.Outcome = outcome
	}
	return snapshot
}

// MatchBuy walks asks from bestAsk upward until price limit.
// Returns the first non-empty bucket at or above startPrice, or nil.
func (ob *OrderBookDirect) FirstAskBucket() *Bucket {
	if ob.bestAsk == 0 {
		return nil
	}
	return &ob.askBuckets[ob.bestAsk]
}

func (ob *OrderBookDirect) NextAskBucket(currentPrice int64) *Bucket {
	for p := currentPrice + 1; p < maxPrice; p++ {
		if !ob.askBuckets[p].IsEmpty() {
			return &ob.askBuckets[p]
		}
	}
	return nil
}

func (ob *OrderBookDirect) FirstBidBucket() *Bucket {
	if ob.bestBid == 0 {
		return nil
	}
	return &ob.bidBuckets[ob.bestBid]
}

func (ob *OrderBookDirect) NextBidBucket(currentPrice int64) *Bucket {
	for p := currentPrice - 1; p >= 1; p-- {
		if !ob.bidBuckets[p].IsEmpty() {
			return &ob.bidBuckets[p]
		}
	}
	return nil
}

func (ob *OrderBookDirect) OrderCount() int {
	return len(ob.orderIndex)
}

func (ob *OrderBookDirect) scanBestAsk() int64 {
	for p := int64(1); p < maxPrice; p++ {
		if !ob.askBuckets[p].IsEmpty() {
			return p
		}
	}
	return 0
}

func (ob *OrderBookDirect) scanBestBid() int64 {
	for p := int64(maxPrice - 1); p >= 1; p-- {
		if !ob.bidBuckets[p].IsEmpty() {
			return p
		}
	}
	return 0
}
