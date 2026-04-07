package assets

import (
	"fmt"
	"math"
	"strings"
)

const DefaultCollateralAsset = "USDT"
const DefaultWinningSharePayout = int64(100)

func NormalizeAsset(asset string) string {
	normalized := strings.ToUpper(strings.TrimSpace(asset))
	if normalized == "" {
		return DefaultCollateralAsset
	}
	return normalized
}

func PositionAsset(marketID int64, outcome string) string {
	return fmt.Sprintf("POSITION:%d:%s", marketID, strings.ToUpper(strings.TrimSpace(outcome)))
}

func WinningPayoutAmount(quantity int64) (int64, error) {
	if quantity <= 0 {
		return 0, fmt.Errorf("winning quantity must be positive")
	}
	if quantity > math.MaxInt64/DefaultWinningSharePayout {
		return 0, fmt.Errorf("winning payout overflows int64")
	}
	return quantity * DefaultWinningSharePayout, nil
}
