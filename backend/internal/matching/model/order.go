package model

import "fmt"

type Order struct {
	OrderID         string
	ClientOrderID   string
	UserID          int64
	MarketID        int64
	Outcome         string
	Side            OrderSide
	Type            OrderType
	TimeInForce     TimeInForce
	Price           int64
	Quantity        int64
	FilledQuantity  int64
	Status          OrderStatus
	CancelReason    CancelReason
	CreatedAtMillis int64
	UpdatedAtMillis int64
}

// Validate is a lightweight assert-guard for invariants that should have been
// enforced upstream (in the OrderService). It is intentionally minimal so the
// engine hot path stays fast; upstream validation catches user-facing errors.
// The engine only accepts LIMIT orders; MARKET orders are converted to
// LIMIT IOC upstream.
func (o *Order) Validate() error {
	if o.OrderID == "" {
		return fmt.Errorf("assert: order_id is required")
	}
	if o.Quantity <= 0 {
		return fmt.Errorf("assert: quantity must be positive")
	}
	if o.Price < 1 || o.Price > 9999 {
		return fmt.Errorf("assert: price out of range [1, 9999]")
	}
	return nil
}

func (o *Order) BookKey() string {
	return BuildBookKey(o.MarketID, o.Outcome)
}

func (o *Order) IsBuy() bool {
	return o.Side == OrderSideBuy
}

func (o *Order) IsSell() bool {
	return o.Side == OrderSideSell
}

func (o *Order) IsLimit() bool {
	return o.Type == OrderTypeLimit
}

func (o *Order) IsMarket() bool {
	return o.Type == OrderTypeMarket
}

func (o *Order) RemainingQuantity() int64 {
	remaining := o.Quantity - o.FilledQuantity
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (o *Order) ApplyFill(quantity int64) {
	o.FilledQuantity += quantity
	if o.FilledQuantity >= o.Quantity {
		o.Status = OrderStatusFilled
		return
	}
	if o.FilledQuantity > 0 {
		o.Status = OrderStatusPartiallyFilled
	}
}

func (o *Order) Cancel(reason CancelReason) {
	o.CancelReason = reason
	o.Status = OrderStatusCancelled
}

func (o *Order) Reject(reason CancelReason) {
	o.CancelReason = reason
	o.Status = OrderStatusRejected
}

