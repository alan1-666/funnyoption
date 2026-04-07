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

// InputGateway is the IO-thread that sits between Kafka and the Input Ring Buffer.
// It does: FetchMessage → JSON decode → MarketIsTradable → MatchCommand → TryPublish.
// When the RB is full, it pauses the Kafka consumer (backpressure).
type InputGateway struct {
	logger  *slog.Logger
	reader  *kafkago.Reader
	inputRB *ringbuffer.RingBuffer[MatchCommand]
	store   TradableChecker
	idle    *ringbuffer.IdleStrategy

	received atomic.Uint64
	dropped  atomic.Uint64
	paused   atomic.Uint64
}

// TradableChecker is a narrow interface used by the gateway to gate orders.
type TradableChecker interface {
	MarketIsTradable(ctx context.Context, marketID int64) (bool, error)
}

func NewInputGateway(
	logger *slog.Logger,
	brokers []string,
	topic, groupID string,
	inputRB *ringbuffer.RingBuffer[MatchCommand],
	store TradableChecker,
) *InputGateway {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	return &InputGateway{
		logger:  logger,
		reader:  reader,
		inputRB: inputRB,
		store:   store,
		idle:    ringbuffer.NewIdleStrategy(200, 20, 100*time.Microsecond),
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

		if g.store != nil {
			tradable, err := g.store.MarketIsTradable(ctx, cmd.MarketID)
			if err != nil {
				g.logger.Error("gateway: tradable check failed", "err", err, "market_id", cmd.MarketID)
				g.commitMsg(ctx, msg)
				continue
			}
			if !tradable {
				g.dropped.Add(1)
				g.logger.Warn("gateway: non-tradable market, dropping", "market_id", cmd.MarketID, "order_id", cmd.OrderID)
				g.commitMsg(ctx, msg)
				continue
			}
		}

		mc := CommandFromKafka(cmd)

		for !g.inputRB.TryPublish(mc) {
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
