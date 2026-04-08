package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func TestListenAndServe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.Default()
	ListenAndServe(ctx, logger, ":0", "test-svc", "test")

	// Port :0 means the OS picks a port — but our implementation uses the literal addr.
	// Use a fixed ephemeral port for the test instead.
	cancel()

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	addr := "127.0.0.1:18199"
	ListenAndServe(ctx2, logger, addr, "test-svc", "test")

	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get("http://" + addr + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status=ok, got %s", body["status"])
	}
	if body["service"] != "test-svc" {
		t.Fatalf("expected service=test-svc, got %s", body["service"])
	}
}

func TestListenAndServeEmptyAddr(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Should be a no-op, not panic.
	ListenAndServe(ctx, slog.Default(), "", "noop", "test")
}
