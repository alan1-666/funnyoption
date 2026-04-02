package model

type Trade struct {
	Sequence        uint64
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
}
