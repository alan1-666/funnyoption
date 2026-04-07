package marketmaker

import "testing"

func TestSpreadStrategyBasicQuotes(t *testing.T) {
	s := &SpreadStrategy{
		DefaultMid:    50,
		DefaultSpread: 6,
		Levels:        1,
		LevelSpacing:  2,
		BaseQuantity:  100,
		InventorySkew: 0,
	}

	qs := s.ComputeQuotes(1, 50, 200, 200)
	if len(qs.Quotes) != 2 {
		t.Fatalf("expected 2 quotes, got %d", len(qs.Quotes))
	}

	var yes, no *Quote
	for i := range qs.Quotes {
		switch qs.Quotes[i].Outcome {
		case "YES":
			yes = &qs.Quotes[i]
		case "NO":
			no = &qs.Quotes[i]
		}
	}
	if yes == nil || no == nil {
		t.Fatalf("expected both YES and NO quotes")
	}
	// YES ask = 50 + 3 = 53
	if yes.Price != 53 {
		t.Errorf("YES price: want 53, got %d", yes.Price)
	}
	// NO ask = (100 - 50) + 3 = 53
	if no.Price != 53 {
		t.Errorf("NO price: want 53, got %d", no.Price)
	}
	// Total: 53 + 53 = 106, spread = 6 (matches config)
	if yes.Price+no.Price-100 != 6 {
		t.Errorf("effective spread: want 6, got %d", yes.Price+no.Price-100)
	}
}

func TestSpreadStrategyMultipleLevels(t *testing.T) {
	s := &SpreadStrategy{
		DefaultMid:    50,
		DefaultSpread: 6,
		Levels:        3,
		LevelSpacing:  2,
		BaseQuantity:  100,
		InventorySkew: 0,
	}

	qs := s.ComputeQuotes(1, 50, 500, 500)
	if len(qs.Quotes) != 6 {
		t.Fatalf("expected 6 quotes (3 levels × 2 outcomes), got %d", len(qs.Quotes))
	}

	yesPrices := make(map[int64]int64)
	noPrices := make(map[int64]int64)
	for _, q := range qs.Quotes {
		switch q.Outcome {
		case "YES":
			yesPrices[q.Price] = q.Quantity
		case "NO":
			noPrices[q.Price] = q.Quantity
		}
	}

	// Level 0: 53, Level 1: 55, Level 2: 57
	for _, p := range []int64{53, 55, 57} {
		if _, ok := yesPrices[p]; !ok {
			t.Errorf("missing YES quote at price %d", p)
		}
		if _, ok := noPrices[p]; !ok {
			t.Errorf("missing NO quote at price %d", p)
		}
	}

	// Quantities decrease: 100, 50, 25
	if yesPrices[53] != 100 {
		t.Errorf("level 0 quantity: want 100, got %d", yesPrices[53])
	}
	if yesPrices[55] != 50 {
		t.Errorf("level 1 quantity: want 50, got %d", yesPrices[55])
	}
	if yesPrices[57] != 25 {
		t.Errorf("level 2 quantity: want 25, got %d", yesPrices[57])
	}
}

func TestSpreadStrategyInventorySkew(t *testing.T) {
	s := &SpreadStrategy{
		DefaultMid:    50,
		DefaultSpread: 10,
		Levels:        1,
		LevelSpacing:  2,
		BaseQuantity:  100,
		InventorySkew: 1.0,
	}

	// Bot holds 300 YES but only 100 NO → imbalance toward YES
	qs := s.ComputeQuotes(1, 50, 300, 100)

	var yes, no *Quote
	for i := range qs.Quotes {
		switch qs.Quotes[i].Outcome {
		case "YES":
			yes = &qs.Quotes[i]
		case "NO":
			no = &qs.Quotes[i]
		}
	}
	if yes == nil || no == nil {
		t.Fatalf("expected both YES and NO quotes, got %d", len(qs.Quotes))
	}

	// More YES inventory → YES should be cheaper (lower ask) to encourage buying
	// and NO should be more expensive (higher ask) to discourage
	baseYesAsk := int64(55) // mid(50) + halfSpread(5)
	if yes.Price >= baseYesAsk {
		t.Errorf("YES ask should be below base %d due to excess YES inventory, got %d", baseYesAsk, yes.Price)
	}
}

func TestSpreadStrategyInsufficientInventory(t *testing.T) {
	s := &SpreadStrategy{
		DefaultMid:    50,
		DefaultSpread: 6,
		Levels:        1,
		LevelSpacing:  2,
		BaseQuantity:  100,
		InventorySkew: 0,
	}

	// Not enough inventory to cover a 100-unit quote
	qs := s.ComputeQuotes(1, 50, 50, 50)
	if len(qs.Quotes) != 0 {
		t.Errorf("expected 0 quotes when inventory < base quantity, got %d", len(qs.Quotes))
	}
}

func TestSpreadStrategyPriceClamp(t *testing.T) {
	s := &SpreadStrategy{
		DefaultMid:    95,
		DefaultSpread: 10,
		Levels:        1,
		LevelSpacing:  2,
		BaseQuantity:  10,
		InventorySkew: 0,
	}

	qs := s.ComputeQuotes(1, 95, 100, 100)
	for _, q := range qs.Quotes {
		if q.Price < 1 || q.Price > 99 {
			t.Errorf("price %d out of [1,99] range for outcome %s", q.Price, q.Outcome)
		}
	}
}

func TestClampPrice(t *testing.T) {
	tests := []struct {
		input, want int64
	}{
		{0, 1}, {-5, 1}, {1, 1}, {50, 50}, {99, 99}, {100, 99}, {150, 99},
	}
	for _, tt := range tests {
		got := clampPrice(tt.input)
		if got != tt.want {
			t.Errorf("clampPrice(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
