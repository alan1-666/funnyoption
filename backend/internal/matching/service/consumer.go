package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	"funnyoption/internal/posttrade"
	"funnyoption/internal/shared/fee"
	sharedkafka "funnyoption/internal/shared/kafka"
)

// commandStore is the persistence interface for the legacy CommandProcessor.
type commandStore interface {
	PersistResult(ctx context.Context, command sharedkafka.OrderCommand, result engine.Result) error
	MarketIsTradable(ctx context.Context, marketID int64) (bool, error)
}

// CommandProcessor is the legacy single-goroutine command handler.
// It uses AsyncEngine for matching and delegates post-trade processing to
// the posttrade.Service. Kept for backward compatibility; new deployments
// should use the Pipeline (InputGateway → BookEngine → OutputDispatcher).
type CommandProcessor struct {
	logger  *slog.Logger
	matcher *engine.AsyncEngine
	store   commandStore
	pt      *posttrade.Service
}

func NewCommandProcessor(logger *slog.Logger, matcher *engine.AsyncEngine, publisher sharedkafka.Publisher, topics sharedkafka.Topics, store commandStore) *CommandProcessor {
	candles := NewCandleBook(defaultCandleIntervalMillis, defaultCandleHistoryLimit)
	pt := posttrade.New(logger, publisher, topics, store, candles, fee.DefaultSchedule())
	return &CommandProcessor{
		logger:  logger,
		matcher: matcher,
		store:   store,
		pt:      pt,
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
			return p.pt.ProcessResult(ctx, &posttrade.MatchResult{
				Command:  command,
				Result:   result,
				Rejected: true,
			})
		}
	}

	result, err := p.matcher.Submit(ctx, order)
	if err != nil {
		p.logger.Warn("matching rejected command", "command_id", command.CommandID, "order_id", command.OrderID, "err", err)
	}

	return p.pt.ProcessResult(ctx, &posttrade.MatchResult{
		Command:  command,
		Result:   result,
		Rejected: err != nil,
	})
}
