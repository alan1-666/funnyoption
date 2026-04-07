package model

type Bucket struct {
	Price     int64
	Head      *DirectOrder
	Tail      *DirectOrder
	Volume    int64
	NumOrders int32
}

func (b *Bucket) Append(o *DirectOrder) {
	o.bucket = b
	o.prev = b.Tail
	o.next = nil
	if b.Tail != nil {
		b.Tail.next = o
	} else {
		b.Head = o
	}
	b.Tail = o
	b.Volume += o.RemainingQuantity()
	b.NumOrders++
}

func (b *Bucket) Remove(o *DirectOrder) {
	if o.prev != nil {
		o.prev.next = o.next
	} else {
		b.Head = o.next
	}
	if o.next != nil {
		o.next.prev = o.prev
	} else {
		b.Tail = o.prev
	}
	b.Volume -= o.RemainingQuantity()
	b.NumOrders--
	o.bucket = nil
	o.prev = nil
	o.next = nil
}

func (b *Bucket) IsEmpty() bool {
	return b.Head == nil
}

func (b *Bucket) TotalQuantity() int64 {
	return b.Volume
}
