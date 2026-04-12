package model

import (
	"strconv"
	"strings"
)

type OrderSide string

type OrderType string

type TimeInForce string

type OrderStatus string

type CancelReason string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

const (
	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeMarket OrderType = "MARKET"
)

const (
	TimeInForceGTC      TimeInForce = "GTC"
	TimeInForceIOC      TimeInForce = "IOC"
	TimeInForceFOK      TimeInForce = "FOK"
	TimeInForcePostOnly TimeInForce = "POST_ONLY"
)

// STPStrategy controls self-trade prevention behavior when a taker's order
// would match against a maker order from the same user.
type STPStrategy string

const (
	STPNone         STPStrategy = ""             // no STP — allow self-trade
	STPCancelTaker  STPStrategy = "CANCEL_TAKER" // cancel taker, keep maker (protect liquidity)
	STPCancelMaker  STPStrategy = "CANCEL_MAKER" // cancel maker, continue matching (market-maker rebalance)
	STPCancelBoth   STPStrategy = "CANCEL_BOTH"  // cancel both taker and maker
)

const (
	OrderStatusNew             OrderStatus = "NEW"
	OrderStatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	OrderStatusFilled          OrderStatus = "FILLED"
	OrderStatusCancelled       OrderStatus = "CANCELLED"
	OrderStatusRejected        OrderStatus = "REJECTED"
)

const (
	CancelReasonNone              CancelReason = "NONE"
	CancelReasonIOCNoLiquidity    CancelReason = "IOC_NO_LIQUIDITY"
	CancelReasonIOCPartialFill    CancelReason = "IOC_PARTIAL_FILL"
	CancelReasonMarketClosed      CancelReason = "MARKET_CLOSED"
	CancelReasonMarketNoLiquidity CancelReason = "MARKET_NO_LIQUIDITY"
	CancelReasonMarketNotTradable CancelReason = "MARKET_NOT_TRADABLE"
	CancelReasonMarketResolved    CancelReason = "MARKET_RESOLVED"
	CancelReasonValidationFailed  CancelReason = "VALIDATION_FAILED"
	CancelReasonSTPTaker          CancelReason = "STP_CANCEL_TAKER"
	CancelReasonSTPMaker          CancelReason = "STP_CANCEL_MAKER"
	CancelReasonSTPBoth           CancelReason = "STP_CANCEL_BOTH"
	CancelReasonFOKNotFilled      CancelReason = "FOK_NOT_FILLED"
	CancelReasonPostOnlyCross     CancelReason = "POST_ONLY_CROSS"
	CancelReasonAmended           CancelReason = "AMENDED"
)

func (s OrderSide) IsValid() bool {
	return s == OrderSideBuy || s == OrderSideSell
}

func (t OrderType) IsValid() bool {
	return t == OrderTypeLimit || t == OrderTypeMarket
}

func (t TimeInForce) IsValid() bool {
	switch t {
	case TimeInForceGTC, TimeInForceIOC, TimeInForceFOK, TimeInForcePostOnly:
		return true
	}
	return false
}

func BuildBookKey(marketID int64, outcome string) string {
	out := strings.ToUpper(strings.TrimSpace(outcome))
	buf := make([]byte, 0, 20+len(out))
	buf = strconv.AppendInt(buf, marketID, 10)
	buf = append(buf, ':')
	buf = append(buf, out...)
	return string(buf)
}
