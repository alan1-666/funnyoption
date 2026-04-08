package ha

import "funnyoption/internal/matching/model"

// DepthDelta represents a single price level change.
type DepthDelta struct {
	Price    int64 `json:"price"`
	Quantity int64 `json:"quantity"` // 0 means level removed
}

// DepthDiff represents incremental depth changes between two snapshots.
type DepthDiff struct {
	BookKey    string       `json:"book_key"`
	BidDeltas  []DepthDelta `json:"bid_deltas,omitempty"`
	AskDeltas  []DepthDelta `json:"ask_deltas,omitempty"`
	NewBestBid int64        `json:"new_best_bid"`
	NewBestAsk int64        `json:"new_best_ask"`
	SequenceID uint64       `json:"sequence_id"`
}

// ComputeDepthDiff compares an old snapshot and a new snapshot, returning
// only the price levels that have changed. Downstream subscribers apply
// deltas instead of replacing the entire book, reducing bandwidth.
func ComputeDepthDiff(old, new model.BookSnapshot, seqID uint64) DepthDiff {
	diff := DepthDiff{
		BookKey:    new.Key,
		NewBestBid: new.BestBid,
		NewBestAsk: new.BestAsk,
		SequenceID: seqID,
	}

	oldBids := indexLevels(old.Bids)
	newBids := indexLevels(new.Bids)
	diff.BidDeltas = computeLevelDiff(oldBids, newBids)

	oldAsks := indexLevels(old.Asks)
	newAsks := indexLevels(new.Asks)
	diff.AskDeltas = computeLevelDiff(oldAsks, newAsks)

	return diff
}

func indexLevels(levels []model.BookLevel) map[int64]int64 {
	m := make(map[int64]int64, len(levels))
	for _, l := range levels {
		m[l.Price] = l.Quantity
	}
	return m
}

func computeLevelDiff(old, new map[int64]int64) []DepthDelta {
	var deltas []DepthDelta

	for price, newQty := range new {
		if oldQty, exists := old[price]; !exists || oldQty != newQty {
			deltas = append(deltas, DepthDelta{Price: price, Quantity: newQty})
		}
	}
	for price := range old {
		if _, exists := new[price]; !exists {
			deltas = append(deltas, DepthDelta{Price: price, Quantity: 0})
		}
	}

	return deltas
}
