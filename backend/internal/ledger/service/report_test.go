package service

import (
	"testing"

	"funnyoption/internal/ledger/model"
)

func TestBuildReport(t *testing.T) {
	report := BuildReport(
		[]model.LiabilitySnapshot{
			{
				Asset:                "usdt",
				UserAvailable:        100,
				UserFrozen:           50,
				PendingSettlement:    20,
				PendingWithdraw:      10,
				PlatformFeeLiability: 5,
			},
		},
		[]model.ChainSnapshot{
			{
				Asset:          "USDT",
				HotWallet:      185,
				ColdWallet:     0,
				ContractLocked: 0,
			},
		},
	)

	if len(report.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(report.Lines))
	}
	line := report.Lines[0]
	if line.Asset != "USDT" {
		t.Fatalf("unexpected asset: %s", line.Asset)
	}
	if line.InternalTotal != 185 || line.ChainTotal != 185 || line.Difference != 0 {
		t.Fatalf("unexpected report line: %+v", line)
	}
	if line.Status != "BALANCED" {
		t.Fatalf("unexpected status: %s", line.Status)
	}
}
