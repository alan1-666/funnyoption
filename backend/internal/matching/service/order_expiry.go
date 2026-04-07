package service

import (
	"context"
	"log/slog"
	"time"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	"funnyoption/internal/matching/pipeline"
	sharedkafka "funnyoption/internal/shared/kafka"
)

const defaultOrderExpirySweepInterval = 2 * time.Second

type orderExpiryStore interface {
	LoadExpiredRestingOrders(ctx context.Context, nowUnix int64) ([]ExpiredRestingOrder, error)
	PersistResult(ctx context.Context, command sharedkafka.OrderCommand, result engine.Result) error
}

// CancelSubmitter is the interface for submitting cancel commands to the pipeline.
type CancelSubmitter interface {
	SubmitCancel(cmd pipeline.MatchCommand) bool
}

type orderExpirySweeper struct {
	logger    *slog.Logger
	store     orderExpiryStore
	submitter CancelSubmitter
	publisher sharedkafka.Publisher
	topics    sharedkafka.Topics
	interval  time.Duration
}

func newOrderExpirySweeper(
	logger *slog.Logger,
	submitter CancelSubmitter,
	store orderExpiryStore,
	publisher sharedkafka.Publisher,
	topics sharedkafka.Topics,
) *orderExpirySweeper {
	return &orderExpirySweeper{
		logger:    logger,
		store:     store,
		submitter: submitter,
		publisher: publisher,
		topics:    topics,
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

	submitted := 0
	for _, item := range items {
		if item.Order == nil {
			continue
		}
		order := item.Order

		if s.submitter != nil {
			cmd := pipeline.MatchCommand{
				Action:        pipeline.ActionCancel,
				OrderID:       order.OrderID,
				MarketID:      order.MarketID,
				Outcome:       order.Outcome,
				BookKey:       model.BuildBookKey(order.MarketID, order.Outcome),
				Side:          pipeline.SideFlagFrom(order.Side),
				Price:         order.Price,
				CommandID:     item.Command.CommandID,
				TraceID:       "market_close_sweep",
				CollateralAsset: item.Command.CollateralAsset,
				FreezeID:      item.Command.FreezeID,
				FreezeAsset:   item.Command.FreezeAsset,
				FreezeAmount:  item.Command.FreezeAmount,
				CancelReason:  pipeline.CancelReasonMarketClosed,
			}
			if s.submitter.SubmitCancel(cmd) {
				submitted++
			}
		}
	}

	if submitted > 0 {
		s.logger.Info("matching submitted cancel commands for expired orders", "count", submitted, "total", len(items))
	}
	return nil
}
