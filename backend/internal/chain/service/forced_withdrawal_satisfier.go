package service

import (
	"context"
	"crypto/ecdsa"
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
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const forcedWithdrawalSatisfyABI = `[{"inputs":[{"internalType":"uint64","name":"requestId","type":"uint64"},{"internalType":"bytes32","name":"claimId","type":"bytes32"}],"name":"satisfyForcedWithdrawal","outputs":[],"stateMutability":"nonpayable","type":"function"}]`

const (
	ForcedWithdrawalSatisfyActionNoop       = "NOOP"
	ForcedWithdrawalSatisfyActionReady      = "READY"
	ForcedWithdrawalSatisfyActionSubmitted  = "SUBMITTED"
	ForcedWithdrawalSatisfyActionPending    = "PENDING"
	ForcedWithdrawalSatisfyActionSatisfied  = "SATISFIED"
	ForcedWithdrawalSatisfyActionAmbiguous  = "AMBIGUOUS"
	ForcedWithdrawalSatisfyActionFailed     = "FAILED"
	ForcedWithdrawalSatisfyActionRollupStop = "ROLLUP_FROZEN"
)

type forcedWithdrawalSatisfierStore interface {
	ListPendingRollupForcedWithdrawalRequests(ctx context.Context, limit int) ([]chainmodel.RollupForcedWithdrawalRequest, error)
	ListForcedWithdrawalClaimMatches(ctx context.Context, requestID int64, limit int) ([]chainmodel.ForcedWithdrawalClaimMatch, error)
	UpdateRollupForcedWithdrawalMatch(ctx context.Context, requestID int64, withdrawalID, claimID, status, errMsg string) error
	MarkRollupForcedWithdrawalSatisfactionSubmitted(ctx context.Context, requestID int64, txHash string) error
	MarkRollupForcedWithdrawalSatisfactionFailed(ctx context.Context, requestID int64, errMsg string) error
	UpsertRollupForcedWithdrawalRequest(ctx context.Context, request chainmodel.RollupForcedWithdrawalRequest) error
}

type ForcedWithdrawalSatisfactionProgress struct {
	Action    string `json:"action"`
	RequestID int64  `json:"request_id,omitempty"`
	ClaimID   string `json:"claim_id,omitempty"`
	TxHash    string `json:"tx_hash,omitempty"`
	Note      string `json:"note,omitempty"`
}

type ForcedWithdrawalSatisfier struct {
	logger       *slog.Logger
	cfg          config.ServiceConfig
	store        forcedWithdrawalSatisfierStore
	sender       rollupTxSender
	privateKey   *ecdsa.PrivateKey
	fromAddress  common.Address
	rollupCore   common.Address
	contractABI  abi.ABI
	pollInterval time.Duration
}

func NewForcedWithdrawalSatisfier(
	logger *slog.Logger,
	cfg config.ServiceConfig,
	store forcedWithdrawalSatisfierStore,
	sender rollupTxSender,
) (*ForcedWithdrawalSatisfier, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if store == nil {
		return nil, fmt.Errorf("forced withdrawal satisfier store is required")
	}
	if sender == nil {
		return nil, fmt.Errorf("forced withdrawal satisfier sender is required")
	}
	if strings.TrimSpace(cfg.ChainOperatorPrivateKey) == "" {
		return nil, fmt.Errorf("chain operator private key is required")
	}
	if strings.TrimSpace(cfg.RollupCoreAddress) == "" {
		return nil, fmt.Errorf("rollup core address is required")
	}
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(strings.TrimSpace(cfg.ChainOperatorPrivateKey), "0x"))
	if err != nil {
		return nil, err
	}
	contractABI, err := abi.JSON(strings.NewReader(forcedWithdrawalSatisfyABI))
	if err != nil {
		return nil, err
	}
	rollupCore, err := validateClaimAddress("rollup_core_address", cfg.RollupCoreAddress)
	if err != nil {
		return nil, err
	}
	pollInterval := cfg.RollupPollInterval
	if pollInterval <= 0 {
		pollInterval = 10 * time.Second
	}

	return &ForcedWithdrawalSatisfier{
		logger:       logger,
		cfg:          cfg,
		store:        store,
		sender:       sender,
		privateKey:   privateKey,
		fromAddress:  crypto.PubkeyToAddress(privateKey.PublicKey),
		rollupCore:   rollupCore,
		contractABI:  contractABI,
		pollInterval: pollInterval,
	}, nil
}

func (p *ForcedWithdrawalSatisfier) Start(ctx context.Context) {
	if _, err := p.PollOnce(ctx); err != nil {
		p.logger.Error("initial forced-withdrawal satisfier poll failed", "err", err)
	}

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := p.PollOnce(ctx); err != nil {
				p.logger.Error("forced-withdrawal satisfier poll failed", "err", err)
			}
		}
	}
}

func (p *ForcedWithdrawalSatisfier) PollOnce(ctx context.Context) (ForcedWithdrawalSatisfactionProgress, error) {
	reader := &RollupSubmissionProcessor{
		sender:      p.sender,
		fromAddress: p.fromAddress,
		rollupCore:  p.rollupCore,
	}
	freezeState, err := reader.loadFreezeState(ctx, nil)
	if err != nil {
		return ForcedWithdrawalSatisfactionProgress{}, err
	}
	if freezeState.Frozen {
		return ForcedWithdrawalSatisfactionProgress{
			Action: ForcedWithdrawalSatisfyActionRollupStop,
			Note:   "rollup core is frozen; forced-withdrawal satisfaction is no longer active",
		}, nil
	}

	requests, err := p.store.ListPendingRollupForcedWithdrawalRequests(ctx, 20)
	if err != nil {
		return ForcedWithdrawalSatisfactionProgress{}, err
	}
	for _, request := range requests {
		if request.SatisfactionStatus == chainmodel.ForcedWithdrawalSatisfactionStatusSubmitted && request.SatisfactionTxHash != "" {
			return p.reconcileSubmitted(ctx, reader, request)
		}

		rawMatches, err := p.store.ListForcedWithdrawalClaimMatches(ctx, request.RequestID, 8)
		if err != nil {
			return ForcedWithdrawalSatisfactionProgress{}, err
		}
		matches := make([]chainmodel.ForcedWithdrawalClaimMatch, 0, len(rawMatches))
		for _, match := range rawMatches {
			chainAmount, err := assets.AccountingToChainAmount(match.Amount, p.cfg.CollateralDecimals, p.cfg.CollateralDisplayDigits)
			if err != nil {
				return ForcedWithdrawalSatisfactionProgress{}, err
			}
			if chainAmount == request.Amount {
				matches = append(matches, match)
			}
		}
		if len(matches) == 0 {
			if request.SatisfactionStatus != chainmodel.ForcedWithdrawalSatisfactionStatusNone ||
				request.MatchedWithdrawalID != "" || request.MatchedClaimID != "" {
				if err := p.store.UpdateRollupForcedWithdrawalMatch(
					ctx,
					request.RequestID,
					"",
					"",
					chainmodel.ForcedWithdrawalSatisfactionStatusNone,
					"",
				); err != nil {
					return ForcedWithdrawalSatisfactionProgress{}, err
				}
			}
			continue
		}
		if len(matches) > 1 {
			if err := p.store.UpdateRollupForcedWithdrawalMatch(
				ctx,
				request.RequestID,
				"",
				"",
				chainmodel.ForcedWithdrawalSatisfactionStatusAmbiguous,
				"multiple claimed withdrawals match one forced-withdrawal request",
			); err != nil {
				return ForcedWithdrawalSatisfactionProgress{}, err
			}
			return ForcedWithdrawalSatisfactionProgress{
				Action:    ForcedWithdrawalSatisfyActionAmbiguous,
				RequestID: request.RequestID,
				Note:      "request is ambiguous across more than one claimed withdrawal",
			}, nil
		}

		match := matches[0]
		if err := p.store.UpdateRollupForcedWithdrawalMatch(
			ctx,
			request.RequestID,
			match.WithdrawalID,
			match.ClaimID,
			chainmodel.ForcedWithdrawalSatisfactionStatusReady,
			"",
		); err != nil {
			return ForcedWithdrawalSatisfactionProgress{}, err
		}
		return p.submitSatisfaction(ctx, request.RequestID, match.ClaimID)
	}

	return ForcedWithdrawalSatisfactionProgress{
		Action: ForcedWithdrawalSatisfyActionNoop,
		Note:   "no satisfiable forced-withdrawal request",
	}, nil
}

func (p *ForcedWithdrawalSatisfier) submitSatisfaction(
	ctx context.Context,
	requestID int64,
	claimID string,
) (ForcedWithdrawalSatisfactionProgress, error) {
	if requestID <= 0 {
		return ForcedWithdrawalSatisfactionProgress{}, fmt.Errorf("request_id must be positive")
	}
	claimHash := common.HexToHash(strings.TrimSpace(claimID))
	data, err := p.contractABI.Pack("satisfyForcedWithdrawal", uint64(requestID), claimHash)
	if err != nil {
		return ForcedWithdrawalSatisfactionProgress{}, err
	}

	chainID, err := p.sender.ChainID(ctx)
	if err != nil {
		return ForcedWithdrawalSatisfactionProgress{}, err
	}
	nonce, err := p.sender.PendingNonceAt(ctx, p.fromAddress)
	if err != nil {
		return ForcedWithdrawalSatisfactionProgress{}, err
	}
	gasPrice, err := p.sender.SuggestGasPrice(ctx)
	if err != nil {
		return ForcedWithdrawalSatisfactionProgress{}, err
	}

	gasLimit := p.cfg.ChainGasLimit
	if gasLimit == 0 {
		gasLimit = 250000
	}
	estimatedGas, err := p.sender.EstimateGas(ctx, ethereum.CallMsg{
		From:     p.fromAddress,
		To:       &p.rollupCore,
		GasPrice: gasPrice,
		Data:     data,
	})
	if err == nil && estimatedGas > 0 {
		gasLimit = estimatedGas + estimatedGas/5
	}

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &p.rollupCore,
		Value:    big.NewInt(0),
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), p.privateKey)
	if err != nil {
		return ForcedWithdrawalSatisfactionProgress{}, err
	}
	if err := p.sender.SendTransaction(ctx, signedTx); err != nil {
		if markErr := p.store.MarkRollupForcedWithdrawalSatisfactionFailed(ctx, requestID, err.Error()); markErr != nil {
			return ForcedWithdrawalSatisfactionProgress{}, markErr
		}
		return ForcedWithdrawalSatisfactionProgress{
			Action:    ForcedWithdrawalSatisfyActionFailed,
			RequestID: requestID,
			ClaimID:   normalizeChainTxHash(claimHash.Hex()),
			Note:      "satisfyForcedWithdrawal submission failed",
		}, nil
	}
	txHash := normalizeChainTxHash(signedTx.Hash().Hex())
	if err := p.store.MarkRollupForcedWithdrawalSatisfactionSubmitted(ctx, requestID, txHash); err != nil {
		return ForcedWithdrawalSatisfactionProgress{}, err
	}
	return ForcedWithdrawalSatisfactionProgress{
		Action:    ForcedWithdrawalSatisfyActionSubmitted,
		RequestID: requestID,
		ClaimID:   normalizeChainTxHash(claimHash.Hex()),
		TxHash:    txHash,
		Note:      "submitted satisfyForcedWithdrawal transaction",
	}, nil
}

func (p *ForcedWithdrawalSatisfier) reconcileSubmitted(
	ctx context.Context,
	reader *RollupSubmissionProcessor,
	request chainmodel.RollupForcedWithdrawalRequest,
) (ForcedWithdrawalSatisfactionProgress, error) {
	receipt, err := p.sender.TransactionReceipt(ctx, common.HexToHash(normalizeChainTxHash(request.SatisfactionTxHash)))
	if err != nil && !errors.Is(err, ethereum.NotFound) {
		return ForcedWithdrawalSatisfactionProgress{}, err
	}
	if errors.Is(err, ethereum.NotFound) {
		receipt = nil
	}
	if receipt == nil {
		return ForcedWithdrawalSatisfactionProgress{
			Action:    ForcedWithdrawalSatisfyActionPending,
			RequestID: request.RequestID,
			ClaimID:   request.MatchedClaimID,
			TxHash:    request.SatisfactionTxHash,
			Note:      "waiting for satisfyForcedWithdrawal receipt",
		}, nil
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		if err := p.store.MarkRollupForcedWithdrawalSatisfactionFailed(ctx, request.RequestID, "satisfyForcedWithdrawal receipt reverted"); err != nil {
			return ForcedWithdrawalSatisfactionProgress{}, err
		}
		return ForcedWithdrawalSatisfactionProgress{
			Action:    ForcedWithdrawalSatisfyActionFailed,
			RequestID: request.RequestID,
			ClaimID:   request.MatchedClaimID,
			TxHash:    request.SatisfactionTxHash,
			Note:      "satisfyForcedWithdrawal receipt reverted",
		}, nil
	}

	onchainRequest, err := reader.loadForcedWithdrawalRequest(ctx, resolveReceiptBlockNumber(receipt), uint64(request.RequestID))
	if err != nil {
		return ForcedWithdrawalSatisfactionProgress{}, err
	}
	if err := p.store.UpsertRollupForcedWithdrawalRequest(ctx, chainmodel.RollupForcedWithdrawalRequest{
		RequestID:               int64(onchainRequest.RequestID),
		WalletAddress:           strings.ToLower(onchainRequest.Wallet.Hex()),
		RecipientAddress:        strings.ToLower(onchainRequest.Recipient.Hex()),
		Amount:                  onchainRequest.Amount,
		RequestedAt:             int64(onchainRequest.RequestedAt),
		DeadlineAt:              int64(onchainRequest.DeadlineAt),
		SatisfiedClaimID:        normalizeChainTxHash(onchainRequest.SatisfiedClaimID.Hex()),
		SatisfiedAt:             int64(onchainRequest.SatisfiedAt),
		FrozenAt:                int64(onchainRequest.FrozenAt),
		Status:                  mapForcedWithdrawalStatus(onchainRequest.Status),
		MatchedWithdrawalID:     request.MatchedWithdrawalID,
		MatchedClaimID:          request.MatchedClaimID,
		SatisfactionStatus:      request.SatisfactionStatus,
		SatisfactionTxHash:      request.SatisfactionTxHash,
		SatisfactionSubmittedAt: request.SatisfactionSubmittedAt,
		SatisfactionLastError:   request.SatisfactionLastError,
		SatisfactionLastErrorAt: request.SatisfactionLastErrorAt,
	}); err != nil {
		return ForcedWithdrawalSatisfactionProgress{}, err
	}
	satisfiedVisible := onchainRequest.Status == 2 ||
		onchainRequest.SatisfiedAt > 0 ||
		onchainRequest.SatisfiedClaimID != (common.Hash{})
	if !satisfiedVisible {
		return ForcedWithdrawalSatisfactionProgress{
			Action:    ForcedWithdrawalSatisfyActionPending,
			RequestID: request.RequestID,
			ClaimID:   request.MatchedClaimID,
			TxHash:    request.SatisfactionTxHash,
			Note:      "receipt succeeded; waiting for SATISFIED request state to become visible",
		}, nil
	}
	return ForcedWithdrawalSatisfactionProgress{
		Action:    ForcedWithdrawalSatisfyActionSatisfied,
		RequestID: request.RequestID,
		ClaimID:   normalizeChainTxHash(onchainRequest.SatisfiedClaimID.Hex()),
		TxHash:    request.SatisfactionTxHash,
		Note:      "forced-withdrawal request is satisfied onchain",
	}, nil
}
