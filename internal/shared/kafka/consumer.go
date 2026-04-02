package kafka

import (
	"context"
	"errors"
	"log/slog"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type Message struct {
	Topic     string
	Key       string
	Value     []byte
	Partition int
	Offset    int64
	Time      time.Time
}

type Handler func(ctx context.Context, msg Message) error

type JSONConsumer struct {
	logger  *slog.Logger
	topic   string
	groupID string
	reader  *kafkago.Reader
	handler Handler
}

func NewJSONConsumer(logger *slog.Logger, brokers []string, topic, groupID string, handler Handler) *JSONConsumer {
	return &JSONConsumer{
		logger:  logger,
		topic:   topic,
		groupID: groupID,
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:  brokers,
			GroupID:  groupID,
			Topic:    topic,
			MinBytes: 1,
			MaxBytes: 10e6,
		}),
		handler: handler,
	}
}

func (c *JSONConsumer) Start(ctx context.Context) {
	go func() {
		c.logger.Info("kafka consumer started", "topic", c.topic, "group_id", c.groupID)
		defer c.logger.Info("kafka consumer stopped", "topic", c.topic, "group_id", c.groupID)
		for {
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil || errors.Is(err, context.Canceled) {
					return
				}
				c.logger.Error("kafka fetch failed", "topic", c.topic, "err", err)
				time.Sleep(time.Second)
				continue
			}

			envelope := Message{
				Topic:     msg.Topic,
				Key:       string(msg.Key),
				Value:     msg.Value,
				Partition: msg.Partition,
				Offset:    msg.Offset,
				Time:      msg.Time,
			}
			if err := c.handler(ctx, envelope); err != nil {
				c.logger.Error("kafka handler failed", "topic", c.topic, "partition", msg.Partition, "offset", msg.Offset, "err", err)
				continue
			}
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				c.logger.Error("kafka commit failed", "topic", c.topic, "partition", msg.Partition, "offset", msg.Offset, "err", err)
			}
		}
	}()
}

func (c *JSONConsumer) Close() error {
	if c.reader == nil {
		return nil
	}
	return c.reader.Close()
}
