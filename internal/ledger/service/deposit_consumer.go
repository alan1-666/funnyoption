package service

import (
	"context"
	"encoding/json"
	"fmt"

	"funnyoption/internal/ledger/model"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type DepositProcessor struct {
	journal *Journal
}

func NewDepositProcessor(journal *Journal) *DepositProcessor {
	return &DepositProcessor{journal: journal}
}

func (p *DepositProcessor) HandleChainDeposit(ctx context.Context, msg sharedkafka.Message) error {
	_ = ctx

	var event sharedkafka.ChainDepositCreditedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return err
	}

	_, err := p.journal.Append(model.Entry{
		BizType: model.BizTypeDeposit,
		RefID:   event.DepositID,
		Postings: []model.Posting{
			{
				Account:   vaultAccount(event.ChainName, event.NetworkName, event.VaultAddress),
				Asset:     event.Asset,
				Direction: model.DirectionDebit,
				Amount:    event.Amount,
			},
			{
				Account:   availableAccount(event.UserID),
				Asset:     event.Asset,
				Direction: model.DirectionCredit,
				Amount:    event.Amount,
			},
		},
	})
	return err
}

func vaultAccount(chainName, networkName, vaultAddress string) string {
	return fmt.Sprintf("vault:%s:%s:%s", chainName, networkName, vaultAddress)
}
