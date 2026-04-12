package pipeline

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"funnyoption/internal/posttrade"
	"funnyoption/internal/shared/fee"
	sharedkafka "funnyoption/internal/shared/kafka"
)

const (
	// dispatchBatchMax is the maximum number of results to accumulate before
	// flushing a batch. Larger batches amortise the DB fsync and Kafka
	// round-trip cost. 128 items × ~6 events each = ~768 Kafka messages
	// per flush, well within the publisher's BatchSize=1000.
	dispatchBatchMax = 128

	// dispatchBatchTimeout is how long to wait for additional results after
	// receiving the first one before flushing an incomplete batch.
	dispatchBatchTimeout = 5 * time.Millisecond

	// dispatchWorkers is the number of concurrent persist workers.
	// Each worker handles a shard of bookKeys, preserving per-book ordering
	// while parallelising DB writes across different books.
	dispatchWorkers = 4
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
	d.logger.Info("output dispatcher started", "workers", dispatchWorkers)
	defer d.logger.Info("output dispatcher stopped")

	// Start sharded workers — each owns a channel and a batch buffer.
	workerChs := make([]chan MatchResult, dispatchWorkers)
	var wg sync.WaitGroup
	for i := range workerChs {
		workerChs[i] = make(chan MatchResult, dispatchBatchMax*2)
		wg.Add(1)
		go func(ch <-chan MatchResult) {
			defer wg.Done()
			d.runWorker(ctx, ch)
		}(workerChs[i])
	}

	defer func() {
		for _, ch := range workerChs {
			close(ch)
		}
		wg.Wait()
	}()

	// Router: read from the shared fan-in channel and route by bookKey hash
	// so that results for the same book always go to the same worker,
	// preserving per-book ordering.
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
			idx := fnvHash(mr.Command.BookKey) % dispatchWorkers
			workerChs[idx] <- mr
		}
	}
}

// runWorker drains a per-shard channel in batches and flushes them.
func (d *OutputDispatcher) runWorker(ctx context.Context, ch <-chan MatchResult) {
	batch := make([]*posttrade.MatchResult, 0, dispatchBatchMax)
	timer := time.NewTimer(dispatchBatchTimeout)
	timer.Stop() // start stopped; reset on first receive

	for {
		batch = batch[:0]

		// Block on first result.
		select {
		case <-ctx.Done():
			return
		case mr, ok := <-ch:
			if !ok {
				return
			}
			batch = append(batch, d.convertResult(mr))
		}

		// Non-blocking drain with reused timer.
		timer.Reset(dispatchBatchTimeout)
	drain:
		for len(batch) < dispatchBatchMax {
			select {
			case mr, ok := <-ch:
				if !ok {
					break drain
				}
				batch = append(batch, d.convertResult(mr))
			case <-timer.C:
				break drain
			}
		}
		// Stop and drain timer if it hasn't fired (batch filled before timeout).
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}

		if err := d.pt.ProcessBatch(ctx, batch); err != nil {
			d.errors.Add(uint64(len(batch)))
			d.logger.Error("dispatcher: batch failed", "err", err, "batch_size", len(batch))
		}
		d.dispatched.Add(uint64(len(batch)))
	}
}

// fnvHash returns a consistent hash for routing bookKeys to workers.
func fnvHash(s string) int {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return int(h & 0x7fffffff) // ensure non-negative
}

func (d *OutputDispatcher) convertResult(mr MatchResult) *posttrade.MatchResult {
	return &posttrade.MatchResult{
		Command:  mr.Command.ToKafkaCommand(),
		Result:   mr.Result,
		Rejected: mr.Rejected,
		EpochID:  mr.EpochID,
	}
}

func (d *OutputDispatcher) Stats() (dispatched, errors uint64) {
	return d.dispatched.Load(), d.errors.Load()
}

func (d *OutputDispatcher) ShadowedCount() uint64 {
	return d.shadowed.Load()
}
