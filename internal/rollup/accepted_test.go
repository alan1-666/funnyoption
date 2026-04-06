package rollup

import (
	"encoding/json"
	"testing"
)

func TestExtractAcceptedWithdrawals(t *testing.T) {
	entryPayload, err := json.Marshal(WithdrawalRequestedPayload{
		WithdrawalID:     "wdq_1",
		AccountID:        1001,
		WalletAddress:    "0x00000000000000000000000000000000000000aa",
		RecipientAddress: "0x00000000000000000000000000000000000000bb",
		VaultAddress:     "0x00000000000000000000000000000000000000cc",
		Asset:            "USDT",
		Amount:           1250,
		Lane:             "SLOW",
		ChainName:        "bsc",
		NetworkName:      "testnet",
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	batchInput, _, err := EncodeBatchInput([]JournalEntry{{
		Sequence:         7,
		EntryType:        EntryTypeWithdrawalRequested,
		SourceType:       SourceTypeChainWithdraw,
		SourceRef:        "wdq_1",
		OccurredAtMillis: 123456789,
		Payload:          entryPayload,
	}})
	if err != nil {
		t.Fatalf("EncodeBatchInput returned error: %v", err)
	}

	withdrawals, err := ExtractAcceptedWithdrawals(StoredBatch{BatchID: 3, InputData: batchInput})
	if err != nil {
		t.Fatalf("ExtractAcceptedWithdrawals returned error: %v", err)
	}
	if len(withdrawals) != 1 {
		t.Fatalf("expected 1 withdrawal, got %d", len(withdrawals))
	}
	item := withdrawals[0]
	if item.BatchID != 3 {
		t.Fatalf("batch_id = %d, want 3", item.BatchID)
	}
	if item.WithdrawalID != "wdq_1" {
		t.Fatalf("withdrawal_id = %s", item.WithdrawalID)
	}
	if item.ClaimStatus != AcceptedWithdrawalStatusClaimable {
		t.Fatalf("claim_status = %s", item.ClaimStatus)
	}
	if item.ClaimID == "" {
		t.Fatalf("claim_id should be populated")
	}
}
