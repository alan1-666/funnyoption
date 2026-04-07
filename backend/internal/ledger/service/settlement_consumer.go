package service

import (
	"context"
	"encoding/json"
	"fmt"

	"funnyoption/internal/ledger/model"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type SettlementProcessor struct {
	journal *Journal
}

func NewSettlementProcessor(journal *Journal) *SettlementProcessor {
	return &SettlementProcessor{journal: journal}
}

func (p *SettlementProcessor) HandleSettlementCompleted(ctx context.Context, msg sharedkafka.Message) error {
	_ = ctx

	var event sharedkafka.SettlementCompletedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}

	_, err := p.journal.Append(model.Entry{
		BizType: model.BizTypeSettlement,
		RefID:   event.EventID,
		Postings: []model.Posting{
			{
				Account:   positionAccount(event.UserID, event.MarketID, event.WinningOutcome),
				Asset:     event.PositionAsset,
				Direction: model.DirectionDebit,
				Amount:    event.SettledQuantity,
			},
			{
				Account:   resolvedPoolAccount(event.MarketID, event.WinningOutcome),
				Asset:     event.PositionAsset,
				Direction: model.DirectionCredit,
				Amount:    event.SettledQuantity,
			},
			{
				Account:   marketTreasuryAccount(event.MarketID),
				Asset:     event.PayoutAsset,
				Direction: model.DirectionDebit,
				Amount:    event.PayoutAmount,
			},
			{
				Account:   availableAccount(event.UserID),
				Asset:     event.PayoutAsset,
				Direction: model.DirectionCredit,
				Amount:    event.PayoutAmount,
			},
		},
	})
	return err
}

func marketTreasuryAccount(marketID int64) string {
	return fmt.Sprintf("market:%d:treasury", marketID)
}

func resolvedPoolAccount(marketID int64, outcome string) string {
	return fmt.Sprintf("market:%d:resolved:%s", marketID, outcome)
}
