package service

import (
	"context"
	"encoding/json"
	"testing"

	sharedkafka "funnyoption/internal/shared/kafka"
)

func TestWithdrawalProcessorHandleChainWithdrawal(t *testing.T) {
	journal := NewJournal()
	processor := NewWithdrawalProcessor(journal)

	payload, err := json.Marshal(sharedkafka.ChainWithdrawalQueuedEvent{
		EventID:          "evt_wdq_1",
		WithdrawalID:     "wdq_1",
		UserID:           1001,
		VaultAddress:     "0xvault",
		Asset:            "USDT",
		Amount:           1_250,
		ChainName:        "bsc",
		NetworkName:      "testnet",
		OccurredAtMillis: 1_711_000_100_000,
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if err := processor.HandleChainWithdrawal(context.Background(), sharedkafka.Message{Value: payload}); err != nil {
		t.Fatalf("HandleChainWithdrawal returned error: %v", err)
	}

	entries := journal.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if len(entries[0].Postings) != 2 {
		t.Fatalf("expected 2 postings, got %d", len(entries[0].Postings))
	}
}
