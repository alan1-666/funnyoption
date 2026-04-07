package handler

import (
	"encoding/json"
	"testing"

	"funnyoption/internal/api/dto"
)

func TestEffectiveMarketStatusAtClosesExpiredOpenMarkets(t *testing.T) {
	t.Parallel()

	if got := effectiveMarketStatusAt("OPEN", 100, 140, 99, nil); got != "OPEN" {
		t.Fatalf("effectiveMarketStatusAt() before close = %s, want OPEN", got)
	}
	if got := effectiveMarketStatusAt("OPEN", 100, 140, 100, nil); got != "CLOSED" {
		t.Fatalf("effectiveMarketStatusAt() at close boundary = %s, want CLOSED", got)
	}
	if got := effectiveMarketStatusAt("OPEN", 100, 140, 139, nil); got != "CLOSED" {
		t.Fatalf("effectiveMarketStatusAt() before resolve_at = %s, want CLOSED", got)
	}
	if got := effectiveMarketStatusAt("RESOLVED", 100, 140, 140, nil); got != "RESOLVED" {
		t.Fatalf("effectiveMarketStatusAt() for resolved market = %s, want RESOLVED", got)
	}
}

func TestEffectiveMarketStatusAtTransitionsManualMarketsToWaitingResolution(t *testing.T) {
	t.Parallel()

	if got := effectiveMarketStatusAt("OPEN", 100, 120, 120, nil); got != "WAITING_RESOLUTION" {
		t.Fatalf("effectiveMarketStatusAt() at resolve_at = %s, want WAITING_RESOLUTION", got)
	}
	if got := effectiveMarketStatusAt("OPEN", 100, 0, 100, nil); got != "WAITING_RESOLUTION" {
		t.Fatalf("effectiveMarketStatusAt() without resolve_at = %s, want WAITING_RESOLUTION", got)
	}
}

func TestEffectiveMarketStatusAtKeepsOracleMarketsClosedUntilResolutionEvent(t *testing.T) {
	t.Parallel()

	oracleMetadata := json.RawMessage(`{"resolution":{"mode":"ORACLE_PRICE"}}`)
	if got := effectiveMarketStatusAt("OPEN", 100, 120, 120, oracleMetadata); got != "CLOSED" {
		t.Fatalf("effectiveMarketStatusAt() for oracle market = %s, want CLOSED", got)
	}
}

func TestApplyEffectiveMarketStatusClosesUnresolvedResponseAfterCloseAt(t *testing.T) {
	t.Parallel()

	market := dto.MarketResponse{
		MarketID:  77,
		Status:    "OPEN",
		CloseAt:   50,
		ResolveAt: 75,
	}

	applyEffectiveMarketStatus(&market, 50)

	if market.Status != "CLOSED" {
		t.Fatalf("market.Status = %s, want CLOSED", market.Status)
	}
}

func TestApplyEffectiveMarketStatusMarksManualMarketWaitingResolutionAtResolveAt(t *testing.T) {
	t.Parallel()

	market := dto.MarketResponse{
		MarketID:  77,
		Status:    "OPEN",
		CloseAt:   50,
		ResolveAt: 75,
	}

	applyEffectiveMarketStatus(&market, 75)

	if market.Status != "WAITING_RESOLUTION" {
		t.Fatalf("market.Status = %s, want WAITING_RESOLUTION", market.Status)
	}
}
