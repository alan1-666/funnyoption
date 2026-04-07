package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

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
			BatchTimeout:           10 * time.Millisecond,
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
		Time:  time.Now(),
	}
	return p.writer.WriteMessages(ctx, msg)
}

func (p *JSONPublisher) PublishJSONBatch(ctx context.Context, items []BatchItem) error {
	if len(items) == 0 {
		return nil
	}
	now := time.Now()
	msgs := make([]kafkago.Message, 0, len(items))
	for _, item := range items {
		value, err := json.Marshal(item.Payload)
		if err != nil {
			return err
		}
		msgs = append(msgs, kafkago.Message{
			Topic: item.Topic,
			Key:   []byte(item.Key),
			Value: value,
			Time:  now,
		})
	}
	return p.writer.WriteMessages(ctx, msgs...)
}

func (p *JSONPublisher) Close() error {
	if p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
