package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"funnyoption/internal/shared/assets"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type Processor struct {
	store     PositionStore
	publisher sharedkafka.Publisher
	topics    sharedkafka.Topics
}

func NewProcessor(store PositionStore, publisher sharedkafka.Publisher, topics sharedkafka.Topics) *Processor {
	return &Processor{
		store:     store,
		publisher: publisher,
		topics:    topics,
	}
}

func (p *Processor) HandlePositionChanged(ctx context.Context, msg sharedkafka.Message) error {
	var event sharedkafka.PositionChangedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}
	return p.store.ApplyDelta(ctx, event.MarketID, event.UserID, strings.ToUpper(strings.TrimSpace(event.Outcome)), event.PositionAsset, event.DeltaQuantity)
}

func (p *Processor) HandleMarketEvent(ctx context.Context, msg sharedkafka.Message) error {
	var event sharedkafka.MarketEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}
	if strings.ToUpper(strings.TrimSpace(event.Status)) != "RESOLVED" {
		return nil
	}

	resolvedOutcome := strings.ToUpper(strings.TrimSpace(event.ResolvedOutcome))
	if resolvedOutcome == "" {
		return nil
	}

	freshResolve, err := p.store.ResolveMarket(ctx, ResolveMarketInput{
		MarketID:         event.MarketID,
		ResolvedOutcome:  resolvedOutcome,
		OccurredAtMillis: event.OccurredAtMillis,
	})
	if err != nil {
		return err
	}
	if !freshResolve {
		return nil
	}
	cancelledOrders, err := p.store.CancelActiveOrders(ctx, event.MarketID, "MARKET_RESOLVED")
	if err != nil {
		return err
	}
	for _, order := range cancelledOrders {
		orderEvent := sharedkafka.OrderEvent{
			EventID:           sharedkafka.NewID("evt_order"),
			CommandID:         order.CommandID,
			TraceID:           event.TraceID,
			OrderID:           order.OrderID,
			ClientOrderID:     order.ClientOrderID,
			FreezeID:          order.FreezeID,
			FreezeAsset:       order.FreezeAsset,
			FreezeAmount:      order.FreezeAmount,
			CollateralAsset:   order.CollateralAsset,
			UserID:            order.UserID,
			MarketID:          order.MarketID,
			Outcome:           order.Outcome,
			Side:              order.Side,
			Type:              order.OrderType,
			TimeInForce:       order.TimeInForce,
			Status:            order.Status,
			CancelReason:      order.CancelReason,
			Price:             order.Price,
			Quantity:          order.Quantity,
			FilledQuantity:    order.FilledQuantity,
			RemainingQuantity: order.RemainingQuantity,
			OccurredAtMillis:  time.Now().UnixMilli(),
		}
		if err := p.publisher.PublishJSON(ctx, p.topics.OrderEvent, marketBookKey(order.MarketID, order.Outcome), orderEvent); err != nil {
			return err
		}
	}
	positions, err := p.store.WinningPositions(ctx, event.MarketID, resolvedOutcome)
	if err != nil {
		return err
	}
	for _, position := range positions {
		payoutAmount, err := assets.WinningPayoutAmount(position.Quantity)
		if err != nil {
			return err
		}
		settlement := sharedkafka.SettlementCompletedEvent{
			EventID:          settlementEventID(position.MarketID, position.UserID, position.Outcome),
			MarketID:         position.MarketID,
			UserID:           position.UserID,
			WinningOutcome:   position.Outcome,
			PositionAsset:    assets.PositionAsset(position.MarketID, position.Outcome),
			SettledQuantity:  position.Quantity,
			PayoutAsset:      assets.DefaultCollateralAsset,
			PayoutAmount:     payoutAmount,
			OccurredAtMillis: time.Now().UnixMilli(),
		}
		if err := p.publisher.PublishJSON(ctx, p.topics.SettlementDone, settlementKey(position.MarketID, position.UserID), settlement); err != nil {
			return err
		}
		if err := p.store.MarkSettled(ctx, settlement); err != nil {
			return err
		}
	}
	return nil
}

func settlementKey(marketID, userID int64) string {
	return strings.Join([]string{strconv.FormatInt(marketID, 10), strconv.FormatInt(userID, 10)}, ":")
}

func marketBookKey(marketID int64, outcome string) string {
	return fmt.Sprintf("%d:%s", marketID, strings.ToUpper(strings.TrimSpace(outcome)))
}

func settlementEventID(marketID, userID int64, outcome string) string {
	return strings.Join([]string{"evt_settlement", strconv.FormatInt(marketID, 10), strconv.FormatInt(userID, 10), strings.ToUpper(strings.TrimSpace(outcome))}, "_")
}
