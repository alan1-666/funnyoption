package rollup

import (
	"encoding/json"
	"strings"
	"testing"

	sharedauth "funnyoption/internal/shared/auth"
)

func TestReplayStoredBatchesDeterministic(t *testing.T) {
	entries := []JournalEntry{
		mustEntry(t, 1, EntryTypeDepositCredited, SourceTypeChainDeposit, "dep_1", DepositCreditedPayload{
			DepositID: "dep_1",
			AccountID: 1,
			Asset:     "USDT",
			Amount:    1000,
		}),
		mustEntry(t, 2, EntryTypeNonceAdvanced, SourceTypeAPIAuth, "sess_1:7", NonceAdvancedPayload{
			AccountID:        1,
			AuthKeyID:        "sess_1",
			Scope:            "TRADE",
			KeyStatus:        "ACTIVE",
			AcceptedNonce:    7,
			NextNonce:        8,
			OccurredAtMillis: 1775880000000,
		}),
		mustEntry(t, 3, EntryTypeOrderAccepted, SourceTypeMatchingOrder, "ord_buy_1", OrderAcceptedPayload{
			OrderID:         "ord_buy_1",
			CommandID:       "cmd_buy_1",
			AccountID:       1,
			MarketID:        1001,
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
			MarketID:        1001,
			Outcome:         "YES",
			Side:            "SELL",
			OrderType:       "LIMIT",
			TimeInForce:     "GTC",
			CollateralAsset: "USDT",
			ReserveAsset:    "POSITION:1001:YES",
			ReserveAmount:   6,
			Price:           35,
			Quantity:        6,
		}),
		mustEntry(t, 5, EntryTypeTradeMatched, SourceTypeMatchingTrade, "trd_1", TradeMatchedPayload{
			TradeID:         "trd_1",
			Sequence:        1,
			CollateralAsset: "USDT",
			MarketID:        1001,
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
			MarketID:          1001,
			Outcome:           "YES",
			Side:              "BUY",
			ReserveAsset:      "USDT",
			Price:             40,
			RemainingQuantity: 4,
			CancelReason:      "IOC_PARTIAL_FILL",
		}),
		mustEntry(t, 7, EntryTypeWithdrawalRequested, SourceTypeChainWithdraw, "wdq_1", WithdrawalRequestedPayload{
			WithdrawalID:     "wdq_1",
			AccountID:        1,
			Asset:            "USDT",
			Amount:           50,
			RecipientAddress: "0x00000000000000000000000000000000000000aa",
			Lane:             "SLOW",
		}),
	}
	input, hash, err := EncodeBatchInput(entries)
	if err != nil {
		t.Fatalf("EncodeBatchInput returned error: %v", err)
	}
	if hash == "" {
		t.Fatalf("expected non-empty input hash")
	}

	batches := []StoredBatch{{
		BatchID:         1,
		EncodingVersion: BatchEncodingVersion,
		InputData:       input,
		InputHash:       hash,
		PrevStateRoot:   ZeroStateRoot(),
	}}

	first, err := ReplayStoredBatches(batches)
	if err != nil {
		t.Fatalf("ReplayStoredBatches returned error: %v", err)
	}
	second, err := ReplayStoredBatches(batches)
	if err != nil {
		t.Fatalf("ReplayStoredBatches second run returned error: %v", err)
	}

	if first != second {
		t.Fatalf("expected deterministic replay roots, got first=%+v second=%+v", first, second)
	}
	if first.StateRoot == ZeroStateRoot() {
		t.Fatalf("expected non-zero state root")
	}
	if first.OrdersRoot == hashStrings("shadow", "orders", ZeroNonceRoot(), ZeroOpenOrdersRoot()) {
		t.Fatalf("expected nonce_root to contribute to orders_root")
	}
	t.Logf("deterministic roots: %+v", first)
}

func TestReplayStoredBatchesRejectsPrevRootMismatch(t *testing.T) {
	input, _, err := EncodeBatchInput(nil)
	if err != nil {
		t.Fatalf("EncodeBatchInput returned error: %v", err)
	}
	_, err = ReplayStoredBatches([]StoredBatch{{
		BatchID:         1,
		EncodingVersion: BatchEncodingVersion,
		InputData:       input,
		PrevStateRoot:   "bad_prev_root",
	}})
	if err == nil {
		t.Fatalf("expected prev root mismatch error")
	}
}

func TestReplayStoredBatchesSettlementDeterministic(t *testing.T) {
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
		LastSequence:    5,
		EntryCount:      len(tradingEntries),
		InputData:       tradingInput,
		InputHash:       tradingHash,
		PrevStateRoot:   ZeroStateRoot(),
	}
	firstRoots, err := ReplayStoredBatches([]StoredBatch{tradingBatch})
	if err != nil {
		t.Fatalf("ReplayStoredBatches(trading) returned error: %v", err)
	}
	tradingBatch.BalancesRoot = firstRoots.BalancesRoot
	tradingBatch.OrdersRoot = firstRoots.OrdersRoot
	tradingBatch.PositionsFundingRoot = firstRoots.PositionsFundingRoot
	tradingBatch.WithdrawalsRoot = firstRoots.WithdrawalsRoot
	tradingBatch.StateRoot = firstRoots.StateRoot

	settlementEntries := []JournalEntry{
		mustEntry(t, 6, EntryTypeMarketResolved, SourceTypeSettlementMarket, "88", MarketResolvedPayload{
			MarketID:         88,
			ResolvedOutcome:  "YES",
			ResolverType:     "ORACLE_PRICE",
			ResolverRef:      "oracle_price:BINANCE:BTCUSDT:1775886400",
			EvidenceHash:     "4d02819d12f6be4ce9fd857f2f6dc1661888881b606b0dd6f85f0df78fd8f1ac",
			OccurredAtMillis: 1775886400000,
		}),
		mustEntry(t, 7, EntryTypeSettlementPayout, SourceTypeSettlementPayout, "evt_settlement_88_1", SettlementPayoutPayload{
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
		FirstSequence:   6,
		LastSequence:    7,
		EntryCount:      len(settlementEntries),
		InputData:       settlementInput,
		InputHash:       settlementHash,
		PrevStateRoot:   tradingBatch.StateRoot,
	}

	first, err := ReplayStoredBatches([]StoredBatch{tradingBatch, settlementBatch})
	if err != nil {
		t.Fatalf("ReplayStoredBatches returned error: %v", err)
	}
	settlementBatch.BalancesRoot = first.BalancesRoot
	settlementBatch.OrdersRoot = first.OrdersRoot
	settlementBatch.PositionsFundingRoot = first.PositionsFundingRoot
	settlementBatch.WithdrawalsRoot = first.WithdrawalsRoot
	settlementBatch.StateRoot = first.StateRoot

	second, err := ReplayStoredBatches([]StoredBatch{tradingBatch, settlementBatch})
	if err != nil {
		t.Fatalf("ReplayStoredBatches second run returned error: %v", err)
	}
	if first != second {
		t.Fatalf("expected deterministic settlement replay roots, got first=%+v second=%+v", first, second)
	}
	if first.BalancesRoot == firstRoots.BalancesRoot {
		t.Fatalf("expected settlement batch to change balances_root")
	}
	if first.PositionsFundingRoot == firstRoots.PositionsFundingRoot {
		t.Fatalf("expected settlement batch to change positions_funding_root")
	}

	contract, err := BuildShadowBatchContract(settlementBatch)
	if err != nil {
		t.Fatalf("BuildShadowBatchContract returned error: %v", err)
	}
	if contract.PublicInputs.BatchDataHash != canonicalBatchDataHash(settlementBatch) {
		t.Fatalf("expected batch_data_hash %s, got %s", canonicalBatchDataHash(settlementBatch), contract.PublicInputs.BatchDataHash)
	}
	if contract.PublicInputs.PrevStateRoot != tradingBatch.StateRoot {
		t.Fatalf("expected prev_state_root %s, got %s", tradingBatch.StateRoot, contract.PublicInputs.PrevStateRoot)
	}
	if contract.L1BatchMetadata.NextStateRoot != settlementBatch.StateRoot {
		t.Fatalf("expected L1 metadata next_state_root %s, got %s", settlementBatch.StateRoot, contract.L1BatchMetadata.NextStateRoot)
	}
	if len(contract.Witness.Entries) != len(settlementEntries) {
		t.Fatalf("expected %d witness entries, got %d", len(settlementEntries), len(contract.Witness.Entries))
	}
	if !containsNamespaceMode(contract.Witness.NamespaceTruth, "orders_root.nonce_root", TruthModeTruthfulShadow) {
		t.Fatalf("expected nonce_root to be marked as truthful shadow")
	}
	if !containsFragment(contract.Witness.Limitations, "NONCE_ADVANCED.payload.order_authorization") {
		t.Fatalf("expected nonce_root limitation note in witness contract, got %+v", contract.Witness.Limitations)
	}

	t.Logf("settlement deterministic roots: %+v", first)
}

func TestReplayStoredBatchesTradingKeyAuthorizationWitnessIsNoOp(t *testing.T) {
	stateEntries := []JournalEntry{
		mustEntry(t, 1, EntryTypeDepositCredited, SourceTypeChainDeposit, "dep_1", DepositCreditedPayload{
			DepositID: "dep_1",
			AccountID: 1,
			Asset:     "USDT",
			Amount:    1000,
		}),
		mustEntry(t, 2, EntryTypeNonceAdvanced, SourceTypeAPIAuth, "tk_live:7", NonceAdvancedPayload{
			AccountID:        1,
			AuthKeyID:        "tk_live",
			Scope:            "TRADE",
			KeyStatus:        "ACTIVE",
			AcceptedNonce:    7,
			NextNonce:        8,
			OccurredAtMillis: 1775886400000,
		}),
	}
	withWitnessEntries := append(append([]JournalEntry{}, stateEntries...), mustEntry(t, 3, EntryTypeTradingKeyAuthorized, SourceTypeAPIAuth, "tk_live:0xauth", TradingKeyAuthorizedPayload{
		AuthorizationWitness: sharedauth.TradingKeyAuthorizationWitness{
			AuthVersion:              sharedauth.CanonicalTradingKeyAuthVersion,
			VerifierEligible:         true,
			AuthorizationRef:         "tk_live:0xauth",
			TradingKeyID:             "tk_live",
			AccountID:                1,
			WalletAddress:            "0x00000000000000000000000000000000000000aa",
			ChainID:                  97,
			VaultAddress:             "0x00000000000000000000000000000000000000bb",
			TradingPublicKey:         "0x8f931f3d9d6a93f2b05a1e8ef8356d7408be0f2f5f63c2dbcbf6c227f5f1c5d2",
			TradingKeyScheme:         "ED25519",
			Scope:                    "TRADE",
			KeyStatus:                "ACTIVE",
			Challenge:                "0xauth",
			ChallengeExpiresAtMillis: 1775886700000,
			KeyExpiresAtMillis:       0,
			AuthorizedAtMillis:       1775886400000,
			WalletSignatureStandard:  sharedauth.DefaultWalletSignatureStandard,
			WalletTypedDataHash:      "0xhash",
			WalletSignature:          "0xsig",
		},
	}))

	inputWithoutWitness, hashWithoutWitness, err := EncodeBatchInput(stateEntries)
	if err != nil {
		t.Fatalf("EncodeBatchInput(stateEntries) returned error: %v", err)
	}
	inputWithWitness, hashWithWitness, err := EncodeBatchInput(withWitnessEntries)
	if err != nil {
		t.Fatalf("EncodeBatchInput(withWitnessEntries) returned error: %v", err)
	}

	rootsWithoutWitness, err := ReplayStoredBatches([]StoredBatch{{
		BatchID:         1,
		EncodingVersion: BatchEncodingVersion,
		InputData:       inputWithoutWitness,
		InputHash:       hashWithoutWitness,
		PrevStateRoot:   ZeroStateRoot(),
	}})
	if err != nil {
		t.Fatalf("ReplayStoredBatches(without witness) returned error: %v", err)
	}
	rootsWithWitness, err := ReplayStoredBatches([]StoredBatch{{
		BatchID:         1,
		EncodingVersion: BatchEncodingVersion,
		InputData:       inputWithWitness,
		InputHash:       hashWithWitness,
		PrevStateRoot:   ZeroStateRoot(),
	}})
	if err != nil {
		t.Fatalf("ReplayStoredBatches(with witness) returned error: %v", err)
	}

	if rootsWithoutWitness != rootsWithWitness {
		t.Fatalf("expected auth witness entry to be replay no-op, got without=%+v with=%+v", rootsWithoutWitness, rootsWithWitness)
	}
}

func TestReplayStoredBatchesNonceRootRequiresMonotonicAdvances(t *testing.T) {
	input, hash, err := EncodeBatchInput([]JournalEntry{
		mustEntry(t, 1, EntryTypeNonceAdvanced, SourceTypeAPIAuth, "sess_live:7", NonceAdvancedPayload{
			AccountID:        1,
			AuthKeyID:        "sess_live",
			Scope:            "TRADE",
			KeyStatus:        "ACTIVE",
			AcceptedNonce:    7,
			NextNonce:        8,
			OccurredAtMillis: 1775886400000,
		}),
		mustEntry(t, 2, EntryTypeNonceAdvanced, SourceTypeAPIAuth, "sess_live:6", NonceAdvancedPayload{
			AccountID:        1,
			AuthKeyID:        "sess_live",
			Scope:            "TRADE",
			KeyStatus:        "ACTIVE",
			AcceptedNonce:    6,
			NextNonce:        7,
			OccurredAtMillis: 1775886401000,
		}),
	})
	if err != nil {
		t.Fatalf("EncodeBatchInput returned error: %v", err)
	}

	_, err = ReplayStoredBatches([]StoredBatch{{
		BatchID:         1,
		EncodingVersion: BatchEncodingVersion,
		InputData:       input,
		InputHash:       hash,
		PrevStateRoot:   ZeroStateRoot(),
	}})
	if err == nil {
		t.Fatalf("expected replay to reject regressed nonce advances")
	}
}

func TestReplayStoredBatchesSettlementRejectsUnknownPosition(t *testing.T) {
	input, hash, err := EncodeBatchInput([]JournalEntry{
		mustEntry(t, 1, EntryTypeMarketResolved, SourceTypeSettlementMarket, "77", MarketResolvedPayload{
			MarketID:        77,
			ResolvedOutcome: "YES",
		}),
		mustEntry(t, 2, EntryTypeSettlementPayout, SourceTypeSettlementPayout, "evt_settlement_77_1", SettlementPayoutPayload{
			EventID:         "evt_settlement_77_1",
			MarketID:        77,
			AccountID:       1,
			WinningOutcome:  "YES",
			PositionAsset:   "POSITION:77:YES",
			SettledQuantity: 1,
			PayoutAsset:     "USDT",
			PayoutAmount:    100,
		}),
	})
	if err != nil {
		t.Fatalf("EncodeBatchInput returned error: %v", err)
	}
	_, err = ReplayStoredBatches([]StoredBatch{{
		BatchID:         1,
		EncodingVersion: BatchEncodingVersion,
		InputData:       input,
		InputHash:       hash,
		PrevStateRoot:   ZeroStateRoot(),
	}})
	if err == nil {
		t.Fatalf("expected settlement replay to reject payout without a matching position")
	}
}

func mustEntry(t *testing.T, sequence int64, entryType, sourceType, sourceRef string, payload any) JournalEntry {
	t.Helper()
	encoded, err := jsonMarshal(payload)
	if err != nil {
		t.Fatalf("jsonMarshal returned error: %v", err)
	}
	return JournalEntry{
		Sequence:   sequence,
		EntryType:  entryType,
		SourceType: sourceType,
		SourceRef:  sourceRef,
		Payload:    encoded,
	}
}

func jsonMarshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

func containsNamespaceMode(items []NamespaceTruth, namespace, mode string) bool {
	for _, item := range items {
		if item.Namespace == namespace && item.Mode == mode {
			return true
		}
	}
	return false
}

func containsFragment(items []string, fragment string) bool {
	for _, item := range items {
		if strings.Contains(item, fragment) {
			return true
		}
	}
	return false
}
