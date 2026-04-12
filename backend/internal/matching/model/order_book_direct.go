package model

import "math/bits"

// maxPrice is the exclusive upper bound for prices.
// Prediction market prices are 1–9999 (0.0001–0.9999), stored at index [price].
const maxPrice = 10000

// priceBitmapWords is the number of uint64 words needed to cover maxPrice bits.
const priceBitmapWords = (maxPrice + 63) / 64 // 157

// OrderBookDirect uses fixed-size bucket arrays for O(1) price lookup,
// intrusive doubly-linked lists for FIFO order management within each
// price level, and a slab allocator (OrderPool) for zero-GC order storage.
// Prices are 1–9999 (prediction market 4-decimal precision), stored at index [price].
type OrderBookDirect struct {
	Key        string
	askBuckets [maxPrice]Bucket
	bidBuckets [maxPrice]Bucket
	askBitmap  [priceBitmapWords]uint64
	bidBitmap  [priceBitmapWords]uint64
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
		ob.bidBitmap[price/64] |= 1 << (uint(price) % 64)
		if ob.bestBid == 0 || price > ob.bestBid {
			ob.bestBid = price
		}
	} else {
		ob.askBuckets[price].Append(do)
		ob.askBitmap[price/64] |= 1 << (uint(price) % 64)
		if ob.bestAsk == 0 || price < ob.bestAsk {
			ob.bestAsk = price
		}
	}
	ob.orderIndex[order.OrderID] = do
}

func (ob *OrderBookDirect) removeFromSide(do *DirectOrder) {
	price := do.Price
	if do.IsBuy() {
		ob.bidBuckets[price].Remove(do)
		if ob.bidBuckets[price].IsEmpty() {
			ob.bidBitmap[price/64] &^= 1 << (uint(price) % 64)
			if price == ob.bestBid {
				ob.bestBid = ob.scanBestBid()
			}
		}
	} else {
		ob.askBuckets[price].Remove(do)
		if ob.askBuckets[price].IsEmpty() {
			ob.askBitmap[price/64] &^= 1 << (uint(price) % 64)
			if price == ob.bestAsk {
				ob.bestAsk = ob.scanBestAsk()
			}
		}
	}
}

func (ob *OrderBookDirect) RemoveOrder(order *Order) {
	do, ok := ob.orderIndex[order.OrderID]
	if !ok {
		return
	}
	ob.removeFromSide(do)
	delete(ob.orderIndex, order.OrderID)
	ob.pool.Put(do)
}

func (ob *OrderBookDirect) RemoveDirectOrder(do *DirectOrder) {
	if do == nil {
		return
	}
	ob.removeFromSide(do)
	delete(ob.orderIndex, do.OrderID)
	ob.pool.Put(do)
}

func (ob *OrderBookDirect) RemoveFromMap(orderID string) {
	do, ok := ob.orderIndex[orderID]
	if !ok {
		return
	}
	ob.removeFromSide(do)
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

	b := ob.FirstBidBucket()
	for b != nil && len(snapshot.Bids) < limit {
		snapshot.Bids = append(snapshot.Bids, BookLevel{Price: b.Price, Quantity: b.Volume})
		b = ob.NextBidBucket(b.Price)
	}

	a := ob.FirstAskBucket()
	for a != nil && len(snapshot.Asks) < limit {
		snapshot.Asks = append(snapshot.Asks, BookLevel{Price: a.Price, Quantity: a.Volume})
		a = ob.NextAskBucket(a.Price)
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
	p := currentPrice + 1
	if p >= maxPrice {
		return nil
	}
	word := int(p / 64)
	bit := uint(p) % 64
	// Mask off bits below 'bit' in the current word.
	masked := ob.askBitmap[word] >> bit << bit
	for word < priceBitmapWords {
		if masked != 0 {
			price := int64(word*64) + int64(bits.TrailingZeros64(masked))
			if price >= maxPrice {
				return nil
			}
			return &ob.askBuckets[price]
		}
		word++
		if word < priceBitmapWords {
			masked = ob.askBitmap[word]
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
	p := currentPrice - 1
	if p < 1 {
		return nil
	}
	word := int(p / 64)
	bit := uint(p) % 64
	// Mask off bits above 'bit' in the current word.
	masked := ob.bidBitmap[word] & ((1 << (bit + 1)) - 1)
	for word >= 0 {
		if masked != 0 {
			price := int64(word*64) + int64(63-bits.LeadingZeros64(masked))
			if price < 1 {
				return nil
			}
			return &ob.bidBuckets[price]
		}
		word--
		if word >= 0 {
			masked = ob.bidBitmap[word]
		}
	}
	return nil
}

func (ob *OrderBookDirect) OrderCount() int {
	return len(ob.orderIndex)
}

// RestingOrders exports all resting orders for snapshot/recovery.
func (ob *OrderBookDirect) RestingOrders() []*Order {
	orders := make([]*Order, 0, len(ob.orderIndex))
	for _, do := range ob.orderIndex {
		orders = append(orders, do.ToOrder())
	}
	return orders
}

func (ob *OrderBookDirect) scanBestAsk() int64 {
	// Start from word 0; skip price 0 by masking.
	for w := 0; w < priceBitmapWords; w++ {
		word := ob.askBitmap[w]
		if w == 0 {
			word &^= 1 // clear bit 0 (price 0 is invalid)
		}
		if word != 0 {
			price := int64(w*64) + int64(bits.TrailingZeros64(word))
			if price < maxPrice {
				return price
			}
		}
	}
	return 0
}

func (ob *OrderBookDirect) scanBestBid() int64 {
	for w := priceBitmapWords - 1; w >= 0; w-- {
		word := ob.bidBitmap[w]
		if word != 0 {
			price := int64(w*64) + int64(63-bits.LeadingZeros64(word))
			if price >= 1 && price < maxPrice {
				return price
			}
		}
	}
	return 0
}
