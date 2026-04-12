package service

import (
	"context"
	"time"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/posttrade"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type CachedCommandStore struct {
	inner *SQLStore
	cache *MarketTradableCache
}

func NewCachedCommandStore(store *SQLStore) *CachedCommandStore {
	cache := NewMarketTradableCache(
		5*time.Second,
		store.MarketTradableNoFreeze,
		store.RollupFrozen,
	)
	return &CachedCommandStore{inner: store, cache: cache}
}

func (c *CachedCommandStore) MarketIsTradable(ctx context.Context, marketID int64) (bool, error) {
	return c.cache.IsTradable(ctx, marketID)
}

func (c *CachedCommandStore) PersistResult(ctx context.Context, command sharedkafka.OrderCommand, result engine.Result) error {
	return c.inner.PersistResult(ctx, command, result)
}

func (c *CachedCommandStore) PersistBatch(ctx context.Context, items []posttrade.PersistItem) error {
	return c.inner.PersistBatch(ctx, items)
}
