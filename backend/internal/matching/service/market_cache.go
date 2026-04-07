package service

import (
	"context"
	"sync"
	"time"
)

type marketCacheEntry struct {
	tradable  bool
	expiresAt time.Time
}

type MarketTradableCache struct {
	mu    sync.RWMutex
	items map[int64]marketCacheEntry
	ttl   time.Duration
	fetch func(ctx context.Context, marketID int64) (bool, error)

	frozenMu    sync.RWMutex
	frozen      bool
	frozenValid bool
	frozenExp   time.Time
	fetchFrozen func(ctx context.Context) (bool, error)
}

func NewMarketTradableCache(
	ttl time.Duration,
	fetch func(ctx context.Context, marketID int64) (bool, error),
	fetchFrozen func(ctx context.Context) (bool, error),
) *MarketTradableCache {
	if ttl <= 0 {
		ttl = 5 * time.Second
	}
	return &MarketTradableCache{
		items:       make(map[int64]marketCacheEntry),
		ttl:         ttl,
		fetch:       fetch,
		fetchFrozen: fetchFrozen,
	}
}

func (c *MarketTradableCache) IsTradable(ctx context.Context, marketID int64) (bool, error) {
	frozen, err := c.isFrozen(ctx)
	if err != nil {
		return false, err
	}
	if frozen {
		return false, nil
	}

	now := time.Now()
	c.mu.RLock()
	entry, ok := c.items[marketID]
	c.mu.RUnlock()
	if ok && now.Before(entry.expiresAt) {
		return entry.tradable, nil
	}

	tradable, err := c.fetch(ctx, marketID)
	if err != nil {
		return false, err
	}

	c.mu.Lock()
	c.items[marketID] = marketCacheEntry{tradable: tradable, expiresAt: now.Add(c.ttl)}
	c.mu.Unlock()
	return tradable, nil
}

func (c *MarketTradableCache) isFrozen(ctx context.Context) (bool, error) {
	now := time.Now()
	c.frozenMu.RLock()
	if c.frozenValid && now.Before(c.frozenExp) {
		val := c.frozen
		c.frozenMu.RUnlock()
		return val, nil
	}
	c.frozenMu.RUnlock()

	frozen, err := c.fetchFrozen(ctx)
	if err != nil {
		return false, err
	}

	c.frozenMu.Lock()
	c.frozen = frozen
	c.frozenValid = true
	c.frozenExp = now.Add(c.ttl)
	c.frozenMu.Unlock()
	return frozen, nil
}

func (c *MarketTradableCache) Invalidate(marketID int64) {
	c.mu.Lock()
	delete(c.items, marketID)
	c.mu.Unlock()
}

func (c *MarketTradableCache) InvalidateAll() {
	c.mu.Lock()
	c.items = make(map[int64]marketCacheEntry)
	c.mu.Unlock()
	c.frozenMu.Lock()
	c.frozenValid = false
	c.frozenMu.Unlock()
}
