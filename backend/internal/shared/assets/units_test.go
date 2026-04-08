package assets

import "testing"

func TestChainToAccountingAmountFloor_nativeLikeRemainder(t *testing.T) {
	// 1.234567 USDT chain (6 decimals) → accounting 2 decimals: floor to 123 cents
	got, err := ChainToAccountingAmountFloor(1234567, 6, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got != 123 {
		t.Fatalf("got %d want 123", got)
	}

	// Strict conversion must reject remainder
	_, err = ChainToAccountingAmount(1234567, 6, 2)
	if err == nil {
		t.Fatal("expected strict conversion error for non-aligned amount")
	}
}

func TestChainToAccountingAmountFloor_exactMultiple(t *testing.T) {
	got, err := ChainToAccountingAmountFloor(1_000_000, 6, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got != 100 {
		t.Fatalf("got %d want 100", got)
	}
}

func TestChainToAccountingAmountFloor_belowOneCent(t *testing.T) {
	got, err := ChainToAccountingAmountFloor(9999, 6, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Fatalf("got %d want 0", got)
	}
}
