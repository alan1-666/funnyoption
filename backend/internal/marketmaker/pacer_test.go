package marketmaker

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestWritePacerWaitSpacesRequests(t *testing.T) {
	now := time.Unix(0, 0)
	var sleeps []time.Duration

	pacer := newWritePacer(5 * time.Second)
	pacer.now = func() time.Time { return now }
	pacer.sleep = func(ctx context.Context, delay time.Duration) error {
		sleeps = append(sleeps, delay)
		now = now.Add(delay)
		return nil
	}

	if err := pacer.Wait(context.Background()); err != nil {
		t.Fatalf("first wait failed: %v", err)
	}
	if len(sleeps) != 0 {
		t.Fatalf("first wait should not sleep, got %v", sleeps)
	}

	now = now.Add(1 * time.Second)
	if err := pacer.Wait(context.Background()); err != nil {
		t.Fatalf("second wait failed: %v", err)
	}
	if len(sleeps) != 1 || sleeps[0] != 4*time.Second {
		t.Fatalf("second wait sleep = %v, want [4s]", sleeps)
	}

	if err := pacer.Wait(context.Background()); err != nil {
		t.Fatalf("third wait failed: %v", err)
	}
	if len(sleeps) != 2 || sleeps[1] != 5*time.Second {
		t.Fatalf("third wait sleep = %v, want [4s 5s]", sleeps)
	}
}

func TestWritePacerPushBackHonorsRetryAfter(t *testing.T) {
	now := time.Unix(0, 0)
	var sleeps []time.Duration

	pacer := newWritePacer(5 * time.Second)
	pacer.now = func() time.Time { return now }
	pacer.sleep = func(ctx context.Context, delay time.Duration) error {
		sleeps = append(sleeps, delay)
		now = now.Add(delay)
		return nil
	}

	pacer.PushBack(30 * time.Second)

	if err := pacer.Wait(context.Background()); err != nil {
		t.Fatalf("wait failed: %v", err)
	}
	if len(sleeps) != 1 || sleeps[0] != 30*time.Second {
		t.Fatalf("sleep = %v, want [30s]", sleeps)
	}
}

func TestRetryAfterDelaySeconds(t *testing.T) {
	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.Header.Set("Retry-After", "9")

	if got := retryAfterDelay(resp); got != 9*time.Second {
		t.Fatalf("retryAfterDelay() = %v, want 9s", got)
	}
}

func TestRetryAfterDelayInvalidHeader(t *testing.T) {
	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.Header.Set("Retry-After", "nope")

	if got := retryAfterDelay(resp); got != 0 {
		t.Fatalf("retryAfterDelay() = %v, want 0", got)
	}
}
