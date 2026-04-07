package fee

import (
	"fmt"
	"math"
)

// BasisPoint is one hundredth of a percent (1 bp = 0.01%).
const BasisPoint = int64(1)

// Schedule holds maker and taker fee rates and provides fee calculation.
type Schedule struct {
	TakerFeeBps int64 // taker fee in basis points (e.g. 50 = 0.50%)
	MakerFeeBps int64 // maker fee in basis points (negative = rebate)
}

// DefaultSchedule returns a standard prediction-market fee schedule:
//   - Taker pays 2% (200 bps) of notional
//   - Maker receives 0.5% rebate (-50 bps)
func DefaultSchedule() Schedule {
	return Schedule{
		TakerFeeBps: 200,
		MakerFeeBps: -50,
	}
}

// FeeResult holds the computed fees for both sides of a trade.
type FeeResult struct {
	TakerFee int64 // positive = taker pays; negative = taker receives
	MakerFee int64 // positive = maker pays; negative = maker receives rebate
}

// NetTakerCredit returns the collateral the taker should receive after fees.
// Only applicable when taker is seller (receiving collateral).
func (r FeeResult) NetTakerCredit(grossCredit int64) int64 {
	return grossCredit - r.TakerFee
}

// NetMakerCredit returns the collateral the maker should receive after fees.
// Only applicable when maker is seller (receiving collateral).
func (r FeeResult) NetMakerCredit(grossCredit int64) int64 {
	return grossCredit - r.MakerFee
}

// PlatformRevenue returns the total fee revenue the platform collects.
func (r FeeResult) PlatformRevenue() int64 {
	return r.TakerFee + r.MakerFee
}

// Compute calculates maker and taker fees for a trade.
// notional = price × quantity (the collateral amount exchanged).
func (s Schedule) Compute(notional int64) (FeeResult, error) {
	if notional <= 0 {
		return FeeResult{}, nil
	}
	takerFee, err := applyBps(notional, s.TakerFeeBps)
	if err != nil {
		return FeeResult{}, fmt.Errorf("taker fee overflow: %w", err)
	}
	makerFee, err := applyBps(notional, s.MakerFeeBps)
	if err != nil {
		return FeeResult{}, fmt.Errorf("maker fee overflow: %w", err)
	}
	return FeeResult{TakerFee: takerFee, MakerFee: makerFee}, nil
}

// applyBps calculates amount × bps / 10000, rounding toward zero.
// Supports negative bps (rebates).
func applyBps(amount, bps int64) (int64, error) {
	if bps == 0 {
		return 0, nil
	}
	absBps := bps
	if absBps < 0 {
		absBps = -absBps
	}
	if amount > math.MaxInt64/absBps {
		return 0, fmt.Errorf("fee calculation overflow: %d × %d", amount, bps)
	}
	result := amount * bps / 10000
	return result, nil
}
