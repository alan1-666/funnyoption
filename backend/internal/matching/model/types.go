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
	TimeInForceGTC TimeInForce = "GTC"
	TimeInForceIOC TimeInForce = "IOC"
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
)

func (s OrderSide) IsValid() bool {
	return s == OrderSideBuy || s == OrderSideSell
}

func (t OrderType) IsValid() bool {
	return t == OrderTypeLimit || t == OrderTypeMarket
}

func (t TimeInForce) IsValid() bool {
	return t == TimeInForceGTC || t == TimeInForceIOC
}

func BuildBookKey(marketID int64, outcome string) string {
	out := strings.ToUpper(strings.TrimSpace(outcome))
	buf := make([]byte, 0, 20+len(out))
	buf = strconv.AppendInt(buf, marketID, 10)
	buf = append(buf, ':')
	buf = append(buf, out...)
	return string(buf)
}
