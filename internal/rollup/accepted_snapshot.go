package rollup

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"funnyoption/internal/shared/assets"
)

type settledPositionAccumulator struct {
	MarketID         int64
	AccountID        int64
	Outcome          string
	PositionAsset    string
	SettledQuantity  int64
	LastAppliedBatch int64
}

func BuildAcceptedReplaySnapshot(batches []StoredBatch) (AcceptedReplaySnapshot, error) {
	state := shadowState{
		balances:      make(map[string]balanceLeaf),
		nonces:        make(map[string]nonceLeaf),
		openOrders:    make(map[string]orderLeaf),
		positions:     make(map[string]positionLeaf),
		marketFunding: make(map[string]marketFundingLeaf),
		withdrawals:   make(map[string]withdrawalLeaf),
	}
	roots := state.roots()

	payoutOrder := make([]string, 0)
	payoutByEventID := make(map[string]AcceptedPayoutRecord)
	settledByPosition := make(map[string]settledPositionAccumulator)

	for _, batch := range batches {
		if batch.InputHash != "" && hashStrings("shadow", "batch_input", batch.InputData) != batch.InputHash {
			return AcceptedReplaySnapshot{}, fmt.Errorf("rollup batch input hash mismatch for batch %d", batch.BatchID)
		}
		expectedPrev := strings.TrimSpace(batch.PrevStateRoot)
		if expectedPrev == "" {
			expectedPrev = ZeroStateRoot()
		}
		if roots.StateRoot != expectedPrev {
			return AcceptedReplaySnapshot{}, fmt.Errorf("rollup batch prev_state_root mismatch: have %s want %s", roots.StateRoot, expectedPrev)
		}

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

		roots = state.roots()
		if strings.TrimSpace(batch.StateRoot) != "" && roots.StateRoot != batch.StateRoot {
			return AcceptedReplaySnapshot{}, fmt.Errorf("accepted replay state_root mismatch for batch %d: have %s want %s", batch.BatchID, roots.StateRoot, batch.StateRoot)
		}
	}

	latestBatchID := int64(0)
	if len(batches) > 0 {
		latestBatchID = batches[len(batches)-1].BatchID
	}

	return AcceptedReplaySnapshot{
		BatchID:   latestBatchID,
		Roots:     roots,
		Balances:  exportAcceptedBalances(latestBatchID, state.balances),
		Positions: exportAcceptedPositions(latestBatchID, state.positions, settledByPosition),
		Payouts:   exportAcceptedPayouts(payoutOrder, payoutByEventID),
	}, nil
}

func decodeAcceptedSettlementPayout(batchID int64, payloadJSON json.RawMessage) (SettlementPayoutPayload, AcceptedPayoutRecord, error) {
	var payload SettlementPayoutPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return SettlementPayoutPayload{}, AcceptedPayoutRecord{}, err
	}
	positionAsset := normalizeAsset(payload.PositionAsset)
	if positionAsset == "" {
		positionAsset = assets.PositionAsset(payload.MarketID, payload.WinningOutcome)
	}
	payoutAsset := normalizeAsset(payload.PayoutAsset)
	if payoutAsset == "" {
		payoutAsset = assets.DefaultCollateralAsset
	}
	record := AcceptedPayoutRecord{
		EventID:         normalizeText(payload.EventID),
		BatchID:         batchID,
		MarketID:        payload.MarketID,
		UserID:          payload.AccountID,
		WinningOutcome:  normalizeOutcome(payload.WinningOutcome),
		PositionAsset:   positionAsset,
		SettledQuantity: payload.SettledQuantity,
		PayoutAsset:     payoutAsset,
		PayoutAmount:    payload.PayoutAmount,
		Status:          "COMPLETED",
	}
	return payload, record, nil
}

func exportAcceptedBalances(batchID int64, balances map[string]balanceLeaf) []AcceptedBalanceRecord {
	keys := make([]string, 0, len(balances))
	for key := range balances {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	items := make([]AcceptedBalanceRecord, 0, len(keys))
	for _, key := range keys {
		leaf := balances[key]
		items = append(items, AcceptedBalanceRecord{
			BatchID:    batchID,
			AccountID:  leaf.AccountID,
			Asset:      leaf.AssetID,
			Available:  leaf.FreeBalance,
			Frozen:     leaf.LockedBalance,
			SequenceNo: leaf.LastAppliedSequence,
		})
	}
	return items
}

func exportAcceptedPositions(batchID int64, positions map[string]positionLeaf, settled map[string]settledPositionAccumulator) []AcceptedPositionRecord {
	keys := make([]string, 0, len(positions)+len(settled))
	seen := make(map[string]struct{}, len(positions)+len(settled))
	for key := range positions {
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	for key := range settled {
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	items := make([]AcceptedPositionRecord, 0, len(keys))
	for _, key := range keys {
		leaf, hasLeaf := positions[key]
		settle, hasSettlement := settled[key]

		var (
			accountID     int64
			marketID      int64
			outcome       string
			positionAsset string
			remaining     int64
			sequenceNo    int64
			status        string
		)
		if hasLeaf {
			accountID = leaf.AccountID
			marketID = leaf.MarketID
			outcome = leaf.Outcome
			positionAsset = assets.PositionAsset(leaf.MarketID, leaf.Outcome)
			remaining = leaf.Quantity
			sequenceNo = leaf.LastAppliedSequence
			status = normalizeText(leaf.SettlementStatus)
		}
		if hasSettlement {
			accountID = settle.AccountID
			marketID = settle.MarketID
			outcome = settle.Outcome
			positionAsset = settle.PositionAsset
			if settle.LastAppliedBatch > 0 && sequenceNo == 0 {
				sequenceNo = settle.LastAppliedBatch
			}
		}
		if positionAsset == "" {
			positionAsset = assets.PositionAsset(marketID, outcome)
		}
		settledQuantity := int64(0)
		if hasSettlement {
			settledQuantity = settle.SettledQuantity
		}
		totalQuantity := remaining + settledQuantity
		if totalQuantity == 0 && settledQuantity == 0 {
			continue
		}
		if status == "" {
			if settledQuantity > 0 {
				status = "SETTLED"
			} else {
				status = "OPEN"
			}
		}
		items = append(items, AcceptedPositionRecord{
			BatchID:          batchID,
			AccountID:        accountID,
			MarketID:         marketID,
			Outcome:          outcome,
			PositionAsset:    positionAsset,
			Quantity:         totalQuantity,
			SettledQuantity:  settledQuantity,
			SettlementStatus: status,
			SequenceNo:       sequenceNo,
		})
	}
	return items
}

func exportAcceptedPayouts(order []string, payouts map[string]AcceptedPayoutRecord) []AcceptedPayoutRecord {
	items := make([]AcceptedPayoutRecord, 0, len(order))
	for _, eventID := range order {
		items = append(items, payouts[eventID])
	}
	return items
}
