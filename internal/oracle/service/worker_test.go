package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	sharedkafka "funnyoption/internal/shared/kafka"
)

type captureStore struct {
	markets []EligibleMarket
	updates []ResolutionUpdate
}

func (s *captureStore) ListEligibleMarkets(ctx context.Context, now int64, limit int) ([]EligibleMarket, error) {
	_ = ctx
	_ = now
	_ = limit
	return s.markets, nil
}

func (s *captureStore) UpsertResolution(ctx context.Context, update ResolutionUpdate) error {
	_ = ctx
	s.updates = append(s.updates, update)
	for index := range s.markets {
		if s.markets[index].MarketID != update.MarketID {
			continue
		}
		s.markets[index].ResolutionStatus = update.Status
		s.markets[index].ResolvedOutcome = update.ResolvedOutcome
		s.markets[index].ResolverType = update.ResolverType
		s.markets[index].ResolverRef = update.ResolverRef
		s.markets[index].Evidence = update.Evidence
	}
	return nil
}

type capturePublisher struct {
	events   []sharedkafka.MarketEvent
	failures int
}

func (p *capturePublisher) PublishJSON(ctx context.Context, topic, key string, payload any) error {
	_ = ctx
	_ = topic
	_ = key
	if p.failures > 0 {
		p.failures--
		return errors.New("publish failed")
	}
	if event, ok := payload.(sharedkafka.MarketEvent); ok {
		p.events = append(p.events, event)
	}
	return nil
}

func (p *capturePublisher) Close() error { return nil }

func TestWorkerPollOnceObservesAndPublishes(t *testing.T) {
	now := time.Now().Unix()
	resolveAt := now - 1
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":1,"price":"85123.45","qty":"1.0","quoteQty":"85123.45","time":` + jsonNumber((resolveAt+1)*1000) + `,"isBuyerMaker":true,"isBestMatch":true}]`))
	}))
	defer server.Close()

	store := &captureStore{
		markets: []EligibleMarket{
			{
				MarketID:     88,
				ResolveAt:    resolveAt,
				MarketStatus: "OPEN",
				CategoryKey:  "CRYPTO",
				Metadata:     supportedOracleMetadata("BTCUSDT", "85000.00000000"),
				OptionSchema: json.RawMessage(`[{"key":"YES"},{"key":"NO"}]`),
			},
		},
	}
	publisher := &capturePublisher{}
	worker := NewWorker(nil, store, NewBinanceProvider(server.URL, server.Client()), publisher, sharedkafka.NewTopics("funnyoption."), time.Second)

	if err := worker.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce returned error: %v", err)
	}
	if len(store.updates) != 2 {
		t.Fatalf("expected OBSERVED write plus dispatched marker, got %d updates", len(store.updates))
	}
	if store.updates[0].Status != "OBSERVED" || store.updates[0].ResolvedOutcome != "YES" {
		t.Fatalf("unexpected initial resolution update: %+v", store.updates[0])
	}
	if store.updates[1].Status != "OBSERVED" || store.updates[1].ResolvedOutcome != "YES" {
		t.Fatalf("unexpected dispatched resolution update: %+v", store.updates[1])
	}
	if store.updates[1].ResolverType != ResolverTypeOraclePrice {
		t.Fatalf("expected resolver type %s, got %s", ResolverTypeOraclePrice, store.updates[1].ResolverType)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("expected one market event, got %d", len(publisher.events))
	}
	if publisher.events[0].ResolvedOutcome != "YES" {
		t.Fatalf("unexpected market event: %+v", publisher.events[0])
	}
	dispatch := loadDispatch(t, store.markets[0].Evidence)
	if dispatch == nil || dispatch.Status != DispatchStatusDispatched || dispatch.AttemptCount != 1 {
		t.Fatalf("expected dispatched marker after successful publish, got %+v", dispatch)
	}
}

func TestWorkerPollOnceWritesRetryableWindowError(t *testing.T) {
	now := time.Now().Unix()
	resolveAt := now - 1
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":1,"price":"85123.45","qty":"1.0","quoteQty":"85123.45","time":` + jsonNumber((resolveAt-301)*1000) + `,"isBuyerMaker":true,"isBestMatch":true}]`))
	}))
	defer server.Close()

	store := &captureStore{
		markets: []EligibleMarket{
			{
				MarketID:     89,
				ResolveAt:    resolveAt,
				MarketStatus: "OPEN",
				CategoryKey:  "CRYPTO",
				Metadata:     supportedOracleMetadata("BTCUSDT", "85000.00000000"),
				OptionSchema: json.RawMessage(`[{"key":"YES"},{"key":"NO"}]`),
			},
		},
	}
	publisher := &capturePublisher{}
	worker := NewWorker(nil, store, NewBinanceProvider(server.URL, server.Client()), publisher, sharedkafka.NewTopics("funnyoption."), time.Second)

	if err := worker.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce returned error: %v", err)
	}
	if len(store.updates) != 1 {
		t.Fatalf("expected one resolution update, got %d", len(store.updates))
	}
	if store.updates[0].Status != "RETRYABLE_ERROR" {
		t.Fatalf("expected RETRYABLE_ERROR, got %+v", store.updates[0])
	}
	if len(publisher.events) != 0 {
		t.Fatalf("expected no market events for retryable error, got %+v", publisher.events)
	}
}

func TestWorkerPollOnceWritesTerminalUnsupportedSymbol(t *testing.T) {
	now := time.Now().Unix()
	resolveAt := now - 1
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":-1121,"msg":"Invalid symbol."}`))
	}))
	defer server.Close()

	store := &captureStore{
		markets: []EligibleMarket{
			{
				MarketID:     90,
				ResolveAt:    resolveAt,
				MarketStatus: "OPEN",
				CategoryKey:  "CRYPTO",
				Metadata:     supportedOracleMetadata("BTCBROKEN", "85000.00000000"),
				OptionSchema: json.RawMessage(`[{"key":"YES"},{"key":"NO"}]`),
			},
		},
	}
	publisher := &capturePublisher{}
	worker := NewWorker(nil, store, NewBinanceProvider(server.URL, server.Client()), publisher, sharedkafka.NewTopics("funnyoption."), time.Second)

	if err := worker.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce returned error: %v", err)
	}
	if len(store.updates) != 1 {
		t.Fatalf("expected one resolution update, got %d", len(store.updates))
	}
	if store.updates[0].Status != "TERMINAL_ERROR" {
		t.Fatalf("expected TERMINAL_ERROR, got %+v", store.updates[0])
	}
	if len(publisher.events) != 0 {
		t.Fatalf("expected no market events for terminal error, got %+v", publisher.events)
	}
}

func TestWorkerPollOnceSkipsDuplicateEmitWhileObservedDispatched(t *testing.T) {
	now := time.Now().Unix()
	resolveAt := now - 1

	store := &captureStore{
		markets: []EligibleMarket{
			{
				MarketID:         91,
				ResolveAt:        resolveAt,
				MarketStatus:     "OPEN",
				CategoryKey:      "CRYPTO",
				Metadata:         supportedOracleMetadata("BTCUSDT", "85000.00000000"),
				OptionSchema:     json.RawMessage(`[{"key":"YES"},{"key":"NO"}]`),
				ResolutionStatus: "OBSERVED",
				ResolvedOutcome:  "YES",
				ResolverType:     ResolverTypeOraclePrice,
				ResolverRef:      "oracle_price:BINANCE:BTCUSDT:" + strconv.FormatInt(resolveAt, 10),
				Evidence:         json.RawMessage(`{"version":1,"dispatch":{"status":"DISPATCHED","attempt_count":1,"last_attempt_at":1,"dispatched_at":1}}`),
			},
		},
	}
	publisher := &capturePublisher{}
	worker := NewWorker(nil, store, nil, publisher, sharedkafka.NewTopics("funnyoption."), time.Second)

	if err := worker.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce returned error: %v", err)
	}
	if len(store.updates) != 0 {
		t.Fatalf("expected no resolution updates for duplicate observed poll, got %+v", store.updates)
	}
	if len(publisher.events) != 0 {
		t.Fatalf("expected no duplicate market.event publish while OBSERVED, got %+v", publisher.events)
	}
}

func TestWorkerPollOnceRetriesPendingDispatchAfterPublishFailure(t *testing.T) {
	now := time.Now().Unix()
	resolveAt := now - 1
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":1,"price":"85123.45","qty":"1.0","quoteQty":"85123.45","time":` + jsonNumber((resolveAt+1)*1000) + `,"isBuyerMaker":true,"isBestMatch":true}]`))
	}))
	defer server.Close()

	store := &captureStore{
		markets: []EligibleMarket{
			{
				MarketID:     92,
				ResolveAt:    resolveAt,
				MarketStatus: "OPEN",
				CategoryKey:  "CRYPTO",
				Metadata:     supportedOracleMetadata("BTCUSDT", "85000.00000000"),
				OptionSchema: json.RawMessage(`[{"key":"YES"},{"key":"NO"}]`),
			},
		},
	}
	publisher := &capturePublisher{failures: 1}
	worker := NewWorker(nil, store, NewBinanceProvider(server.URL, server.Client()), publisher, sharedkafka.NewTopics("funnyoption."), time.Second)

	if err := worker.pollOnce(context.Background()); err == nil {
		t.Fatalf("expected publish failure to bubble up")
	}
	if len(store.updates) != 2 {
		t.Fatalf("expected OBSERVED write plus pending dispatch retry marker, got %d updates", len(store.updates))
	}
	dispatch := loadDispatch(t, store.markets[0].Evidence)
	if dispatch == nil || dispatch.Status != DispatchStatusPending || dispatch.AttemptCount != 1 {
		t.Fatalf("expected pending dispatch after failed publish, got %+v", dispatch)
	}
	if len(publisher.events) != 0 {
		t.Fatalf("expected failed publish to record no market events, got %+v", publisher.events)
	}

	if err := worker.pollOnce(context.Background()); err != nil {
		t.Fatalf("retry pollOnce returned error: %v", err)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("expected retry poll to publish exactly one market.event, got %d", len(publisher.events))
	}
	dispatch = loadDispatch(t, store.markets[0].Evidence)
	if dispatch == nil || dispatch.Status != DispatchStatusDispatched || dispatch.AttemptCount != 2 || dispatch.DispatchedAt <= 0 {
		t.Fatalf("expected dispatched marker after retry success, got %+v", dispatch)
	}

	updateCount := len(store.updates)
	if err := worker.pollOnce(context.Background()); err != nil {
		t.Fatalf("post-dispatch pollOnce returned error: %v", err)
	}
	if len(publisher.events) != 1 {
		t.Fatalf("expected dispatched observation to skip duplicate publish, got %d events", len(publisher.events))
	}
	if len(store.updates) != updateCount {
		t.Fatalf("expected no extra resolution writes after dispatch marker is set, got %d -> %d", updateCount, len(store.updates))
	}
}

func supportedOracleMetadata(symbol, threshold string) json.RawMessage {
	return json.RawMessage(`{
		"resolution": {
			"version": 1,
			"mode": "ORACLE_PRICE",
			"market_kind": "CRYPTO_PRICE_THRESHOLD",
			"manual_fallback_allowed": true,
			"oracle": {
				"source_kind": "HTTP_JSON",
				"provider_key": "BINANCE",
				"instrument": {
					"kind": "SPOT",
					"base_asset": "BTC",
					"quote_asset": "USDT",
					"symbol": "` + symbol + `"
				},
				"price": {
					"field": "LAST_PRICE",
					"scale": 8,
					"rounding_mode": "ROUND_HALF_UP",
					"max_data_age_sec": 120
				},
				"window": {
					"anchor": "RESOLVE_AT",
					"before_sec": 300,
					"after_sec": 300
				},
				"rule": {
					"type": "PRICE_THRESHOLD",
					"comparator": "GTE",
					"threshold_price": "` + threshold + `"
				}
			}
		}
	}`)
}

func jsonNumber(value int64) string {
	return strconv.FormatInt(value, 10)
}

func loadDispatch(t *testing.T, raw json.RawMessage) *EvidenceDispatch {
	t.Helper()
	var stored StoredEvidence
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("failed to decode evidence: %v", err)
	}
	return stored.Dispatch
}
