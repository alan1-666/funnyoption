package pipeline

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	json "github.com/goccy/go-json"

	"funnyoption/internal/matching/ringbuffer"
	sharedkafka "funnyoption/internal/shared/kafka"

	kafkago "github.com/segmentio/kafka-go"
)

// CommandRouter is the interface the gateway uses to route commands.
// BookSupervisor implements this.
type CommandRouter interface {
	Route(cmd MatchCommand) bool
}

// InputGateway is the IO-thread that sits between Kafka and the BookSupervisor.
// It does: FetchMessage → JSON decode → MatchCommand → Route.
// Market-tradable checks are performed upstream in OrderService; the gateway is
// a pure Kafka→RingBuffer forwarder with zero DB dependency.
// When the book's RB is full, it spins/yields (backpressure).
//
// Offset commits run in a background goroutine so CommitMessages never blocks
// the fetch→decode→route hot path.
type InputGateway struct {
	logger *slog.Logger
	reader *kafkago.Reader
	router CommandRouter
	idle   *ringbuffer.IdleStrategy

	received atomic.Uint64
	dropped  atomic.Uint64
	paused   atomic.Uint64
}

func NewInputGateway(
	logger *slog.Logger,
	brokers []string,
	topic, groupID string,
	router CommandRouter,
) *InputGateway {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:       brokers,
		GroupID:       groupID,
		Topic:         topic,
		MinBytes:      10e3,                  // 10 KB — broker batches ~20 msgs before responding
		MaxBytes:      10e6,                  // 10 MB
		MaxWait:       10 * time.Millisecond, // ceiling on broker-side wait
		QueueCapacity: 1000,                  // internal pre-fetch buffer (default 100)
	})
	return &InputGateway{
		logger: logger,
		reader: reader,
		router: router,
		idle:   ringbuffer.NewIdleStrategy(200, 20, 100*time.Microsecond),
	}
}

func (g *InputGateway) Start(ctx context.Context) {
	go g.run(ctx)
}

func (g *InputGateway) run(ctx context.Context) {
	g.logger.Info("input gateway started")
	defer g.logger.Info("input gateway stopped")

	// Async commit goroutine — CommitMessages is a synchronous Kafka RPC
	// that takes 5-20 ms. Running it in the hot loop blocks ~256 fetches
	// worth of processing each time. Moving it here removes that stall.
	commitCh := make(chan kafkago.Message, 512)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		g.commitLoop(ctx, commitCh)
	}()
	defer func() {
		close(commitCh)
		wg.Wait()
	}()

	for {
		if ctx.Err() != nil {
			return
		}

		msg, err := g.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			g.logger.Error("gateway: kafka fetch failed", "err", err)
			time.Sleep(time.Second)
			continue
		}

		var cmd sharedkafka.OrderCommand
		if err := json.Unmarshal(msg.Value, &cmd); err != nil {
			g.logger.Error("gateway: json decode failed", "err", err, "offset", msg.Offset)
			commitCh <- msg
			continue
		}

		mc := CommandFromKafka(cmd)

		for !g.router.Route(mc) {
			g.paused.Add(1)
			g.idle.Idle()
			if ctx.Err() != nil {
				return
			}
		}
		g.idle.Reset()
		g.received.Add(1)

		commitCh <- msg
	}
}

// commitLoop batches Kafka offset commits in a dedicated goroutine.
// Flushes on batch-full (256) or time (200 ms), whichever comes first.
func (g *InputGateway) commitLoop(ctx context.Context, ch <-chan kafkago.Message) {
	const batchCap = 256
	batch := make([]kafkago.Message, 0, batchCap)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := g.reader.CommitMessages(ctx, batch...); err != nil {
			if ctx.Err() == nil {
				g.logger.Error("gateway: batch commit failed", "err", err)
			}
		}
		batch = batch[:0]
	}

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				flush()
				return
			}
			batch = append(batch, msg)
			// Greedily drain channel without blocking.
			for len(batch) < batchCap {
				select {
				case m, ok := <-ch:
					if !ok {
						flush()
						return
					}
					batch = append(batch, m)
				default:
					goto checkFlush
				}
			}
		checkFlush:
			if len(batch) >= batchCap {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-ctx.Done():
			flush()
			return
		}
	}
}

func (g *InputGateway) Close() error {
	return g.reader.Close()
}

// Stats returns counters for monitoring.
func (g *InputGateway) Stats() (received, dropped, paused uint64) {
	return g.received.Load(), g.dropped.Load(), g.paused.Load()
}
