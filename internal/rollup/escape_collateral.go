package rollup

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"funnyoption/internal/shared/assets"
	sharedauth "funnyoption/internal/shared/auth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	EscapeCollateralAnchorStatusReady     = "READY"
	EscapeCollateralAnchorStatusSubmitted = "SUBMITTED"
	EscapeCollateralAnchorStatusAnchored  = "ANCHORED"
	EscapeCollateralAnchorStatusFailed    = "FAILED"

	EscapeCollateralClaimStatusClaimable = "CLAIMABLE"
	EscapeCollateralClaimStatusSubmitted = "CLAIM_SUBMITTED"
	EscapeCollateralClaimStatusClaimed   = "CLAIMED"
	EscapeCollateralClaimStatusFailed    = "CLAIM_FAILED"
)

type acceptedEscapeLeafInput struct {
	AccountID       int64
	WalletAddress   string
	CollateralAsset string
	ClaimAmount     int64
}

func BuildAcceptedEscapeCollateralSnapshot(
	batchID int64,
	stateRoot string,
	balances []AcceptedBalanceRecord,
	walletByAccount map[int64]string,
) (AcceptedEscapeCollateralRootRecord, []AcceptedEscapeCollateralLeafRecord, error) {
	inputs := make([]acceptedEscapeLeafInput, 0)
	for _, balance := range balances {
		if assets.NormalizeAsset(balance.Asset) != assets.DefaultCollateralAsset {
			continue
		}
		accountingTotal := balance.Available + balance.Frozen
		if accountingTotal <= 0 {
			continue
		}
		claimAmount, err := assets.AccountingToAssetChainAmount(assets.DefaultCollateralAsset, accountingTotal)
		if err != nil {
			return AcceptedEscapeCollateralRootRecord{}, nil, fmt.Errorf(
				"convert accepted escape collateral amount for account %d: %w",
				balance.AccountID,
				err,
			)
		}
		if claimAmount <= 0 {
			continue
		}
		walletAddress := sharedauth.NormalizeHex(walletByAccount[balance.AccountID])
		if walletAddress == "" {
			return AcceptedEscapeCollateralRootRecord{}, nil, fmt.Errorf("wallet address is required for accepted escape collateral account %d", balance.AccountID)
		}
		inputs = append(inputs, acceptedEscapeLeafInput{
			AccountID:       balance.AccountID,
			WalletAddress:   walletAddress,
			CollateralAsset: assets.DefaultCollateralAsset,
			ClaimAmount:     claimAmount,
		})
	}

	sort.Slice(inputs, func(i, j int) bool {
		if inputs[i].AccountID != inputs[j].AccountID {
			return inputs[i].AccountID < inputs[j].AccountID
		}
		return inputs[i].WalletAddress < inputs[j].WalletAddress
	})

	leafHashes := make([][]byte, 0, len(inputs))
	leaves := make([]AcceptedEscapeCollateralLeafRecord, 0, len(inputs))
	totalAmount := int64(0)
	for index, input := range inputs {
		leafHash := hashAcceptedEscapeCollateralLeaf(batchID, int64(index), input)
		claimID := buildEscapeCollateralClaimID(batchID, int64(index), input.WalletAddress, input.ClaimAmount)
		leafHashes = append(leafHashes, leafHash)
		leaves = append(leaves, AcceptedEscapeCollateralLeafRecord{
			BatchID:         batchID,
			AccountID:       input.AccountID,
			WalletAddress:   input.WalletAddress,
			CollateralAsset: input.CollateralAsset,
			ClaimAmount:     input.ClaimAmount,
			LeafIndex:       int64(index),
			LeafHash:        "0x" + hex.EncodeToString(leafHash),
			ProofHashes:     nil,
			ClaimID:         claimID,
			ClaimStatus:     EscapeCollateralClaimStatusClaimable,
		})
		totalAmount += input.ClaimAmount
	}

	rootBytes, proofs := buildAcceptedEscapeMerkleTree(leafHashes)
	for index := range leaves {
		leaves[index].ProofHashes = proofs[index]
	}

	root := AcceptedEscapeCollateralRootRecord{
		BatchID:           batchID,
		StateRoot:         normalizeHex32(stateRoot),
		CollateralAsset:   assets.DefaultCollateralAsset,
		MerkleRoot:        "0x" + hex.EncodeToString(rootBytes),
		LeafCount:         int64(len(leaves)),
		TotalAmount:       totalAmount,
		AnchorStatus:      EscapeCollateralAnchorStatusReady,
		AnchorTxHash:      "",
		AnchorSubmittedAt: 0,
		AnchoredAt:        0,
	}
	return root, leaves, nil
}

func hashAcceptedEscapeCollateralLeaf(batchID, leafIndex int64, input acceptedEscapeLeafInput) []byte {
	var batchBytes [8]byte
	var leafBytes [8]byte
	binary.BigEndian.PutUint64(batchBytes[:], uint64(batchID))
	binary.BigEndian.PutUint64(leafBytes[:], uint64(leafIndex))
	walletAddress := common.HexToAddress(sharedauth.NormalizeHex(input.WalletAddress))
	return crypto.Keccak256(
		[]byte("funny-rollup-escape-collateral-v1"),
		batchBytes[:],
		leafBytes[:],
		walletAddress.Bytes(),
		int64ToBytes(input.ClaimAmount),
	)
}

func buildEscapeCollateralClaimID(batchID, leafIndex int64, walletAddress string, amount int64) string {
	var batchBytes [8]byte
	var leafBytes [8]byte
	binary.BigEndian.PutUint64(batchBytes[:], uint64(batchID))
	binary.BigEndian.PutUint64(leafBytes[:], uint64(leafIndex))
	wallet := common.HexToAddress(sharedauth.NormalizeHex(walletAddress))
	return crypto.Keccak256Hash(
		[]byte("funny-rollup-escape-claim-v1"),
		batchBytes[:],
		leafBytes[:],
		wallet.Bytes(),
		int64ToBytes(amount),
	).Hex()
}

func buildAcceptedEscapeMerkleTree(leaves [][]byte) ([]byte, [][]string) {
	if len(leaves) == 0 {
		root := crypto.Keccak256([]byte("funny-rollup-empty-escape-collateral-root"))
		return root, [][]string{}
	}

	proofs := make([][]string, len(leaves))
	level := make([][]byte, len(leaves))
	copy(level, leaves)
	indexes := make([]int, len(leaves))
	for i := range indexes {
		indexes[i] = i
	}

	for len(level) > 1 {
		if len(level)%2 == 1 {
			level = append(level, level[len(level)-1])
			indexes = append(indexes, indexes[len(indexes)-1])
		}

		nextLevel := make([][]byte, 0, len(level)/2)
		nextIndexes := make([]int, 0, len(indexes)/2)
		for i := 0; i < len(level); i += 2 {
			left := level[i]
			right := level[i+1]
			leftIndex := indexes[i]
			rightIndex := indexes[i+1]
			if leftIndex < len(proofs) {
				proofs[leftIndex] = append(proofs[leftIndex], "0x"+hex.EncodeToString(right))
			}
			if rightIndex < len(proofs) && rightIndex != leftIndex {
				proofs[rightIndex] = append(proofs[rightIndex], "0x"+hex.EncodeToString(left))
			}
			nextLevel = append(nextLevel, crypto.Keccak256(left, right))
			nextIndexes = append(nextIndexes, minInt(leftIndex, rightIndex))
		}
		level = nextLevel
		indexes = nextIndexes
	}

	return level[0], proofs
}

func int64ToBytes(value int64) []byte {
	var buf [32]byte
	binary.BigEndian.PutUint64(buf[24:], uint64(value))
	return buf[:]
}

func normalizeHex32(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "0x") {
		return trimmed
	}
	return "0x" + trimmed
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func DecodeEscapeProofHex(proof []string) ([][]byte, error) {
	items := make([][]byte, 0, len(proof))
	for _, item := range proof {
		trimmed := strings.TrimPrefix(strings.TrimSpace(item), "0x")
		decoded, err := hex.DecodeString(trimmed)
		if err != nil {
			return nil, err
		}
		items = append(items, decoded)
	}
	return items, nil
}

func VerifyAcceptedEscapeCollateralProof(rootHex, leafHex string, proof []string, leafIndex int64) bool {
	root, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(rootHex), "0x"))
	if err != nil {
		return false
	}
	current, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(leafHex), "0x"))
	if err != nil {
		return false
	}
	decodedProof, err := DecodeEscapeProofHex(proof)
	if err != nil {
		return false
	}
	index := leafIndex
	for _, sibling := range decodedProof {
		if index%2 == 0 {
			current = crypto.Keccak256(current, sibling)
		} else {
			current = crypto.Keccak256(sibling, current)
		}
		index /= 2
	}
	return bytes.Equal(current, root)
}
