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
		Price:       60,
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
		Price:       55,
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
		Price:       60,
		Quantity:    60,
	}
	result, err := eng.PlaceOrder(taker)
	if err != nil {
		t.Fatalf("place taker order: %v", err)
	}
	if len(result.Trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(result.Trades))
	}
	if result.Trades[0].Price != 55 {
		t.Fatalf("expected first trade at best ask 55, got %d", result.Trades[0].Price)
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
		Price:       50,
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
		Price:       50,
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

func TestAsyncEngineCancelOrdersRemovesRestingLiquidity(t *testing.T) {
	async := NewAsync(slog.New(slog.NewTextHandler(io.Discard, nil)), 8)
	if err := async.Restore(3, []*model.Order{
		{
			OrderID:         "resting-yes",
			UserID:          1001,
			MarketID:        21,
			Outcome:         "YES",
			Side:            model.OrderSideBuy,
			Type:            model.OrderTypeLimit,
			TimeInForce:     model.TimeInForceGTC,
			Price:           52,
			Quantity:        15,
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

	cancelled, err := async.CancelOrders(ctx, []*model.Order{{
		OrderID:  "resting-yes",
		MarketID: 21,
		Outcome:  "YES",
		Side:     model.OrderSideBuy,
	}}, model.CancelReasonMarketClosed)
	if err != nil {
		t.Fatalf("cancel orders: %v", err)
	}
	if len(cancelled.Orders) != 1 {
		t.Fatalf("expected one cancelled order, got %d", len(cancelled.Orders))
	}
	if cancelled.Orders[0].Status != model.OrderStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", cancelled.Orders[0].Status)
	}
	if cancelled.Orders[0].CancelReason != model.CancelReasonMarketClosed {
		t.Fatalf("unexpected cancel reason: %s", cancelled.Orders[0].CancelReason)
	}
	if len(cancelled.Books) != 1 {
		t.Fatalf("expected one book snapshot, got %d", len(cancelled.Books))
	}
	if len(cancelled.Books[0].Bids) != 0 || len(cancelled.Books[0].Asks) != 0 {
		t.Fatalf("expected empty book snapshot after cancellation, got %+v", cancelled.Books[0])
	}
	if async.BookCount() != 0 {
		t.Fatalf("expected empty matcher after cancellation, got %d books", async.BookCount())
	}
}

// ---- STP strategy tests ----

func TestSTPCancelTaker(t *testing.T) {
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	eng.PlaceOrder(&model.Order{
		OrderID: "maker-1", UserID: 100, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideSell, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 50, Quantity: 10,
	})

	taker := &model.Order{
		OrderID: "taker-1", UserID: 100, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 50, Quantity: 5,
		STPStrategy: model.STPCancelTaker,
	}
	result, _ := eng.PlaceOrder(taker)

	if taker.Status != model.OrderStatusCancelled {
		t.Fatalf("expected taker cancelled, got %s", taker.Status)
	}
	if taker.CancelReason != model.CancelReasonSTPTaker {
		t.Fatalf("expected STP_CANCEL_TAKER reason, got %s", taker.CancelReason)
	}
	if len(result.Trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(result.Trades))
	}
}

func TestSTPCancelMaker(t *testing.T) {
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	eng.PlaceOrder(&model.Order{
		OrderID: "maker-1", UserID: 100, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideSell, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 50, Quantity: 10,
	})
	// Second maker from a different user at the same price.
	eng.PlaceOrder(&model.Order{
		OrderID: "maker-2", UserID: 200, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideSell, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 50, Quantity: 10,
	})

	taker := &model.Order{
		OrderID: "taker-1", UserID: 100, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 50, Quantity: 5,
		STPStrategy: model.STPCancelMaker,
	}
	result, _ := eng.PlaceOrder(taker)

	// maker-1 (same user) should be cancelled; taker should match against maker-2.
	if taker.Status != model.OrderStatusFilled {
		t.Fatalf("expected taker filled, got %s", taker.Status)
	}
	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}
	if result.Trades[0].MakerOrderID != "maker-2" {
		t.Fatalf("expected trade with maker-2, got %s", result.Trades[0].MakerOrderID)
	}
	// Affected should include cancelled maker-1 AND filled maker-2.
	makerCancelled := false
	for _, a := range result.Affected {
		if a.OrderID == "maker-1" && a.Status == model.OrderStatusCancelled {
			makerCancelled = true
		}
	}
	if !makerCancelled {
		t.Fatal("expected maker-1 to be cancelled by STP")
	}
}

func TestSTPCancelBoth(t *testing.T) {
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	eng.PlaceOrder(&model.Order{
		OrderID: "maker-1", UserID: 100, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideSell, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 50, Quantity: 10,
	})

	taker := &model.Order{
		OrderID: "taker-1", UserID: 100, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 50, Quantity: 5,
		STPStrategy: model.STPCancelBoth,
	}
	result, _ := eng.PlaceOrder(taker)

	if taker.Status != model.OrderStatusCancelled {
		t.Fatalf("expected taker cancelled, got %s", taker.Status)
	}
	if len(result.Trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(result.Trades))
	}
	makerCancelled := false
	for _, a := range result.Affected {
		if a.OrderID == "maker-1" && a.Status == model.OrderStatusCancelled {
			makerCancelled = true
		}
	}
	if !makerCancelled {
		t.Fatal("expected maker-1 to be cancelled by STP_CANCEL_BOTH")
	}
}

func TestSTPNoneAllowsSelfTrade(t *testing.T) {
	// Legacy behavior: STPStrategy="" skips same-user (no trade, no cancel).
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	eng.PlaceOrder(&model.Order{
		OrderID: "maker-1", UserID: 100, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideSell, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 50, Quantity: 10,
	})

	taker := &model.Order{
		OrderID: "taker-1", UserID: 100, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 50, Quantity: 5,
		// STPStrategy is empty — legacy skip behavior.
	}
	result, _ := eng.PlaceOrder(taker)

	// Should skip maker-1 (same user), no trade, taker rests on book.
	if len(result.Trades) != 0 {
		t.Fatalf("expected 0 trades (legacy skip), got %d", len(result.Trades))
	}
	if taker.Status != model.OrderStatusNew {
		t.Fatalf("expected taker resting NEW, got %s", taker.Status)
	}
}

// ---- POST_ONLY tests ----

func TestPostOnlyNoCross(t *testing.T) {
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	// Book is empty — no crossing possible.
	order := &model.Order{
		OrderID: "po-1", UserID: 1, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForcePostOnly, Price: 50, Quantity: 10,
	}
	result, err := eng.PlaceOrder(order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.Status != model.OrderStatusNew {
		t.Fatalf("expected NEW (resting), got %s", order.Status)
	}
	if len(result.Trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(result.Trades))
	}
}

func TestPostOnlyCrossCancelled(t *testing.T) {
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	eng.PlaceOrder(&model.Order{
		OrderID: "maker-1", UserID: 1, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideSell, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 50, Quantity: 10,
	})

	order := &model.Order{
		OrderID: "po-1", UserID: 2, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForcePostOnly, Price: 50, Quantity: 5,
	}
	result, _ := eng.PlaceOrder(order)

	if order.Status != model.OrderStatusCancelled {
		t.Fatalf("expected CANCELLED, got %s", order.Status)
	}
	if order.CancelReason != model.CancelReasonPostOnlyCross {
		t.Fatalf("expected POST_ONLY_CROSS reason, got %s", order.CancelReason)
	}
	if len(result.Trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(result.Trades))
	}
}

// ---- FOK tests ----

func TestFOKFullFill(t *testing.T) {
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	eng.PlaceOrder(&model.Order{
		OrderID: "maker-1", UserID: 1, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideSell, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 50, Quantity: 100,
	})

	order := &model.Order{
		OrderID: "fok-1", UserID: 2, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceFOK, Price: 50, Quantity: 50,
	}
	result, _ := eng.PlaceOrder(order)

	if order.Status != model.OrderStatusFilled {
		t.Fatalf("expected FILLED, got %s", order.Status)
	}
	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}
	if result.Trades[0].Quantity != 50 {
		t.Fatalf("expected trade qty 50, got %d", result.Trades[0].Quantity)
	}
}

func TestFOKNotEnoughLiquidity(t *testing.T) {
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	eng.PlaceOrder(&model.Order{
		OrderID: "maker-1", UserID: 1, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideSell, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 50, Quantity: 10,
	})

	order := &model.Order{
		OrderID: "fok-1", UserID: 2, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceFOK, Price: 50, Quantity: 50,
	}
	result, _ := eng.PlaceOrder(order)

	if order.Status != model.OrderStatusCancelled {
		t.Fatalf("expected CANCELLED, got %s", order.Status)
	}
	if order.CancelReason != model.CancelReasonFOKNotFilled {
		t.Fatalf("expected FOK_NOT_FILLED, got %s", order.CancelReason)
	}
	if len(result.Trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(result.Trades))
	}
}

func TestFOKNoCross(t *testing.T) {
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))

	order := &model.Order{
		OrderID: "fok-1", UserID: 2, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceFOK, Price: 50, Quantity: 10,
	}
	result, _ := eng.PlaceOrder(order)

	if order.Status != model.OrderStatusCancelled {
		t.Fatalf("expected CANCELLED (no cross), got %s", order.Status)
	}
	if len(result.Trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(result.Trades))
	}
}

func TestFOKWithSTPCancelTakerRejects(t *testing.T) {
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	eng.PlaceOrder(&model.Order{
		OrderID: "maker-1", UserID: 100, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideSell, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 50, Quantity: 100,
	})

	// FOK from same user with STP_CANCEL_TAKER — can't fill because taker
	// would be cancelled on STP encounter.
	order := &model.Order{
		OrderID: "fok-1", UserID: 100, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceFOK, Price: 50, Quantity: 10,
		STPStrategy: model.STPCancelTaker,
	}
	result, _ := eng.PlaceOrder(order)

	if order.Status != model.OrderStatusCancelled {
		t.Fatalf("expected CANCELLED, got %s", order.Status)
	}
	if len(result.Trades) != 0 {
		t.Fatalf("expected 0 trades, got %d", len(result.Trades))
	}
}

func TestFOKDoesNotModifyBookOnReject(t *testing.T) {
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	eng.PlaceOrder(&model.Order{
		OrderID: "maker-1", UserID: 1, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideSell, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 50, Quantity: 10,
	})

	// FOK wants 50 but only 10 available — should be rejected without modifying maker.
	eng.PlaceOrder(&model.Order{
		OrderID: "fok-fail", UserID: 2, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceFOK, Price: 50, Quantity: 50,
	})

	// Now a regular IOC should still match maker-1's full 10.
	ioc := &model.Order{
		OrderID: "ioc-1", UserID: 3, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceIOC, Price: 50, Quantity: 10,
	}
	result, _ := eng.PlaceOrder(ioc)
	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}
	if result.Trades[0].Quantity != 10 {
		t.Fatalf("maker should still have full qty 10, traded %d", result.Trades[0].Quantity)
	}
}

// ---- AmendOrder tests ----

func TestAmendOrderChangePrice(t *testing.T) {
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	eng.PlaceOrder(&model.Order{
		OrderID: "resting-1", UserID: 1, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 40, Quantity: 10,
	})

	original := &model.Order{
		OrderID: "resting-1", MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Price: 40,
	}
	result, err := eng.AmendOrder(original, 45, 0)
	if err != nil {
		t.Fatalf("amend failed: %v", err)
	}

	// First affected should be the cancelled original.
	if result.Affected[0].OrderID != "resting-1" {
		t.Fatalf("expected cancelled original in affected, got %s", result.Affected[0].OrderID)
	}
	if result.Affected[0].Status != model.OrderStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", result.Affected[0].Status)
	}
	if result.Affected[0].CancelReason != model.CancelReasonAmended {
		t.Fatalf("expected AMENDED reason, got %s", result.Affected[0].CancelReason)
	}
	// Result order should be the new resting order at new price.
	if result.Order.Price != 45 {
		t.Fatalf("expected amended price 45, got %d", result.Order.Price)
	}
	if result.Order.Status != model.OrderStatusNew {
		t.Fatalf("expected NEW, got %s", result.Order.Status)
	}
}

func TestAmendOrderTriggersMatch(t *testing.T) {
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	// Resting ask at 50.
	eng.PlaceOrder(&model.Order{
		OrderID: "ask-1", UserID: 1, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideSell, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 50, Quantity: 10,
	})
	// Resting bid at 40 (no cross).
	eng.PlaceOrder(&model.Order{
		OrderID: "bid-1", UserID: 2, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 40, Quantity: 5,
	})

	// Amend bid from 40 to 50 — should now cross and match.
	original := &model.Order{
		OrderID: "bid-1", MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Price: 40,
	}
	result, err := eng.AmendOrder(original, 50, 0)
	if err != nil {
		t.Fatalf("amend failed: %v", err)
	}
	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade after amend, got %d", len(result.Trades))
	}
	if result.Trades[0].MakerOrderID != "ask-1" {
		t.Fatalf("expected match against ask-1, got %s", result.Trades[0].MakerOrderID)
	}
}

func TestAmendOrderNotFound(t *testing.T) {
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	_, err := eng.AmendOrder(&model.Order{
		OrderID: "nonexistent", MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Price: 50,
	}, 55, 0)
	if err == nil {
		t.Fatal("expected error for non-existent order")
	}
}

func TestTradeSliceDoesNotAliasReusableBuffer(t *testing.T) {
	eng := New(slog.New(slog.NewTextHandler(io.Discard, nil)))

	eng.PlaceOrder(&model.Order{
		OrderID: "maker-1", UserID: 1, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideSell, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC, Price: 5000, Quantity: 1_000_000,
	})

	r1, err := eng.PlaceOrder(&model.Order{
		OrderID: "taker-1", UserID: 2, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceIOC, Price: 5000, Quantity: 1,
	})
	if err != nil {
		t.Fatalf("taker-1 failed: %v", err)
	}
	if len(r1.Trades) != 1 || r1.Trades[0].TakerOrderID != "taker-1" {
		t.Fatalf("expected 1 trade with TakerOrderID=taker-1, got %+v", r1.Trades)
	}

	r2, err := eng.PlaceOrder(&model.Order{
		OrderID: "taker-2", UserID: 2, MarketID: 1, Outcome: "YES",
		Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
		TimeInForce: model.TimeInForceIOC, Price: 5000, Quantity: 1,
	})
	if err != nil {
		t.Fatalf("taker-2 failed: %v", err)
	}
	if len(r2.Trades) != 1 || r2.Trades[0].TakerOrderID != "taker-2" {
		t.Fatalf("expected 1 trade with TakerOrderID=taker-2, got %+v", r2.Trades)
	}

	if r1.Trades[0].TakerOrderID != "taker-1" {
		t.Fatalf("r1 trade was corrupted by subsequent PlaceOrder: TakerOrderID=%q, want taker-1", r1.Trades[0].TakerOrderID)
	}
	if r1.Order.OrderID != "taker-1" {
		t.Fatalf("r1 order was corrupted: OrderID=%q, want taker-1", r1.Order.OrderID)
	}
}
