package rollup

import "testing"

func TestBuildAcceptedReplaySnapshotExportsBalancesPositionsAndPayouts(t *testing.T) {
	tradingEntries := []JournalEntry{
		mustEntry(t, 1, EntryTypeDepositCredited, SourceTypeChainDeposit, "dep_1", DepositCreditedPayload{
			DepositID: "dep_1",
			AccountID: 1,
			Asset:     "USDT",
			Amount:    1000,
		}),
		mustEntry(t, 2, EntryTypeNonceAdvanced, SourceTypeAPIAuth, "sess_live:7", NonceAdvancedPayload{
			AccountID:        1,
			AuthKeyID:        "sess_live",
			Scope:            "TRADE",
			KeyStatus:        "ACTIVE",
			AcceptedNonce:    7,
			NextNonce:        8,
			OccurredAtMillis: 1775886400000,
		}),
		mustEntry(t, 3, EntryTypeOrderAccepted, SourceTypeMatchingOrder, "ord_buy_1", OrderAcceptedPayload{
			OrderID:         "ord_buy_1",
			CommandID:       "cmd_buy_1",
			AccountID:       1,
			MarketID:        88,
			Outcome:         "YES",
			Side:            "BUY",
			OrderType:       "LIMIT",
			TimeInForce:     "GTC",
			CollateralAsset: "USDT",
			ReserveAsset:    "USDT",
			ReserveAmount:   400,
			Price:           40,
			Quantity:        10,
		}),
		mustEntry(t, 4, EntryTypeOrderAccepted, SourceTypeMatchingOrder, "ord_sell_1", OrderAcceptedPayload{
			OrderID:         "ord_sell_1",
			CommandID:       "cmd_sell_1",
			AccountID:       2,
			MarketID:        88,
			Outcome:         "YES",
			Side:            "SELL",
			OrderType:       "LIMIT",
			TimeInForce:     "GTC",
			CollateralAsset: "USDT",
			ReserveAsset:    "POSITION:88:YES",
			ReserveAmount:   6,
			Price:           35,
			Quantity:        6,
		}),
		mustEntry(t, 5, EntryTypeTradeMatched, SourceTypeMatchingTrade, "trd_1", TradeMatchedPayload{
			TradeID:         "trd_1",
			Sequence:        1,
			CollateralAsset: "USDT",
			MarketID:        88,
			Outcome:         "YES",
			Price:           35,
			Quantity:        6,
			TakerOrderID:    "ord_buy_1",
			MakerOrderID:    "ord_sell_1",
			TakerAccountID:  1,
			MakerAccountID:  2,
			TakerSide:       "BUY",
			MakerSide:       "SELL",
		}),
		mustEntry(t, 6, EntryTypeOrderCancelled, SourceTypeMatchingOrder, "ord_buy_1", OrderCancelledPayload{
			OrderID:           "ord_buy_1",
			AccountID:         1,
			MarketID:          88,
			Outcome:           "YES",
			Side:              "BUY",
			ReserveAsset:      "USDT",
			Price:             40,
			RemainingQuantity: 4,
			CancelReason:      "IOC_PARTIAL_FILL",
		}),
	}
	tradingInput, tradingHash, err := EncodeBatchInput(tradingEntries)
	if err != nil {
		t.Fatalf("EncodeBatchInput(trading) returned error: %v", err)
	}
	tradingBatch := StoredBatch{
		BatchID:         1,
		EncodingVersion: BatchEncodingVersion,
		FirstSequence:   1,
		LastSequence:    6,
		EntryCount:      len(tradingEntries),
		InputData:       tradingInput,
		InputHash:       tradingHash,
		PrevStateRoot:   ZeroStateRoot(),
	}
	tradingRoots, err := ReplayStoredBatches([]StoredBatch{tradingBatch})
	if err != nil {
		t.Fatalf("ReplayStoredBatches(trading) returned error: %v", err)
	}
	tradingBatch.BalancesRoot = tradingRoots.BalancesRoot
	tradingBatch.OrdersRoot = tradingRoots.OrdersRoot
	tradingBatch.PositionsFundingRoot = tradingRoots.PositionsFundingRoot
	tradingBatch.WithdrawalsRoot = tradingRoots.WithdrawalsRoot
	tradingBatch.StateRoot = tradingRoots.StateRoot

	settlementEntries := []JournalEntry{
		mustEntry(t, 7, EntryTypeMarketResolved, SourceTypeSettlementMarket, "88", MarketResolvedPayload{
			MarketID:         88,
			ResolvedOutcome:  "YES",
			ResolverType:     "ORACLE_PRICE",
			ResolverRef:      "oracle_price:BINANCE:BTCUSDT:1775886400",
			EvidenceHash:     "4d02819d12f6be4ce9fd857f2f6dc1661888881b606b0dd6f85f0df78fd8f1ac",
			OccurredAtMillis: 1775886400000,
		}),
		mustEntry(t, 8, EntryTypeSettlementPayout, SourceTypeSettlementPayout, "evt_settlement_88_1", SettlementPayoutPayload{
			EventID:          "evt_settlement_88_1",
			MarketID:         88,
			AccountID:        1,
			WinningOutcome:   "YES",
			PositionAsset:    "POSITION:88:YES",
			SettledQuantity:  6,
			PayoutAsset:      "USDT",
			PayoutAmount:     600,
			OccurredAtMillis: 1775886401000,
		}),
	}
	settlementInput, settlementHash, err := EncodeBatchInput(settlementEntries)
	if err != nil {
		t.Fatalf("EncodeBatchInput(settlement) returned error: %v", err)
	}
	settlementBatch := StoredBatch{
		BatchID:         2,
		EncodingVersion: BatchEncodingVersion,
		FirstSequence:   7,
		LastSequence:    8,
		EntryCount:      len(settlementEntries),
		InputData:       settlementInput,
		InputHash:       settlementHash,
		PrevStateRoot:   tradingBatch.StateRoot,
	}
	settlementRoots, err := ReplayStoredBatches([]StoredBatch{tradingBatch, settlementBatch})
	if err != nil {
		t.Fatalf("ReplayStoredBatches(settlement) returned error: %v", err)
	}
	settlementBatch.BalancesRoot = settlementRoots.BalancesRoot
	settlementBatch.OrdersRoot = settlementRoots.OrdersRoot
	settlementBatch.PositionsFundingRoot = settlementRoots.PositionsFundingRoot
	settlementBatch.WithdrawalsRoot = settlementRoots.WithdrawalsRoot
	settlementBatch.StateRoot = settlementRoots.StateRoot

	snapshot, err := BuildAcceptedReplaySnapshot([]StoredBatch{tradingBatch, settlementBatch})
	if err != nil {
		t.Fatalf("BuildAcceptedReplaySnapshot returned error: %v", err)
	}

	if snapshot.BatchID != 2 {
		t.Fatalf("snapshot.BatchID = %d, want 2", snapshot.BatchID)
	}
	if snapshot.Roots.StateRoot != settlementBatch.StateRoot {
		t.Fatalf("snapshot.StateRoot = %s, want %s", snapshot.Roots.StateRoot, settlementBatch.StateRoot)
	}

	if len(snapshot.Balances) != 3 {
		t.Fatalf("accepted balances len = %d, want 3", len(snapshot.Balances))
	}
	if snapshot.Balances[0].AccountID != 1 || snapshot.Balances[0].Asset != "USDT" || snapshot.Balances[0].Available != 1390 {
		t.Fatalf("unexpected first accepted balance: %+v", snapshot.Balances[0])
	}
	if snapshot.Balances[1].AccountID != 2 || snapshot.Balances[1].Asset != "POSITION:88:YES" || snapshot.Balances[1].Available != -6 {
		t.Fatalf("unexpected second accepted balance: %+v", snapshot.Balances[1])
	}
	if snapshot.Balances[2].AccountID != 2 || snapshot.Balances[2].Asset != "USDT" || snapshot.Balances[2].Available != 210 {
		t.Fatalf("unexpected third accepted balance: %+v", snapshot.Balances[2])
	}

	if len(snapshot.Positions) != 2 {
		t.Fatalf("accepted positions len = %d, want 2", len(snapshot.Positions))
	}
	if snapshot.Positions[0].AccountID != 1 || snapshot.Positions[0].MarketID != 88 || snapshot.Positions[0].Quantity != 6 || snapshot.Positions[0].SettledQuantity != 6 {
		t.Fatalf("unexpected settled winner position: %+v", snapshot.Positions[0])
	}
	if snapshot.Positions[1].AccountID != 2 || snapshot.Positions[1].Quantity != -6 || snapshot.Positions[1].SettledQuantity != 0 {
		t.Fatalf("unexpected short position: %+v", snapshot.Positions[1])
	}

	if len(snapshot.Payouts) != 1 {
		t.Fatalf("accepted payouts len = %d, want 1", len(snapshot.Payouts))
	}
	if snapshot.Payouts[0].EventID != "evt_settlement_88_1" || snapshot.Payouts[0].PayoutAmount != 600 || snapshot.Payouts[0].Status != "COMPLETED" {
		t.Fatalf("unexpected accepted payout: %+v", snapshot.Payouts[0])
	}
}
