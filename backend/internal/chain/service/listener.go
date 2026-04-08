package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	chainmodel "funnyoption/internal/chain/model"
	"funnyoption/internal/shared/assets"
	"funnyoption/internal/shared/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const maxVaultScanSpan uint64 = 500

var depositEventTopic = crypto.Keccak256Hash([]byte("Deposited(address,uint256)"))
var withdrawalEventTopic = crypto.Keccak256Hash([]byte("WithdrawalQueued(bytes32,address,uint256,address)"))
var claimProcessedEventTopic = crypto.Keccak256Hash([]byte("ClaimProcessed(bytes32,address,uint256,address)"))

type logReader interface {
	BlockNumber(ctx context.Context) (uint64, error)
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
	Close()
}

type DepositListener struct {
	logger        *slog.Logger
	cfg           config.ServiceConfig
	store         DepositStore
	processor     *Processor
	reader        logReader
	vaultAddress  common.Address
	confirmations uint64
	startBlock    uint64
	nextBlock     uint64
	cursorReady   bool
	pollInterval  time.Duration
}

func NewDepositListener(ctx context.Context, logger *slog.Logger, cfg config.ServiceConfig, store DepositStore, processor *Processor) (*DepositListener, error) {
	client, err := newRPCPool(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return NewDepositListenerWithReader(logger, cfg, store, processor, client)
}

func NewDepositListenerWithReader(logger *slog.Logger, cfg config.ServiceConfig, store DepositStore, processor *Processor, reader logReader) (*DepositListener, error) {
	if strings.TrimSpace(cfg.VaultAddress) == "" {
		return nil, fmt.Errorf("vault address is required")
	}
	if store == nil || processor == nil || reader == nil {
		return nil, fmt.Errorf("deposit listener dependencies are not ready")
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &DepositListener{
		logger:        logger,
		cfg:           cfg,
		store:         store,
		processor:     processor,
		reader:        reader,
		vaultAddress:  common.HexToAddress(cfg.VaultAddress),
		confirmations: uint64(maxInt64(cfg.Confirmations, 0)),
		startBlock:    uint64(maxInt64(cfg.StartBlock, 0)),
		nextBlock:     uint64(maxInt64(cfg.StartBlock, 0)),
		pollInterval:  cfg.PollInterval,
	}, nil
}

func (l *DepositListener) Close() {
	if l.reader != nil {
		l.reader.Close()
	}
}

func (l *DepositListener) Start(ctx context.Context) {
	if err := l.pollOnce(ctx); err != nil {
		l.logger.Error("initial deposit poll failed", "err", err)
	}

	ticker := time.NewTicker(l.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := l.pollOnce(ctx); err != nil {
				l.logger.Error("deposit poll failed", "err", err)
			}
		}
	}
}

func (l *DepositListener) pollOnce(ctx context.Context) error {
	head, err := l.reader.BlockNumber(ctx)
	if err != nil {
		return err
	}

	safeHead, ok := confirmedHead(head, l.confirmations)
	if !ok {
		return nil
	}
	if err := l.ensureScanCursor(ctx, safeHead); err != nil {
		return err
	}
	if l.nextBlock > safeHead {
		return nil
	}

	toBlock := minUint64(l.nextBlock+maxVaultScanSpan-1, safeHead)
	logs, err := l.reader.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(l.nextBlock),
		ToBlock:   new(big.Int).SetUint64(toBlock),
		Addresses: []common.Address{l.vaultAddress},
		Topics:    [][]common.Hash{{depositEventTopic, withdrawalEventTopic, claimProcessedEventTopic}},
	})
	if err != nil {
		if isPrunedHistoryError(err) {
			skippedFromBlock := l.nextBlock
			l.nextBlock = safeHead + 1
			if saveErr := l.persistScanCursor(ctx); saveErr != nil {
				return saveErr
			}
			l.logger.Warn(
				"skip pruned vault history",
				"from_block", skippedFromBlock,
				"to_block", safeHead,
				"next_block", l.nextBlock,
				"vault_address", normalizeVaultAddress(l.vaultAddress.Hex()),
				"chain_name", normalizeChainName(l.cfg.ChainName),
				"network_name", normalizeNetworkName(l.cfg.NetworkName),
				"err", err,
			)
			return nil
		}
		return err
	}

	for _, logEntry := range logs {
		if err := l.handleVaultLog(ctx, logEntry); err != nil {
			l.logger.Error(
				"handle vault log failed",
				"tx_hash", logEntry.TxHash.Hex(),
				"log_index", logEntry.Index,
				"err", err,
			)
		}
	}

	l.nextBlock = toBlock + 1
	if err := l.persistScanCursor(ctx); err != nil {
		return err
	}
	return nil
}

func (l *DepositListener) ensureScanCursor(ctx context.Context, safeHead uint64) error {
	if l.cursorReady {
		return nil
	}

	nextBlock := l.startBlock
	checkpoint, ok, err := l.store.LoadVaultScanCursor(ctx, l.cfg.ChainName, l.cfg.NetworkName, l.vaultAddress.Hex())
	if err != nil {
		return err
	}
	if ok && checkpoint > safeHead+1 {
		l.logger.Warn(
			"vault scan cursor ahead of safe head; resetting from configured start block",
			"checkpoint_next_block", checkpoint,
			"safe_head", safeHead,
			"configured_start_block", l.startBlock,
			"vault_address", normalizeVaultAddress(l.vaultAddress.Hex()),
			"chain_name", normalizeChainName(l.cfg.ChainName),
			"network_name", normalizeNetworkName(l.cfg.NetworkName),
		)
	} else if ok && checkpoint > nextBlock {
		nextBlock = checkpoint
	}
	if nextBlock == 0 {
		nextBlock = safeHead + 1
	}

	l.nextBlock = nextBlock
	l.cursorReady = true
	if err := l.persistScanCursor(ctx); err != nil {
		l.cursorReady = false
		return err
	}

	l.logger.Info(
		"vault scan cursor initialized",
		"configured_start_block", l.startBlock,
		"checkpoint_next_block", checkpoint,
		"next_block", l.nextBlock,
		"safe_head", safeHead,
		"vault_address", normalizeVaultAddress(l.vaultAddress.Hex()),
		"chain_name", normalizeChainName(l.cfg.ChainName),
		"network_name", normalizeNetworkName(l.cfg.NetworkName),
	)
	return nil
}

func (l *DepositListener) persistScanCursor(ctx context.Context) error {
	return l.store.SaveVaultScanCursor(ctx, l.cfg.ChainName, l.cfg.NetworkName, l.vaultAddress.Hex(), l.nextBlock)
}

func (l *DepositListener) handleVaultLog(ctx context.Context, logEntry types.Log) error {
	switch logEntry.Topics[0] {
	case depositEventTopic:
		return l.handleDepositLog(ctx, logEntry)
	case withdrawalEventTopic:
		return l.handleWithdrawalLog(ctx, logEntry)
	case claimProcessedEventTopic:
		return l.handleClaimProcessedLog(ctx, logEntry)
	default:
		return fmt.Errorf("unsupported vault topic: %s", logEntry.Topics[0].Hex())
	}
}

func (l *DepositListener) handleDepositLog(ctx context.Context, logEntry types.Log) error {
	if len(logEntry.Topics) < 2 {
		return fmt.Errorf("deposit log missing wallet topic")
	}

	amount := new(big.Int).SetBytes(logEntry.Data)
	if amount.Sign() <= 0 {
		return fmt.Errorf("deposit amount must be positive")
	}
	if !amount.IsInt64() {
		return fmt.Errorf("deposit amount exceeds int64")
	}
	// Native deposits (and any oracle-derived credits) use arbitrary 6-decimal collateral amounts;
	// strict ChainToAccountingAmount rejects values not divisible by 10^(chain−display), so we floor.
	accountingAmount, err := assets.ChainToAccountingAmountFloor(amount.Int64(), l.cfg.CollateralDecimals, l.cfg.CollateralDisplayDigits)
	if err != nil {
		return err
	}
	if accountingAmount <= 0 {
		return fmt.Errorf("deposit amount resolves to non-positive accounting amount")
	}

	walletAddress := strings.ToLower(common.BytesToAddress(logEntry.Topics[1].Bytes()).Hex())
	userID, err := l.store.LookupActiveUserByWallet(ctx, walletAddress)
	if err != nil {
		if errors.Is(err, ErrWalletSessionNotFound) {
			l.logger.Warn("skip deposit without durable wallet binding", "wallet_address", walletAddress, "tx_hash", logEntry.TxHash.Hex())
			return nil
		}
		return err
	}

	deposit := chainmodel.Deposit{
		DepositID:     buildChainEventID("dep", logEntry.TxHash.Hex(), logEntry.Index),
		UserID:        userID,
		WalletAddress: walletAddress,
		VaultAddress:  strings.ToLower(l.vaultAddress.Hex()),
		Asset:         assets.NormalizeAsset(l.cfg.CollateralSymbol),
		Amount:        accountingAmount,
		ChainName:     l.cfg.ChainName,
		NetworkName:   l.cfg.NetworkName,
		TxHash:        normalizeChainTxHash(logEntry.TxHash.Hex()),
		LogIndex:      int64(logEntry.Index),
		BlockNumber:   int64(logEntry.BlockNumber),
		Status:        "CONFIRMED",
	}
	return l.processor.ApplyConfirmedDeposit(ctx, deposit)
}

func (l *DepositListener) handleWithdrawalLog(ctx context.Context, logEntry types.Log) error {
	if len(logEntry.Topics) < 3 {
		return fmt.Errorf("withdrawal log missing indexed topics")
	}
	if len(logEntry.Data) < 64 {
		return fmt.Errorf("withdrawal log payload is too short")
	}

	withdrawalID := normalizeChainTxHash(logEntry.Topics[1].Hex())
	walletAddress := strings.ToLower(common.BytesToAddress(logEntry.Topics[2].Bytes()).Hex())
	amount := new(big.Int).SetBytes(logEntry.Data[:32])
	if amount.Sign() <= 0 {
		return fmt.Errorf("withdrawal amount must be positive")
	}
	if !amount.IsInt64() {
		return fmt.Errorf("withdrawal amount exceeds int64")
	}
	accountingAmount, err := assets.ChainToAccountingAmount(amount.Int64(), l.cfg.CollateralDecimals, l.cfg.CollateralDisplayDigits)
	if err != nil {
		return err
	}
	if accountingAmount <= 0 {
		return fmt.Errorf("withdrawal amount resolves to non-positive accounting amount")
	}
	recipientAddress := strings.ToLower(common.BytesToAddress(logEntry.Data[32:64]).Hex())

	userID, err := l.store.LookupActiveUserByWallet(ctx, walletAddress)
	if err != nil {
		if errors.Is(err, ErrWalletSessionNotFound) {
			l.logger.Warn("skip withdrawal without durable wallet binding", "wallet_address", walletAddress, "tx_hash", logEntry.TxHash.Hex())
			return nil
		}
		return err
	}

	withdrawal := chainmodel.Withdrawal{
		WithdrawalID:     withdrawalID,
		UserID:           userID,
		WalletAddress:    walletAddress,
		RecipientAddress: recipientAddress,
		VaultAddress:     strings.ToLower(l.vaultAddress.Hex()),
		Asset:            assets.NormalizeAsset(l.cfg.CollateralSymbol),
		Amount:           accountingAmount,
		ChainName:        l.cfg.ChainName,
		NetworkName:      l.cfg.NetworkName,
		TxHash:           normalizeChainTxHash(logEntry.TxHash.Hex()),
		LogIndex:         int64(logEntry.Index),
		BlockNumber:      int64(logEntry.BlockNumber),
		Status:           "QUEUED",
	}
	return l.processor.ApplyConfirmedWithdrawal(ctx, withdrawal)
}

func (l *DepositListener) handleClaimProcessedLog(ctx context.Context, logEntry types.Log) error {
	if len(logEntry.Topics) < 3 {
		return fmt.Errorf("claim processed log missing indexed topics")
	}
	if len(logEntry.Data) < 64 {
		return fmt.Errorf("claim processed log payload is too short")
	}
	claimID := strings.ToLower(common.BytesToHash(logEntry.Topics[1].Bytes()).Hex())
	txHash := normalizeChainTxHash(logEntry.TxHash.Hex())
	if err := l.store.MarkClaimConfirmedByTxHash(ctx, txHash); err != nil {
		return err
	}
	return l.store.MarkAcceptedEscapeClaimConfirmed(ctx, claimID, txHash)
}

func confirmedHead(head uint64, confirmations uint64) (uint64, bool) {
	if confirmations == 0 {
		return head, true
	}
	if head < confirmations {
		return 0, false
	}
	return head - confirmations, true
}

func minUint64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func maxInt64(value int64, floor int64) int64 {
	if value < floor {
		return floor
	}
	return value
}

func isPrunedHistoryError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "history has been pruned")
}
