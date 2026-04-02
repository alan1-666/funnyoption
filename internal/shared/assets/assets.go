package assets

import (
	"fmt"
	"strings"
)

const DefaultCollateralAsset = "USDT"

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
