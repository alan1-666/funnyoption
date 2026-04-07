package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	chainmodel "funnyoption/internal/chain/model"
	"funnyoption/internal/shared/config"

	"github.com/ethereum/go-ethereum/common"
)

const (
	forcedWithdrawalStatusNone      = "NONE"
	forcedWithdrawalStatusRequested = "REQUESTED"
	forcedWithdrawalStatusSatisfied = "SATISFIED"
	forcedWithdrawalStatusFrozen    = "FROZEN"
)

type forcedWithdrawalMirrorStore interface {
	UpsertRollupForcedWithdrawalRequest(ctx context.Context, request chainmodel.RollupForcedWithdrawalRequest) error
	UpsertRollupFreezeState(ctx context.Context, state chainmodel.RollupFreezeState) error
}

type ForcedWithdrawalMirrorProgress struct {
	RequestCount    int   `json:"request_count"`
	Frozen          bool  `json:"frozen"`
	FreezeRequestID int64 `json:"freeze_request_id"`
}

type ForcedWithdrawalMirrorProcessor struct {
	logger       *slog.Logger
	store        forcedWithdrawalMirrorStore
	sender       rollupTxSender
	fromAddress  common.Address
	rollupCore   common.Address
	pollInterval time.Duration
}

func NewForcedWithdrawalMirrorProcessor(
	logger *slog.Logger,
	cfg config.ServiceConfig,
	store forcedWithdrawalMirrorStore,
	sender rollupTxSender,
) (*ForcedWithdrawalMirrorProcessor, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if store == nil {
		return nil, fmt.Errorf("forced withdrawal mirror store is required")
	}
	if sender == nil {
		return nil, fmt.Errorf("forced withdrawal mirror sender is required")
	}
	if strings.TrimSpace(cfg.RollupCoreAddress) == "" {
		return nil, fmt.Errorf("rollup core address is required")
	}
	rollupCore, err := validateClaimAddress("rollup_core_address", cfg.RollupCoreAddress)
	if err != nil {
		return nil, err
	}
	pollInterval := cfg.RollupPollInterval
	if pollInterval <= 0 {
		pollInterval = 10 * time.Second
	}

	return &ForcedWithdrawalMirrorProcessor{
		logger:       logger,
		store:        store,
		sender:       sender,
		fromAddress:  common.Address{},
		rollupCore:   rollupCore,
		pollInterval: pollInterval,
	}, nil
}

func (p *ForcedWithdrawalMirrorProcessor) Start(ctx context.Context) {
	if _, err := p.PollOnce(ctx); err != nil {
		p.logger.Error("initial forced-withdrawal mirror poll failed", "err", err)
	}

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := p.PollOnce(ctx); err != nil {
				p.logger.Error("forced-withdrawal mirror poll failed", "err", err)
			}
		}
	}
}

func (p *ForcedWithdrawalMirrorProcessor) PollOnce(ctx context.Context) (ForcedWithdrawalMirrorProgress, error) {
	reader := &RollupSubmissionProcessor{
		sender:      p.sender,
		fromAddress: p.fromAddress,
		rollupCore:  p.rollupCore,
	}

	freezeState, err := reader.loadFreezeState(ctx, nil)
	if err != nil {
		return ForcedWithdrawalMirrorProgress{}, err
	}
	if err := p.store.UpsertRollupFreezeState(ctx, chainmodel.RollupFreezeState{
		Frozen:    freezeState.Frozen,
		FrozenAt:  int64(freezeState.FrozenAt),
		RequestID: int64(freezeState.FreezeRequestID),
	}); err != nil {
		return ForcedWithdrawalMirrorProgress{}, err
	}

	requestCount, err := reader.loadForcedWithdrawalRequestCount(ctx, nil)
	if err != nil {
		return ForcedWithdrawalMirrorProgress{}, err
	}
	for requestID := uint64(1); requestID <= requestCount; requestID++ {
		request, err := reader.loadForcedWithdrawalRequest(ctx, nil, requestID)
		if err != nil {
			return ForcedWithdrawalMirrorProgress{}, err
		}
		if err := p.store.UpsertRollupForcedWithdrawalRequest(ctx, chainmodel.RollupForcedWithdrawalRequest{
			RequestID:        int64(request.RequestID),
			WalletAddress:    strings.ToLower(request.Wallet.Hex()),
			RecipientAddress: strings.ToLower(request.Recipient.Hex()),
			Amount:           request.Amount,
			RequestedAt:      int64(request.RequestedAt),
			DeadlineAt:       int64(request.DeadlineAt),
			SatisfiedClaimID: normalizeChainTxHash(request.SatisfiedClaimID.Hex()),
			SatisfiedAt:      int64(request.SatisfiedAt),
			FrozenAt:         int64(request.FrozenAt),
			Status:           mapForcedWithdrawalStatus(request.Status),
		}); err != nil {
			return ForcedWithdrawalMirrorProgress{}, err
		}
	}

	return ForcedWithdrawalMirrorProgress{
		RequestCount:    int(requestCount),
		Frozen:          freezeState.Frozen,
		FreezeRequestID: int64(freezeState.FreezeRequestID),
	}, nil
}

func mapForcedWithdrawalStatus(status uint8) string {
	switch status {
	case 1:
		return forcedWithdrawalStatusRequested
	case 2:
		return forcedWithdrawalStatusSatisfied
	case 3:
		return forcedWithdrawalStatusFrozen
	default:
		return forcedWithdrawalStatusNone
	}
}
