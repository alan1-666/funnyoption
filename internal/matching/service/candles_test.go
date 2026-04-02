package service

import (
	"testing"

	"funnyoption/internal/matching/model"
)

func TestCandleBookAggregatesTradesIntoBuckets(t *testing.T) {
	book := NewCandleBook(60_000, 8)

	first := book.ApplyTrade(model.Trade{
		MarketID:        1001,
		Outcome:         "YES",
		BookKey:         "1001:YES",
		Price:           58,
		Quantity:        100,
		MatchedAtMillis: 1_770_000_000_000,
	})
	if len(first.Candles) != 1 {
		t.Fatalf("expected 1 candle, got %d", len(first.Candles))
	}
	if first.Candles[0].Open != 58 || first.Candles[0].Close != 58 || first.Candles[0].Volume != 100 {
		t.Fatalf("unexpected first candle: %+v", first.Candles[0])
	}

	second := book.ApplyTrade(model.Trade{
		MarketID:        1001,
		Outcome:         "YES",
		BookKey:         "1001:YES",
		Price:           61,
		Quantity:        70,
		MatchedAtMillis: 1_770_000_020_000,
	})
	if len(second.Candles) != 1 {
		t.Fatalf("expected same bucket update, got %d candles", len(second.Candles))
	}
	if second.Candles[0].High != 61 || second.Candles[0].Low != 58 || second.Candles[0].Close != 61 || second.Candles[0].Volume != 170 {
		t.Fatalf("unexpected merged candle: %+v", second.Candles[0])
	}

	third := book.ApplyTrade(model.Trade{
		MarketID:        1001,
		Outcome:         "YES",
		BookKey:         "1001:YES",
		Price:           59,
		Quantity:        30,
		MatchedAtMillis: 1_770_000_090_000,
	})
	if len(third.Candles) != 2 {
		t.Fatalf("expected new bucket, got %d candles", len(third.Candles))
	}
	if third.Candles[1].Open != 59 || third.Candles[1].Close != 59 || third.Candles[1].Volume != 30 {
		t.Fatalf("unexpected second candle: %+v", third.Candles[1])
	}
}
