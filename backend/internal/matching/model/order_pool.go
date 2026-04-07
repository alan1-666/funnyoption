package model

const defaultPoolSize = 64

type OrderPool struct {
	slab []DirectOrder
	free []int
	next int
}

func NewOrderPool(initialSize int) *OrderPool {
	if initialSize <= 0 {
		initialSize = defaultPoolSize
	}
	p := &OrderPool{
		slab: make([]DirectOrder, initialSize),
		free: make([]int, 0, 256),
	}
	for i := range p.slab {
		p.slab[i].poolID = i
	}
	return p
}

func (p *OrderPool) Get() *DirectOrder {
	if len(p.free) > 0 {
		idx := p.free[len(p.free)-1]
		p.free = p.free[:len(p.free)-1]
		o := &p.slab[idx]
		o.reset()
		return o
	}
	if p.next >= len(p.slab) {
		p.grow()
	}
	o := &p.slab[p.next]
	o.poolID = p.next
	p.next++
	return o
}

func (p *OrderPool) Put(o *DirectOrder) {
	if o == nil {
		return
	}
	o.reset()
	p.free = append(p.free, o.poolID)
}

func (p *OrderPool) grow() {
	oldLen := len(p.slab)
	newLen := oldLen * 2
	newSlab := make([]DirectOrder, newLen)
	copy(newSlab, p.slab)
	for i := oldLen; i < newLen; i++ {
		newSlab[i].poolID = i
	}
	p.slab = newSlab
}
