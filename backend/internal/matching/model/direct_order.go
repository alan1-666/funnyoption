package model

type DirectOrder struct {
	OrderID         string
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
	ClientOrderID   string
	CreatedAtMillis int64
	UpdatedAtMillis int64

	bucket *Bucket
	prev   *DirectOrder
	next   *DirectOrder
	poolID int
}

func (d *DirectOrder) RemainingQuantity() int64 {
	r := d.Quantity - d.FilledQuantity
	if r < 0 {
		return 0
	}
	return r
}

func (d *DirectOrder) ApplyFill(quantity int64) {
	d.FilledQuantity += quantity
	if d.FilledQuantity >= d.Quantity {
		d.Status = OrderStatusFilled
		return
	}
	if d.FilledQuantity > 0 {
		d.Status = OrderStatusPartiallyFilled
	}
}

func (d *DirectOrder) Cancel(reason CancelReason) {
	d.CancelReason = reason
	d.Status = OrderStatusCancelled
}

func (d *DirectOrder) IsBuy() bool       { return d.Side == OrderSideBuy }
func (d *DirectOrder) IsSell() bool      { return d.Side == OrderSideSell }
func (d *DirectOrder) Next() *DirectOrder { return d.next }
func (d *DirectOrder) Prev() *DirectOrder { return d.prev }

func (d *DirectOrder) BookKey() string {
	return BuildBookKey(d.MarketID, d.Outcome)
}

func (d *DirectOrder) FromOrder(o *Order) *DirectOrder {
	d.OrderID = o.OrderID
	d.UserID = o.UserID
	d.MarketID = o.MarketID
	d.Outcome = o.Outcome
	d.Side = o.Side
	d.Type = o.Type
	d.TimeInForce = o.TimeInForce
	d.Price = o.Price
	d.Quantity = o.Quantity
	d.FilledQuantity = o.FilledQuantity
	d.Status = o.Status
	d.CancelReason = o.CancelReason
	d.ClientOrderID = o.ClientOrderID
	d.CreatedAtMillis = o.CreatedAtMillis
	d.UpdatedAtMillis = o.UpdatedAtMillis
	d.bucket = nil
	d.prev = nil
	d.next = nil
	return d
}

func (d *DirectOrder) ToOrder() *Order {
	return &Order{
		OrderID:         d.OrderID,
		UserID:          d.UserID,
		MarketID:        d.MarketID,
		Outcome:         d.Outcome,
		Side:            d.Side,
		Type:            d.Type,
		TimeInForce:     d.TimeInForce,
		Price:           d.Price,
		Quantity:        d.Quantity,
		FilledQuantity:  d.FilledQuantity,
		Status:          d.Status,
		CancelReason:    d.CancelReason,
		ClientOrderID:   d.ClientOrderID,
		CreatedAtMillis: d.CreatedAtMillis,
		UpdatedAtMillis: d.UpdatedAtMillis,
	}
}

func (d *DirectOrder) reset() {
	d.OrderID = ""
	d.UserID = 0
	d.MarketID = 0
	d.Outcome = ""
	d.Side = ""
	d.Type = ""
	d.TimeInForce = ""
	d.Price = 0
	d.Quantity = 0
	d.FilledQuantity = 0
	d.Status = ""
	d.CancelReason = ""
	d.ClientOrderID = ""
	d.CreatedAtMillis = 0
	d.UpdatedAtMillis = 0
	d.bucket = nil
	d.prev = nil
	d.next = nil
}
