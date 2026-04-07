package model

// Book is the interface that both OrderBook (sorted-slice) and
// OrderBookDirect (array-indexed buckets) implement.
type Book interface {
	HasOrder(orderID string) bool
	AddOrder(order *Order)
	RemoveOrder(order *Order)
	RemoveFromMap(orderID string)
	BestBidPrice() (int64, bool)
	BestAskPrice() (int64, bool)
	IsCross(order *Order) bool
	IsCrossWithPrice(order *Order, price int64) bool
	Snapshot(limit int) BookSnapshot
	OrderCount() int
}
