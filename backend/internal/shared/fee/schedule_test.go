package fee

import "testing"

func TestDefaultScheduleCompute(t *testing.T) {
	s := DefaultSchedule()
	// notional = 55 * 10 = 550 (price 55, qty 10)
	result, err := s.Compute(550)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// taker: 550 × 200 / 10000 = 11
	if result.TakerFee != 11 {
		t.Errorf("taker fee: want 11, got %d", result.TakerFee)
	}
	// maker: 550 × (-50) / 10000 = -2 (rebate)
	if result.MakerFee != -2 {
		t.Errorf("maker fee: want -2, got %d", result.MakerFee)
	}
	// platform revenue: 11 + (-2) = 9
	if result.PlatformRevenue() != 9 {
		t.Errorf("platform revenue: want 9, got %d", result.PlatformRevenue())
	}
}

func TestZeroNotional(t *testing.T) {
	s := DefaultSchedule()
	result, err := s.Compute(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TakerFee != 0 || result.MakerFee != 0 {
		t.Errorf("expected zero fees for zero notional, got taker=%d maker=%d", result.TakerFee, result.MakerFee)
	}
}

func TestZeroFeeSchedule(t *testing.T) {
	s := Schedule{TakerFeeBps: 0, MakerFeeBps: 0}
	result, err := s.Compute(1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TakerFee != 0 || result.MakerFee != 0 {
		t.Errorf("expected zero fees, got taker=%d maker=%d", result.TakerFee, result.MakerFee)
	}
}

func TestMakerOnlyRebate(t *testing.T) {
	s := Schedule{TakerFeeBps: 100, MakerFeeBps: -100}
	result, err := s.Compute(10000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// taker: 10000 × 100 / 10000 = 100
	if result.TakerFee != 100 {
		t.Errorf("taker fee: want 100, got %d", result.TakerFee)
	}
	// maker: 10000 × (-100) / 10000 = -100 (rebate)
	if result.MakerFee != -100 {
		t.Errorf("maker fee: want -100, got %d", result.MakerFee)
	}
	// net zero to platform
	if result.PlatformRevenue() != 0 {
		t.Errorf("platform revenue: want 0, got %d", result.PlatformRevenue())
	}
}

func TestSmallNotionalRounding(t *testing.T) {
	s := Schedule{TakerFeeBps: 50, MakerFeeBps: -25}
	// notional = 10 → taker: 10×50/10000 = 0 (rounds to zero)
	result, err := s.Compute(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TakerFee != 0 {
		t.Errorf("expected 0 taker fee on small notional, got %d", result.TakerFee)
	}
}

func TestNetCreditHelpers(t *testing.T) {
	result := FeeResult{TakerFee: 10, MakerFee: -5}
	if net := result.NetTakerCredit(100); net != 90 {
		t.Errorf("net taker credit: want 90, got %d", net)
	}
	if net := result.NetMakerCredit(100); net != 105 {
		t.Errorf("net maker credit: want 105, got %d", net)
	}
}
