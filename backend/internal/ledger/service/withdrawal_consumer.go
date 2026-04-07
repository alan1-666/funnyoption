package service

import (
	"context"
	"encoding/json"

	"funnyoption/internal/ledger/model"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type WithdrawalProcessor struct {
	journal *Journal
}

func NewWithdrawalProcessor(journal *Journal) *WithdrawalProcessor {
	return &WithdrawalProcessor{journal: journal}
}

func (p *WithdrawalProcessor) HandleChainWithdrawal(ctx context.Context, msg sharedkafka.Message) error {
	_ = ctx

	var event sharedkafka.ChainWithdrawalQueuedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}

	_, err := p.journal.Append(model.Entry{
		BizType: model.BizTypeWithdraw,
		RefID:   event.WithdrawalID,
		Postings: []model.Posting{
			{
				Account:   availableAccount(event.UserID),
				Asset:     event.Asset,
				Direction: model.DirectionDebit,
				Amount:    event.Amount,
			},
			{
				Account:   vaultAccount(event.ChainName, event.NetworkName, event.VaultAddress),
				Asset:     event.Asset,
				Direction: model.DirectionCredit,
				Amount:    event.Amount,
			},
		},
	})
	return err
}
