package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"funnyoption/internal/ledger/model"
	"funnyoption/internal/shared/assets"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type TradeProcessor struct {
	journal *Journal
}

func NewTradeProcessor(journal *Journal) *TradeProcessor {
	return &TradeProcessor{journal: journal}
}

func (p *TradeProcessor) HandleTradeMatched(ctx context.Context, msg sharedkafka.Message) error {
	_ = ctx

	var event sharedkafka.TradeMatchedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}

	notional, err := tradeNotional(event.Price, event.Quantity)
	if err != nil {
		return err
	}
	asset := strings.ToUpper(strings.TrimSpace(event.CollateralAsset))
	if asset == "" {
		asset = "USDT"
	}

	payer, receiver, err := transferParties(event)
	if err != nil {
		return err
	}
	buyer, seller, err := positionParties(event)
	if err != nil {
		return err
	}
	positionAsset := assets.PositionAsset(event.MarketID, event.Outcome)

	_, err = p.journal.Append(model.Entry{
		BizType: model.BizTypeTrade,
		RefID:   event.EventID,
		Postings: []model.Posting{
			{
				Account:   availableAccount(payer),
				Asset:     asset,
				Direction: model.DirectionDebit,
				Amount:    notional,
			},
			{
				Account:   availableAccount(receiver),
				Asset:     asset,
				Direction: model.DirectionCredit,
				Amount:    notional,
			},
			{
				Account:   positionAccount(seller, event.MarketID, event.Outcome),
				Asset:     positionAsset,
				Direction: model.DirectionDebit,
				Amount:    event.Quantity,
			},
			{
				Account:   positionAccount(buyer, event.MarketID, event.Outcome),
				Asset:     positionAsset,
				Direction: model.DirectionCredit,
				Amount:    event.Quantity,
			},
		},
	})
	return err
}

func transferParties(event sharedkafka.TradeMatchedEvent) (payer int64, receiver int64, err error) {
	switch strings.ToUpper(strings.TrimSpace(event.TakerSide)) {
	case "BUY":
		return event.TakerUserID, event.MakerUserID, nil
	case "SELL":
		return event.MakerUserID, event.TakerUserID, nil
	default:
		return 0, 0, fmt.Errorf("unsupported taker side: %s", event.TakerSide)
	}
}

func positionParties(event sharedkafka.TradeMatchedEvent) (buyer int64, seller int64, err error) {
	switch strings.ToUpper(strings.TrimSpace(event.TakerSide)) {
	case "BUY":
		return event.TakerUserID, event.MakerUserID, nil
	case "SELL":
		return event.MakerUserID, event.TakerUserID, nil
	default:
		return 0, 0, fmt.Errorf("unsupported taker side: %s", event.TakerSide)
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

func availableAccount(userID int64) string {
	return fmt.Sprintf("user:%d:available", userID)
}

func positionAccount(userID, marketID int64, outcome string) string {
	return fmt.Sprintf("user:%d:position:%d:%s", userID, marketID, strings.ToUpper(strings.TrimSpace(outcome)))
}
