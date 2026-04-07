package kafka

import "strings"

type Topics struct {
	OrderCommand   string
	OrderEvent     string
	TradeMatched   string
	PositionChange string
	QuoteDepth     string
	QuoteTicker    string
	QuoteCandle    string
	MarketEvent    string
	SettlementDone string
	ChainDeposit   string
	ChainWithdraw  string
}

func NewTopics(prefix string) Topics {
	normalized := strings.TrimSpace(prefix)
	if normalized == "" {
		normalized = "funnyoption."
	}
	if !strings.HasSuffix(normalized, ".") {
		normalized += "."
	}

	return Topics{
		OrderCommand:   normalized + "order.command",
		OrderEvent:     normalized + "order.event",
		TradeMatched:   normalized + "trade.matched",
		PositionChange: normalized + "position.changed",
		QuoteDepth:     normalized + "quote.depth",
		QuoteTicker:    normalized + "quote.ticker",
		QuoteCandle:    normalized + "quote.candle",
		MarketEvent:    normalized + "market.event",
		SettlementDone: normalized + "settlement.completed",
		ChainDeposit:   normalized + "chain.deposit",
		ChainWithdraw:  normalized + "chain.withdrawal",
	}
}
