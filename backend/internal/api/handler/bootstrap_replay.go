package handler

import (
	"context"
	"errors"
	"strings"
	"sync"
)

type bootstrapReplayGate struct {
	mu      sync.Mutex
	entries map[string]*bootstrapReplayGateEntry
}

type bootstrapReplayGateEntry struct {
	refs int
	mu   sync.Mutex
}

func newBootstrapReplayGate() *bootstrapReplayGate {
	return &bootstrapReplayGate{
		entries: map[string]*bootstrapReplayGateEntry{},
	}
}

func (g *bootstrapReplayGate) Lock(key string) func() {
	g.mu.Lock()
	entry := g.entries[key]
	if entry == nil {
		entry = &bootstrapReplayGateEntry{}
		g.entries[key] = entry
	}
	entry.refs++
	g.mu.Unlock()

	entry.mu.Lock()

	return func() {
		entry.mu.Unlock()

		g.mu.Lock()
		entry.refs--
		if entry.refs == 0 {
			delete(g.entries, key)
		}
		g.mu.Unlock()
	}
}

func (h *OrderHandler) bootstrapOrderAlreadyAccepted(ctx context.Context, orderID string) (bool, error) {
	if _, err := h.store.GetOrder(ctx, orderID); err == nil {
		return true, nil
	} else if !errors.Is(err, ErrNotFound) {
		return false, err
	}

	freeze, err := h.store.GetLatestFreezeByRef(ctx, "ORDER", orderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	switch strings.ToUpper(strings.TrimSpace(freeze.Status)) {
	case "", "RELEASED":
		return false, nil
	default:
		return true, nil
	}
}
