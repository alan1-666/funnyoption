package handler

import (
	"strings"

	"funnyoption/internal/api/dto"
)

func normalizeLifecycleMarketStatus(status string) string {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	switch normalized {
	case "", "OPEN":
		return "OPEN"
	case "DRAFT", "PAUSED", "CLOSED", "RESOLVED":
		return normalized
	default:
		return "OPEN"
	}
}

func effectiveMarketStatusAt(status string, closeAt, nowUnix int64) string {
	normalized := normalizeLifecycleMarketStatus(status)
	if normalized == "OPEN" && closeAt > 0 && nowUnix >= closeAt {
		return "CLOSED"
	}
	return normalized
}

func marketIsOpenForTrading(market dto.MarketResponse, nowUnix int64) bool {
	return effectiveMarketStatusAt(market.Status, market.CloseAt, nowUnix) == "OPEN"
}

func applyEffectiveMarketStatus(market *dto.MarketResponse, nowUnix int64) {
	if market == nil {
		return
	}
	market.Status = effectiveMarketStatusAt(market.Status, market.CloseAt, nowUnix)
}
