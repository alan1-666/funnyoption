package marketmaker

// Quote represents a single order to be placed or maintained.
type Quote struct {
	Outcome  string
	Price    int64
	Quantity int64
}

// QuoteSet is the full set of quotes the strategy wants on a single market.
type QuoteSet struct {
	MarketID int64
	Quotes   []Quote
}

// SpreadStrategy computes bilateral SELL quotes for a binary prediction market.
//
// In a binary market, selling YES at Py and NO at Pn creates a two-sided market
// where the effective YES bid = 100 - Pn and YES ask = Py. The maker earns
// (Py + Pn - 100) per complete round-trip.
type SpreadStrategy struct {
	DefaultMid    int64
	DefaultSpread int64
	Levels        int
	LevelSpacing  int64
	BaseQuantity  int64
	InventorySkew float64
}

func NewSpreadStrategy(cfg Config) *SpreadStrategy {
	spacing := int64(2)
	if cfg.DefaultSpread > 4 {
		spacing = cfg.DefaultSpread / 3
	}
	return &SpreadStrategy{
		DefaultMid:    cfg.DefaultMidPrice,
		DefaultSpread: cfg.DefaultSpread,
		Levels:        cfg.Levels,
		LevelSpacing:  spacing,
		BaseQuantity:  cfg.DefaultQuantity,
		InventorySkew: cfg.InventorySkew,
	}
}

// ComputeQuotes returns the desired SELL orders for both YES and NO outcomes.
// mid is the current mid-probability for YES (1-99). If 0, DefaultMid is used.
// yesInventory / noInventory are the current holdings of each outcome.
func (s *SpreadStrategy) ComputeQuotes(marketID int64, mid int64, yesInventory, noInventory int64) QuoteSet {
	if mid <= 0 || mid >= 100 {
		mid = s.DefaultMid
	}

	halfSpread := s.DefaultSpread / 2
	if halfSpread < 1 {
		halfSpread = 1
	}

	skewYes, skewNo := s.inventoryAdjustment(yesInventory, noInventory)

	var quotes []Quote

	for i := 0; i < s.Levels; i++ {
		offset := halfSpread + int64(i)*s.LevelSpacing
		qty := s.levelQuantity(i)

		yesAsk := mid + offset + skewYes
		if yesAsk = clampPrice(yesAsk); yesAsk > 0 && qty > 0 && int64(qty) <= yesInventory {
			quotes = append(quotes, Quote{Outcome: "YES", Price: yesAsk, Quantity: qty})
		}

		noAsk := (100 - mid) + offset + skewNo
		if noAsk = clampPrice(noAsk); noAsk > 0 && qty > 0 && int64(qty) <= noInventory {
			quotes = append(quotes, Quote{Outcome: "NO", Price: noAsk, Quantity: qty})
		}
	}

	return QuoteSet{MarketID: marketID, Quotes: quotes}
}

// inventoryAdjustment returns price adjustments for YES and NO sides based on
// current inventory imbalance. When one side is depleted, its ask rises to slow
// further consumption while the other side becomes cheaper.
func (s *SpreadStrategy) inventoryAdjustment(yesInv, noInv int64) (skewYes, skewNo int64) {
	if s.InventorySkew <= 0 || (yesInv == 0 && noInv == 0) {
		return 0, 0
	}
	total := float64(yesInv + noInv)
	if total == 0 {
		return 0, 0
	}
	yesFraction := float64(yesInv) / total
	// yesFraction > 0.5 means we have more YES than NO → lower YES ask (cheaper
	// to buy YES), raise NO ask (more expensive to buy NO)
	imbalance := (yesFraction - 0.5) * 2 // range [-1, 1]
	adj := int64(imbalance * s.InventorySkew * float64(s.DefaultSpread))
	return -adj, adj
}

func (s *SpreadStrategy) levelQuantity(level int) int64 {
	// Deeper levels get smaller quantity (outer quotes are less aggressive)
	divisor := int64(1) << uint(level)
	qty := s.BaseQuantity / divisor
	if qty < 1 {
		qty = 1
	}
	return qty
}

func clampPrice(price int64) int64 {
	if price < 1 {
		return 1
	}
	if price > 99 {
		return 99
	}
	return price
}
