package pipeline

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"time"

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
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 1,
		MaxBytes: 10e6,
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
			g.commitMsg(ctx, msg)
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

		g.commitMsg(ctx, msg)
	}
}

func (g *InputGateway) commitMsg(ctx context.Context, msg kafkago.Message) {
	if err := g.reader.CommitMessages(ctx, msg); err != nil {
		g.logger.Error("gateway: commit failed", "err", err)
	}
}

func (g *InputGateway) Close() error {
	return g.reader.Close()
}

// Stats returns counters for monitoring.
func (g *InputGateway) Stats() (received, dropped, paused uint64) {
	return g.received.Load(), g.dropped.Load(), g.paused.Load()
}
