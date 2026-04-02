package service

import (
	"sync"

	"funnyoption/internal/matching/model"
	sharedkafka "funnyoption/internal/shared/kafka"
)

const (
	defaultCandleIntervalMillis = int64(60_000)
	defaultCandleHistoryLimit   = 48
)

type CandleBook struct {
	mu             sync.Mutex
	intervalMillis int64
	historyLimit   int
	series         map[string][]sharedkafka.QuoteCandle
}

func NewCandleBook(intervalMillis int64, historyLimit int) *CandleBook {
	if intervalMillis <= 0 {
		intervalMillis = defaultCandleIntervalMillis
	}
	if historyLimit <= 0 {
		historyLimit = defaultCandleHistoryLimit
	}
	return &CandleBook{
		intervalMillis: intervalMillis,
		historyLimit:   historyLimit,
		series:         make(map[string][]sharedkafka.QuoteCandle),
	}
}

func (b *CandleBook) ApplyTrade(trade model.Trade) sharedkafka.QuoteCandleEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	bucketStart := trade.MatchedAtMillis - (trade.MatchedAtMillis % b.intervalMillis)
	bucketEnd := bucketStart + b.intervalMillis

	candles := append([]sharedkafka.QuoteCandle(nil), b.series[trade.BookKey]...)
	lastIdx := len(candles) - 1
	if lastIdx >= 0 && candles[lastIdx].BucketStartMillis == bucketStart {
		candle := candles[lastIdx]
		if trade.Price > candle.High {
			candle.High = trade.Price
		}
		if trade.Price < candle.Low {
			candle.Low = trade.Price
		}
		candle.Close = trade.Price
		candle.Volume += trade.Quantity
		candle.TradeCount++
		candles[lastIdx] = candle
	} else {
		candles = append(candles, sharedkafka.QuoteCandle{
			BucketStartMillis: bucketStart,
			BucketEndMillis:   bucketEnd,
			Open:              trade.Price,
			High:              trade.Price,
			Low:               trade.Price,
			Close:             trade.Price,
			Volume:            trade.Quantity,
			TradeCount:        1,
		})
	}

	if len(candles) > b.historyLimit {
		candles = candles[len(candles)-b.historyLimit:]
	}
	b.series[trade.BookKey] = candles

	return sharedkafka.QuoteCandleEvent{
		EventID:          sharedkafka.NewID("evt_candle"),
		MarketID:         trade.MarketID,
		Outcome:          trade.Outcome,
		BookKey:          trade.BookKey,
		IntervalSec:      b.intervalMillis / 1000,
		Candles:          append([]sharedkafka.QuoteCandle(nil), candles...),
		OccurredAtMillis: trade.MatchedAtMillis,
	}
}
