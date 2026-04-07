package handler

import (
	"encoding/json"
	"strings"

	"funnyoption/internal/api/dto"
	oracleservice "funnyoption/internal/oracle/service"
)

func normalizeLifecycleMarketStatus(status string) string {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	switch normalized {
	case "", "OPEN":
		return "OPEN"
	case "DRAFT", "PAUSED", "CLOSED", "WAITING_RESOLUTION", "RESOLVED":
		return normalized
	default:
		return "OPEN"
	}
}

func effectiveMarketStatusAt(status string, closeAt, resolveAt, nowUnix int64, metadata json.RawMessage) string {
	normalized := normalizeLifecycleMarketStatus(status)
	if normalized != "OPEN" {
		return normalized
	}
	if closeAt <= 0 || nowUnix < closeAt {
		return "OPEN"
	}
	if marketUsesOracleResolution(metadata) {
		return "CLOSED"
	}
	if resolveAt <= 0 || nowUnix >= resolveAt {
		return "WAITING_RESOLUTION"
	}
	return "CLOSED"
}

func marketUsesOracleResolution(metadata json.RawMessage) bool {
	return oracleservice.HasOracleResolutionMode(metadata)
}

func marketIsOpenForTrading(market dto.MarketResponse, nowUnix int64) bool {
	return effectiveMarketStatusAt(market.Status, market.CloseAt, market.ResolveAt, nowUnix, market.Metadata) == "OPEN"
}

func applyEffectiveMarketStatus(market *dto.MarketResponse, nowUnix int64) {
	if market == nil {
		return
	}
	market.Status = effectiveMarketStatusAt(market.Status, market.CloseAt, market.ResolveAt, nowUnix, market.Metadata)
}
