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

	"github.com/ethereum/go-ethereum/crypto"
)

const (
	WithdrawalClaimStatusClaimable = "CLAIMABLE"
	WithdrawalClaimStatusSubmitted = "CLAIM_SUBMITTED"
	WithdrawalClaimStatusClaimed   = "CLAIMED"
	WithdrawalClaimStatusFailed    = "CLAIM_FAILED"
)

type AcceptedWithdrawalRootRecord struct {
	BatchID    int64  `json:"batch_id"`
	MerkleRoot string `json:"merkle_root"`
	LeafCount  int64  `json:"leaf_count"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type AcceptedWithdrawalLeafRecord struct {
	BatchID          int64    `json:"batch_id"`
	WithdrawalID     string   `json:"withdrawal_id"`
	AccountID        int64    `json:"account_id"`
	WalletAddress    string   `json:"wallet_address"`
	RecipientAddress string   `json:"recipient_address"`
	Amount           int64    `json:"amount"`
	LeafIndex        int64    `json:"leaf_index"`
	LeafHash         string   `json:"leaf_hash"`
	ProofHashes      []string `json:"proof_hashes"`
	ClaimID          string   `json:"claim_id"`
	ClaimStatus      string   `json:"claim_status"`
	ClaimTxHash      string   `json:"claim_tx_hash"`
	ClaimSubmittedAt int64    `json:"claim_submitted_at"`
	ClaimedAt        int64    `json:"claimed_at"`
	LastError        string   `json:"last_error"`
	LastErrorAt      int64    `json:"last_error_at"`
	CreatedAt        int64    `json:"created_at"`
	UpdatedAt        int64    `json:"updated_at"`
}

func BuildAcceptedWithdrawalMerkleTree(
	batchID int64,
	withdrawals []AcceptedWithdrawalRecord,
) (AcceptedWithdrawalRootRecord, []AcceptedWithdrawalLeafRecord, error) {
	sort.Slice(withdrawals, func(i, j int) bool {
		return withdrawals[i].RequestSequence < withdrawals[j].RequestSequence
	})

	leafHashes := make([][]byte, 0, len(withdrawals))
	leaves := make([]AcceptedWithdrawalLeafRecord, 0, len(withdrawals))

	for index, w := range withdrawals {
		chainAmount, err := assets.AccountingToAssetChainAmount(
			assets.NormalizeAsset(w.Asset), w.Amount,
		)
		if err != nil {
			return AcceptedWithdrawalRootRecord{}, nil, fmt.Errorf(
				"convert withdrawal amount for %s: %w", w.WithdrawalID, err,
			)
		}
		recipientAddr := sharedauth.NormalizeHex(w.RecipientAddress)
		walletAddr := sharedauth.NormalizeHex(w.WalletAddress)

		leafHash := hashWithdrawalLeaf(batchID, int64(index), w.WithdrawalID, walletAddr, chainAmount, recipientAddr)
		claimID := buildWithdrawalClaimID(batchID, int64(index), w.WithdrawalID, walletAddr)

		leafHashes = append(leafHashes, leafHash)
		leaves = append(leaves, AcceptedWithdrawalLeafRecord{
			BatchID:          batchID,
			WithdrawalID:     w.WithdrawalID,
			AccountID:        w.AccountID,
			WalletAddress:    walletAddr,
			RecipientAddress: recipientAddr,
			Amount:           chainAmount,
			LeafIndex:        int64(index),
			LeafHash:         "0x" + hex.EncodeToString(leafHash),
			ClaimID:          claimID,
			ClaimStatus:      WithdrawalClaimStatusClaimable,
		})
	}

	rootBytes, proofs := buildWithdrawalMerkleTree(leafHashes)
	for index := range leaves {
		leaves[index].ProofHashes = proofs[index]
	}

	root := AcceptedWithdrawalRootRecord{
		BatchID:    batchID,
		MerkleRoot: "0x" + hex.EncodeToString(rootBytes),
		LeafCount:  int64(len(leaves)),
	}
	return root, leaves, nil
}

func hashWithdrawalLeaf(batchID, leafIndex int64, withdrawalID, walletAddress string, amount int64, recipientAddress string) []byte {
	var batchBytes [8]byte
	var leafBytes [8]byte
	binary.BigEndian.PutUint64(batchBytes[:], uint64(batchID))
	binary.BigEndian.PutUint64(leafBytes[:], uint64(leafIndex))

	withdrawalIDHash := crypto.Keccak256Hash([]byte(withdrawalID))

	return crypto.Keccak256(
		[]byte("funny-rollup-withdrawal-leaf-v1"),
		batchBytes[:],
		leafBytes[:],
		withdrawalIDHash[:],
		[]byte(sharedauth.NormalizeHex(walletAddress)),
		int64ToBytes(amount),
		[]byte(sharedauth.NormalizeHex(recipientAddress)),
	)
}

func buildWithdrawalClaimID(batchID, leafIndex int64, withdrawalID, walletAddress string) string {
	var batchBytes [8]byte
	var leafBytes [8]byte
	binary.BigEndian.PutUint64(batchBytes[:], uint64(batchID))
	binary.BigEndian.PutUint64(leafBytes[:], uint64(leafIndex))

	withdrawalIDHash := crypto.Keccak256Hash([]byte(withdrawalID))

	return crypto.Keccak256Hash(
		[]byte("funny-rollup-withdrawal-claim-v1"),
		batchBytes[:],
		leafBytes[:],
		withdrawalIDHash[:],
		[]byte(sharedauth.NormalizeHex(walletAddress)),
	).Hex()
}

func buildWithdrawalMerkleTree(leaves [][]byte) ([]byte, [][]string) {
	if len(leaves) == 0 {
		root := crypto.Keccak256([]byte("funny-rollup-empty-withdrawal-root"))
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

func VerifyAcceptedWithdrawalProof(rootHex, leafHex string, proof []string, leafIndex int64) bool {
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
