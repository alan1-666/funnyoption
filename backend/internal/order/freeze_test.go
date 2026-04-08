package order

import "testing"

func TestCalculateFreezeBuyLimit(t *testing.T) {
	asset, amount, err := CalculateFreeze("BUY", "LIMIT", 1, "YES", 55, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asset != "USDT" {
		t.Errorf("asset: want USDT, got %s", asset)
	}
	if amount != 550 {
		t.Errorf("amount: want 550, got %d", amount)
	}
}

func TestCalculateFreezeSell(t *testing.T) {
	asset, amount, err := CalculateFreeze("SELL", "LIMIT", 42, "YES", 55, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asset != "POSITION:42:YES" {
		t.Errorf("asset: want POSITION:42:YES, got %s", asset)
	}
	if amount != 10 {
		t.Errorf("amount: want 10, got %d", amount)
	}
}

func TestCalculateFreezeZeroQuantity(t *testing.T) {
	_, _, err := CalculateFreeze("BUY", "LIMIT", 1, "YES", 55, 0)
	if err == nil {
		t.Fatal("expected error for zero quantity")
	}
}

func TestCalculateFreezeSellNoOutcome(t *testing.T) {
	_, _, err := CalculateFreeze("SELL", "LIMIT", 1, "", 55, 10)
	if err == nil {
		t.Fatal("expected error for empty outcome on sell")
	}
}

func TestCalculateFreezeOverflow(t *testing.T) {
	_, _, err := CalculateFreeze("BUY", "LIMIT", 1, "YES", 1<<50, 1<<50)
	if err == nil {
		t.Fatal("expected overflow error")
	}
}

func TestCalculateFreezeUnsupportedSide(t *testing.T) {
	_, _, err := CalculateFreeze("SWAP", "LIMIT", 1, "YES", 55, 10)
	if err == nil {
		t.Fatal("expected error for unsupported side")
	}
}

func TestCalculateFreezeBuyMarketOrder(t *testing.T) {
	asset, amount, err := CalculateFreeze("BUY", "MARKET", 1, "YES", 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asset != "USDT" {
		t.Errorf("asset: want USDT, got %s", asset)
	}
	if amount != 1000 {
		t.Errorf("amount: want 1000 (100*10), got %d", amount)
	}
}

func TestCalculateFreezeSellMarketOrder(t *testing.T) {
	asset, amount, err := CalculateFreeze("SELL", "MARKET", 42, "YES", 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asset != "POSITION:42:YES" {
		t.Errorf("asset: want POSITION:42:YES, got %s", asset)
	}
	if amount != 10 {
		t.Errorf("amount: want 10, got %d", amount)
	}
}
