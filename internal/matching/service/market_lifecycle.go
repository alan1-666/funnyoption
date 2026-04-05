package service

import "strings"

func effectiveMarketStatusAt(status string, closeAt, nowUnix int64) string {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	switch normalized {
	case "", "OPEN":
		normalized = "OPEN"
	case "DRAFT", "PAUSED", "CLOSED", "WAITING_RESOLUTION", "RESOLVED":
	default:
		normalized = "OPEN"
	}
	if normalized == "OPEN" && closeAt > 0 && nowUnix >= closeAt {
		return "CLOSED"
	}
	return normalized
}

func marketTradingOpen(status string, closeAt, nowUnix int64) bool {
	return effectiveMarketStatusAt(status, closeAt, nowUnix) == "OPEN"
}
