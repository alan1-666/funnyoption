package service

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"

	sharedkafka "funnyoption/internal/shared/kafka"
)

type subscriber struct {
	ch chan []byte
}

type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]map[*subscriber]struct{}
}

type streamEnvelope struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[string]map[*subscriber]struct{}),
	}
}

func (h *Hub) Subscribe(topic string) (*subscriber, func()) {
	sub := &subscriber{ch: make(chan []byte, 64)}
	h.mu.Lock()
	if _, ok := h.subscribers[topic]; !ok {
		h.subscribers[topic] = make(map[*subscriber]struct{})
	}
	h.subscribers[topic][sub] = struct{}{}
	h.mu.Unlock()

	return sub, func() {
		h.mu.Lock()
		if subs, ok := h.subscribers[topic]; ok {
			delete(subs, sub)
			if len(subs) == 0 {
				delete(h.subscribers, topic)
			}
		}
		h.mu.Unlock()
		close(sub.ch)
	}
}

func (h *Hub) Broadcast(topic string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for sub := range h.subscribers[topic] {
		select {
		case sub.ch <- body:
		default:
		}
	}
	return nil
}

func (h *Hub) HandleDepth(_ context.Context, msg sharedkafka.Message) error {
	var event sharedkafka.QuoteDepthEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}
	return h.Broadcast("depth:"+event.BookKey, event)
}

func (h *Hub) HandleTicker(_ context.Context, msg sharedkafka.Message) error {
	var event sharedkafka.QuoteTickerEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}
	return h.Broadcast("ticker:"+event.BookKey, event)
}

func (h *Hub) HandleCandle(_ context.Context, msg sharedkafka.Message) error {
	var event sharedkafka.QuoteCandleEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}
	return h.Broadcast("candle:"+event.BookKey, event)
}

func (h *Hub) HandleMarketEvent(_ context.Context, msg sharedkafka.Message) error {
	var event sharedkafka.MarketEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}
	return h.Broadcast("market:"+formatMarketKey(event.MarketID), streamEnvelope{
		Type:    "market.event",
		Payload: event,
	})
}

func (h *Hub) HandleSettlementCompleted(_ context.Context, msg sharedkafka.Message) error {
	var event sharedkafka.SettlementCompletedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}
	return h.Broadcast("market:"+formatMarketKey(event.MarketID), streamEnvelope{
		Type:    "settlement.completed",
		Payload: event,
	})
}

func formatMarketKey(marketID int64) string {
	return strconv.FormatInt(marketID, 10)
}
