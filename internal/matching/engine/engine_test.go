package engine

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"funnyoption/internal/matching/model"
)

func TestEnginePriceTimePriority(t *testing.T) {
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))

	makerOne := &model.Order{
		OrderID:     "maker-1",
		UserID:      101,
		MarketID:    1,
		Outcome:     "YES",
		Side:        model.OrderSideSell,
		Type:        model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC,
		Price:       120,
		Quantity:    50,
	}
	makerTwo := &model.Order{
		OrderID:     "maker-2",
		UserID:      102,
		MarketID:    1,
		Outcome:     "YES",
		Side:        model.OrderSideSell,
		Type:        model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC,
		Price:       110,
		Quantity:    30,
	}
	_, _ = eng.PlaceOrder(makerOne)
	_, _ = eng.PlaceOrder(makerTwo)

	taker := &model.Order{
		OrderID:     "taker-1",
		UserID:      201,
		MarketID:    1,
		Outcome:     "YES",
		Side:        model.OrderSideBuy,
		Type:        model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC,
		Price:       120,
		Quantity:    60,
	}
	result, err := eng.PlaceOrder(taker)
	if err != nil {
		t.Fatalf("place taker order: %v", err)
	}
	if len(result.Trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(result.Trades))
	}
	if result.Trades[0].Price != 110 {
		t.Fatalf("expected first trade at best ask 110, got %d", result.Trades[0].Price)
	}
	if result.Trades[0].MakerOrderID != "maker-2" {
		t.Fatalf("expected maker-2 to match first, got %s", result.Trades[0].MakerOrderID)
	}
	if result.Trades[1].MakerOrderID != "maker-1" {
		t.Fatalf("expected maker-1 to match second, got %s", result.Trades[1].MakerOrderID)
	}
	if taker.Status != model.OrderStatusFilled {
		t.Fatalf("expected taker filled, got %s", taker.Status)
	}
}

func TestEngineIOCResidualCancelled(t *testing.T) {
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	_, _ = eng.PlaceOrder(&model.Order{
		OrderID:     "maker-1",
		UserID:      101,
		MarketID:    2,
		Outcome:     "NO",
		Side:        model.OrderSideSell,
		Type:        model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC,
		Price:       100,
		Quantity:    20,
	})

	order := &model.Order{
		OrderID:     "ioc-1",
		UserID:      201,
		MarketID:    2,
		Outcome:     "NO",
		Side:        model.OrderSideBuy,
		Type:        model.OrderTypeLimit,
		TimeInForce: model.TimeInForceIOC,
		Price:       100,
		Quantity:    50,
	}
	result, err := eng.PlaceOrder(order)
	if err != nil {
		t.Fatalf("place ioc order: %v", err)
	}
	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}
	if order.Status != model.OrderStatusCancelled {
		t.Fatalf("expected cancelled residual, got %s", order.Status)
	}
	if order.CancelReason != model.CancelReasonIOCPartialFill {
		t.Fatalf("unexpected cancel reason: %s", order.CancelReason)
	}
}

func TestAsyncEngineSubmit(t *testing.T) {
	async := NewAsync(slog.New(slog.NewTextHandler(io.Discard, nil)), 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	async.Start(ctx)

	result, err := async.Submit(ctx, &model.Order{
		OrderID:     "async-1",
		UserID:      1,
		MarketID:    3,
		Outcome:     "YES",
		Side:        model.OrderSideBuy,
		Type:        model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC,
		Price:       99,
		Quantity:    10,
	})
	if err != nil {
		t.Fatalf("submit order: %v", err)
	}
	if result.Order.Status != model.OrderStatusNew {
		t.Fatalf("expected resting order status NEW, got %s", result.Order.Status)
	}
}

func TestAsyncEngineRestoreSeedsSequenceAndRestingBook(t *testing.T) {
	async := NewAsync(slog.New(slog.NewTextHandler(io.Discard, nil)), 8)
	if err := async.Restore(7, []*model.Order{
		{
			OrderID:         "maker-resting",
			UserID:          101,
			MarketID:        11,
			Outcome:         "YES",
			Side:            model.OrderSideBuy,
			Type:            model.OrderTypeLimit,
			TimeInForce:     model.TimeInForceGTC,
			Price:           64,
			Quantity:        20,
			Status:          model.OrderStatusNew,
			CreatedAtMillis: 1000,
			UpdatedAtMillis: 1000,
		},
	}); err != nil {
		t.Fatalf("restore state: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	async.Start(ctx)

	result, err := async.Submit(ctx, &model.Order{
		OrderID:         "taker-cross",
		UserID:          202,
		MarketID:        11,
		Outcome:         "YES",
		Side:            model.OrderSideSell,
		Type:            model.OrderTypeLimit,
		TimeInForce:     model.TimeInForceGTC,
		Price:           64,
		Quantity:        5,
		CreatedAtMillis: 2000,
		UpdatedAtMillis: 2000,
	})
	if err != nil {
		t.Fatalf("submit order after restore: %v", err)
	}
	if async.BookCount() != 1 {
		t.Fatalf("expected one restored book, got %d", async.BookCount())
	}
	if len(result.Trades) != 1 {
		t.Fatalf("expected one trade against restored maker, got %d", len(result.Trades))
	}
	if result.Trades[0].Sequence != 8 {
		t.Fatalf("expected restored sequence to continue at 8, got %d", result.Trades[0].Sequence)
	}
	if result.Trades[0].MakerOrderID != "maker-resting" {
		t.Fatalf("expected restored maker to match, got %s", result.Trades[0].MakerOrderID)
	}
}
