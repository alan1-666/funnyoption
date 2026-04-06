package rollup

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

const (
	AcceptedWithdrawalStatusClaimable      = "CLAIMABLE"
	AcceptedWithdrawalStatusClaimSubmitted = "CLAIM_SUBMITTED"
	AcceptedWithdrawalStatusClaimed        = "CLAIMED"
	AcceptedWithdrawalStatusClaimFailed    = "FAILED"
)

func ExtractAcceptedWithdrawals(batch StoredBatch) ([]AcceptedWithdrawalRecord, error) {
	input, err := DecodeBatchInput(batch.InputData)
	if err != nil {
		return nil, err
	}

	byWithdrawalID := make(map[string]AcceptedWithdrawalRecord)
	order := make([]string, 0)
	for _, entry := range input.Entries {
		if entry.EntryType != EntryTypeWithdrawalRequested {
			continue
		}
		var payload WithdrawalRequestedPayload
		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			return nil, fmt.Errorf("decode withdrawal payload: %w", err)
		}
		withdrawalID := normalizeText(payload.WithdrawalID)
		if withdrawalID == "" {
			return nil, fmt.Errorf("accepted withdrawal is missing withdrawal_id")
		}
		record := AcceptedWithdrawalRecord{
			WithdrawalID:     withdrawalID,
			BatchID:          batch.BatchID,
			AccountID:        payload.AccountID,
			WalletAddress:    normalizeText(payload.WalletAddress),
			RecipientAddress: normalizeText(payload.RecipientAddress),
			VaultAddress:     normalizeText(payload.VaultAddress),
			Asset:            normalizeAsset(payload.Asset),
			Amount:           payload.Amount,
			Lane:             normalizeText(payload.Lane),
			ChainName:        normalizeText(payload.ChainName),
			NetworkName:      normalizeText(payload.NetworkName),
			RequestSequence:  entry.Sequence,
			ClaimID:          acceptedWithdrawalClaimID(withdrawalID),
			ClaimStatus:      AcceptedWithdrawalStatusClaimable,
		}
		if _, exists := byWithdrawalID[withdrawalID]; !exists {
			order = append(order, withdrawalID)
		}
		byWithdrawalID[withdrawalID] = record
	}

	result := make([]AcceptedWithdrawalRecord, 0, len(order))
	for _, withdrawalID := range order {
		result = append(result, byWithdrawalID[withdrawalID])
	}
	return result, nil
}

func acceptedWithdrawalClaimID(withdrawalID string) string {
	sum := crypto.Keccak256([]byte(strings.TrimSpace(withdrawalID)))
	return "0x" + hex.EncodeToString(sum)
}
