package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type expiryStoreStub struct {
	items     []ExpiredRestingOrder
	persisted []engine.Result
	commands  []sharedkafka.OrderCommand
}

func (s *expiryStoreStub) LoadExpiredRestingOrders(_ context.Context, _ int64) ([]ExpiredRestingOrder, error) {
	return s.items, nil
}

func (s *expiryStoreStub) PersistResult(_ context.Context, command sharedkafka.OrderCommand, result engine.Result) error {
	s.commands = append(s.commands, command)
	s.persisted = append(s.persisted, result)
	return nil
}

type publishCall struct {
	topic   string
	key     string
	payload any
}

type capturePublisherMulti struct {
	calls []publishCall
}

func (p *capturePublisherMulti) PublishJSON(_ context.Context, topic, key string, payload any) error {
	p.calls = append(p.calls, publishCall{topic: topic, key: key, payload: payload})
	return nil
}

func (p *capturePublisherMulti) PublishJSONBatch(ctx context.Context, items []sharedkafka.BatchItem) error {
	for _, item := range items {
		if err := p.PublishJSON(ctx, item.Topic, item.Key, item.Payload); err != nil {
			return err
		}
	}
	return nil
}

func (p *capturePublisherMulti) Close() error { return nil }

func TestOrderExpirySweeperCancelsPastCloseRestingOrders(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	matcher := engine.NewAsync(logger, 8)
	resting := &model.Order{
		OrderID:         "ord_close_1",
		ClientOrderID:   "client_close_1",
		UserID:          1001,
		MarketID:        77,
		Outcome:         "YES",
		Side:            model.OrderSideBuy,
		Type:            model.OrderTypeLimit,
		TimeInForce:     model.TimeInForceGTC,
		Price:           50,
		Quantity:        10,
		Status:          model.OrderStatusNew,
		CreatedAtMillis: 1000,
		UpdatedAtMillis: 1000,
	}
	if err := matcher.Restore(0, []*model.Order{resting}); err != nil {
		t.Fatalf("restore resting order: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	matcher.Start(ctx)

	store := &expiryStoreStub{
		items: []ExpiredRestingOrder{{
			Command: sharedkafka.OrderCommand{
				CommandID:         "cmd_original_1",
				OrderID:           resting.OrderID,
				ClientOrderID:     resting.ClientOrderID,
				UserID:            resting.UserID,
				MarketID:          resting.MarketID,
				Outcome:           resting.Outcome,
				Side:              string(resting.Side),
				Type:              string(resting.Type),
				TimeInForce:       string(resting.TimeInForce),
				CollateralAsset:   "USDT",
				FreezeID:          "frz_close_1",
				FreezeAsset:       "USDT",
				FreezeAmount:      500,
				Price:             resting.Price,
				Quantity:          resting.Quantity,
				RequestedAtMillis: resting.CreatedAtMillis,
			},
			Order: resting,
		}},
	}
	publisher := &capturePublisherMulti{}
	sweeper := newOrderExpirySweeper(logger, matcher, store, publisher, sharedkafka.NewTopics("funnyoption."))

	if err := sweeper.sweepOnce(ctx, time.Unix(123, 0)); err != nil {
		t.Fatalf("sweepOnce returned error: %v", err)
	}

	if len(store.persisted) != 1 {
		t.Fatalf("expected one persisted cancellation, got %d", len(store.persisted))
	}
	if len(store.persisted[0].Affected) != 1 {
		t.Fatalf("expected one affected order, got %+v", store.persisted[0])
	}
	cancelled := store.persisted[0].Affected[0]
	if cancelled.Status != model.OrderStatusCancelled {
		t.Fatalf("expected cancelled order status, got %s", cancelled.Status)
	}
	if cancelled.CancelReason != model.CancelReasonMarketClosed {
		t.Fatalf("expected market closed cancel reason, got %s", cancelled.CancelReason)
	}

	if len(publisher.calls) != 3 {
		t.Fatalf("expected order + depth + ticker events, got %d calls", len(publisher.calls))
	}
	orderEvent, ok := publisher.calls[0].payload.(sharedkafka.OrderEvent)
	if !ok {
		t.Fatalf("expected first payload to be order event, got %T", publisher.calls[0].payload)
	}
	if orderEvent.Status != "CANCELLED" || orderEvent.CancelReason != "MARKET_CLOSED" {
		t.Fatalf("unexpected lifecycle cancel event: %+v", orderEvent)
	}
	if orderEvent.FreezeID != "frz_close_1" || orderEvent.FreezeAmount != 500 {
		t.Fatalf("expected freeze metadata to survive lifecycle cancel, got %+v", orderEvent)
	}
	depthEvent, ok := publisher.calls[1].payload.(sharedkafka.QuoteDepthEvent)
	if !ok {
		t.Fatalf("expected second payload to be depth event, got %T", publisher.calls[1].payload)
	}
	if len(depthEvent.Bids) != 0 || len(depthEvent.Asks) != 0 {
		t.Fatalf("expected empty depth after sweep, got %+v", depthEvent)
	}
}
