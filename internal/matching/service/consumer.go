package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	"funnyoption/internal/shared/assets"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type commandStore interface {
	PersistResult(ctx context.Context, command sharedkafka.OrderCommand, result engine.Result) error
	MarketIsTradable(ctx context.Context, marketID int64) (bool, error)
}

type CommandProcessor struct {
	logger    *slog.Logger
	matcher   *engine.AsyncEngine
	publisher sharedkafka.Publisher
	topics    sharedkafka.Topics
	store     commandStore
	candles   *CandleBook
}

func NewCommandProcessor(logger *slog.Logger, matcher *engine.AsyncEngine, publisher sharedkafka.Publisher, topics sharedkafka.Topics, store commandStore) *CommandProcessor {
	return &CommandProcessor{
		logger:    logger,
		matcher:   matcher,
		publisher: publisher,
		topics:    topics,
		store:     store,
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

	if err := p.publishOrderEvent(ctx, command, result.Order); err != nil {
		return err
	}
	for _, affected := range result.Affected {
		if err := p.publishPassiveOrderEvent(ctx, command.TraceID, command.CollateralAsset, affected); err != nil {
			return err
		}
	}
	for _, trade := range result.Trades {
		if err := p.publishTradeEvent(ctx, command.TraceID, command.CollateralAsset, trade); err != nil {
			return err
		}
		if err := p.publishPositionEvents(ctx, command.TraceID, trade); err != nil {
			return err
		}
		if err := p.publishCandleEvent(ctx, command.TraceID, trade); err != nil {
			return err
		}
	}
	if err := p.publishQuoteEvents(ctx, command.TraceID, result); err != nil {
		return err
	}
	return nil
}

func (p *CommandProcessor) publishOrderEvent(ctx context.Context, command sharedkafka.OrderCommand, order *model.Order) error {
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
	return p.publisher.PublishJSON(ctx, p.topics.OrderEvent, order.BookKey(), event)
}

func (p *CommandProcessor) publishPassiveOrderEvent(ctx context.Context, traceID, collateralAsset string, order *model.Order) error {
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
	return p.publisher.PublishJSON(ctx, p.topics.OrderEvent, order.BookKey(), event)
}

func (p *CommandProcessor) publishTradeEvent(ctx context.Context, traceID, collateralAsset string, trade model.Trade) error {
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
		OccurredAtMillis: trade.MatchedAtMillis,
	}
	return p.publisher.PublishJSON(ctx, p.topics.TradeMatched, trade.BookKey, event)
}

func (p *CommandProcessor) publishPositionEvents(ctx context.Context, traceID string, trade model.Trade) error {
	buyerUserID, sellerUserID := positionSides(trade)
	events := []sharedkafka.PositionChangedEvent{
		{
			EventID:          sharedkafka.NewID("evt_pos"),
			TraceID:          traceID,
			SourceTradeID:    trade.TakerOrderID + ":" + trade.MakerOrderID,
			UserID:           buyerUserID,
			MarketID:         trade.MarketID,
			Outcome:          trade.Outcome,
			PositionAsset:    assets.PositionAsset(trade.MarketID, trade.Outcome),
			DeltaQuantity:    trade.Quantity,
			OccurredAtMillis: trade.MatchedAtMillis,
		},
		{
			EventID:          sharedkafka.NewID("evt_pos"),
			TraceID:          traceID,
			SourceTradeID:    trade.TakerOrderID + ":" + trade.MakerOrderID,
			UserID:           sellerUserID,
			MarketID:         trade.MarketID,
			Outcome:          trade.Outcome,
			PositionAsset:    assets.PositionAsset(trade.MarketID, trade.Outcome),
			DeltaQuantity:    -trade.Quantity,
			OccurredAtMillis: trade.MatchedAtMillis,
		},
	}

	for _, event := range events {
		if err := p.publisher.PublishJSON(ctx, p.topics.PositionChange, trade.BookKey, event); err != nil {
			return err
		}
	}
	return nil
}

func positionSides(trade model.Trade) (buyerUserID, sellerUserID int64) {
	if trade.TakerSide == model.OrderSideBuy {
		return trade.TakerUserID, trade.MakerUserID
	}
	return trade.MakerUserID, trade.TakerUserID
}

func (p *CommandProcessor) publishQuoteEvents(ctx context.Context, traceID string, result engine.Result) error {
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
	if err := p.publisher.PublishJSON(ctx, p.topics.QuoteDepth, result.Book.Key, depth); err != nil {
		return err
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
	return p.publisher.PublishJSON(ctx, p.topics.QuoteTicker, result.Book.Key, ticker)
}

func (p *CommandProcessor) publishCandleEvent(ctx context.Context, traceID string, trade model.Trade) error {
	if p.candles == nil {
		return nil
	}
	event := p.candles.ApplyTrade(trade)
	event.TraceID = traceID
	return p.publisher.PublishJSON(ctx, p.topics.QuoteCandle, trade.BookKey, event)
}
