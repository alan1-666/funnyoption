package service

import (
	"context"
	"encoding/json"
	"testing"

	sharedkafka "funnyoption/internal/shared/kafka"
)

func TestSettlementProcessorHandleSettlementCompleted(t *testing.T) {
	journal := NewJournal()
	processor := NewSettlementProcessor(journal)

	payload, err := json.Marshal(sharedkafka.SettlementCompletedEvent{
		EventID:         "evt_settle_1",
		MarketID:        1001,
		UserID:          2001,
		WinningOutcome:  "YES",
		PositionAsset:   "POSITION:1001:YES",
		SettledQuantity: 10,
		PayoutAsset:     "USDT",
		PayoutAmount:    10,
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if err := processor.HandleSettlementCompleted(context.Background(), sharedkafka.Message{Value: payload}); err != nil {
		t.Fatalf("HandleSettlementCompleted returned error: %v", err)
	}

	if got := journal.BalanceOf("user:2001:position:1001:yes", "POSITION:1001:YES"); got != -10 {
		t.Fatalf("unexpected position settlement delta: %d", got)
	}
	if got := journal.BalanceOf("market:1001:resolved:yes", "POSITION:1001:YES"); got != 10 {
		t.Fatalf("unexpected resolved pool delta: %d", got)
	}
	if got := journal.BalanceOf("market:1001:treasury", "USDT"); got != -10 {
		t.Fatalf("unexpected treasury payout delta: %d", got)
	}
	if got := journal.BalanceOf("user:2001:available", "USDT"); got != 10 {
		t.Fatalf("unexpected user payout delta: %d", got)
	}
}
