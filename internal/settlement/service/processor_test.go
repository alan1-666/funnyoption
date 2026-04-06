package service

import (
	"context"
	"encoding/json"
	"testing"

	sharedkafka "funnyoption/internal/shared/kafka"
)

type fakePublisher struct {
	settlements []sharedkafka.SettlementCompletedEvent
	orders      []sharedkafka.OrderEvent
}

func (f *fakePublisher) PublishJSON(ctx context.Context, topic, key string, payload any) error {
	_ = ctx
	_ = topic
	_ = key
	switch event := payload.(type) {
	case sharedkafka.SettlementCompletedEvent:
		f.settlements = append(f.settlements, event)
	case sharedkafka.OrderEvent:
		f.orders = append(f.orders, event)
	}
	return nil
}

func (f *fakePublisher) Close() error { return nil }

func TestProcessorResolveMarket(t *testing.T) {
	store := newPositionStore()
	publisher := &fakePublisher{}
	processor := NewProcessor(store, publisher, sharedkafka.NewTopics("funnyoption."))

	posPayload, _ := json.Marshal(sharedkafka.PositionChangedEvent{
		EventID:       "pos_1",
		SourceTradeID: "trade_1",
		UserID:        1001,
		MarketID:      88,
		Outcome:       "YES",
		PositionAsset: "POSITION:88:YES",
		DeltaQuantity: 25,
	})
	if err := processor.HandlePositionChanged(context.Background(), sharedkafka.Message{Value: posPayload}); err != nil {
		t.Fatalf("HandlePositionChanged returned error: %v", err)
	}

	marketPayload, _ := json.Marshal(sharedkafka.MarketEvent{
		EventID:         "mkt_1",
		MarketID:        88,
		Status:          "RESOLVED",
		ResolvedOutcome: "YES",
	})
	if err := processor.HandleMarketEvent(context.Background(), sharedkafka.Message{Value: marketPayload}); err != nil {
		t.Fatalf("HandleMarketEvent returned error: %v", err)
	}

	if len(publisher.settlements) != 1 {
		t.Fatalf("expected 1 settlement event, got %d", len(publisher.settlements))
	}
	if publisher.settlements[0].UserID != 1001 || publisher.settlements[0].PayoutAmount != 2500 {
		t.Fatalf("unexpected settlement event: %+v", publisher.settlements[0])
	}
}

type cancelOrderStore struct {
	cancelled []cancelledOrder
	resolved  []int64
	frozen    bool
	deltas    int
}

func (s *cancelOrderStore) ApplyDelta(_ context.Context, _, _ int64, _, _ string, _ int64) error {
	s.deltas++
	return nil
}

func (s *cancelOrderStore) ResolveMarket(_ context.Context, input ResolveMarketInput) (bool, error) {
	s.resolved = append(s.resolved, input.MarketID)
	return true, nil
}

func (s *cancelOrderStore) CancelActiveOrders(_ context.Context, _ int64, _ string) ([]cancelledOrder, error) {
	return s.cancelled, nil
}

func (s *cancelOrderStore) WinningPositions(_ context.Context, _ int64, _ string) ([]winningPosition, error) {
	return nil, nil
}

func (s *cancelOrderStore) MarkSettled(_ context.Context, _ sharedkafka.SettlementCompletedEvent) error {
	return nil
}

func (s *cancelOrderStore) RollupFrozen(_ context.Context) (bool, error) {
	return s.frozen, nil
}

func TestProcessorResolveMarketCancelsActiveOrders(t *testing.T) {
	store := &cancelOrderStore{
		cancelled: []cancelledOrder{
			{
				OrderID:           "ord_resting_1",
				CommandID:         "cmd_resting_1",
				ClientOrderID:     "cli_resting_1",
				UserID:            1002,
				MarketID:          88,
				Outcome:           "YES",
				Side:              "BUY",
				OrderType:         "LIMIT",
				TimeInForce:       "GTC",
				CollateralAsset:   "USDT",
				FreezeID:          "frz_resting_1",
				FreezeAsset:       "USDT",
				FreezeAmount:      500,
				Price:             50,
				Quantity:          10,
				FilledQuantity:    0,
				RemainingQuantity: 10,
				Status:            "CANCELLED",
				CancelReason:      "MARKET_RESOLVED",
			},
		},
	}
	publisher := &fakePublisher{}
	processor := NewProcessor(store, publisher, sharedkafka.NewTopics("funnyoption."))

	marketPayload, _ := json.Marshal(sharedkafka.MarketEvent{
		EventID:         "mkt_2",
		TraceID:         "trace_mkt_2",
		MarketID:        88,
		Status:          "RESOLVED",
		ResolvedOutcome: "YES",
	})
	if err := processor.HandleMarketEvent(context.Background(), sharedkafka.Message{Value: marketPayload}); err != nil {
		t.Fatalf("HandleMarketEvent returned error: %v", err)
	}

	if len(publisher.orders) != 1 {
		t.Fatalf("expected 1 cancellation order event, got %d", len(publisher.orders))
	}
	if publisher.orders[0].Status != "CANCELLED" || publisher.orders[0].CancelReason != "MARKET_RESOLVED" {
		t.Fatalf("unexpected cancellation order event: %+v", publisher.orders[0])
	}
}

func TestProcessorResolveMarketSkipsDuplicateResolvedEvent(t *testing.T) {
	store := newPositionStore()
	publisher := &fakePublisher{}
	processor := NewProcessor(store, publisher, sharedkafka.NewTopics("funnyoption."))

	posPayload, _ := json.Marshal(sharedkafka.PositionChangedEvent{
		EventID:       "pos_2",
		SourceTradeID: "trade_2",
		UserID:        1001,
		MarketID:      99,
		Outcome:       "YES",
		PositionAsset: "POSITION:99:YES",
		DeltaQuantity: 25,
	})
	if err := processor.HandlePositionChanged(context.Background(), sharedkafka.Message{Value: posPayload}); err != nil {
		t.Fatalf("HandlePositionChanged returned error: %v", err)
	}

	marketPayload, _ := json.Marshal(sharedkafka.MarketEvent{
		EventID:         "mkt_dup_1",
		MarketID:        99,
		Status:          "RESOLVED",
		ResolvedOutcome: "YES",
	})
	if err := processor.HandleMarketEvent(context.Background(), sharedkafka.Message{Value: marketPayload}); err != nil {
		t.Fatalf("first HandleMarketEvent returned error: %v", err)
	}
	if err := processor.HandleMarketEvent(context.Background(), sharedkafka.Message{Value: marketPayload}); err != nil {
		t.Fatalf("second HandleMarketEvent returned error: %v", err)
	}

	if len(publisher.settlements) != 1 {
		t.Fatalf("expected duplicate resolved event to publish 1 settlement, got %d", len(publisher.settlements))
	}
}

func TestProcessorSkipsResolutionSideEffectsWhileFrozen(t *testing.T) {
	store := &cancelOrderStore{
		frozen: true,
		cancelled: []cancelledOrder{
			{
				OrderID:      "ord_resting_1",
				MarketID:     88,
				Status:       "CANCELLED",
				CancelReason: "MARKET_RESOLVED",
			},
		},
	}
	publisher := &fakePublisher{}
	processor := NewProcessor(store, publisher, sharedkafka.NewTopics("funnyoption."))

	marketPayload, _ := json.Marshal(sharedkafka.MarketEvent{
		EventID:         "mkt_frozen_1",
		TraceID:         "trace_frozen_1",
		MarketID:        88,
		Status:          "RESOLVED",
		ResolvedOutcome: "YES",
	})
	if err := processor.HandleMarketEvent(context.Background(), sharedkafka.Message{Value: marketPayload}); err != nil {
		t.Fatalf("HandleMarketEvent returned error: %v", err)
	}

	if len(store.resolved) != 0 {
		t.Fatalf("expected frozen settlement lane to skip resolve writes, got %+v", store.resolved)
	}
	if store.deltas != 0 {
		t.Fatalf("expected frozen settlement lane to skip position deltas, got %d", store.deltas)
	}
	if len(publisher.orders) != 0 || len(publisher.settlements) != 0 {
		t.Fatalf("expected no settlement publishes while frozen, got orders=%+v settlements=%+v", publisher.orders, publisher.settlements)
	}
}

func TestProcessorSkipsPositionDeltaWhileFrozen(t *testing.T) {
	store := &cancelOrderStore{frozen: true}
	publisher := &fakePublisher{}
	processor := NewProcessor(store, publisher, sharedkafka.NewTopics("funnyoption."))

	posPayload, _ := json.Marshal(sharedkafka.PositionChangedEvent{
		EventID:       "pos_frozen_1",
		SourceTradeID: "trade_frozen_1",
		UserID:        1001,
		MarketID:      88,
		Outcome:       "YES",
		PositionAsset: "POSITION:88:YES",
		DeltaQuantity: 25,
	})
	if err := processor.HandlePositionChanged(context.Background(), sharedkafka.Message{Value: posPayload}); err != nil {
		t.Fatalf("HandlePositionChanged returned error: %v", err)
	}

	if store.deltas != 0 {
		t.Fatalf("expected frozen position lane to skip deltas, got %d", store.deltas)
	}
	if len(publisher.orders) != 0 || len(publisher.settlements) != 0 {
		t.Fatalf("expected no publishes while frozen, got orders=%+v settlements=%+v", publisher.orders, publisher.settlements)
	}
}
