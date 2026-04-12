package pipeline

import (
	"context"
	"log/slog"
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

// InputGateway sits between Kafka and the BookSupervisor.
// It uses a ConsumerGroup to consume from multiple partitions in parallel —
// one goroutine per assigned partition. Because the upstream producer keys
// messages by bookKey (Hash partitioner), all orders for a given book always
// land on the same partition. This guarantees that each per-book SPSC
// ringbuffer has exactly one producer goroutine at any time.
//
// Offset commits run in a background goroutine per partition so
// CommitOffsets never blocks the fetch→decode→route hot path.
type InputGateway struct {
	logger  *slog.Logger
	brokers []string
	topic   string
	groupID string
	router  CommandRouter

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
	return &InputGateway{
		logger:  logger,
		brokers: brokers,
		topic:   topic,
		groupID: groupID,
		router:  router,
	}
}

func (g *InputGateway) Start(ctx context.Context) {
	go g.run(ctx)
}

func (g *InputGateway) run(ctx context.Context) {
	g.logger.Info("input gateway started (consumer group mode)")
	defer g.logger.Info("input gateway stopped")

	group, err := kafkago.NewConsumerGroup(kafkago.ConsumerGroupConfig{
		ID:                     g.groupID,
		Brokers:                g.brokers,
		Topics:                 []string{g.topic},
		HeartbeatInterval:      3 * time.Second,
		RebalanceTimeout:       30 * time.Second,
		SessionTimeout:         30 * time.Second,
		WatchPartitionChanges:  true,
		PartitionWatchInterval: 10 * time.Second,
	})
	if err != nil {
		g.logger.Error("gateway: failed to create consumer group", "err", err)
		return
	}
	defer group.Close()

	for {
		gen, err := group.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			g.logger.Error("gateway: consumer group next failed", "err", err)
			time.Sleep(time.Second)
			continue
		}

		assignments := gen.Assignments[g.topic]
		g.logger.Info("gateway: new generation",
			"generation", gen.ID,
			"member", gen.MemberID,
			"partitions", len(assignments),
		)

		for _, assignment := range assignments {
			partition := assignment.ID
			offset := assignment.Offset
			gen.Start(func(ctx context.Context) {
				g.runPartition(ctx, gen, partition, offset)
			})
		}
	}
}

// runPartition runs the fetch→decode→route loop for a single partition.
// It exits when the generation context is cancelled (rebalance or shutdown).
func (g *InputGateway) runPartition(ctx context.Context, gen *kafkago.Generation, partition int, startOffset int64) {
	logger := g.logger.With("partition", partition)
	logger.Info("partition worker started", "offset", startOffset)
	defer logger.Info("partition worker stopped")

	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:       g.brokers,
		Topic:         g.topic,
		Partition:     partition,
		MinBytes:      10e3,                  // 10 KB — broker batches ~20 msgs
		MaxBytes:      10e6,                  // 10 MB
		MaxWait:       10 * time.Millisecond, // ceiling on broker-side wait
		QueueCapacity: 1000,                  // pre-fetch buffer
	})
	defer reader.Close()

	// Seek to the assigned offset.
	if startOffset >= 0 {
		if err := reader.SetOffset(startOffset); err != nil {
			logger.Error("partition worker: set offset failed", "err", err, "offset", startOffset)
			return
		}
	}

	// Async offset commit goroutine.
	commitCh := make(chan int64, 512)
	go g.partitionCommitLoop(ctx, gen, partition, commitCh)

	idle := ringbuffer.NewIdleStrategy(200, 20, 100*time.Microsecond)

	for {
		if ctx.Err() != nil {
			return
		}

		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error("partition worker: fetch failed", "err", err)
			time.Sleep(time.Second)
			continue
		}

		var cmd sharedkafka.OrderCommand
		if err := json.Unmarshal(msg.Value, &cmd); err != nil {
			logger.Error("partition worker: json decode failed", "err", err, "offset", msg.Offset)
			commitCh <- msg.Offset + 1
			continue
		}

		mc := CommandFromKafka(cmd)

		for !g.router.Route(mc) {
			g.paused.Add(1)
			idle.Idle()
			if ctx.Err() != nil {
				return
			}
		}
		idle.Reset()
		g.received.Add(1)

		commitCh <- msg.Offset + 1
	}
}

// partitionCommitLoop batches offset commits for a single partition.
func (g *InputGateway) partitionCommitLoop(ctx context.Context, gen *kafkago.Generation, partition int, ch <-chan int64) {
	const flushInterval = 200 * time.Millisecond
	const batchCap = 256

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	var maxOffset int64
	pending := 0

	flush := func() {
		if pending == 0 {
			return
		}
		offsets := map[string]map[int]int64{
			g.topic: {partition: maxOffset},
		}
		if err := gen.CommitOffsets(offsets); err != nil {
			if ctx.Err() == nil {
				g.logger.Error("partition commit failed", "partition", partition, "err", err)
			}
		}
		pending = 0
	}

	for {
		select {
		case off, ok := <-ch:
			if !ok {
				flush()
				return
			}
			if off > maxOffset {
				maxOffset = off
			}
			pending++
			// Greedily drain.
			for pending < batchCap {
				select {
				case off, ok := <-ch:
					if !ok {
						flush()
						return
					}
					if off > maxOffset {
						maxOffset = off
					}
					pending++
				default:
					goto checkFlush
				}
			}
		checkFlush:
			if pending >= batchCap {
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
	// The consumer group is owned by run() and closed there.
	return nil
}

// Stats returns counters for monitoring.
func (g *InputGateway) Stats() (received, dropped, paused uint64) {
	return g.received.Load(), g.dropped.Load(), g.paused.Load()
}
