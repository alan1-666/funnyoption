package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	"funnyoption/internal/matching/pipeline"
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

type captureCancelSubmitter struct {
	commands []pipeline.MatchCommand
}

func (c *captureCancelSubmitter) SubmitCancel(cmd pipeline.MatchCommand) bool {
	c.commands = append(c.commands, cmd)
	return true
}

type capturePublisherMulti struct {
	calls []struct {
		topic   string
		key     string
		payload any
	}
}

func (p *capturePublisherMulti) PublishJSON(_ context.Context, topic, key string, payload any) error {
	p.calls = append(p.calls, struct {
		topic   string
		key     string
		payload any
	}{topic: topic, key: key, payload: payload})
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

func TestOrderExpirySweeperSubmitsCancelCommands(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

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

	submitter := &captureCancelSubmitter{}
	publisher := &capturePublisherMulti{}
	sweeper := newOrderExpirySweeper(logger, submitter, store, publisher, sharedkafka.NewTopics("funnyoption."))

	ctx := context.Background()
	if err := sweeper.sweepOnce(ctx, time.Unix(123, 0)); err != nil {
		t.Fatalf("sweepOnce returned error: %v", err)
	}

	if len(submitter.commands) != 1 {
		t.Fatalf("expected 1 cancel command submitted, got %d", len(submitter.commands))
	}

	cmd := submitter.commands[0]
	if cmd.Action != pipeline.ActionCancel {
		t.Fatalf("expected ActionCancel, got %d", cmd.Action)
	}
	if cmd.OrderID != "ord_close_1" {
		t.Fatalf("expected order_id ord_close_1, got %s", cmd.OrderID)
	}
	if cmd.MarketID != 77 {
		t.Fatalf("expected market_id 77, got %d", cmd.MarketID)
	}
	if cmd.CancelReason != pipeline.CancelReasonMarketClosed {
		t.Fatalf("expected CancelReasonMarketClosed, got %d", cmd.CancelReason)
	}
	if cmd.FreezeID != "frz_close_1" {
		t.Fatalf("expected freeze metadata, got %s", cmd.FreezeID)
	}
}
