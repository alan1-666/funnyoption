package ringbuffer

import (
	"runtime"
	"time"
)

// IdleStrategy controls how a consumer thread waits when no data is available.
// Follows the spin → yield → park pattern from Aeron.
type IdleStrategy struct {
	spinCount  int
	yieldCount int
	parkNs     time.Duration

	state int
	count int
}

func NewIdleStrategy(spinCount, yieldCount int, parkDuration time.Duration) *IdleStrategy {
	if spinCount <= 0 {
		spinCount = 100
	}
	if yieldCount <= 0 {
		yieldCount = 10
	}
	if parkDuration <= 0 {
		parkDuration = 50 * time.Microsecond
	}
	return &IdleStrategy{
		spinCount:  spinCount,
		yieldCount: yieldCount,
		parkNs:     parkDuration,
	}
}

func DefaultIdleStrategy() *IdleStrategy {
	return NewIdleStrategy(100, 10, 50*time.Microsecond)
}

// Idle is called when the consumer loop found no work.
func (s *IdleStrategy) Idle() {
	switch s.state {
	case 0: // spin
		s.count++
		if s.count >= s.spinCount {
			s.state = 1
			s.count = 0
		}
	case 1: // yield
		runtime.Gosched()
		s.count++
		if s.count >= s.yieldCount {
			s.state = 2
			s.count = 0
		}
	case 2: // park
		time.Sleep(s.parkNs)
	}
}

// Reset should be called when work was found, resetting back to spin state.
func (s *IdleStrategy) Reset() {
	s.state = 0
	s.count = 0
}
