package order

import "fmt"

// ValidateOrderFields performs upstream business validation that was previously
// handled inside the matching engine. Calling this before publishing to Kafka
// ensures invalid orders never reach the engine, keeping the hot path clean.
func ValidateOrderFields(outcome, side, orderType, tif string, price, quantity int64) error {
	if outcome == "" {
		return fmt.Errorf("outcome is required")
	}
	if side != "BUY" && side != "SELL" {
		return fmt.Errorf("invalid side: %s", side)
	}
	if orderType != "LIMIT" && orderType != "MARKET" {
		return fmt.Errorf("invalid order type: %s", orderType)
	}
	if tif != "GTC" && tif != "IOC" {
		return fmt.Errorf("invalid time_in_force: %s", tif)
	}
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if orderType == "LIMIT" && (price < 1 || price > 99) {
		return fmt.Errorf("limit order price must be between 1 and 99")
	}
	return nil
}
