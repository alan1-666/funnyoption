package rollup

import "testing"

func TestBuildAcceptedEscapeCollateralSnapshotUsesChainUnitsAndVerifiableProof(t *testing.T) {
	root, leaves, err := BuildAcceptedEscapeCollateralSnapshot(
		7,
		"abcd1234",
		[]AcceptedBalanceRecord{
			{
				AccountID: 1,
				Asset:     "USDT",
				Available: 1390,
				Frozen:    10,
			},
			{
				AccountID: 2,
				Asset:     "POSITION:88:YES",
				Available: 6,
			},
		},
		map[int64]string{
			1: "0x1111111111111111111111111111111111111111",
		},
	)
	if err != nil {
		t.Fatalf("BuildAcceptedEscapeCollateralSnapshot returned error: %v", err)
	}
	if root.BatchID != 7 {
		t.Fatalf("root.BatchID = %d, want 7", root.BatchID)
	}
	if root.StateRoot != "0xabcd1234" {
		t.Fatalf("root.StateRoot = %s, want 0xabcd1234", root.StateRoot)
	}
	if root.CollateralAsset != "USDT" {
		t.Fatalf("root.CollateralAsset = %s, want USDT", root.CollateralAsset)
	}
	if root.LeafCount != 1 {
		t.Fatalf("root.LeafCount = %d, want 1", root.LeafCount)
	}
	if root.TotalAmount != 14000000 {
		t.Fatalf("root.TotalAmount = %d, want 14000000", root.TotalAmount)
	}
	if len(leaves) != 1 {
		t.Fatalf("len(leaves) = %d, want 1", len(leaves))
	}
	leaf := leaves[0]
	if leaf.ClaimAmount != 14000000 {
		t.Fatalf("leaf.ClaimAmount = %d, want 14000000", leaf.ClaimAmount)
	}
	if leaf.ClaimStatus != EscapeCollateralClaimStatusClaimable {
		t.Fatalf("leaf.ClaimStatus = %s, want %s", leaf.ClaimStatus, EscapeCollateralClaimStatusClaimable)
	}
	if len(leaf.ProofHashes) != 0 {
		t.Fatalf("len(leaf.ProofHashes) = %d, want 0 for single-leaf tree", len(leaf.ProofHashes))
	}
	if !VerifyAcceptedEscapeCollateralProof(root.MerkleRoot, leaf.LeafHash, leaf.ProofHashes, leaf.LeafIndex) {
		t.Fatalf("expected single-leaf escape proof to verify")
	}
}

func TestBuildAcceptedEscapeCollateralSnapshotRequiresWalletMirror(t *testing.T) {
	_, _, err := BuildAcceptedEscapeCollateralSnapshot(
		1,
		"abcd1234",
		[]AcceptedBalanceRecord{{
			AccountID: 1,
			Asset:     "USDT",
			Available: 100,
		}},
		nil,
	)
	if err == nil {
		t.Fatalf("expected missing wallet address to fail")
	}
}
