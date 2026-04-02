package model

type DepthLevel struct {
	Price  int64
	Orders []*Order
}

func (l *DepthLevel) Append(order *Order) {
	l.Orders = append(l.Orders, order)
}

func (l *DepthLevel) IsEmpty() bool {
	return len(l.Orders) == 0
}

func (l *DepthLevel) TotalQuantity() int64 {
	var total int64
	for _, order := range l.Orders {
		total += order.RemainingQuantity()
	}
	return total
}
