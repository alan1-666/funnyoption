package model

import (
	"strconv"
	"strings"
)

type BookLevel struct {
	Price    int64
	Quantity int64
}

type BookSnapshot struct {
	Key      string
	MarketID int64
	Outcome  string
	Bids     []BookLevel
	Asks     []BookLevel
	BestBid  int64
	BestAsk  int64
}

func aggregateLevels(levels []*DepthLevel, limit int) []BookLevel {
	if len(levels) == 0 {
		return nil
	}
	if len(levels) < limit {
		limit = len(levels)
	}
	result := make([]BookLevel, 0, limit)
	for _, level := range levels[:limit] {
		qty := level.TotalQuantity()
		if qty <= 0 {
			continue
		}
		result = append(result, BookLevel{Price: level.Price, Quantity: qty})
	}
	return result
}

func parseBookKey(key string) (int64, string, bool) {
	parts := strings.SplitN(strings.TrimSpace(key), ":", 2)
	if len(parts) != 2 {
		return 0, "", false
	}
	marketID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", false
	}
	return marketID, strings.ToUpper(strings.TrimSpace(parts[1])), true
}
