package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	"funnyoption/internal/shared/assets"
	"funnyoption/internal/shared/fee"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type commandStore interface {
	PersistResult(ctx context.Context, command sharedkafka.OrderCommand, result engine.Result) error
	MarketIsTradable(ctx context.Context, marketID int64) (bool, error)
}

type CommandProcessor struct {
	logger      *slog.Logger
	matcher     *engine.AsyncEngine
	publisher   sharedkafka.Publisher
	topics      sharedkafka.Topics
	store       commandStore
	candles     *CandleBook
	feeSchedule fee.Schedule
}

func NewCommandProcessor(logger *slog.Logger, matcher *engine.AsyncEngine, publisher sharedkafka.Publisher, topics sharedkafka.Topics, store commandStore) *CommandProcessor {
	return &CommandProcessor{
		logger:      logger,
		matcher:     matcher,
		publisher:   publisher,
		topics:      topics,
		store:       store,
		feeSchedule: fee.DefaultSchedule(),
		candles:   NewCandleBook(defaultCandleIntervalMillis, defaultCandleHistoryLimit),
	}
}

func (p *CommandProcessor) HandleOrderCommand(ctx context.Context, msg sharedkafka.Message) error {
	var command sharedkafka.OrderCommand
	if err := json.Unmarshal(msg.Value, &command); err != nil {
		return err
	}

	order := &model.Order{
		OrderID:         command.OrderID,
		ClientOrderID:   command.ClientOrderID,
		UserID:          command.UserID,
		MarketID:        command.MarketID,
		Outcome:         command.Outcome,
		Side:            model.OrderSide(command.Side),
		Type:            model.OrderType(command.Type),
		TimeInForce:     model.TimeInForce(command.TimeInForce),
		Price:           command.Price,
		Quantity:        command.Quantity,
		CreatedAtMillis: command.RequestedAtMillis,
		UpdatedAtMillis: time.Now().UnixMilli(),
	}

	if p.store != nil {
		tradable, err := p.store.MarketIsTradable(ctx, command.MarketID)
		if err != nil {
			return err
		}
		if !tradable {
			order.Reject(model.CancelReasonMarketNotTradable)
			result := engine.Result{Order: order}
			if err := p.store.PersistResult(ctx, command, result); err != nil {
				return err
			}
			p.logger.Warn("matching rejected command for non-tradable market", "command_id", command.CommandID, "order_id", command.OrderID, "market_id", command.MarketID)
			return p.publishOrderEvent(ctx, command, result.Order)
		}
	}

	result, err := p.matcher.Submit(ctx, order)
	if err != nil {
		p.logger.Warn("matching rejected command", "command_id", command.CommandID, "order_id", command.OrderID, "err", err)
		return p.publishOrderEvent(ctx, command, result.Order)
	}

	if p.store != nil {
		if err := p.store.PersistResult(ctx, command, result); err != nil {
			return err
		}
	}

	batch := make([]sharedkafka.BatchItem, 0, 4+len(result.Affected)+len(result.Trades)*4)
	if item, err := p.buildOrderEvent(command, result.Order); err != nil {
		return err
	} else if item != nil {
		batch = append(batch, *item)
	}
	for _, affected := range result.Affected {
		if item, err := p.buildPassiveOrderEvent(command.TraceID, command.CollateralAsset, affected); err != nil {
			return err
		} else if item != nil {
			batch = append(batch, *item)
		}
	}
	for _, trade := range result.Trades {
		batch = append(batch, p.buildTradeEvent(command.TraceID, command.CollateralAsset, trade))
		batch = append(batch, p.buildPositionEvents(command.TraceID, trade)...)
		batch = append(batch, p.buildCandleEvent(command.TraceID, trade))
	}
	batch = append(batch, p.buildQuoteEvents(command.TraceID, result)...)
	return p.publisher.PublishJSONBatch(ctx, batch)
}

func (p *CommandProcessor) buildOrderEvent(command sharedkafka.OrderCommand, order *model.Order) (*sharedkafka.BatchItem, error) {
	if order == nil {
		return nil, nil
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
	return &sharedkafka.BatchItem{Topic: p.topics.OrderEvent, Key: order.BookKey(), Payload: event}, nil
}

func (p *CommandProcessor) publishOrderEvent(ctx context.Context, command sharedkafka.OrderCommand, order *model.Order) error {
	item, err := p.buildOrderEvent(command, order)
	if err != nil || item == nil {
		return err
	}
	return p.publisher.PublishJSON(ctx, item.Topic, item.Key, item.Payload)
}

func (p *CommandProcessor) buildPassiveOrderEvent(traceID, collateralAsset string, order *model.Order) (*sharedkafka.BatchItem, error) {
	if order == nil {
		return nil, nil
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
	return &sharedkafka.BatchItem{Topic: p.topics.OrderEvent, Key: order.BookKey(), Payload: event}, nil
}

func (p *CommandProcessor) buildTradeEvent(traceID, collateralAsset string, trade model.Trade) sharedkafka.BatchItem {
	notional := trade.Price * trade.Quantity
	feeResult, err := p.feeSchedule.Compute(notional)
	if err != nil {
		p.logger.Warn("fee computation failed, using zero fees", "err", err)
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
		TakerFeeBps:      p.feeSchedule.TakerFeeBps,
		MakerFeeBps:      p.feeSchedule.MakerFeeBps,
		TakerFee:         feeResult.TakerFee,
		MakerFee:         feeResult.MakerFee,
		OccurredAtMillis: trade.MatchedAtMillis,
	}
	return sharedkafka.BatchItem{Topic: p.topics.TradeMatched, Key: trade.BookKey, Payload: event}
}

func (p *CommandProcessor) buildPositionEvents(traceID string, trade model.Trade) []sharedkafka.BatchItem {
	buyerUserID, sellerUserID := positionSides(trade)
	return []sharedkafka.BatchItem{
		{Topic: p.topics.PositionChange, Key: trade.BookKey, Payload: sharedkafka.PositionChangedEvent{
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
		{Topic: p.topics.PositionChange, Key: trade.BookKey, Payload: sharedkafka.PositionChangedEvent{
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

func (p *CommandProcessor) buildQuoteEvents(traceID string, result engine.Result) []sharedkafka.BatchItem {
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
		{Topic: p.topics.QuoteDepth, Key: result.Book.Key, Payload: depth},
		{Topic: p.topics.QuoteTicker, Key: result.Book.Key, Payload: ticker},
	}
}

func (p *CommandProcessor) publishQuoteEvents(ctx context.Context, traceID string, result engine.Result) error {
	items := p.buildQuoteEvents(traceID, result)
	return p.publisher.PublishJSONBatch(ctx, items)
}

func (p *CommandProcessor) buildCandleEvent(traceID string, trade model.Trade) sharedkafka.BatchItem {
	if p.candles == nil {
		return sharedkafka.BatchItem{Topic: p.topics.QuoteCandle, Key: trade.BookKey, Payload: sharedkafka.QuoteCandleEvent{}}
	}
	event := p.candles.ApplyTrade(trade)
	event.TraceID = traceID
	return sharedkafka.BatchItem{Topic: p.topics.QuoteCandle, Key: trade.BookKey, Payload: event}
}
