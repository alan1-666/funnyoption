package model

import "fmt"

type Trade struct {
	Sequence        uint64
	TradeID         string
	MarketID        int64
	Outcome         string
	BookKey         string
	Price           int64
	Quantity        int64
	TakerOrderID    string
	MakerOrderID    string
	TakerUserID     int64
	MakerUserID     int64
	TakerSide       OrderSide
	MakerSide       OrderSide
	MatchedAtMillis int64
	EpochID         uint64
}

// DeterministicTradeID generates a deterministic trade ID from bookKey and
// per-book local sequence. Identical command sequences produce identical IDs,
// enabling safe Kafka replay on standby nodes.
func DeterministicTradeID(bookKey string, localSeq uint64) string {
	return fmt.Sprintf("%s:%08d", bookKey, localSeq)
}
