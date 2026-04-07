package order

import "fmt"

// SubmitRequest is the transport-agnostic representation of an incoming order.
type SubmitRequest struct {
	UserID        int64
	MarketID      int64
	Outcome       string
	Side          string
	Type          string
	TimeInForce   string
	Price         int64
	Quantity      int64
	TraceID       string
	ClientOrderID string
	RequestedAt   int64
	// OrderID, if non-empty, is used as the deterministic order ID (e.g. for
	// bootstrap replay protection). When empty, a random ID is generated.
	OrderID string
}

// SubmitResult is the outcome of a successfully queued order.
type SubmitResult struct {
	CommandID string
	OrderID   string
	FreezeID  string
	Asset     string
	Amount    int64
}

// SeedLiquidityRequest represents a request to mint a complete set and place a
// bootstrap sell order for initial market liquidity.
type SeedLiquidityRequest struct {
	OperatorUserID int64
	MarketID       int64
	Outcome        string
	Quantity       int64
	Price          int64
	TraceID        string
}

// SeedLiquidityResult is the outcome of a successful liquidity seed.
type SeedLiquidityResult struct {
	FirstLiquidityID string
	OrderID          string
	CollateralDebit  int64
	Inventory        []InventoryItem
}

// InventoryItem describes one outcome's inventory created during seeding.
type InventoryItem struct {
	Outcome       string
	PositionAsset string
	Quantity      int64
}

// Errors returned by the order service.
var (
	ErrMarketNotFound     = fmt.Errorf("market not found")
	ErrMarketNotTradable  = fmt.Errorf("market is not tradable")
	ErrRollupFrozen       = fmt.Errorf("rollup is frozen")
	ErrInsufficientFunds  = fmt.Errorf("insufficient funds")
	ErrAlreadySeeded      = fmt.Errorf("liquidity already seeded")
	ErrValidationFailed   = fmt.Errorf("order validation failed")
	ErrPublishFailed      = fmt.Errorf("order publish failed")
)
