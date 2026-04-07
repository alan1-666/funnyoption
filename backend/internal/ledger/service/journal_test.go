package service

import (
	"errors"
	"testing"

	"funnyoption/internal/ledger/model"
)

func TestJournalAppendAndReplayBalance(t *testing.T) {
	journal := NewJournal()

	entry, err := journal.Append(model.Entry{
		BizType: model.BizTypeDeposit,
		RefID:   "dep_1",
		Postings: []model.Posting{
			{Account: "external:deposit", Asset: "USDT", Direction: model.DirectionDebit, Amount: 10_000},
			{Account: "user:1001:available", Asset: "USDT", Direction: model.DirectionCredit, Amount: 10_000},
		},
	})
	if err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	if entry.EntryID == "" {
		t.Fatalf("expected generated entry id")
	}

	trade, err := journal.Append(model.Entry{
		BizType: model.BizTypeTrade,
		RefID:   "trade_1",
		Postings: []model.Posting{
			{Account: "user:1001:available", Asset: "USDT", Direction: model.DirectionDebit, Amount: 1_500},
			{Account: "market:escrow:100", Asset: "USDT", Direction: model.DirectionCredit, Amount: 1_500},
		},
	})
	if err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	if trade.Status != model.EntryStatusConfirmed {
		t.Fatalf("unexpected entry status: %s", trade.Status)
	}

	balance := journal.BalanceOf("user:1001:available", "USDT")
	if balance != 8_500 {
		t.Fatalf("unexpected replay balance: %d", balance)
	}
}

func TestJournalRejectsUnbalancedEntry(t *testing.T) {
	journal := NewJournal()

	_, err := journal.Append(model.Entry{
		BizType: model.BizTypeFee,
		RefID:   "fee_1",
		Postings: []model.Posting{
			{Account: "user:1001:available", Asset: "USDT", Direction: model.DirectionDebit, Amount: 10},
			{Account: "platform:fee", Asset: "USDT", Direction: model.DirectionCredit, Amount: 9},
		},
	})
	if !errors.Is(err, ErrUnbalancedEntry) {
		t.Fatalf("expected ErrUnbalancedEntry, got %v", err)
	}
}
