package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	accountclient "funnyoption/internal/account/client"
	chainmodel "funnyoption/internal/chain/model"
	"funnyoption/internal/rollup"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type Processor struct {
	logger    *slog.Logger
	store     DepositStore
	account   accountclient.AccountClient
	publisher sharedkafka.Publisher
	topics    sharedkafka.Topics
	rollup    *rollup.Store
}

func NewProcessor(logger *slog.Logger, store DepositStore, account accountclient.AccountClient, publisher sharedkafka.Publisher, topics sharedkafka.Topics) *Processor {
	return &Processor{
		logger:    logger,
		store:     store,
		account:   account,
		publisher: publisher,
		topics:    topics,
	}
}

func (p *Processor) WithRollup(store *rollup.Store) *Processor {
	p.rollup = store
	return p
}

func (p *Processor) ApplyConfirmedDeposit(ctx context.Context, deposit chainmodel.Deposit) error {
	if deposit.UserID <= 0 || deposit.Amount <= 0 {
		return fmt.Errorf("invalid deposit payload")
	}
	deposit.Asset = normalizeChainAsset(deposit.Asset)
	deposit.ChainName = normalizeChainName(deposit.ChainName)
	deposit.NetworkName = normalizeNetworkName(deposit.NetworkName)
	deposit.WalletAddress = strings.ToLower(strings.TrimSpace(deposit.WalletAddress))
	deposit.VaultAddress = strings.ToLower(strings.TrimSpace(deposit.VaultAddress))
	deposit.TxHash = normalizeChainTxHash(deposit.TxHash)
	if strings.TrimSpace(deposit.DepositID) == "" {
		deposit.DepositID = sharedkafka.NewID("dep")
	}
	if strings.TrimSpace(deposit.Status) == "" {
		deposit.Status = "CONFIRMED"
	}

	stored, err := p.store.UpsertDeposit(ctx, deposit)
	if err != nil {
		return err
	}
	if stored.CreditedAt > 0 {
		if p.logger != nil {
			p.logger.Info("deposit already credited", "deposit_id", stored.DepositID, "tx_hash", stored.TxHash, "log_index", stored.LogIndex)
		}
		return nil
	}

	result, err := p.account.CreditBalance(ctx, stored.UserID, stored.Asset, stored.Amount, "DEPOSIT", stored.DepositID)
	if err != nil {
		return err
	}

	event := sharedkafka.ChainDepositCreditedEvent{
		EventID:          "evt_" + stored.DepositID,
		DepositID:        stored.DepositID,
		UserID:           stored.UserID,
		WalletAddress:    stored.WalletAddress,
		VaultAddress:     stored.VaultAddress,
		Asset:            stored.Asset,
		Amount:           stored.Amount,
		ChainName:        stored.ChainName,
		NetworkName:      stored.NetworkName,
		TxHash:           stored.TxHash,
		LogIndex:         stored.LogIndex,
		BlockNumber:      stored.BlockNumber,
		OccurredAtMillis: time.Now().UnixMilli(),
	}
	if result.Applied {
		if err := p.publisher.PublishJSON(ctx, p.topics.ChainDeposit, stored.DepositID, event); err != nil {
			return err
		}
	}
	if p.rollup != nil {
		if err := p.rollup.AppendEntries(ctx, []rollup.JournalAppend{{
			EntryType:        rollup.EntryTypeDepositCredited,
			SourceType:       rollup.SourceTypeChainDeposit,
			SourceRef:        stored.DepositID,
			OccurredAtMillis: event.OccurredAtMillis,
			Payload: rollup.DepositCreditedPayload{
				DepositID:        stored.DepositID,
				AccountID:        stored.UserID,
				WalletAddress:    stored.WalletAddress,
				VaultAddress:     stored.VaultAddress,
				Asset:            stored.Asset,
				Amount:           stored.Amount,
				ChainName:        stored.ChainName,
				NetworkName:      stored.NetworkName,
				TxHash:           stored.TxHash,
				LogIndex:         stored.LogIndex,
				BlockNumber:      stored.BlockNumber,
				OccurredAtMillis: event.OccurredAtMillis,
			},
		}}); err != nil {
			return err
		}
	}
	if err := p.store.MarkDepositCredited(ctx, stored.DepositID, time.Now().Unix()); err != nil {
		return err
	}
	return nil
}

func (p *Processor) ApplyConfirmedWithdrawal(ctx context.Context, withdrawal chainmodel.Withdrawal) error {
	if withdrawal.UserID <= 0 || withdrawal.Amount <= 0 {
		return fmt.Errorf("invalid withdrawal payload")
	}
	withdrawal.Asset = normalizeChainAsset(withdrawal.Asset)
	withdrawal.ChainName = normalizeChainName(withdrawal.ChainName)
	withdrawal.NetworkName = normalizeNetworkName(withdrawal.NetworkName)
	withdrawal.WalletAddress = strings.ToLower(strings.TrimSpace(withdrawal.WalletAddress))
	withdrawal.RecipientAddress = strings.ToLower(strings.TrimSpace(withdrawal.RecipientAddress))
	withdrawal.VaultAddress = strings.ToLower(strings.TrimSpace(withdrawal.VaultAddress))
	withdrawal.TxHash = normalizeChainTxHash(withdrawal.TxHash)
	if strings.TrimSpace(withdrawal.WithdrawalID) == "" {
		withdrawal.WithdrawalID = sharedkafka.NewID("wdq")
	}
	if strings.TrimSpace(withdrawal.Status) == "" {
		withdrawal.Status = "QUEUED"
	}

	stored, err := p.store.UpsertWithdrawal(ctx, withdrawal)
	if err != nil {
		return err
	}
	if stored.DebitedAt > 0 {
		if p.logger != nil {
			p.logger.Info("withdrawal already debited", "withdrawal_id", stored.WithdrawalID, "tx_hash", stored.TxHash, "log_index", stored.LogIndex)
		}
		return nil
	}

	result, err := p.account.DebitBalance(ctx, stored.UserID, stored.Asset, stored.Amount, "WITHDRAWAL", stored.WithdrawalID)
	if err != nil {
		return err
	}

	event := sharedkafka.ChainWithdrawalQueuedEvent{
		EventID:          "evt_" + stored.WithdrawalID,
		WithdrawalID:     stored.WithdrawalID,
		UserID:           stored.UserID,
		WalletAddress:    stored.WalletAddress,
		RecipientAddress: stored.RecipientAddress,
		VaultAddress:     stored.VaultAddress,
		Asset:            stored.Asset,
		Amount:           stored.Amount,
		ChainName:        stored.ChainName,
		NetworkName:      stored.NetworkName,
		TxHash:           stored.TxHash,
		LogIndex:         stored.LogIndex,
		BlockNumber:      stored.BlockNumber,
		OccurredAtMillis: time.Now().UnixMilli(),
	}
	if result.Applied {
		if err := p.publisher.PublishJSON(ctx, p.topics.ChainWithdraw, stored.WithdrawalID, event); err != nil {
			return err
		}
	}
	if p.rollup != nil {
		if err := p.rollup.AppendEntries(ctx, []rollup.JournalAppend{{
			EntryType:        rollup.EntryTypeWithdrawalRequested,
			SourceType:       rollup.SourceTypeChainWithdraw,
			SourceRef:        stored.WithdrawalID,
			OccurredAtMillis: event.OccurredAtMillis,
			Payload: rollup.WithdrawalRequestedPayload{
				WithdrawalID:     stored.WithdrawalID,
				AccountID:        stored.UserID,
				WalletAddress:    stored.WalletAddress,
				RecipientAddress: stored.RecipientAddress,
				VaultAddress:     stored.VaultAddress,
				Asset:            stored.Asset,
				Amount:           stored.Amount,
				Lane:             "SLOW",
				ChainName:        stored.ChainName,
				NetworkName:      stored.NetworkName,
				TxHash:           stored.TxHash,
				LogIndex:         stored.LogIndex,
				BlockNumber:      stored.BlockNumber,
				OccurredAtMillis: event.OccurredAtMillis,
			},
		}}); err != nil {
			return err
		}
	}
	if err := p.store.MarkWithdrawalDebited(ctx, stored.WithdrawalID, time.Now().Unix()); err != nil {
		return err
	}
	return nil
}

func normalizeChainAsset(asset string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(asset))
	if trimmed == "" {
		return "USDT"
	}
	return trimmed
}

func normalizeChainName(chainName string) string {
	trimmed := strings.ToLower(strings.TrimSpace(chainName))
	if trimmed == "" {
		return "bsc"
	}
	return trimmed
}

func normalizeNetworkName(networkName string) string {
	trimmed := strings.ToLower(strings.TrimSpace(networkName))
	if trimmed == "" {
		return "testnet"
	}
	return trimmed
}

func normalizeChainTxHash(txHash string) string {
	trimmed := strings.ToLower(strings.TrimSpace(txHash))
	return strings.TrimPrefix(trimmed, "0x")
}

func buildChainEventID(prefix, txHash string, logIndex uint) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", prefix, normalizeChainTxHash(txHash), logIndex)))
	return prefix + "_" + hex.EncodeToString(sum[:16])
}
