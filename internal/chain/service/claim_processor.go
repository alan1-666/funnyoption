package service

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	chainmodel "funnyoption/internal/chain/model"
	"funnyoption/internal/shared/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const vaultClaimABI = `[{"inputs":[{"internalType":"bytes32","name":"claimId","type":"bytes32"},{"internalType":"address","name":"wallet","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"address","name":"recipient","type":"address"}],"name":"processClaim","outputs":[],"stateMutability":"nonpayable","type":"function"}]`

type claimStore interface {
	ListPendingClaims(ctx context.Context, limit int) ([]chainmodel.ClaimTask, error)
	MarkClaimSubmitted(ctx context.Context, id int64, txHash string) error
	MarkClaimFailed(ctx context.Context, id int64, errMsg string) error
}

type txSender interface {
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error)
	SendTransaction(ctx context.Context, tx *types.Transaction) error
	ChainID(ctx context.Context) (*big.Int, error)
}

type ClaimProcessor struct {
	logger       *slog.Logger
	cfg          config.ServiceConfig
	store        claimStore
	sender       txSender
	privateKey   *ecdsa.PrivateKey
	fromAddress  common.Address
	vaultAddress common.Address
	contractABI  abi.ABI
}

func NewClaimProcessor(logger *slog.Logger, cfg config.ServiceConfig, store claimStore, sender txSender) (*ClaimProcessor, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if strings.TrimSpace(cfg.ChainOperatorPrivateKey) == "" {
		return nil, fmt.Errorf("chain operator private key is required")
	}
	if strings.TrimSpace(cfg.VaultAddress) == "" {
		return nil, fmt.Errorf("vault address is required")
	}
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(strings.TrimSpace(cfg.ChainOperatorPrivateKey), "0x"))
	if err != nil {
		return nil, err
	}
	contractABI, err := abi.JSON(strings.NewReader(vaultClaimABI))
	if err != nil {
		return nil, err
	}
	vaultAddress, err := validateClaimAddress("vault_address", cfg.VaultAddress)
	if err != nil {
		return nil, err
	}

	return &ClaimProcessor{
		logger:       logger,
		cfg:          cfg,
		store:        store,
		sender:       sender,
		privateKey:   privateKey,
		fromAddress:  crypto.PubkeyToAddress(privateKey.PublicKey),
		vaultAddress: vaultAddress,
		contractABI:  contractABI,
	}, nil
}

func (p *ClaimProcessor) Start(ctx context.Context) {
	ticker := time.NewTicker(p.cfg.ClaimPollInterval)
	defer ticker.Stop()

	if err := p.pollOnce(ctx); err != nil {
		p.logger.Error("initial claim poll failed", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.pollOnce(ctx); err != nil {
				p.logger.Error("claim poll failed", "err", err)
			}
		}
	}
}

func (p *ClaimProcessor) pollOnce(ctx context.Context) error {
	tasks, err := p.store.ListPendingClaims(ctx, 20)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		txHash, err := p.submitClaim(ctx, task)
		if err != nil {
			if markErr := p.store.MarkClaimFailed(ctx, task.ID, err.Error()); markErr != nil {
				p.logger.Error("mark claim failed status error", "id", task.ID, "err", markErr)
			}
			continue
		}
		if err := p.store.MarkClaimSubmitted(ctx, task.ID, txHash); err != nil {
			return err
		}
	}
	return nil
}

func (p *ClaimProcessor) submitClaim(ctx context.Context, task chainmodel.ClaimTask) (string, error) {
	if strings.TrimSpace(task.RefID) == "" {
		return "", fmt.Errorf("ref_id is required")
	}
	wallet, err := validateClaimAddress("wallet_address", task.WalletAddress)
	if err != nil {
		return "", err
	}
	recipient, err := validateClaimAddress("recipient_address", task.RecipientAddress)
	if err != nil {
		return "", err
	}
	if task.PayoutAmount <= 0 {
		return "", fmt.Errorf("payout_amount must be positive")
	}

	claimID := crypto.Keccak256Hash([]byte(task.RefID))
	data, err := p.contractABI.Pack("processClaim", claimID, wallet, big.NewInt(task.PayoutAmount), recipient)
	if err != nil {
		return "", err
	}

	chainID, err := p.sender.ChainID(ctx)
	if err != nil {
		return "", err
	}
	nonce, err := p.sender.PendingNonceAt(ctx, p.fromAddress)
	if err != nil {
		return "", err
	}
	gasPrice, err := p.sender.SuggestGasPrice(ctx)
	if err != nil {
		return "", err
	}

	gasLimit := p.cfg.ChainGasLimit
	if gasLimit == 0 {
		gasLimit = 250000
	}
	estimatedGas, err := p.sender.EstimateGas(ctx, ethereum.CallMsg{
		From:     p.fromAddress,
		To:       &p.vaultAddress,
		GasPrice: gasPrice,
		Data:     data,
	})
	if err == nil && estimatedGas > 0 {
		gasLimit = estimatedGas + estimatedGas/5
	}

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &p.vaultAddress,
		Value:    big.NewInt(0),
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), p.privateKey)
	if err != nil {
		return "", err
	}
	if err := p.sender.SendTransaction(ctx, signedTx); err != nil {
		return "", err
	}
	return normalizeChainTxHash(signedTx.Hash().Hex()), nil
}

func validateClaimAddress(field, value string) (common.Address, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return common.Address{}, fmt.Errorf("%s is required", field)
	}
	if !common.IsHexAddress(trimmed) {
		return common.Address{}, fmt.Errorf("%s must be a valid EVM address", field)
	}

	address := common.HexToAddress(trimmed)
	if address == (common.Address{}) {
		return common.Address{}, fmt.Errorf("%s must not be zero address", field)
	}
	return address, nil
}
