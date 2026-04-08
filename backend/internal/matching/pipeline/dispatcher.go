package pipeline

import (
	"context"
	"log/slog"
	"sync/atomic"

	"funnyoption/internal/posttrade"
	"funnyoption/internal/shared/fee"
	sharedkafka "funnyoption/internal/shared/kafka"
)

// CandleTracker applies trades and returns candle events.
type CandleTracker = posttrade.CandleTracker

// PersistStore is the interface the dispatcher needs for DB writes.
type PersistStore = posttrade.PersistStore

// DispatchMode controls whether the dispatcher emits real output or silently drains.
type DispatchMode int32

const (
	DispatchModeActive DispatchMode = 0
	DispatchModeShadow DispatchMode = 1
)

// OutputDispatcher drains MatchResults from the shared fan-in channel and
// delegates all post-trade IO to the posttrade.Service.
// In Shadow mode it drains and counts but does not process, enabling a
// standby node to keep its engine warm.
type OutputDispatcher struct {
	logger   *slog.Logger
	outputCh <-chan MatchResult
	pt       *posttrade.Service

	mode       atomic.Int32
	dispatched atomic.Uint64
	shadowed   atomic.Uint64
	errors     atomic.Uint64
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
	pt := posttrade.New(logger, publisher, topics, store, candles, feeSched)
	return &OutputDispatcher{
		logger:   logger,
		outputCh: outputCh,
		pt:       pt,
	}
}

// SetMode switches the dispatcher between ACTIVE and SHADOW mode.
func (d *OutputDispatcher) SetMode(mode DispatchMode) {
	d.mode.Store(int32(mode))
	d.logger.Info("dispatcher mode changed", "mode", mode)
}

// Mode returns the current dispatch mode.
func (d *OutputDispatcher) Mode() DispatchMode {
	return DispatchMode(d.mode.Load())
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
			if d.Mode() == DispatchModeShadow {
				d.shadowed.Add(1)
				continue
			}
			ptResult := posttrade.MatchResult{
				Command:  mr.Command.ToKafkaCommand(),
				Result:   mr.Result,
				Rejected: mr.Rejected,
				EpochID:  mr.EpochID,
			}
			if err := d.pt.ProcessResult(ctx, &ptResult); err != nil {
				d.errors.Add(1)
				d.logger.Error("dispatcher: failed", "err", err, "order_id", mr.Command.OrderID)
			}
			d.dispatched.Add(1)
		}
	}
}

func (d *OutputDispatcher) Stats() (dispatched, errors uint64) {
	return d.dispatched.Load(), d.errors.Load()
}

func (d *OutputDispatcher) ShadowedCount() uint64 {
	return d.shadowed.Load()
}
