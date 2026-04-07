package rollup

import (
	"encoding/json"
	"fmt"
	"sort"
)

func BuildStateTransitionWitnessMaterial(history []StoredBatch, batch StoredBatch) (StateTransitionWitnessMaterial, error) {
	shadowContract, err := BuildShadowBatchContract(batch)
	if err != nil {
		return StateTransitionWitnessMaterial{}, err
	}

	replayBatches := make([]StoredBatch, 0, len(history)+1)
	replayBatches = append(replayBatches, history...)
	replayBatches = append(replayBatches, batch)

	snapshot, err := buildTransitionAcceptedReplaySnapshot(replayBatches)
	if err != nil {
		return StateTransitionWitnessMaterial{}, err
	}
	walletByAccount, err := buildWitnessWalletMirror(replayBatches)
	if err != nil {
		return StateTransitionWitnessMaterial{}, err
	}
	escapeRoot, escapeLeaves, err := BuildAcceptedEscapeCollateralSnapshot(
		snapshot.BatchID,
		defaultStateRoot(batch.StateRoot),
		snapshot.Balances,
		walletByAccount,
	)
	if err != nil {
		return StateTransitionWitnessMaterial{}, err
	}
	withdrawals, err := ExtractAcceptedWithdrawals(batch)
	if err != nil {
		return StateTransitionWitnessMaterial{}, err
	}
	withdrawalRoot, withdrawalLeaves, err := BuildAcceptedWithdrawalMerkleTree(batch.BatchID, withdrawals)
	if err != nil {
		return StateTransitionWitnessMaterial{}, err
	}

	return StateTransitionWitnessMaterial{
		EntrySetHash:                 hashJournalEntries(shadowContract.Witness.Entries),
		AcceptedBalancesHash:         hashAcceptedBalances(snapshot.Balances),
		AcceptedPositionsHash:        hashAcceptedPositions(snapshot.Positions),
		AcceptedPayoutsHash:          hashAcceptedPayouts(snapshot.Payouts),
		AcceptedWithdrawalRootHash:   hashAcceptedWithdrawalRoot(withdrawalRoot),
		AcceptedWithdrawalLeavesHash: hashAcceptedWithdrawalLeaves(withdrawalLeaves),
		EscapeCollateralRootHash:     hashAcceptedEscapeRoot(escapeRoot),
		EscapeCollateralLeavesHash:   hashAcceptedEscapeLeaves(escapeLeaves),
	}, nil
}

func BuildDeterministicStateTransitionWitnessMaterial(context SolidityVerifierGateContext) StateTransitionWitnessMaterial {
	return StateTransitionWitnessMaterial{
		EntrySetHash:                 hashStrings("shadow", "transition", "placeholder", "entries", context.PublicInputs.BatchDataHash),
		AcceptedBalancesHash:         hashStrings("shadow", "transition", "placeholder", "balances", context.PublicInputs.BalancesRoot),
		AcceptedPositionsHash:        hashStrings("shadow", "transition", "placeholder", "positions", context.PublicInputs.PositionsFundingRoot),
		AcceptedPayoutsHash:          hashStrings("shadow", "transition", "placeholder", "payouts", context.PublicInputs.NextStateRoot),
		AcceptedWithdrawalRootHash:   hashStrings("shadow", "transition", "placeholder", "withdrawal_root", context.PublicInputs.WithdrawalsRoot),
		AcceptedWithdrawalLeavesHash: hashStrings("shadow", "transition", "placeholder", "withdrawal_leaves", context.PublicInputs.WithdrawalsRoot, context.VerifierGateHash),
		EscapeCollateralRootHash:     hashStrings("shadow", "transition", "placeholder", "escape_root", context.PublicInputs.BalancesRoot, context.PublicInputs.NextStateRoot),
		EscapeCollateralLeavesHash:   hashStrings("shadow", "transition", "placeholder", "escape_leaves", context.AuthProofHash, context.VerifierGateHash),
	}
}

func buildTransitionAcceptedReplaySnapshot(batches []StoredBatch) (AcceptedReplaySnapshot, error) {
	state := shadowState{
		balances:      make(map[string]balanceLeaf),
		nonces:        make(map[string]nonceLeaf),
		openOrders:    make(map[string]orderLeaf),
		positions:     make(map[string]positionLeaf),
		marketFunding: make(map[string]marketFundingLeaf),
		withdrawals:   make(map[string]withdrawalLeaf),
	}

	payoutOrder := make([]string, 0)
	payoutByEventID := make(map[string]AcceptedPayoutRecord)
	settledByPosition := make(map[string]settledPositionAccumulator)

	for _, batch := range batches {
		input, err := DecodeBatchInput(batch.InputData)
		if err != nil {
			return AcceptedReplaySnapshot{}, err
		}
		for _, entry := range input.Entries {
			if entry.EntryType == EntryTypeSettlementPayout {
				payload, payoutRecord, err := decodeAcceptedSettlementPayout(batch.BatchID, entry.Payload)
				if err != nil {
					return AcceptedReplaySnapshot{}, err
				}
				if _, exists := payoutByEventID[payoutRecord.EventID]; !exists {
					payoutOrder = append(payoutOrder, payoutRecord.EventID)
				}
				payoutByEventID[payoutRecord.EventID] = payoutRecord
				key := positionKey(payload.AccountID, payload.MarketID, payload.WinningOutcome)
				current := settledByPosition[key]
				current.MarketID = payload.MarketID
				current.AccountID = payload.AccountID
				current.Outcome = normalizeOutcome(payload.WinningOutcome)
				current.PositionAsset = payoutRecord.PositionAsset
				current.SettledQuantity += payload.SettledQuantity
				current.LastAppliedBatch = batch.BatchID
				settledByPosition[key] = current
			}
			if err := state.apply(entry); err != nil {
				return AcceptedReplaySnapshot{}, err
			}
		}
	}

	latestBatchID := int64(0)
	if len(batches) > 0 {
		latestBatchID = batches[len(batches)-1].BatchID
	}

	return AcceptedReplaySnapshot{
		BatchID:                latestBatchID,
		Roots:                  state.roots(),
		Balances:               exportAcceptedBalances(latestBatchID, state.balances),
		Positions:              exportAcceptedPositions(latestBatchID, state.positions, settledByPosition),
		Payouts:                exportAcceptedPayouts(payoutOrder, payoutByEventID),
		EscapeCollateralRoot:   AcceptedEscapeCollateralRootRecord{},
		EscapeCollateralLeaves: []AcceptedEscapeCollateralLeafRecord{},
	}, nil
}

func buildWitnessWalletMirror(batches []StoredBatch) (map[int64]string, error) {
	walletByAccount := make(map[int64]string)
	for _, batch := range batches {
		input, err := DecodeBatchInput(batch.InputData)
		if err != nil {
			return nil, err
		}
		for _, entry := range input.Entries {
			switch entry.EntryType {
			case EntryTypeTradingKeyAuthorized:
				var payload TradingKeyAuthorizedPayload
				if err := json.Unmarshal(entry.Payload, &payload); err != nil {
					return nil, err
				}
				walletByAccount[payload.AuthorizationWitness.AccountID] = payload.AuthorizationWitness.WalletAddress
			case EntryTypeDepositCredited:
				var payload DepositCreditedPayload
				if err := json.Unmarshal(entry.Payload, &payload); err != nil {
					return nil, err
				}
				walletByAccount[payload.AccountID] = payload.WalletAddress
			case EntryTypeWithdrawalRequested:
				var payload WithdrawalRequestedPayload
				if err := json.Unmarshal(entry.Payload, &payload); err != nil {
					return nil, err
				}
				walletByAccount[payload.AccountID] = payload.WalletAddress
			case EntryTypeNonceAdvanced:
				var payload NonceAdvancedPayload
				if err := json.Unmarshal(entry.Payload, &payload); err != nil {
					return nil, err
				}
				if payload.OrderAuthorization != nil {
					walletByAccount[payload.AccountID] = payload.OrderAuthorization.WalletAddress
				}
			}
		}
	}
	return walletByAccount, nil
}

func hashJournalEntries(entries []JournalEntry) string {
	if len(entries) == 0 {
		return hashStrings("shadow", "transition", "entries", "empty")
	}
	hashes := make([]string, 0, len(entries))
	for _, entry := range entries {
		sum, err := hashStruct("shadow_transition_entry", fmt.Sprintf("%d:%s", entry.Sequence, entry.EntryID), entry)
		if err != nil {
			return hashStrings("shadow", "transition", "entries", "encode_error", fmt.Sprintf("%d", entry.Sequence))
		}
		hashes = append(hashes, sum)
	}
	return hashStrings(append([]string{"shadow", "transition", "entries"}, hashes...)...)
}

func hashAcceptedBalances(items []AcceptedBalanceRecord) string {
	sorted := append([]AcceptedBalanceRecord(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].AccountID != sorted[j].AccountID {
			return sorted[i].AccountID < sorted[j].AccountID
		}
		return sorted[i].Asset < sorted[j].Asset
	})
	return hashAcceptedItems("balances", sorted, func(item AcceptedBalanceRecord) string {
		return fmt.Sprintf("%d:%s", item.AccountID, item.Asset)
	})
}

func hashAcceptedPositions(items []AcceptedPositionRecord) string {
	sorted := append([]AcceptedPositionRecord(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].AccountID != sorted[j].AccountID {
			return sorted[i].AccountID < sorted[j].AccountID
		}
		if sorted[i].MarketID != sorted[j].MarketID {
			return sorted[i].MarketID < sorted[j].MarketID
		}
		return sorted[i].Outcome < sorted[j].Outcome
	})
	return hashAcceptedItems("positions", sorted, func(item AcceptedPositionRecord) string {
		return fmt.Sprintf("%d:%d:%s", item.AccountID, item.MarketID, item.Outcome)
	})
}

func hashAcceptedPayouts(items []AcceptedPayoutRecord) string {
	sorted := append([]AcceptedPayoutRecord(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].EventID < sorted[j].EventID
	})
	return hashAcceptedItems("payouts", sorted, func(item AcceptedPayoutRecord) string {
		return item.EventID
	})
}

func hashAcceptedWithdrawalRoot(item AcceptedWithdrawalRootRecord) string {
	sum, err := hashStruct("shadow_transition_withdrawal_root", fmt.Sprintf("%d", item.BatchID), item)
	if err != nil {
		return hashStrings("shadow", "transition", "withdrawal_root", "encode_error")
	}
	return hashStrings("shadow", "transition", "withdrawal_root", sum)
}

func hashAcceptedWithdrawalLeaves(items []AcceptedWithdrawalLeafRecord) string {
	sorted := append([]AcceptedWithdrawalLeafRecord(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LeafIndex < sorted[j].LeafIndex
	})
	return hashAcceptedItems("withdrawal_leaves", sorted, func(item AcceptedWithdrawalLeafRecord) string {
		return fmt.Sprintf("%d:%s", item.LeafIndex, item.ClaimID)
	})
}

func hashAcceptedEscapeRoot(item AcceptedEscapeCollateralRootRecord) string {
	sum, err := hashStruct("shadow_transition_escape_root", fmt.Sprintf("%d", item.BatchID), item)
	if err != nil {
		return hashStrings("shadow", "transition", "escape_root", "encode_error")
	}
	return hashStrings("shadow", "transition", "escape_root", sum)
}

func hashAcceptedEscapeLeaves(items []AcceptedEscapeCollateralLeafRecord) string {
	sorted := append([]AcceptedEscapeCollateralLeafRecord(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LeafIndex < sorted[j].LeafIndex
	})
	return hashAcceptedItems("escape_leaves", sorted, func(item AcceptedEscapeCollateralLeafRecord) string {
		return fmt.Sprintf("%d:%s", item.LeafIndex, item.ClaimID)
	})
}

func hashAcceptedItems[T any](namespace string, items []T, key func(T) string) string {
	if len(items) == 0 {
		return hashStrings("shadow", "transition", namespace, "empty")
	}
	hashes := make([]string, 0, len(items))
	for _, item := range items {
		sum, err := hashStruct("shadow_transition_"+namespace, key(item), item)
		if err != nil {
			return hashStrings("shadow", "transition", namespace, "encode_error", key(item))
		}
		hashes = append(hashes, sum)
	}
	return hashStrings(append([]string{"shadow", "transition", namespace}, hashes...)...)
}
