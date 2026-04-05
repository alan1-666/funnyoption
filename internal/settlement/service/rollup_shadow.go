package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"funnyoption/internal/rollup"
	sharedkafka "funnyoption/internal/shared/kafka"
)

func buildMarketResolvedEntry(input ResolveMarketInput, record resolutionRecord) rollup.JournalAppend {
	return rollup.JournalAppend{
		EntryType:        rollup.EntryTypeMarketResolved,
		SourceType:       rollup.SourceTypeSettlementMarket,
		SourceRef:        fmt.Sprintf("%d", input.MarketID),
		OccurredAtMillis: input.OccurredAtMillis,
		Payload: rollup.MarketResolvedPayload{
			MarketID:         input.MarketID,
			ResolvedOutcome:  input.ResolvedOutcome,
			ResolverType:     strings.ToUpper(strings.TrimSpace(record.ResolverType)),
			ResolverRef:      strings.TrimSpace(record.ResolverRef),
			EvidenceHash:     hashResolutionEvidence(record.Evidence),
			OccurredAtMillis: input.OccurredAtMillis,
		},
	}
}

func buildSettlementCancellationEntries(cancelled []cancelledOrder) []rollup.JournalAppend {
	entries := make([]rollup.JournalAppend, 0, len(cancelled))
	for _, order := range cancelled {
		entries = append(entries, rollup.JournalAppend{
			EntryType:        rollup.EntryTypeOrderCancelled,
			SourceType:       rollup.SourceTypeSettlementOrder,
			SourceRef:        order.OrderID,
			OccurredAtMillis: order.UpdatedAtMillis,
			Payload: rollup.OrderCancelledPayload{
				OrderID:           order.OrderID,
				AccountID:         order.UserID,
				MarketID:          order.MarketID,
				Outcome:           order.Outcome,
				Side:              order.Side,
				ReserveAsset:      settlementReserveAsset(order),
				Price:             order.Price,
				RemainingQuantity: order.RemainingQuantity,
				CancelReason:      order.CancelReason,
			},
		})
	}
	return entries
}

func buildSettlementPayoutEntry(event sharedkafka.SettlementCompletedEvent) rollup.JournalAppend {
	return rollup.JournalAppend{
		EntryType:        rollup.EntryTypeSettlementPayout,
		SourceType:       rollup.SourceTypeSettlementPayout,
		SourceRef:        strings.TrimSpace(event.EventID),
		OccurredAtMillis: event.OccurredAtMillis,
		Payload: rollup.SettlementPayoutPayload{
			EventID:          event.EventID,
			MarketID:         event.MarketID,
			AccountID:        event.UserID,
			WinningOutcome:   event.WinningOutcome,
			PositionAsset:    event.PositionAsset,
			SettledQuantity:  event.SettledQuantity,
			PayoutAsset:      event.PayoutAsset,
			PayoutAmount:     event.PayoutAmount,
			OccurredAtMillis: event.OccurredAtMillis,
		},
	}
}

func hashResolutionEvidence(raw json.RawMessage) string {
	normalized := normalizeResolutionEvidence(raw)
	sum := sha256.Sum256(normalized)
	return hex.EncodeToString(sum[:])
}

func settlementReserveAsset(order cancelledOrder) string {
	if strings.ToUpper(strings.TrimSpace(order.Side)) == "SELL" {
		return fmt.Sprintf("POSITION:%d:%s", order.MarketID, strings.ToUpper(strings.TrimSpace(order.Outcome)))
	}
	asset := strings.ToUpper(strings.TrimSpace(order.FreezeAsset))
	if asset != "" {
		return asset
	}
	asset = strings.ToUpper(strings.TrimSpace(order.CollateralAsset))
	if asset != "" {
		return asset
	}
	return "USDT"
}
