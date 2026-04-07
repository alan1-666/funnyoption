package pipeline

import (
	"context"
	"log/slog"
	"time"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	"funnyoption/internal/matching/ringbuffer"
	"funnyoption/internal/shared/fee"
	sharedkafka "funnyoption/internal/shared/kafka"
)

// Config tunes the ring buffer sizes.
type Config struct {
	InputRBSize  int // default 8192
	OutputRBSize int // default 8192
}

func (c Config) withDefaults() Config {
	if c.InputRBSize <= 0 {
		c.InputRBSize = 8192
	}
	if c.OutputRBSize <= 0 {
		c.OutputRBSize = 8192
	}
	return c
}

// Pipeline owns the three-stage matching pipeline:
// InputGateway → MatchLoop → OutputDispatcher
type Pipeline struct {
	gateway    *InputGateway
	matchLoop  *MatchLoop
	dispatcher *OutputDispatcher
	eng        *engine.Engine
	inputRB    *ringbuffer.RingBuffer[MatchCommand]

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
) *Pipeline {
	cfg = cfg.withDefaults()

	inputRB := ringbuffer.New[MatchCommand](cfg.InputRBSize)
	outputRB := ringbuffer.New[MatchResult](cfg.OutputRBSize)

	eng := engine.New(logger)

	gateway := NewInputGateway(logger, brokers, topic, groupID, inputRB, tradableStore)
	matchLoop := NewMatchLoop(logger, eng, inputRB, outputRB)
	dispatcher := NewOutputDispatcher(logger, outputRB, publisher, topics, persistStore, candles, feeSched)

	return &Pipeline{
		gateway:    gateway,
		matchLoop:  matchLoop,
		dispatcher: dispatcher,
		eng:        eng,
		inputRB:    inputRB,
	}
}

// Restore loads resting orders into the engine and sets the trade sequence.
func (p *Pipeline) Restore(sequence uint64, orders []*model.Order) error {
	p.eng.SetSequence(sequence)
	for _, order := range orders {
		if err := p.matchLoop.RestoreOrder(order); err != nil {
			return err
		}
	}
	return nil
}

// Start launches all three goroutines.
func (p *Pipeline) Start(ctx context.Context) {
	ctx, p.cancel = context.WithCancel(ctx)
	p.gateway.Start(ctx)
	go p.matchLoop.Run(ctx)
	go p.dispatcher.Run(ctx)
}

// SubmitCancel injects a cancel command into the input ring buffer.
// Used by the expiry sweeper to cancel expired resting orders through the pipeline.
func (p *Pipeline) SubmitCancel(cmd MatchCommand) bool {
	cmd.Action = ActionCancel
	return p.inputRB.TryPublish(cmd)
}

// Close shuts down the pipeline gracefully.
func (p *Pipeline) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	return p.gateway.Close()
}

// LogStats logs counters from all three stages. Call periodically.
func (p *Pipeline) LogStats(logger *slog.Logger) {
	recv, drop, pause := p.gateway.Stats()
	matched, batches, outStall := p.matchLoop.Stats()
	dispatched, errs := p.dispatcher.Stats()
	logger.Info("pipeline stats",
		"gw_received", recv,
		"gw_dropped", drop,
		"gw_paused", pause,
		"ml_matched", matched,
		"ml_batches", batches,
		"ml_out_stall", outStall,
		"disp_dispatched", dispatched,
		"disp_errors", errs,
	)
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
