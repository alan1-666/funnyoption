package service

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	"funnyoption/internal/posttrade"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type stubCommandStore struct {
	tradable bool
	command  sharedkafka.OrderCommand
	result   engine.Result
}

func (s *stubCommandStore) PersistResult(_ context.Context, command sharedkafka.OrderCommand, result engine.Result) error {
	s.command = command
	s.result = result
	return nil
}

func (s *stubCommandStore) PersistBatch(_ context.Context, items []posttrade.PersistItem) error {
	if len(items) > 0 {
		s.command = items[0].Command
		s.result = items[0].Result
	}
	return nil
}

func (s *stubCommandStore) MarketIsTradable(_ context.Context, _ int64) (bool, error) {
	return s.tradable, nil
}

type capturePublisher struct {
	items []sharedkafka.BatchItem
}

func (p *capturePublisher) PublishJSON(_ context.Context, topic, key string, payload any) error {
	p.items = append(p.items, sharedkafka.BatchItem{Topic: topic, Key: key, Payload: payload})
	return nil
}

func (p *capturePublisher) PublishJSONBatch(ctx context.Context, items []sharedkafka.BatchItem) error {
	p.items = append(p.items, items...)
	return nil
}

func (p *capturePublisher) Close() error { return nil }

func (p *capturePublisher) findOrderEvent() *sharedkafka.OrderEvent {
	for _, item := range p.items {
		if ev, ok := item.Payload.(sharedkafka.OrderEvent); ok {
			return &ev
		}
	}
	return nil
}

func TestHandleOrderCommandRejectsNonTradableMarket(t *testing.T) {
	store := &stubCommandStore{}
	publisher := &capturePublisher{}
	processor := NewCommandProcessor(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		nil,
		publisher,
		sharedkafka.NewTopics("funnyoption."),
		store,
	)

	payload, err := json.Marshal(sharedkafka.OrderCommand{
		CommandID:       "cmd_reject_1",
		OrderID:         "ord_reject_1",
		FreezeID:        "frz_reject_1",
		FreezeAsset:     "USDT",
		FreezeAmount:    610,
		CollateralAsset: "USDT",
		UserID:          1001,
		MarketID:        1101,
		Outcome:         "YES",
		Side:            "BUY",
		Type:            "LIMIT",
		TimeInForce:     "GTC",
		Price:           61,
		Quantity:        10,
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if err := processor.HandleOrderCommand(context.Background(), sharedkafka.Message{Value: payload}); err != nil {
		t.Fatalf("HandleOrderCommand returned error: %v", err)
	}

	if store.command.OrderID != "ord_reject_1" {
		t.Fatalf("expected rejected order to be persisted, got command=%+v", store.command)
	}
	if store.result.Order == nil || store.result.Order.Status != model.OrderStatusRejected {
		t.Fatalf("expected rejected order result, got %+v", store.result.Order)
	}
	if store.result.Order.CancelReason != model.CancelReasonMarketNotTradable {
		t.Fatalf("unexpected cancel reason: %s", store.result.Order.CancelReason)
	}

	event := publisher.findOrderEvent()
	if event == nil {
		t.Fatalf("expected order event in published batch, got %d items", len(publisher.items))
	}
	if event.Status != "REJECTED" || event.CancelReason != "MARKET_NOT_TRADABLE" {
		t.Fatalf("unexpected reject order event: %+v", event)
	}
	if event.FreezeID != "frz_reject_1" || event.FreezeAmount != 610 {
		t.Fatalf("expected freeze metadata to survive reject, got %+v", event)
	}
}
