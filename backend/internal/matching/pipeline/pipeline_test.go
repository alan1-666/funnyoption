package pipeline

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	"funnyoption/internal/shared/fee"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type capturePersistStore struct {
	results []engine.Result
}

func (s *capturePersistStore) PersistResult(_ context.Context, _ sharedkafka.OrderCommand, result engine.Result) error {
	s.results = append(s.results, result)
	return nil
}

type nullCandle struct{}

func (n *nullCandle) ApplyTrade(trade model.Trade) sharedkafka.QuoteCandleEvent {
	return sharedkafka.QuoteCandleEvent{}
}

type constEpoch uint64

func (c constEpoch) Current() uint64 { return uint64(c) }

func TestBookEngineProcessesPlaceCommand(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	outputCh := make(chan MatchResult, 64)
	var seq uint64
	be := NewBookEngine("1:YES", logger, &seq, outputCh, constEpoch(1))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go be.Run(ctx)

	makerCmd := MatchCommand{
		Action:          ActionPlace,
		UserID:          1001,
		MarketID:        1,
		Outcome:         "YES",
		BookKey:         model.BuildBookKey(1, "YES"),
		Side:            SideSell,
		Type:            TypeLimit,
		TimeInForce:     TIFGTC,
		Price:           55,
		Quantity:        10,
		OrderID:         "ord_maker_1",
		CollateralAsset: "USDT",
	}
	be.TryPublish(makerCmd)

	time.Sleep(50 * time.Millisecond)

	takerCmd := MatchCommand{
		Action:          ActionPlace,
		UserID:          1002,
		MarketID:        1,
		Outcome:         "YES",
		BookKey:         model.BuildBookKey(1, "YES"),
		Side:            SideBuy,
		Type:            TypeLimit,
		TimeInForce:     TIFGTC,
		Price:           55,
		Quantity:        5,
		OrderID:         "ord_taker_1",
		CollateralAsset: "USDT",
	}
	be.TryPublish(takerCmd)

	deadline := time.After(2 * time.Second)
	results := make([]MatchResult, 0)
	for len(results) < 2 {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for results, got %d", len(results))
		case r := <-outputCh:
			results = append(results, r)
		}
	}

	if results[0].Command.OrderID != "ord_maker_1" {
		t.Fatalf("expected first result to be maker, got %s", results[0].Command.OrderID)
	}
	if results[0].Result.Order == nil {
		t.Fatal("expected maker order in result")
	}

	takerResult := results[1]
	if takerResult.Result.Order == nil {
		t.Fatal("expected taker order in result")
	}
	if takerResult.Command.OrderID != "ord_taker_1" {
		t.Fatalf("expected taker order id, got %s", takerResult.Command.OrderID)
	}
	if len(takerResult.Result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(takerResult.Result.Trades))
	}
	trade := takerResult.Result.Trades[0]
	if trade.Price != 55 || trade.Quantity != 5 {
		t.Fatalf("unexpected trade: price=%d qty=%d", trade.Price, trade.Quantity)
	}
	if trade.TakerOrderID != "ord_taker_1" || trade.MakerOrderID != "ord_maker_1" {
		t.Fatalf("unexpected trade participants: taker=%s maker=%s", trade.TakerOrderID, trade.MakerOrderID)
	}
}

func TestSupervisorRoutesToCorrectBook(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	sv := NewBookSupervisor(logger, constEpoch(1))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sv.Start(ctx)

	// Place ask on book "1:YES"
	sv.Route(MatchCommand{
		Action:   ActionPlace,
		UserID:   1,
		MarketID: 1,
		Outcome:  "YES",
		BookKey:  "1:YES",
		Side:     SideSell,
		Type:     TypeLimit,
		TimeInForce: TIFGTC,
		Price:    60,
		Quantity: 10,
		OrderID:  "ask-1",
	})

	// Place ask on book "2:NO"
	sv.Route(MatchCommand{
		Action:   ActionPlace,
		UserID:   2,
		MarketID: 2,
		Outcome:  "NO",
		BookKey:  "2:NO",
		Side:     SideSell,
		Type:     TypeLimit,
		TimeInForce: TIFGTC,
		Price:    40,
		Quantity: 5,
		OrderID:  "ask-2",
	})

	time.Sleep(50 * time.Millisecond)

	if sv.BookCount() != 2 {
		t.Fatalf("expected 2 books, got %d", sv.BookCount())
	}

	// Cross on book "1:YES" — should match against ask-1
	sv.Route(MatchCommand{
		Action:   ActionPlace,
		UserID:   3,
		MarketID: 1,
		Outcome:  "YES",
		BookKey:  "1:YES",
		Side:     SideBuy,
		Type:     TypeLimit,
		TimeInForce: TIFGTC,
		Price:    60,
		Quantity: 5,
		OrderID:  "bid-1",
	})

	deadline := time.After(2 * time.Second)
	var tradeResult *MatchResult
	collected := 0
	for collected < 3 {
		select {
		case <-deadline:
			t.Fatalf("timed out, collected %d results", collected)
		case r := <-sv.OutputCh():
			collected++
			if len(r.Result.Trades) > 0 {
				tradeResult = &r
			}
		}
	}

	if tradeResult == nil {
		t.Fatal("expected a trade result")
	}
	if tradeResult.Result.Trades[0].MakerOrderID != "ask-1" {
		t.Fatalf("expected trade with ask-1, got %s", tradeResult.Result.Trades[0].MakerOrderID)
	}
}

func TestSupervisorSubmitCancel(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	sv := NewBookSupervisor(logger, constEpoch(1))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Restore a resting order
	sv.Restore(0, []*model.Order{{
		OrderID:     "resting-1",
		UserID:      100,
		MarketID:    1,
		Outcome:     "YES",
		Side:        model.OrderSideBuy,
		Type:        model.OrderTypeLimit,
		TimeInForce: model.TimeInForceGTC,
		Price:       50,
		Quantity:    10,
		Status:      model.OrderStatusNew,
	}})

	sv.Start(ctx)

	// Submit a cancel
	ok := sv.SubmitCancel(MatchCommand{
		OrderID:  "resting-1",
		MarketID: 1,
		Outcome:  "YES",
		BookKey:  "1:YES",
		Side:     SideBuy,
		Price:    50,
		CancelReason: CancelReasonMarketClosed,
	})
	if !ok {
		t.Fatal("SubmitCancel returned false")
	}

	deadline := time.After(2 * time.Second)
	gotCancel := false
	for !gotCancel {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for cancel result")
		case r := <-sv.OutputCh():
			if r.Result.Order != nil && r.Result.Order.Status == model.OrderStatusCancelled {
				gotCancel = true
			}
		}
	}
}

func TestDeterministicTradeIDInBookEngine(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	outputCh := make(chan MatchResult, 64)
	var seq uint64
	be := NewBookEngine("1:YES", logger, &seq, outputCh, constEpoch(5))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go be.Run(ctx)

	// Maker: sell at 55
	be.TryPublish(MatchCommand{
		Action: ActionPlace, UserID: 1001, MarketID: 1, Outcome: "YES",
		BookKey: "1:YES", Side: SideSell, Type: TypeLimit, TimeInForce: TIFGTC,
		Price: 55, Quantity: 10, OrderID: "m1",
	})
	time.Sleep(30 * time.Millisecond)

	// Taker: buy at 55
	be.TryPublish(MatchCommand{
		Action: ActionPlace, UserID: 1002, MarketID: 1, Outcome: "YES",
		BookKey: "1:YES", Side: SideBuy, Type: TypeLimit, TimeInForce: TIFGTC,
		Price: 55, Quantity: 3, OrderID: "t1",
	})

	deadline := time.After(2 * time.Second)
	var tradeResult *MatchResult
	for tradeResult == nil {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for trade")
		case r := <-outputCh:
			if len(r.Result.Trades) > 0 {
				tradeResult = &r
			}
		}
	}

	trade := tradeResult.Result.Trades[0]
	if trade.TradeID == "" {
		t.Fatal("expected deterministic trade ID")
	}
	if trade.TradeID != "1:YES:00000001" {
		t.Fatalf("unexpected trade ID: %s (expected 1:YES:00000001)", trade.TradeID)
	}
	if trade.EpochID != 5 {
		t.Fatalf("expected epoch 5, got %d", trade.EpochID)
	}
	if tradeResult.EpochID != 5 {
		t.Fatalf("expected result epoch 5, got %d", tradeResult.EpochID)
	}
}

func TestShadowModeDispatcher(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	outputCh := make(chan MatchResult, 64)
	store := &capturePersistStore{}
	d := NewOutputDispatcher(
		logger, outputCh, nil, sharedkafka.Topics{}, store, &nullCandle{},
		fee.Schedule{},
	)
	d.SetMode(DispatchModeShadow)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go d.Run(ctx)

	outputCh <- MatchResult{
		Command: MatchCommand{OrderID: "ord-1", MarketID: 1, Outcome: "YES", BookKey: "1:YES"},
		Result: engine.Result{
			Order: &model.Order{OrderID: "ord-1", MarketID: 1, Outcome: "YES", Status: model.OrderStatusNew},
		},
	}

	time.Sleep(50 * time.Millisecond)

	if d.ShadowedCount() != 1 {
		t.Fatalf("expected 1 shadowed, got %d", d.ShadowedCount())
	}
	dispatched, _ := d.Stats()
	if dispatched != 0 {
		t.Fatalf("expected 0 dispatched in shadow mode, got %d", dispatched)
	}
	if len(store.results) != 0 {
		t.Fatalf("expected no persistence in shadow mode, got %d", len(store.results))
	}
}

func TestSupervisorSnapshot(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	sv := NewBookSupervisor(logger, constEpoch(3))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sv.Restore(100, []*model.Order{
		{
			OrderID: "resting-1", UserID: 100, MarketID: 1, Outcome: "YES",
			Side: model.OrderSideBuy, Type: model.OrderTypeLimit,
			TimeInForce: model.TimeInForceGTC, Price: 50, Quantity: 10,
			Status: model.OrderStatusNew,
		},
		{
			OrderID: "resting-2", UserID: 200, MarketID: 2, Outcome: "NO",
			Side: model.OrderSideSell, Type: model.OrderTypeLimit,
			TimeInForce: model.TimeInForceGTC, Price: 40, Quantity: 5,
			Status: model.OrderStatusNew,
		},
	})

	sv.Start(ctx)

	snap := sv.TakeSnapshot()

	if snap.GlobalSequence != 100 {
		t.Fatalf("expected global seq 100, got %d", snap.GlobalSequence)
	}
	if len(snap.Books) != 2 {
		t.Fatalf("expected 2 books in snapshot, got %d", len(snap.Books))
	}

	totalOrders := 0
	for _, book := range snap.Books {
		totalOrders += len(book.Orders)
	}
	if totalOrders != 2 {
		t.Fatalf("expected 2 total orders, got %d", totalOrders)
	}
}

func TestProtocolRoundTrip(t *testing.T) {
	original := sharedkafka.OrderCommand{
		CommandID:         "cmd_1",
		TraceID:           "trace_1",
		OrderID:           "ord_1",
		ClientOrderID:     "client_1",
		FreezeID:          "frz_1",
		FreezeAsset:       "USDT",
		FreezeAmount:      550,
		CollateralAsset:   "USDT",
		UserID:            1001,
		MarketID:          42,
		Outcome:           "YES",
		Side:              "BUY",
		Type:              "LIMIT",
		TimeInForce:       "GTC",
		Price:             55,
		Quantity:          10,
		RequestedAtMillis: 1234567890,
	}

	mc := CommandFromKafka(original)
	roundtripped := mc.ToKafkaCommand()

	if roundtripped.CommandID != original.CommandID {
		t.Errorf("CommandID mismatch: %s != %s", roundtripped.CommandID, original.CommandID)
	}
	if roundtripped.OrderID != original.OrderID {
		t.Errorf("OrderID mismatch")
	}
	if roundtripped.FreezeID != original.FreezeID {
		t.Errorf("FreezeID mismatch")
	}
	if roundtripped.FreezeAmount != original.FreezeAmount {
		t.Errorf("FreezeAmount mismatch")
	}
	if roundtripped.Price != original.Price {
		t.Errorf("Price mismatch: %d != %d", roundtripped.Price, original.Price)
	}
	if roundtripped.Side != original.Side {
		t.Errorf("Side mismatch: %s != %s", roundtripped.Side, original.Side)
	}
}
