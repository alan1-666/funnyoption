package order

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"funnyoption/internal/shared/assets"
)

// CalculateFreeze determines the asset and amount that must be frozen for an
// order before it can be submitted to the matching engine.
//
// BUY LIMIT  → freeze collateral = price × quantity
// SELL       → freeze position   = quantity
func CalculateFreeze(side, orderType string, marketID int64, outcome string, price, quantity int64) (asset string, amount int64, err error) {
	if quantity <= 0 {
		return "", 0, errors.New("quantity must be positive")
	}
	switch side {
	case "BUY":
		switch orderType {
		case "LIMIT":
			if price <= 0 {
				return "", 0, errors.New("limit order requires positive price")
			}
			if quantity > 0 && price > math.MaxInt64/quantity {
				return "", 0, errors.New("freeze amount overflow")
			}
			return assets.DefaultCollateralAsset, price * quantity, nil
		case "MARKET":
			// Market buy freezes the maximum possible cost (price=100 × quantity).
			maxPrice := int64(100)
			if quantity > 0 && maxPrice > math.MaxInt64/quantity {
				return "", 0, errors.New("freeze amount overflow")
			}
			return assets.DefaultCollateralAsset, maxPrice * quantity, nil
		default:
			return "", 0, fmt.Errorf("unsupported order type: %s", orderType)
		}
	case "SELL":
		if strings.TrimSpace(outcome) == "" {
			return "", 0, errors.New("sell order requires outcome")
		}
		return assets.PositionAsset(marketID, outcome), quantity, nil
	default:
		return "", 0, fmt.Errorf("unsupported side: %s", side)
	}
}
