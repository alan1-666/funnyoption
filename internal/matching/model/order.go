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

func (o *Order) Validate() error {
	if o.OrderID == "" {
		return fmt.Errorf("order_id is required")
	}
	if o.MarketID <= 0 {
		return fmt.Errorf("market_id must be positive")
	}
	if stringsTrim(o.Outcome) == "" {
		return fmt.Errorf("outcome is required")
	}
	if !o.Side.IsValid() {
		return fmt.Errorf("invalid side: %s", o.Side)
	}
	if !o.Type.IsValid() {
		return fmt.Errorf("invalid type: %s", o.Type)
	}
	if !o.TimeInForce.IsValid() {
		return fmt.Errorf("invalid tif: %s", o.TimeInForce)
	}
	if o.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if o.Type == OrderTypeLimit && (o.Price < 1 || o.Price > 99) {
		return fmt.Errorf("limit order price must be between 1 and 99")
	}
	if o.Type == OrderTypeMarket && o.Price != 0 {
		return fmt.Errorf("market order price must be zero")
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

func stringsTrim(value string) string {
	var trimmed []rune
	for _, r := range value {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			trimmed = append(trimmed, r)
		}
	}
	return string(trimmed)
}
