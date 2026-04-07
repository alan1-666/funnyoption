package rollup

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

const ConservationVersion = "funny-rollup-conservation-v1"

type ConservationRecord struct {
	Version         string                    `json:"version"`
	BatchID         int64                     `json:"batch_id"`
	AssetDeltas     []ConservationAssetDelta  `json:"asset_deltas"`
	ConservationHash string                   `json:"conservation_hash"`
	Conserved       bool                      `json:"conserved"`
}

type ConservationAssetDelta struct {
	Asset      string `json:"asset"`
	TotalDebit int64  `json:"total_debit"`
	TotalCredit int64 `json:"total_credit"`
	NetDelta   int64  `json:"net_delta"`
}

func BuildConservationRecord(batchID int64, entries []JournalEntry) (ConservationRecord, error) {
	credits := make(map[string]int64)
	debits := make(map[string]int64)

	for _, entry := range entries {
		switch entry.EntryType {
		case EntryTypeDepositCredited:
			var p DepositCreditedPayload
			if err := json.Unmarshal(entry.Payload, &p); err != nil {
				return ConservationRecord{}, fmt.Errorf("decode deposit payload seq=%d: %w", entry.Sequence, err)
			}
			asset := normalizeConservationAsset(p.Asset)
			credits[asset] += p.Amount

		case EntryTypeWithdrawalRequested:
			var p WithdrawalRequestedPayload
			if err := json.Unmarshal(entry.Payload, &p); err != nil {
				return ConservationRecord{}, fmt.Errorf("decode withdrawal payload seq=%d: %w", entry.Sequence, err)
			}
			asset := normalizeConservationAsset(p.Asset)
			debits[asset] += p.Amount

		case EntryTypeTradeMatched:
			var p TradeMatchedPayload
			if err := json.Unmarshal(entry.Payload, &p); err != nil {
				return ConservationRecord{}, fmt.Errorf("decode trade payload seq=%d: %w", entry.Sequence, err)
			}
			asset := normalizeConservationAsset(p.CollateralAsset)
			cost := p.Price * p.Quantity
			credits[asset] += cost
			debits[asset] += cost

		case EntryTypeSettlementPayout:
			var p SettlementPayoutPayload
			if err := json.Unmarshal(entry.Payload, &p); err != nil {
				return ConservationRecord{}, fmt.Errorf("decode settlement payload seq=%d: %w", entry.Sequence, err)
			}
			asset := normalizeConservationAsset(p.PayoutAsset)
			credits[asset] += p.PayoutAmount
			debits[asset] += p.PayoutAmount
		}
	}

	allAssets := make(map[string]bool)
	for a := range credits {
		allAssets[a] = true
	}
	for a := range debits {
		allAssets[a] = true
	}
	sortedAssets := make([]string, 0, len(allAssets))
	for a := range allAssets {
		sortedAssets = append(sortedAssets, a)
	}
	sort.Strings(sortedAssets)

	deltas := make([]ConservationAssetDelta, 0, len(sortedAssets))
	conserved := true
	for _, asset := range sortedAssets {
		credit := credits[asset]
		debit := debits[asset]
		net := credit - debit
		if net != 0 {
			conserved = false
		}
		deltas = append(deltas, ConservationAssetDelta{
			Asset:       asset,
			TotalDebit:  debit,
			TotalCredit: credit,
			NetDelta:    net,
		})
	}

	hash := computeConservationHash(batchID, deltas)
	return ConservationRecord{
		Version:          ConservationVersion,
		BatchID:          batchID,
		AssetDeltas:      deltas,
		ConservationHash: hash,
		Conserved:        conserved,
	}, nil
}

func computeConservationHash(batchID int64, deltas []ConservationAssetDelta) string {
	var batchBytes [8]byte
	binary.BigEndian.PutUint64(batchBytes[:], uint64(batchID))

	preimage := make([]byte, 0, 256)
	preimage = append(preimage, []byte(ConservationVersion)...)
	preimage = append(preimage, batchBytes[:]...)

	var countBytes [4]byte
	binary.BigEndian.PutUint32(countBytes[:], uint32(len(deltas)))
	preimage = append(preimage, countBytes[:]...)

	for _, delta := range deltas {
		assetHash := crypto.Keccak256([]byte(delta.Asset))
		preimage = append(preimage, assetHash...)

		var netBytes [32]byte
		if delta.NetDelta >= 0 {
			binary.BigEndian.PutUint64(netBytes[24:], uint64(delta.NetDelta))
		} else {
			for i := range netBytes {
				netBytes[i] = 0xff
			}
			binary.BigEndian.PutUint64(netBytes[24:], uint64(delta.NetDelta))
		}
		preimage = append(preimage, netBytes[:]...)
	}

	hash := crypto.Keccak256(preimage)
	return "0x" + hex.EncodeToString(hash)
}

func normalizeConservationAsset(asset string) string {
	return strings.ToUpper(strings.TrimSpace(asset))
}
