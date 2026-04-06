package service

import "testing"

func TestMarketTradingOpenRespectsCloseAtBoundary(t *testing.T) {
	t.Parallel()

	if !marketTradingOpen("OPEN", 100, 99) {
		t.Fatalf("expected open market before close_at to remain tradable")
	}
	if marketTradingOpen("OPEN", 100, 100) {
		t.Fatalf("expected close_at boundary to stop trading")
	}
	if marketTradingOpen("RESOLVED", 0, 10) {
		t.Fatalf("expected resolved market to remain non-tradable")
	}
	if marketTradingOpen("WAITING_RESOLUTION", 0, 10) {
		t.Fatalf("expected waiting-resolution market to remain non-tradable")
	}
}

func TestMarketTradingEnabledRespectsFrozenState(t *testing.T) {
	t.Parallel()

	if !marketTradingEnabled(false, "OPEN", 100, 99) {
		t.Fatalf("expected unfrozen open market to remain tradable")
	}
	if marketTradingEnabled(true, "OPEN", 100, 99) {
		t.Fatalf("expected frozen rollup to stop trading even before close_at")
	}
}
