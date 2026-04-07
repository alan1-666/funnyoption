package pipeline

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

// OutputDispatcher drains MatchResults from the shared fan-in channel
// and performs all IO: DB persist + Kafka publish.
type OutputDispatcher struct {
	logger    *slog.Logger
	outputCh  <-chan MatchResult
	publisher sharedkafka.Publisher
	topics    sharedkafka.Topics
	store     PersistStore
	candles   CandleTracker
	feeSched  fee.Schedule

	dispatched atomic.Uint64
	errors     atomic.Uint64
}

// PersistStore is the interface the dispatcher needs for DB writes.
type PersistStore interface {
	PersistResult(ctx context.Context, command sharedkafka.OrderCommand, result engine.Result) error
}

func NewOutputDispatcher(
	logger *slog.Logger,
	outputCh <-chan MatchResult,
	publisher sharedkafka.Publisher,
	topics sharedkafka.Topics,
	store PersistStore,
	candles CandleTracker,
	feeSched fee.Schedule,
) *OutputDispatcher {
	return &OutputDispatcher{
		logger:    logger,
		outputCh:  outputCh,
		publisher: publisher,
		topics:    topics,
		store:     store,
		candles:   candles,
		feeSched:  feeSched,
	}
}

func (d *OutputDispatcher) Run(ctx context.Context) {
	d.logger.Info("output dispatcher started")
	defer d.logger.Info("output dispatcher stopped")

	for {
		select {
		case <-ctx.Done():
			return
		case mr, ok := <-d.outputCh:
			if !ok {
				return
			}
			if err := d.dispatch(ctx, &mr); err != nil {
				d.errors.Add(1)
				d.logger.Error("dispatcher: failed", "err", err, "order_id", mr.Command.OrderID)
			}
			d.dispatched.Add(1)
		}
	}
}

func (d *OutputDispatcher) dispatch(ctx context.Context, mr *MatchResult) error {
	cmd := mr.Command.ToKafkaCommand()
	result := mr.Result

	if d.store != nil {
		if err := d.store.PersistResult(ctx, cmd, result); err != nil {
			return err
		}
	}

	batch := make([]sharedkafka.BatchItem, 0, 4+len(result.Affected)+len(result.Trades)*4)

	if item := d.buildOrderEvent(cmd, result.Order); item != nil {
		batch = append(batch, *item)
	}
	for _, affected := range result.Affected {
		if item := d.buildPassiveOrderEvent(cmd.TraceID, cmd.CollateralAsset, affected); item != nil {
			batch = append(batch, *item)
		}
	}
	for _, trade := range result.Trades {
		batch = append(batch, d.buildTradeEvent(cmd.TraceID, cmd.CollateralAsset, trade))
		batch = append(batch, d.buildPositionEvents(cmd.TraceID, trade)...)
		batch = append(batch, d.buildCandleEvent(cmd.TraceID, trade))
	}
	batch = append(batch, d.buildQuoteEvents(cmd.TraceID, result)...)

	if len(batch) == 0 {
		return nil
	}
	return d.publisher.PublishJSONBatch(ctx, batch)
}

func (d *OutputDispatcher) buildOrderEvent(command sharedkafka.OrderCommand, order *model.Order) *sharedkafka.BatchItem {
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
	return &sharedkafka.BatchItem{Topic: d.topics.OrderEvent, Key: order.BookKey(), Payload: event}
}

func (d *OutputDispatcher) buildPassiveOrderEvent(traceID, collateralAsset string, order *model.Order) *sharedkafka.BatchItem {
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
	return &sharedkafka.BatchItem{Topic: d.topics.OrderEvent, Key: order.BookKey(), Payload: event}
}

func (d *OutputDispatcher) buildTradeEvent(traceID, collateralAsset string, trade model.Trade) sharedkafka.BatchItem {
	notional := trade.Price * trade.Quantity
	feeResult, err := d.feeSched.Compute(notional)
	if err != nil {
		d.logger.Warn("fee computation failed, using zero fees", "err", err)
		feeResult = fee.FeeResult{}
	}

	event := sharedkafka.TradeMatchedEvent{
		EventID:          sharedkafka.NewID("evt_trade"),
		Sequence:         trade.Sequence,
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
		TakerFeeBps:      d.feeSched.TakerFeeBps,
		MakerFeeBps:      d.feeSched.MakerFeeBps,
		TakerFee:         feeResult.TakerFee,
		MakerFee:         feeResult.MakerFee,
		OccurredAtMillis: trade.MatchedAtMillis,
	}
	return sharedkafka.BatchItem{Topic: d.topics.TradeMatched, Key: trade.BookKey, Payload: event}
}

func (d *OutputDispatcher) buildPositionEvents(traceID string, trade model.Trade) []sharedkafka.BatchItem {
	buyerUserID, sellerUserID := positionSides(trade)
	return []sharedkafka.BatchItem{
		{Topic: d.topics.PositionChange, Key: trade.BookKey, Payload: sharedkafka.PositionChangedEvent{
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
		{Topic: d.topics.PositionChange, Key: trade.BookKey, Payload: sharedkafka.PositionChangedEvent{
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

func (d *OutputDispatcher) buildQuoteEvents(traceID string, result engine.Result) []sharedkafka.BatchItem {
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
		{Topic: d.topics.QuoteDepth, Key: result.Book.Key, Payload: depth},
		{Topic: d.topics.QuoteTicker, Key: result.Book.Key, Payload: ticker},
	}
}

func (d *OutputDispatcher) buildCandleEvent(traceID string, trade model.Trade) sharedkafka.BatchItem {
	if d.candles == nil {
		return sharedkafka.BatchItem{Topic: d.topics.QuoteCandle, Key: trade.BookKey, Payload: sharedkafka.QuoteCandleEvent{}}
	}
	event := d.candles.ApplyTrade(trade)
	event.TraceID = traceID
	return sharedkafka.BatchItem{Topic: d.topics.QuoteCandle, Key: trade.BookKey, Payload: event}
}

func (d *OutputDispatcher) Stats() (dispatched, errors uint64) {
	return d.dispatched.Load(), d.errors.Load()
}
