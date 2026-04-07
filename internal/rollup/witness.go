package rollup

import (
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

const (
	TruthModeTruthfulShadow   = "TRUTHFUL_SHADOW"
	TruthModeZeroPlaceholder  = "ZERO_PLACEHOLDER"
	TruthModeShadowTransition = "SHADOW_ONLY_TRANSITIONAL"
)

func BuildShadowBatchContract(batch StoredBatch) (ShadowBatchContract, error) {
	witness, err := BuildShadowBatchWitness(batch)
	if err != nil {
		return ShadowBatchContract{}, err
	}
	return ShadowBatchContract{
		Witness:         witness,
		PublicInputs:    BuildShadowBatchPublicInputs(batch),
		L1BatchMetadata: BuildL1BatchMetadata(batch),
	}, nil
}

func BuildShadowBatchWitness(batch StoredBatch) (ShadowBatchWitness, error) {
	input, err := DecodeBatchInput(batch.InputData)
	if err != nil {
		return ShadowBatchWitness{}, err
	}
	return ShadowBatchWitness{
		EncodingVersion: input.EncodingVersion,
		Entries:         input.Entries,
		NamespaceTruth:  shadowNamespaceTruth(),
		Limitations: []string{
			"orders_root.nonce_root truthfully shadows API/auth accepted order-nonce advances as a monotonic next_nonce floor; verifier-eligible paths now also carry NONCE_ADVANCED.payload.order_authorization bound to canonical V2 TRADING_KEY_AUTHORIZED witness refs, while deprecated blank-vault /api/v1/sessions rows remain compatibility-only.",
			"TRADING_KEY_AUTHORIZED and NONCE_ADVANCED.payload.order_authorization are witness-only prep for a future prover lane; current replay and public inputs still do not verify wallet auth or Ed25519 order signatures inside the state transition.",
			"withdrawals_root still mirrors direct-vault queueWithdrawal requests and does not yet commit canonical claim-nullifier truth.",
			"positions_funding_root.insurance_root remains a deterministic zero placeholder in the current binary-market shadow lane.",
		},
	}, nil
}

func BuildShadowBatchPublicInputs(batch StoredBatch) ShadowBatchPublicInputs {
	encodingVersion := strings.TrimSpace(batch.EncodingVersion)
	if encodingVersion == "" {
		encodingVersion = BatchEncodingVersion
	}
	conservationHash := ZeroConservationHash()
	if input, err := DecodeBatchInput(batch.InputData); err == nil {
		if record, err := BuildConservationRecord(batch.BatchID, input.Entries); err == nil {
			conservationHash = record.ConservationHash
		}
	}
	return ShadowBatchPublicInputs{
		EncodingVersion:      encodingVersion,
		BatchID:              batch.BatchID,
		FirstSequence:        batch.FirstSequence,
		LastSequence:         batch.LastSequence,
		EntryCount:           batch.EntryCount,
		BatchDataHash:        canonicalBatchDataHash(batch),
		PrevStateRoot:        defaultStateRoot(batch.PrevStateRoot),
		BalancesRoot:         defaultComponentRoot(batch.BalancesRoot, ZeroBalancesRoot()),
		OrdersRoot:           defaultComponentRoot(batch.OrdersRoot, hashStrings("shadow", "orders", ZeroNonceRoot(), ZeroOpenOrdersRoot())),
		PositionsFundingRoot: defaultComponentRoot(batch.PositionsFundingRoot, hashStrings("shadow", "positions_funding", hashStrings("shadow", "positions", "leafs", "empty"), ZeroMarketFundingRoot(), ZeroInsuranceRoot())),
		WithdrawalsRoot:      defaultComponentRoot(batch.WithdrawalsRoot, ZeroWithdrawalsRoot()),
		NextStateRoot:        defaultStateRoot(batch.StateRoot),
		ConservationHash:     conservationHash,
	}
}

func BuildL1BatchMetadata(batch StoredBatch) L1BatchMetadata {
	return L1BatchMetadata{
		BatchID:       batch.BatchID,
		BatchDataHash: canonicalBatchDataHash(batch),
		PrevStateRoot: defaultStateRoot(batch.PrevStateRoot),
		NextStateRoot: defaultStateRoot(batch.StateRoot),
	}
}

func shadowNamespaceTruth() []NamespaceTruth {
	return []NamespaceTruth{
		{
			Namespace: "balances_root",
			Mode:      TruthModeTruthfulShadow,
			Detail:    "Deterministically mirrors deposits, order reserves/releases, matched-trade balance moves, settlement payouts, and direct-vault withdrawal requests.",
		},
		{
			Namespace: "orders_root.nonce_root",
			Mode:      TruthModeTruthfulShadow,
			Detail:    "Truthfully mirrors API/auth accepted order-nonce advances as a `(account_id, auth_key_id)` keyed next_nonce floor, with verifier-eligible paths also carrying canonical V2 trading-key order-authorization witness material in each NONCE_ADVANCED payload.",
		},
		{
			Namespace: "orders_root.open_orders_root",
			Mode:      TruthModeTruthfulShadow,
			Detail:    "Mirrors accepted resting orders plus matching- and settlement-triggered cancellations from ordered journal inputs.",
		},
		{
			Namespace: "positions_funding_root.position_root",
			Mode:      TruthModeTruthfulShadow,
			Detail:    "Mirrors matched-trade position deltas and consumes winning position quantity when settlement payouts are applied.",
		},
		{
			Namespace: "positions_funding_root.market_funding_root",
			Mode:      TruthModeTruthfulShadow,
			Detail:    "Tracks market resolution state for settled markets while keeping cumulative funding index fixed at zero in the current binary-market lane.",
		},
		{
			Namespace: "positions_funding_root.insurance_root",
			Mode:      TruthModeZeroPlaceholder,
			Detail:    "Insurance accounting is still out of scope for the current shadow batch version.",
		},
		{
			Namespace: "withdrawals_root",
			Mode:      TruthModeShadowTransition,
			Detail:    "Mirrors direct-vault withdrawal requests today and is not yet the future claim-nullifier-based canonical withdrawal truth.",
		},
	}
}

func canonicalBatchDataHash(batch StoredBatch) string {
	return strings.TrimPrefix(strings.ToLower(crypto.Keccak256Hash([]byte(batch.InputData)).Hex()), "0x")
}

func defaultStateRoot(root string) string {
	return defaultComponentRoot(root, ZeroStateRoot())
}

func defaultComponentRoot(root, fallback string) string {
	if strings.TrimSpace(root) == "" {
		return fallback
	}
	return root
}
