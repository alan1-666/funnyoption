package handler

import (
	"testing"

	"funnyoption/internal/api/dto"
)

func TestEffectiveMarketStatusAtClosesExpiredOpenMarkets(t *testing.T) {
	t.Parallel()

	if got := effectiveMarketStatusAt("OPEN", 100, 99); got != "OPEN" {
		t.Fatalf("effectiveMarketStatusAt() before close = %s, want OPEN", got)
	}
	if got := effectiveMarketStatusAt("OPEN", 100, 100); got != "CLOSED" {
		t.Fatalf("effectiveMarketStatusAt() at close boundary = %s, want CLOSED", got)
	}
	if got := effectiveMarketStatusAt("RESOLVED", 100, 100); got != "RESOLVED" {
		t.Fatalf("effectiveMarketStatusAt() for resolved market = %s, want RESOLVED", got)
	}
}

func TestApplyEffectiveMarketStatusClosesUnresolvedResponseAfterCloseAt(t *testing.T) {
	t.Parallel()

	market := dto.MarketResponse{
		MarketID: 77,
		Status:   "OPEN",
		CloseAt:  50,
	}

	applyEffectiveMarketStatus(&market, 50)

	if market.Status != "CLOSED" {
		t.Fatalf("market.Status = %s, want CLOSED", market.Status)
	}
}
