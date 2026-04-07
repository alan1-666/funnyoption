package ringbuffer

import (
	"sync/atomic"
)

// RingBuffer is a lock-free Single-Producer Single-Consumer (SPSC) queue.
// Cache-line padding prevents false sharing between producer and consumer.
type RingBuffer[T any] struct {
	_     [64]byte       // cache line pad before write cursor
	write atomic.Uint64  // only modified by producer
	_     [56]byte       // pad between write and read
	read  atomic.Uint64  // only modified by consumer
	_     [56]byte       // pad after read
	mask  uint64
	slots []T
}

// New creates a ring buffer whose capacity is rounded up to the next power of two.
// Minimum capacity is 16.
func New[T any](requestedCap int) *RingBuffer[T] {
	cap := nextPowerOfTwo(requestedCap)
	if cap < 16 {
		cap = 16
	}
	return &RingBuffer[T]{
		mask:  uint64(cap - 1),
		slots: make([]T, cap),
	}
}

// TryPublish writes one item. Returns false if the buffer is full (back-pressure signal).
func (rb *RingBuffer[T]) TryPublish(item T) bool {
	w := rb.write.Load()
	if w-rb.read.Load() > rb.mask {
		return false
	}
	rb.slots[w&rb.mask] = item
	rb.write.Store(w + 1)
	return true
}

// TryConsume reads one item. Returns the zero value and false if empty.
func (rb *RingBuffer[T]) TryConsume() (T, bool) {
	r := rb.read.Load()
	if r >= rb.write.Load() {
		var zero T
		return zero, false
	}
	item := rb.slots[r&rb.mask]
	rb.read.Store(r + 1)
	return item, true
}

// DrainTo reads up to maxItems into dst, returning the count actually read.
// This amortises the cost of atomic loads across a batch.
func (rb *RingBuffer[T]) DrainTo(dst []T, maxItems int) int {
	r := rb.read.Load()
	w := rb.write.Load()
	avail := int(w - r)
	if avail == 0 {
		return 0
	}
	n := avail
	if n > maxItems {
		n = maxItems
	}
	if n > len(dst) {
		n = len(dst)
	}
	for i := 0; i < n; i++ {
		dst[i] = rb.slots[(r+uint64(i))&rb.mask]
	}
	rb.read.Store(r + uint64(n))
	return n
}

// Size returns the current number of unconsumed items.
func (rb *RingBuffer[T]) Size() int {
	return int(rb.write.Load() - rb.read.Load())
}

// Cap returns the total slot capacity.
func (rb *RingBuffer[T]) Cap() int {
	return int(rb.mask + 1)
}

// IsFull returns true when no more items can be published.
func (rb *RingBuffer[T]) IsFull() bool {
	return rb.write.Load()-rb.read.Load() > rb.mask
}

func nextPowerOfTwo(v int) int {
	if v <= 0 {
		return 1
	}
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v |= v >> 32
	return v + 1
}
