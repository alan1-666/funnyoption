package ringbuffer

import (
	"sync"
	"testing"
)

func TestBasicPublishConsume(t *testing.T) {
	rb := New[int](16)
	if rb.Cap() != 16 {
		t.Fatalf("expected cap 16, got %d", rb.Cap())
	}

	for i := 0; i < 16; i++ {
		if !rb.TryPublish(i) {
			t.Fatalf("publish %d failed", i)
		}
	}
	if rb.TryPublish(99) {
		t.Fatal("expected full")
	}

	for i := 0; i < 16; i++ {
		v, ok := rb.TryConsume()
		if !ok || v != i {
			t.Fatalf("expected (%d, true), got (%d, %v)", i, v, ok)
		}
	}
	_, ok := rb.TryConsume()
	if ok {
		t.Fatal("expected empty")
	}
}

func TestDrainTo(t *testing.T) {
	rb := New[int](16)
	for i := 0; i < 10; i++ {
		rb.TryPublish(i)
	}
	dst := make([]int, 16)
	n := rb.DrainTo(dst, 5)
	if n != 5 {
		t.Fatalf("expected 5, got %d", n)
	}
	for i := 0; i < 5; i++ {
		if dst[i] != i {
			t.Fatalf("dst[%d] = %d", i, dst[i])
		}
	}
	n = rb.DrainTo(dst, 16)
	if n != 5 {
		t.Fatalf("expected 5 remaining, got %d", n)
	}
}

func TestConcurrentSPSC(t *testing.T) {
	const N = 1_000_000
	rb := New[int64](4096)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := int64(0); i < N; i++ {
			for !rb.TryPublish(i) {
			}
		}
	}()

	go func() {
		defer wg.Done()
		expected := int64(0)
		for expected < N {
			v, ok := rb.TryConsume()
			if !ok {
				continue
			}
			if v != expected {
				t.Errorf("expected %d, got %d", expected, v)
				return
			}
			expected++
		}
	}()

	wg.Wait()
}

func BenchmarkTryPublishConsume(b *testing.B) {
	rb := New[int64](65536)
	for i := 0; i < b.N; i++ {
		rb.TryPublish(int64(i))
		rb.TryConsume()
	}
}
