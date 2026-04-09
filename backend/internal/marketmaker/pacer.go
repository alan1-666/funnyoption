package marketmaker

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type writePacer struct {
	mu          sync.Mutex
	minInterval time.Duration
	nextAllowed time.Time
	now         func() time.Time
	sleep       func(context.Context, time.Duration) error
}

func newWritePacer(minInterval time.Duration) *writePacer {
	return &writePacer{
		minInterval: minInterval,
		now:         time.Now,
		sleep:       sleepContext,
	}
}

func (p *writePacer) Wait(ctx context.Context) error {
	if p == nil || p.minInterval <= 0 {
		return nil
	}

	p.mu.Lock()
	now := p.now()
	reserveAt := now
	if p.nextAllowed.After(reserveAt) {
		reserveAt = p.nextAllowed
	}
	delay := reserveAt.Sub(now)
	p.nextAllowed = reserveAt.Add(p.minInterval)
	sleep := p.sleep
	p.mu.Unlock()

	if delay <= 0 {
		return nil
	}
	return sleep(ctx, delay)
}

func (p *writePacer) PushBack(delay time.Duration) {
	if p == nil || delay <= 0 {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	until := p.now().Add(delay)
	if until.After(p.nextAllowed) {
		p.nextAllowed = until
	}
}

func retryAfterDelay(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}

	value := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if value == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	when, err := http.ParseTime(value)
	if err != nil {
		return 0
	}

	delay := time.Until(when)
	if delay < 0 {
		return 0
	}
	return delay
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
