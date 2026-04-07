package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	sharedkafka "funnyoption/internal/shared/kafka"
)

const defaultOrderExpirySweepInterval = 2 * time.Second

type orderExpiryStore interface {
	LoadExpiredRestingOrders(ctx context.Context, nowUnix int64) ([]ExpiredRestingOrder, error)
	PersistResult(ctx context.Context, command sharedkafka.OrderCommand, result engine.Result) error
}

type orderExpirySweeper struct {
	logger    *slog.Logger
	matcher   *engine.AsyncEngine
	store     orderExpiryStore
	processor *CommandProcessor
	interval  time.Duration
}

func newOrderExpirySweeper(
	logger *slog.Logger,
	matcher *engine.AsyncEngine,
	store orderExpiryStore,
	publisher sharedkafka.Publisher,
	topics sharedkafka.Topics,
) *orderExpirySweeper {
	return &orderExpirySweeper{
		logger:    logger,
		matcher:   matcher,
		store:     store,
		processor: NewCommandProcessor(logger, matcher, publisher, topics, nil),
		interval:  defaultOrderExpirySweepInterval,
	}
}

func (s *orderExpirySweeper) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.sweepOnce(ctx, time.Now()); err != nil {
					s.logger.Warn("matching close_at sweep failed", "err", err)
				}
			}
		}
	}()
}

func (s *orderExpirySweeper) sweepOnce(ctx context.Context, now time.Time) error {
	items, err := s.store.LoadExpiredRestingOrders(ctx, now.Unix())
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}

	orders := make([]*model.Order, 0, len(items))
	byOrderID := make(map[string]ExpiredRestingOrder, len(items))
	for _, item := range items {
		if item.Order == nil {
			continue
		}
		orders = append(orders, item.Order)
		byOrderID[item.Order.OrderID] = item
	}
	if len(orders) == 0 {
		return nil
	}

	cancelled, err := s.matcher.CancelOrders(ctx, orders, model.CancelReasonMarketClosed)
	if err != nil {
		return err
	}
	if len(cancelled.Orders) == 0 {
		return nil
	}

	for _, order := range cancelled.Orders {
		item, ok := byOrderID[order.OrderID]
		if !ok {
			continue
		}
		command := item.Command
		command.TraceID = fmt.Sprintf("market_close:%d", order.MarketID)
		command.RequestedAtMillis = order.UpdatedAtMillis

		if err := s.store.PersistResult(ctx, command, engine.Result{
			Affected: []*model.Order{order},
		}); err != nil {
			return err
		}
		if err := s.processor.publishOrderEvent(ctx, command, order); err != nil {
			return err
		}
	}

	for _, book := range cancelled.Books {
		if err := s.processor.publishQuoteEvents(ctx, "market_close_sweep", engine.Result{Book: book}); err != nil {
			return err
		}
	}

	s.logger.Info("matching cancelled expired resting orders", "count", len(cancelled.Orders))
	return nil
}
