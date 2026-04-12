package kafka

type OrderCommand struct {
	CommandID         string `json:"command_id"`
	TraceID           string `json:"trace_id,omitempty"`
	OrderID           string `json:"order_id"`
	ClientOrderID     string `json:"client_order_id,omitempty"`
	FreezeID          string `json:"freeze_id,omitempty"`
	FreezeAsset       string `json:"freeze_asset,omitempty"`
	FreezeAmount      int64  `json:"freeze_amount,omitempty"`
	CollateralAsset   string `json:"collateral_asset"`
	UserID            int64  `json:"user_id"`
	MarketID          int64  `json:"market_id"`
	Outcome           string `json:"outcome"`
	BookKey           string `json:"book_key,omitempty"`
	Side              string `json:"side"`
	Type              string `json:"type"`
	TimeInForce       string `json:"time_in_force"`
	STPStrategy       string `json:"stp_strategy,omitempty"`
	Price             int64  `json:"price"`
	Quantity          int64  `json:"quantity"`
	RequestedAtMillis int64  `json:"requested_at_millis"`
}

type OrderEvent struct {
	EventID           string `json:"event_id"`
	CommandID         string `json:"command_id"`
	TraceID           string `json:"trace_id,omitempty"`
	OrderID           string `json:"order_id"`
	ClientOrderID     string `json:"client_order_id,omitempty"`
	FreezeID          string `json:"freeze_id,omitempty"`
	FreezeAsset       string `json:"freeze_asset,omitempty"`
	FreezeAmount      int64  `json:"freeze_amount,omitempty"`
	CollateralAsset   string `json:"collateral_asset"`
	UserID            int64  `json:"user_id"`
	MarketID          int64  `json:"market_id"`
	Outcome           string `json:"outcome"`
	Side              string `json:"side"`
	Type              string `json:"type"`
	TimeInForce       string `json:"time_in_force"`
	Status            string `json:"status"`
	CancelReason      string `json:"cancel_reason,omitempty"`
	Price             int64  `json:"price"`
	Quantity          int64  `json:"quantity"`
	FilledQuantity    int64  `json:"filled_quantity"`
	RemainingQuantity int64  `json:"remaining_quantity"`
	OccurredAtMillis  int64  `json:"occurred_at_millis"`
}

type TradeMatchedEvent struct {
	EventID          string `json:"event_id"`
	TradeID          string `json:"trade_id"`
	Sequence         uint64 `json:"sequence"`
	EpochID          uint64 `json:"epoch_id,omitempty"`
	TraceID          string `json:"trace_id,omitempty"`
	CollateralAsset  string `json:"collateral_asset"`
	MarketID         int64  `json:"market_id"`
	Outcome          string `json:"outcome"`
	BookKey          string `json:"book_key"`
	Price            int64  `json:"price"`
	Quantity         int64  `json:"quantity"`
	TakerOrderID     string `json:"taker_order_id"`
	MakerOrderID     string `json:"maker_order_id"`
	TakerUserID      int64  `json:"taker_user_id"`
	MakerUserID      int64  `json:"maker_user_id"`
	TakerSide        string `json:"taker_side"`
	MakerSide        string `json:"maker_side"`
	TakerFeeBps      int64  `json:"taker_fee_bps"`
	MakerFeeBps      int64  `json:"maker_fee_bps"`
	TakerFee         int64  `json:"taker_fee"`
	MakerFee         int64  `json:"maker_fee"`
	OccurredAtMillis int64  `json:"occurred_at_millis"`
}

type PositionChangedEvent struct {
	EventID          string `json:"event_id"`
	TraceID          string `json:"trace_id,omitempty"`
	SourceTradeID    string `json:"source_trade_id"`
	UserID           int64  `json:"user_id"`
	MarketID         int64  `json:"market_id"`
	Outcome          string `json:"outcome"`
	PositionAsset    string `json:"position_asset"`
	DeltaQuantity    int64  `json:"delta_quantity"`
	OccurredAtMillis int64  `json:"occurred_at_millis"`
}

type MarketEvent struct {
	EventID          string `json:"event_id"`
	TraceID          string `json:"trace_id,omitempty"`
	MarketID         int64  `json:"market_id"`
	Status           string `json:"status"`
	ResolvedOutcome  string `json:"resolved_outcome,omitempty"`
	OccurredAtMillis int64  `json:"occurred_at_millis"`
}

type SettlementCompletedEvent struct {
	EventID          string `json:"event_id"`
	MarketID         int64  `json:"market_id"`
	UserID           int64  `json:"user_id"`
	WinningOutcome   string `json:"winning_outcome"`
	PositionAsset    string `json:"position_asset"`
	SettledQuantity  int64  `json:"settled_quantity"`
	PayoutAsset      string `json:"payout_asset"`
	PayoutAmount     int64  `json:"payout_amount"`
	OccurredAtMillis int64  `json:"occurred_at_millis"`
}

type QuoteLevel struct {
	Price    int64 `json:"price"`
	Quantity int64 `json:"quantity"`
}

type QuoteDepthEvent struct {
	EventID          string       `json:"event_id"`
	TraceID          string       `json:"trace_id,omitempty"`
	MarketID         int64        `json:"market_id"`
	Outcome          string       `json:"outcome"`
	BookKey          string       `json:"book_key"`
	Bids             []QuoteLevel `json:"bids"`
	Asks             []QuoteLevel `json:"asks"`
	OccurredAtMillis int64        `json:"occurred_at_millis"`
}

type QuoteTickerEvent struct {
	EventID          string `json:"event_id"`
	TraceID          string `json:"trace_id,omitempty"`
	MarketID         int64  `json:"market_id"`
	Outcome          string `json:"outcome"`
	BookKey          string `json:"book_key"`
	LastPrice        int64  `json:"last_price"`
	LastQuantity     int64  `json:"last_quantity"`
	BestBid          int64  `json:"best_bid"`
	BestAsk          int64  `json:"best_ask"`
	OccurredAtMillis int64  `json:"occurred_at_millis"`
}

type QuoteCandle struct {
	BucketStartMillis int64 `json:"bucket_start_millis"`
	BucketEndMillis   int64 `json:"bucket_end_millis"`
	Open              int64 `json:"open"`
	High              int64 `json:"high"`
	Low               int64 `json:"low"`
	Close             int64 `json:"close"`
	Volume            int64 `json:"volume"`
	TradeCount        int64 `json:"trade_count"`
}

type QuoteCandleEvent struct {
	EventID          string        `json:"event_id"`
	TraceID          string        `json:"trace_id,omitempty"`
	MarketID         int64         `json:"market_id"`
	Outcome          string        `json:"outcome"`
	BookKey          string        `json:"book_key"`
	IntervalSec      int64         `json:"interval_sec"`
	Candles          []QuoteCandle `json:"candles"`
	OccurredAtMillis int64         `json:"occurred_at_millis"`
}

type NotificationCreatedEvent struct {
	NotificationID int64  `json:"notification_id"`
	UserID         int64  `json:"user_id"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	CreatedAt      int64  `json:"created_at"`
}

type CustodyDepositEvent struct {
	DepositID    string `json:"deposit_id"`
	UserID       int64  `json:"user_id"`
	Asset        string `json:"asset"`
	CreditAsset  string `json:"credit_asset"`
	CreditAmount int64  `json:"credit_amount"`
	ChainAmount  string `json:"chain_amount"`
	TxHash       string `json:"tx_hash"`
	CreatedAt    int64  `json:"created_at"`
}
