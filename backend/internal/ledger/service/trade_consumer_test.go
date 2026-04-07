package service

import (
	"context"
	"encoding/json"
	"testing"

	sharedkafka "funnyoption/internal/shared/kafka"
)

func TestTradeProcessorHandleTradeMatched(t *testing.T) {
	journal := NewJournal()
	processor := NewTradeProcessor(journal)

	payload, err := json.Marshal(sharedkafka.TradeMatchedEvent{
		EventID:         "evt_trade_1",
		CollateralAsset: "USDT",
		MarketID:        1001,
		Outcome:         "YES",
		BookKey:         "1001:YES",
		Price:           65,
		Quantity:        10,
		TakerOrderID:    "ord_taker",
		MakerOrderID:    "ord_maker",
		TakerUserID:     2001,
		MakerUserID:     2002,
		TakerSide:       "BUY",
		MakerSide:       "SELL",
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if err := processor.HandleTradeMatched(context.Background(), sharedkafka.Message{Value: payload}); err != nil {
		t.Fatalf("HandleTradeMatched returned error: %v", err)
	}

	if got := journal.BalanceOf("user:2001:available", "USDT"); got != -650 {
		t.Fatalf("unexpected taker balance delta: %d", got)
	}
	if got := journal.BalanceOf("user:2002:available", "USDT"); got != 650 {
		t.Fatalf("unexpected maker balance delta: %d", got)
	}
	if got := journal.BalanceOf("user:2001:position:1001:yes", "POSITION:1001:YES"); got != 10 {
		t.Fatalf("unexpected buyer position delta: %d", got)
	}
	if got := journal.BalanceOf("user:2002:position:1001:yes", "POSITION:1001:YES"); got != -10 {
		t.Fatalf("unexpected seller position delta: %d", got)
	}
}
