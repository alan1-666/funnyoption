package model

import "strconv"

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
// Uses strconv.AppendUint instead of fmt.Sprintf for lower overhead.
func DeterministicTradeID(bookKey string, localSeq uint64) string {
	// bookKey + ":" + 8-digit zero-padded sequence
	buf := make([]byte, 0, len(bookKey)+9)
	buf = append(buf, bookKey...)
	buf = append(buf, ':')
	// Compute digit count for zero-padding.
	start := len(buf)
	buf = strconv.AppendUint(buf, localSeq, 10)
	digits := len(buf) - start
	if pad := 8 - digits; pad > 0 {
		// Shift digits right and insert zeros.
		buf = append(buf, make([]byte, pad)...)
		copy(buf[start+pad:], buf[start:start+digits])
		for i := 0; i < pad; i++ {
			buf[start+i] = '0'
		}
	}
	return string(buf)
}
