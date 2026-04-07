package pipeline

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	"funnyoption/internal/matching/ringbuffer"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type stubTradableChecker struct{}

func (s *stubTradableChecker) MarketIsTradable(_ context.Context, _ int64) (bool, error) {
	return true, nil
}

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

func TestMatchLoopProcessesPlaceCommand(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := engine.New(logger)

	inputRB := ringbuffer.New[MatchCommand](64)
	outputRB := ringbuffer.New[MatchResult](64)

	loop := NewMatchLoop(logger, eng, inputRB, outputRB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go loop.Run(ctx)

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
	inputRB.TryPublish(makerCmd)

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
	inputRB.TryPublish(takerCmd)

	deadline := time.After(2 * time.Second)
	results := make([]MatchResult, 0)
	for len(results) < 2 {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for results, got %d", len(results))
		default:
		}
		if r, ok := outputRB.TryConsume(); ok {
			results = append(results, r)
		} else {
			time.Sleep(10 * time.Millisecond)
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
