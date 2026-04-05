package service

import (
	"fmt"
	"strings"

	"funnyoption/internal/matching/engine"
	"funnyoption/internal/matching/model"
	"funnyoption/internal/rollup"
	"funnyoption/internal/shared/assets"
	sharedkafka "funnyoption/internal/shared/kafka"
)

func buildRollupEntries(command sharedkafka.OrderCommand, result engine.Result) []rollup.JournalAppend {
	entries := make([]rollup.JournalAppend, 0, 2+len(result.Affected)+len(result.Trades))
	if result.Order != nil && result.Order.Status != model.OrderStatusRejected {
		entries = append(entries, rollup.JournalAppend{
			EntryType:        rollup.EntryTypeOrderAccepted,
			SourceType:       rollup.SourceTypeMatchingOrder,
			SourceRef:        result.Order.OrderID,
			OccurredAtMillis: command.RequestedAtMillis,
			Payload: rollup.OrderAcceptedPayload{
				OrderID:           result.Order.OrderID,
				CommandID:         command.CommandID,
				ClientOrderID:     command.ClientOrderID,
				AccountID:         command.UserID,
				MarketID:          command.MarketID,
				Outcome:           command.Outcome,
				Side:              command.Side,
				OrderType:         command.Type,
				TimeInForce:       command.TimeInForce,
				CollateralAsset:   command.CollateralAsset,
				ReserveAsset:      reserveAsset(command),
				ReserveAmount:     command.FreezeAmount,
				Price:             command.Price,
				Quantity:          command.Quantity,
				RequestedAtMillis: command.RequestedAtMillis,
			},
		})
		if result.Order.Status == model.OrderStatusCancelled {
			entries = append(entries, cancellationEntry(result.Order, result.Order.OrderID))
		}
	}
	for _, affected := range result.Affected {
		if affected != nil && affected.Status == model.OrderStatusCancelled {
			entries = append(entries, cancellationEntry(affected, affected.OrderID))
		}
	}
	for _, trade := range result.Trades {
		entries = append(entries, rollup.JournalAppend{
			EntryType:        rollup.EntryTypeTradeMatched,
			SourceType:       rollup.SourceTypeMatchingTrade,
			SourceRef:        fmt.Sprintf("%d", trade.Sequence),
			OccurredAtMillis: trade.MatchedAtMillis,
			Payload: rollup.TradeMatchedPayload{
				TradeID:          fmt.Sprintf("trd_%d", trade.Sequence),
				Sequence:         trade.Sequence,
				CollateralAsset:  command.CollateralAsset,
				MarketID:         trade.MarketID,
				Outcome:          trade.Outcome,
				Price:            trade.Price,
				Quantity:         trade.Quantity,
				TakerOrderID:     trade.TakerOrderID,
				MakerOrderID:     trade.MakerOrderID,
				TakerAccountID:   trade.TakerUserID,
				MakerAccountID:   trade.MakerUserID,
				TakerSide:        string(trade.TakerSide),
				MakerSide:        string(trade.MakerSide),
				OccurredAtMillis: trade.MatchedAtMillis,
			},
		})
	}
	return entries
}

func cancellationEntry(order *model.Order, sourceRef string) rollup.JournalAppend {
	return rollup.JournalAppend{
		EntryType:        rollup.EntryTypeOrderCancelled,
		SourceType:       rollup.SourceTypeMatchingOrder,
		SourceRef:        sourceRef,
		OccurredAtMillis: order.UpdatedAtMillis,
		Payload: rollup.OrderCancelledPayload{
			OrderID:           order.OrderID,
			AccountID:         order.UserID,
			MarketID:          order.MarketID,
			Outcome:           order.Outcome,
			Side:              string(order.Side),
			ReserveAsset:      cancellationReserveAsset(order),
			Price:             order.Price,
			RemainingQuantity: order.RemainingQuantity(),
			CancelReason:      string(order.CancelReason),
		},
	}
}

func reserveAsset(command sharedkafka.OrderCommand) string {
	if strings.TrimSpace(command.FreezeAsset) != "" {
		return command.FreezeAsset
	}
	return command.CollateralAsset
}

func cancellationReserveAsset(order *model.Order) string {
	if order == nil {
		return ""
	}
	switch order.Side {
	case model.OrderSideSell:
		return fmt.Sprintf("POSITION:%d:%s", order.MarketID, strings.ToUpper(strings.TrimSpace(order.Outcome)))
	default:
		return assetsOrFallback(order)
	}
}

func assetsOrFallback(order *model.Order) string {
	if order == nil {
		return ""
	}
	return assets.DefaultCollateralAsset
}
