package posttrade

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	"funnyoption/internal/shared/assets"
	"funnyoption/internal/shared/fee"
	sharedkafka "funnyoption/internal/shared/kafka"
)

// CandleTracker applies trades and returns candle events.
type CandleTracker interface {
	ApplyTrade(trade model.Trade) sharedkafka.QuoteCandleEvent
}

// PersistItem pairs a command with its engine result for batch persistence.
type PersistItem struct {
	Command sharedkafka.OrderCommand
	Result  engine.Result
}

// PersistStore is the interface the post-trade service needs for DB writes.
type PersistStore interface {
	PersistResult(ctx context.Context, command sharedkafka.OrderCommand, result engine.Result) error
	PersistBatch(ctx context.Context, items []PersistItem) error
}

// MatchResult is the input the post-trade service processes.
// It mirrors the pipeline's MatchResult but is owned by this package to avoid
// a circular import. The pipeline adapter converts between the two.
type MatchResult struct {
	Command  sharedkafka.OrderCommand
	Result   engine.Result
	Rejected bool
	EpochID  uint64
}

// Service performs all post-trade IO: persistence, fee computation, position
// events, quote events, and candle events. It is decoupled from the matching
// engine so the engine hot path is pure in-memory matching.
type Service struct {
	logger    *slog.Logger
	publisher sharedkafka.Publisher
	topics    sharedkafka.Topics
	store     PersistStore
	candles   CandleTracker
	feeSched  fee.Schedule

	dispatched atomic.Uint64
	errors     atomic.Uint64
}

func New(
	logger *slog.Logger,
	publisher sharedkafka.Publisher,
	topics sharedkafka.Topics,
	store PersistStore,
	candles CandleTracker,
	feeSched fee.Schedule,
) *Service {
	return &Service{
		logger:    logger,
		publisher: publisher,
		topics:    topics,
		store:     store,
		candles:   candles,
		feeSched:  feeSched,
	}
}

// ProcessResult handles a single match result: persist + publish downstream events.
func (s *Service) ProcessResult(ctx context.Context, mr *MatchResult) error {
	cmd := mr.Command
	result := mr.Result

	if s.store != nil {
		if err := s.store.PersistResult(ctx, cmd, result); err != nil {
			return err
		}
	}

	batch := make([]sharedkafka.BatchItem, 0, 4+len(result.Affected)+len(result.Trades)*4)

	if item := s.buildOrderEvent(cmd, result.Order); item != nil {
		batch = append(batch, *item)
	}
	for _, affected := range result.Affected {
		if item := s.buildPassiveOrderEvent(cmd.TraceID, cmd.CollateralAsset, affected); item != nil {
			batch = append(batch, *item)
		}
	}
	for _, trade := range result.Trades {
		batch = append(batch, s.buildTradeEvent(cmd.TraceID, cmd.CollateralAsset, trade))
		batch = append(batch, s.buildPositionEvents(cmd.TraceID, trade)...)
		batch = append(batch, s.buildCandleEvent(cmd.TraceID, trade))
	}
	batch = append(batch, s.buildQuoteEvents(cmd.TraceID, result)...)

	if len(batch) == 0 {
		return nil
	}
	return s.publisher.PublishJSONBatch(ctx, batch)
}

func (s *Service) buildOrderEvent(command sharedkafka.OrderCommand, order *model.Order) *sharedkafka.BatchItem {
	if order == nil {
		return nil
	}
	event := sharedkafka.OrderEvent{
		EventID:           sharedkafka.NewID("evt_order"),
		CommandID:         command.CommandID,
		TraceID:           command.TraceID,
		OrderID:           order.OrderID,
		ClientOrderID:     order.ClientOrderID,
		FreezeID:          command.FreezeID,
		FreezeAsset:       command.FreezeAsset,
		FreezeAmount:      command.FreezeAmount,
		CollateralAsset:   command.CollateralAsset,
		UserID:            order.UserID,
		MarketID:          order.MarketID,
		Outcome:           order.Outcome,
		Side:              string(order.Side),
		Type:              string(order.Type),
		TimeInForce:       string(order.TimeInForce),
		Status:            string(order.Status),
		CancelReason:      string(order.CancelReason),
		Price:             order.Price,
		Quantity:          order.Quantity,
		FilledQuantity:    order.FilledQuantity,
		RemainingQuantity: order.RemainingQuantity(),
		OccurredAtMillis:  time.Now().UnixMilli(),
	}
	return &sharedkafka.BatchItem{Topic: s.topics.OrderEvent, Key: order.BookKey(), Payload: event}
}

func (s *Service) buildPassiveOrderEvent(traceID, collateralAsset string, order *model.Order) *sharedkafka.BatchItem {
	if order == nil {
		return nil
	}
	event := sharedkafka.OrderEvent{
		EventID:           sharedkafka.NewID("evt_order"),
		TraceID:           traceID,
		OrderID:           order.OrderID,
		ClientOrderID:     order.ClientOrderID,
		CollateralAsset:   collateralAsset,
		UserID:            order.UserID,
		MarketID:          order.MarketID,
		Outcome:           order.Outcome,
		Side:              string(order.Side),
		Type:              string(order.Type),
		TimeInForce:       string(order.TimeInForce),
		Status:            string(order.Status),
		CancelReason:      string(order.CancelReason),
		Price:             order.Price,
		Quantity:          order.Quantity,
		FilledQuantity:    order.FilledQuantity,
		RemainingQuantity: order.RemainingQuantity(),
		OccurredAtMillis:  time.Now().UnixMilli(),
	}
	return &sharedkafka.BatchItem{Topic: s.topics.OrderEvent, Key: order.BookKey(), Payload: event}
}

func (s *Service) buildTradeEvent(traceID, collateralAsset string, trade model.Trade) sharedkafka.BatchItem {
	notional := trade.Price * trade.Quantity
	feeResult, err := s.feeSched.Compute(notional)
	if err != nil {
		s.logger.Warn("fee computation failed, using zero fees", "err", err)
		feeResult = fee.FeeResult{}
	}

	event := sharedkafka.TradeMatchedEvent{
		EventID:          sharedkafka.NewID("evt_trade"),
		TradeID:          trade.TradeID,
		Sequence:         trade.Sequence,
		EpochID:          trade.EpochID,
		TraceID:          traceID,
		CollateralAsset:  collateralAsset,
		MarketID:         trade.MarketID,
		Outcome:          trade.Outcome,
		BookKey:          trade.BookKey,
		Price:            trade.Price,
		Quantity:         trade.Quantity,
		TakerOrderID:     trade.TakerOrderID,
		MakerOrderID:     trade.MakerOrderID,
		TakerUserID:      trade.TakerUserID,
		MakerUserID:      trade.MakerUserID,
		TakerSide:        string(trade.TakerSide),
		MakerSide:        string(trade.MakerSide),
		TakerFeeBps:      s.feeSched.TakerFeeBps,
		MakerFeeBps:      s.feeSched.MakerFeeBps,
		TakerFee:         feeResult.TakerFee,
		MakerFee:         feeResult.MakerFee,
		OccurredAtMillis: trade.MatchedAtMillis,
	}
	return sharedkafka.BatchItem{Topic: s.topics.TradeMatched, Key: trade.BookKey, Payload: event}
}

func (s *Service) buildPositionEvents(traceID string, trade model.Trade) []sharedkafka.BatchItem {
	buyerUserID, sellerUserID := positionSides(trade)
	return []sharedkafka.BatchItem{
		{Topic: s.topics.PositionChange, Key: trade.BookKey, Payload: sharedkafka.PositionChangedEvent{
			EventID:          sharedkafka.NewID("evt_pos"),
			TraceID:          traceID,
			SourceTradeID:    trade.TakerOrderID + ":" + trade.MakerOrderID,
			UserID:           buyerUserID,
			MarketID:         trade.MarketID,
			Outcome:          trade.Outcome,
			PositionAsset:    assets.PositionAsset(trade.MarketID, trade.Outcome),
			DeltaQuantity:    trade.Quantity,
			OccurredAtMillis: trade.MatchedAtMillis,
		}},
		{Topic: s.topics.PositionChange, Key: trade.BookKey, Payload: sharedkafka.PositionChangedEvent{
			EventID:          sharedkafka.NewID("evt_pos"),
			TraceID:          traceID,
			SourceTradeID:    trade.TakerOrderID + ":" + trade.MakerOrderID,
			UserID:           sellerUserID,
			MarketID:         trade.MarketID,
			Outcome:          trade.Outcome,
			PositionAsset:    assets.PositionAsset(trade.MarketID, trade.Outcome),
			DeltaQuantity:    -trade.Quantity,
			OccurredAtMillis: trade.MatchedAtMillis,
		}},
	}
}

func positionSides(trade model.Trade) (buyerUserID, sellerUserID int64) {
	if trade.TakerSide == model.OrderSideBuy {
		return trade.TakerUserID, trade.MakerUserID
	}
	return trade.MakerUserID, trade.TakerUserID
}

func (s *Service) buildQuoteEvents(traceID string, result engine.Result) []sharedkafka.BatchItem {
	depth := sharedkafka.QuoteDepthEvent{
		EventID:          sharedkafka.NewID("evt_depth"),
		TraceID:          traceID,
		MarketID:         result.Book.MarketID,
		Outcome:          result.Book.Outcome,
		BookKey:          result.Book.Key,
		Bids:             make([]sharedkafka.QuoteLevel, 0, len(result.Book.Bids)),
		Asks:             make([]sharedkafka.QuoteLevel, 0, len(result.Book.Asks)),
		OccurredAtMillis: time.Now().UnixMilli(),
	}
	for _, level := range result.Book.Bids {
		depth.Bids = append(depth.Bids, sharedkafka.QuoteLevel{Price: level.Price, Quantity: level.Quantity})
	}
	for _, level := range result.Book.Asks {
		depth.Asks = append(depth.Asks, sharedkafka.QuoteLevel{Price: level.Price, Quantity: level.Quantity})
	}

	ticker := sharedkafka.QuoteTickerEvent{
		EventID:          sharedkafka.NewID("evt_ticker"),
		TraceID:          traceID,
		MarketID:         result.Book.MarketID,
		Outcome:          result.Book.Outcome,
		BookKey:          result.Book.Key,
		BestBid:          result.Book.BestBid,
		BestAsk:          result.Book.BestAsk,
		OccurredAtMillis: time.Now().UnixMilli(),
	}
	if tradeCount := len(result.Trades); tradeCount > 0 {
		lastTrade := result.Trades[tradeCount-1]
		ticker.LastPrice = lastTrade.Price
		ticker.LastQuantity = lastTrade.Quantity
	}
	return []sharedkafka.BatchItem{
		{Topic: s.topics.QuoteDepth, Key: result.Book.Key, Payload: depth},
		{Topic: s.topics.QuoteTicker, Key: result.Book.Key, Payload: ticker},
	}
}

func (s *Service) buildCandleEvent(traceID string, trade model.Trade) sharedkafka.BatchItem {
	if s.candles == nil {
		return sharedkafka.BatchItem{Topic: s.topics.QuoteCandle, Key: trade.BookKey, Payload: sharedkafka.QuoteCandleEvent{}}
	}
	event := s.candles.ApplyTrade(trade)
	event.TraceID = traceID
	return sharedkafka.BatchItem{Topic: s.topics.QuoteCandle, Key: trade.BookKey, Payload: event}
}

// ProcessBatch handles multiple match results in one DB transaction + one Kafka publish.
func (s *Service) ProcessBatch(ctx context.Context, mrs []*MatchResult) error {
	if len(mrs) == 0 {
		return nil
	}
	// Fast path: single item, use existing ProcessResult.
	if len(mrs) == 1 {
		return s.ProcessResult(ctx, mrs[0])
	}

	// Batch persist: one tx for all results.
	if s.store != nil {
		items := make([]PersistItem, len(mrs))
		for i, mr := range mrs {
			items[i] = PersistItem{Command: mr.Command, Result: mr.Result}
		}
		if err := s.store.PersistBatch(ctx, items); err != nil {
			return err
		}
	}

	// Build one mega-batch of Kafka events.
	batch := make([]sharedkafka.BatchItem, 0, len(mrs)*6)
	for _, mr := range mrs {
		cmd := mr.Command
		result := mr.Result

		if item := s.buildOrderEvent(cmd, result.Order); item != nil {
			batch = append(batch, *item)
		}
		for _, affected := range result.Affected {
			if item := s.buildPassiveOrderEvent(cmd.TraceID, cmd.CollateralAsset, affected); item != nil {
				batch = append(batch, *item)
			}
		}
		for _, trade := range result.Trades {
			batch = append(batch, s.buildTradeEvent(cmd.TraceID, cmd.CollateralAsset, trade))
			batch = append(batch, s.buildPositionEvents(cmd.TraceID, trade)...)
			batch = append(batch, s.buildCandleEvent(cmd.TraceID, trade))
		}
		batch = append(batch, s.buildQuoteEvents(cmd.TraceID, result)...)
	}

	if len(batch) == 0 {
		return nil
	}
	return s.publisher.PublishJSONBatch(ctx, batch)
}

// Stats returns counters for monitoring.
func (s *Service) Stats() (dispatched, errors uint64) {
	return s.dispatched.Load(), s.errors.Load()
}
