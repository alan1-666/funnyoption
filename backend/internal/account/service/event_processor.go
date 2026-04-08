package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"funnyoption/internal/shared/assets"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type EventProcessor struct {
	book     *BalanceBook
	registry *OrderRegistry
	freeze   freezeStateReader
}

type freezeStateReader interface {
	RollupFrozen(ctx context.Context) (bool, error)
}

func NewEventProcessor(book *BalanceBook, registry *OrderRegistry, freezeReaders ...freezeStateReader) *EventProcessor {
	processor := &EventProcessor{
		book:     book,
		registry: registry,
	}
	if len(freezeReaders) > 0 {
		processor.freeze = freezeReaders[0]
	}
	return processor
}

func (p *EventProcessor) rollupFrozen(ctx context.Context) (bool, error) {
	if p.freeze == nil {
		return false, nil
	}
	return p.freeze.RollupFrozen(ctx)
}

func (p *EventProcessor) HandleOrderEvent(ctx context.Context, msg sharedkafka.Message) error {
	frozen, err := p.rollupFrozen(ctx)
	if err != nil {
		return err
	}
	if frozen {
		return nil
	}

	var event sharedkafka.OrderEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}

	state := p.registry.Upsert(OrderState{
		OrderID:           event.OrderID,
		UserID:            event.UserID,
		Side:              event.Side,
		Price:             event.Price,
		FreezeID:          event.FreezeID,
		FreezeAsset:       freezeAssetFromOrderEvent(event),
		Status:            event.Status,
		RemainingQuantity: event.RemainingQuantity,
	})

	if event.FreezeID != "" && !state.FreezeApplied && event.FreezeAmount > 0 {
		if _, err := p.book.ApplyExternalFreeze(FreezeRequest{
			FreezeID: event.FreezeID,
			UserID:   event.UserID,
			Asset:    freezeAsset(event),
			RefType:  "ORDER",
			RefID:    event.OrderID,
			Amount:   event.FreezeAmount,
		}); err != nil && !errors.Is(err, ErrFreezeAlreadyExists) {
			return err
		}
		state = p.registry.Upsert(OrderState{
			OrderID:           event.OrderID,
			UserID:            state.UserID,
			Side:              state.Side,
			Price:             state.Price,
			FreezeID:          event.FreezeID,
			FreezeAsset:       freezeAssetFromOrderEvent(event),
			FreezeApplied:     true,
			Status:            state.Status,
			RemainingQuantity: state.RemainingQuantity,
		})
	}

	if shouldReleaseFreeze(event.Status, event.RemainingQuantity) && state.FreezeID != "" {
		if _, ok := p.book.FreezeAmount(state.FreezeID); ok {
			if err := p.book.ReleaseFreeze(state.FreezeID); err != nil && !errors.Is(err, ErrFreezeAlreadyClosed) {
				return err
			}
		}
	}
	return nil
}

func (p *EventProcessor) HandleTradeMatched(ctx context.Context, msg sharedkafka.Message) error {
	frozen, err := p.rollupFrozen(ctx)
	if err != nil {
		return err
	}
	if frozen {
		return nil
	}

	var event sharedkafka.TradeMatchedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}

	// Use deterministic TradeID as idempotency key for replay safety.
	// Falls back to EventID for backward compatibility with pre-V2 trades.
	idempotencyKey := event.TradeID
	if idempotencyKey == "" {
		idempotencyKey = event.EventID
	}

	notional, err := tradeNotional(event.Price, event.Quantity)
	if err != nil {
		return err
	}

	payerOrderID, receiverUserID := settlementSides(event)
	sellerOrderID, buyerUserID := positionSides(event)
	if payerOrderID != "" {
		if state, ok := p.registry.Get(payerOrderID); ok && state.FreezeID != "" {
			if err := p.book.ConsumeFreeze(state.FreezeID, notional); err != nil && !errors.Is(err, ErrFreezeAlreadyClosed) {
				return err
			}
			if err := p.releaseExcessFreeze(*state); err != nil {
				return err
			}
		}
	}
	if sellerOrderID != "" {
		if state, ok := p.registry.Get(sellerOrderID); ok && state.FreezeID != "" {
			if err := p.book.ConsumeFreeze(state.FreezeID, event.Quantity); err != nil && !errors.Is(err, ErrFreezeAlreadyClosed) {
				return err
			}
			if err := p.releaseExcessFreeze(*state); err != nil {
				return err
			}
		}
	}

	receiverFee, payerFee := tradeFees(event)

	if receiverUserID > 0 {
		netCredit := notional - receiverFee
		if netCredit > 0 {
			if _, _, err := p.book.CreditAvailableWithRef(CreditRequest{
				UserID:  receiverUserID,
				Asset:   collateralAsset(event.CollateralAsset),
				Amount:  netCredit,
				RefType: "TRADE_COLLATERAL",
				RefID:   idempotencyKey,
			}); err != nil {
				return err
			}
		}
	}
	if buyerUserID > 0 {
		if _, _, err := p.book.CreditAvailableWithRef(CreditRequest{
			UserID:  buyerUserID,
			Asset:   assets.PositionAsset(event.MarketID, event.Outcome),
			Amount:  event.Quantity,
			RefType: "TRADE_POSITION",
			RefID:   idempotencyKey,
		}); err != nil {
			return err
		}
	}

	platformRevenue := receiverFee + payerFee
	if platformRevenue > 0 {
		if _, _, err := p.book.CreditAvailableWithRef(CreditRequest{
			UserID:  PlatformFeeUserID,
			Asset:   collateralAsset(event.CollateralAsset),
			Amount:  platformRevenue,
			RefType: "TRADE_FEE",
			RefID:   idempotencyKey,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (p *EventProcessor) HandleSettlementCompleted(ctx context.Context, msg sharedkafka.Message) error {
	frozen, err := p.rollupFrozen(ctx)
	if err != nil {
		return err
	}
	if frozen {
		return nil
	}

	var event sharedkafka.SettlementCompletedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}

	if event.SettledQuantity > 0 {
		if _, _, err := p.book.DebitAvailableWithRef(DebitRequest{
			UserID:  event.UserID,
			Asset:   event.PositionAsset,
			Amount:  event.SettledQuantity,
			RefType: "SETTLEMENT_DEBIT",
			RefID:   event.EventID,
		}); err != nil {
			return err
		}
	}
	if event.PayoutAmount > 0 {
		if _, _, err := p.book.CreditAvailableWithRef(CreditRequest{
			UserID:  event.UserID,
			Asset:   collateralAsset(event.PayoutAsset),
			Amount:  event.PayoutAmount,
			RefType: "SETTLEMENT_CREDIT",
			RefID:   event.EventID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (p *EventProcessor) releaseExcessFreeze(state OrderState) error {
	current, ok := p.book.FreezeAmount(state.FreezeID)
	if !ok {
		return nil
	}

	expected, err := expectedFreezeAmount(state)
	if err != nil {
		return err
	}
	if current <= expected {
		return nil
	}
	return p.book.ReleaseFreezeAmount(state.FreezeID, current-expected)
}

func expectedFreezeAmount(state OrderState) (int64, error) {
	remaining := state.RemainingQuantity
	if remaining < 0 {
		return 0, fmt.Errorf("remaining quantity must not be negative")
	}

	switch strings.ToUpper(strings.TrimSpace(state.Side)) {
	case "BUY":
		if remaining == 0 {
			return 0, nil
		}
		if state.Price < 0 {
			return 0, fmt.Errorf("price must not be negative")
		}
		if state.Price > 0 && remaining > math.MaxInt64/state.Price {
			return 0, fmt.Errorf("freeze reserve overflow")
		}
		return state.Price * remaining, nil
	case "SELL":
		return remaining, nil
	default:
		return 0, fmt.Errorf("unsupported side: %s", state.Side)
	}
}

func freezeAssetFromOrderEvent(event sharedkafka.OrderEvent) string {
	if strings.TrimSpace(event.FreezeID) == "" {
		return ""
	}
	return freezeAsset(event)
}

func shouldReleaseFreeze(status string, remainingQuantity int64) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "CANCELLED", "REJECTED":
		return remainingQuantity > 0
	default:
		return false
	}
}

func freezeAsset(event sharedkafka.OrderEvent) string {
	if strings.TrimSpace(event.FreezeAsset) != "" {
		return collateralAsset(event.FreezeAsset)
	}
	return collateralAsset(event.CollateralAsset)
}

func collateralAsset(asset string) string {
	return assets.NormalizeAsset(asset)
}

func settlementSides(event sharedkafka.TradeMatchedEvent) (payerOrderID string, receiverUserID int64) {
	switch strings.ToUpper(strings.TrimSpace(event.TakerSide)) {
	case "BUY":
		return event.TakerOrderID, event.MakerUserID
	case "SELL":
		return event.MakerOrderID, event.TakerUserID
	default:
		return "", 0
	}
}

func positionSides(event sharedkafka.TradeMatchedEvent) (sellerOrderID string, buyerUserID int64) {
	switch strings.ToUpper(strings.TrimSpace(event.TakerSide)) {
	case "BUY":
		return event.MakerOrderID, event.TakerUserID
	case "SELL":
		return event.TakerOrderID, event.MakerUserID
	default:
		return "", 0
	}
}

func tradeNotional(price, quantity int64) (int64, error) {
	if price <= 0 || quantity <= 0 {
		return 0, fmt.Errorf("trade notional requires positive price and quantity")
	}
	if price > math.MaxInt64/quantity {
		return 0, fmt.Errorf("trade notional overflow")
	}
	return price * quantity, nil
}

// PlatformFeeUserID is the synthetic user account that accumulates trading fees.
const PlatformFeeUserID int64 = 0

// tradeFees extracts the fee amounts for the collateral receiver and payer from
// the trade event. Returns (receiverFee, payerFee) where receiver is the seller
// (who receives collateral) and payer is the buyer (who pays collateral).
// A negative fee means a rebate.
func tradeFees(event sharedkafka.TradeMatchedEvent) (receiverFee, payerFee int64) {
	switch strings.ToUpper(strings.TrimSpace(event.TakerSide)) {
	case "BUY":
		// Taker buys → taker is the payer, maker is the receiver (seller)
		return event.MakerFee, event.TakerFee
	case "SELL":
		// Taker sells → taker is the receiver, maker is the payer
		return event.TakerFee, event.MakerFee
	default:
		return 0, 0
	}
}
