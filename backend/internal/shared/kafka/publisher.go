package kafka

import (
	"context"
	"log/slog"
	"time"

	json "github.com/goccy/go-json"
	kafkago "github.com/segmentio/kafka-go"
)

type Publisher interface {
	PublishJSON(ctx context.Context, topic, key string, payload any) error
	PublishJSONBatch(ctx context.Context, items []BatchItem) error
	Close() error
}

type BatchItem struct {
	Topic   string
	Key     string
	Payload any
}

type JSONPublisher struct {
	logger *slog.Logger
	writer *kafkago.Writer
}

func NewJSONPublisher(logger *slog.Logger, brokers []string) *JSONPublisher {
	return &JSONPublisher{
		logger: logger,
		writer: &kafkago.Writer{
			Addr:                   kafkago.TCP(brokers...),
			Balancer:               &kafkago.Hash{},
			BatchSize:              1000,                  // default ~100; mega-batches (384+ events) flush in one round-trip
			BatchBytes:             10 * 1024 * 1024,      // 10 MB per batch — match consumer MaxBytes
			BatchTimeout:           5 * time.Millisecond,  // tighter flush for lower latency
			Compression:            kafkago.Lz4,           // ~30% smaller payloads, lz4 is cheapest CPU
			RequiredAcks:           kafkago.RequireOne,
			AllowAutoTopicCreation: false,
		},
	}
}

func (p *JSONPublisher) PublishJSON(ctx context.Context, topic, key string, payload any) error {
	value, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := kafkago.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}
	return p.writer.WriteMessages(ctx, msg)
}

func (p *JSONPublisher) PublishJSONBatch(ctx context.Context, items []BatchItem) error {
	if len(items) == 0 {
		return nil
	}
	msgs := make([]kafkago.Message, len(items))
	for i, item := range items {
		value, err := json.Marshal(item.Payload)
		if err != nil {
			return err
		}
		msgs[i] = kafkago.Message{
			Topic: item.Topic,
			Key:   []byte(item.Key),
			Value: value,
		}
	}
	return p.writer.WriteMessages(ctx, msgs...)
}

func (p *JSONPublisher) Close() error {
	if p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
