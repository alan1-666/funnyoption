package pipeline

import (
	"context"
	"log/slog"
	"time"

	"funnyoption/internal/matching/ha"
	"funnyoption/internal/matching/model"
	"funnyoption/internal/shared/fee"
	sharedkafka "funnyoption/internal/shared/kafka"
)

// Config tunes the ring buffer sizes.
type Config struct {
	InputRBSize  int // per-book input RB (default 1024)
	OutputRBSize int // shared output channel size (default 8192)
}

func (c Config) withDefaults() Config {
	if c.InputRBSize <= 0 {
		c.InputRBSize = bookRBSize
	}
	if c.OutputRBSize <= 0 {
		c.OutputRBSize = outputChSize
	}
	return c
}

// Pipeline owns the three-stage matching pipeline:
// InputGateway → BookSupervisor (per-book MatchLoops) → OutputDispatcher
type Pipeline struct {
	gateway    *InputGateway
	supervisor *BookSupervisor
	dispatcher *OutputDispatcher

	cancel context.CancelFunc
}

// New constructs the full pipeline. Call Start() to begin processing.
func New(
	logger *slog.Logger,
	brokers []string,
	topic, groupID string,
	tradableStore TradableChecker,
	persistStore PersistStore,
	publisher sharedkafka.Publisher,
	topics sharedkafka.Topics,
	feeSched fee.Schedule,
	candles CandleTracker,
	cfg Config,
	epoch EpochSource,
) *Pipeline {
	cfg = cfg.withDefaults()

	supervisor := NewBookSupervisor(logger, epoch)
	gateway := NewInputGateway(logger, brokers, topic, groupID, supervisor, tradableStore)
	dispatcher := NewOutputDispatcher(logger, supervisor.OutputCh(), publisher, topics, persistStore, candles, feeSched)

	return &Pipeline{
		gateway:    gateway,
		supervisor: supervisor,
		dispatcher: dispatcher,
	}
}

// Restore loads resting orders into the per-book engines and sets the trade sequence.
func (p *Pipeline) Restore(sequence uint64, orders []*model.Order) error {
	return p.supervisor.Restore(sequence, orders)
}

// Start launches the gateway, all book engines, and the dispatcher.
func (p *Pipeline) Start(ctx context.Context) {
	ctx, p.cancel = context.WithCancel(ctx)
	p.supervisor.Start(ctx)
	p.gateway.Start(ctx)
	go p.dispatcher.Run(ctx)
}

// SubmitCancel injects a cancel command into the correct book engine.
// Used by the expiry sweeper to cancel expired resting orders through the pipeline.
func (p *Pipeline) SubmitCancel(cmd MatchCommand) bool {
	return p.supervisor.SubmitCancel(cmd)
}

// Close shuts down the pipeline gracefully.
func (p *Pipeline) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	return p.gateway.Close()
}

// LogStats logs counters from all stages. Call periodically.
func (p *Pipeline) LogStats(logger *slog.Logger) {
	recv, drop, pause := p.gateway.Stats()
	matched, batches := p.supervisor.Stats()
	dispatched, errs := p.dispatcher.Stats()
	logger.Info("pipeline stats",
		"gw_received", recv,
		"gw_dropped", drop,
		"gw_paused", pause,
		"sv_books", p.supervisor.BookCount(),
		"sv_matched", matched,
		"sv_batches", batches,
		"disp_dispatched", dispatched,
		"disp_errors", errs,
	)
}

// SetDispatchMode switches the output dispatcher between ACTIVE and SHADOW.
func (p *Pipeline) SetDispatchMode(mode DispatchMode) {
	p.dispatcher.SetMode(mode)
}

// TakeSnapshot captures the full engine state for HA recovery.
func (p *Pipeline) TakeSnapshot() ha.FullSnapshot {
	return p.supervisor.TakeSnapshot()
}

// GlobalSequence returns the current trade sequence.
func (p *Pipeline) GlobalSequence() uint64 {
	return p.supervisor.GlobalSequence()
}

// StartStatsReporter logs stats every interval.
func (p *Pipeline) StartStatsReporter(ctx context.Context, logger *slog.Logger, interval time.Duration) {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.LogStats(logger)
			}
		}
	}()
}
