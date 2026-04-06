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

	"funnyoption/internal/rollup"
	"funnyoption/internal/shared/config"
	shareddb "funnyoption/internal/shared/db"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const rollupEscapeCollateralRootABIJSON = `[{
  "type":"function",
  "name":"recordEscapeCollateralRoot",
  "stateMutability":"nonpayable",
  "inputs":[
    {"name":"batchId","type":"uint64"},
    {"name":"merkleRoot","type":"bytes32"},
    {"name":"leafCount","type":"uint64"},
    {"name":"totalAmount","type":"uint256"}
  ],
  "outputs":[]
}]`

var rollupEscapeCollateralRootABI = mustEscapeCollateralRootABI(rollupEscapeCollateralRootABIJSON)

const (
	RollupSubmissionActionNoop            = "NOOP"
	RollupSubmissionActionFrozen          = "FROZEN"
	RollupSubmissionActionEscapeRootSubmitted = "ESCAPE_ROOT_SUBMITTED"
	RollupSubmissionActionEscapeRootPending   = "ESCAPE_ROOT_PENDING"
	RollupSubmissionActionEscapeRootAnchored  = "ESCAPE_ROOT_ANCHORED"
	RollupSubmissionActionEscapeRootFailed    = "ESCAPE_ROOT_FAILED"
	RollupSubmissionActionBlockedAuth     = "BLOCKED_AUTH"
	RollupSubmissionActionFailedBlocked   = "FAILED_BLOCKED"
	RollupSubmissionActionRecordSubmitted = "RECORD_SUBMITTED"
	RollupSubmissionActionRecordPending   = "RECORD_PENDING"
	RollupSubmissionActionPublishSubmitted = "PUBLISH_SUBMITTED"
	RollupSubmissionActionPublishPending   = "PUBLISH_PENDING"
	RollupSubmissionActionAcceptSubmitted = "ACCEPT_SUBMITTED"
	RollupSubmissionActionAcceptPending   = "ACCEPT_PENDING"
	RollupSubmissionActionAccepted        = "ACCEPTED"
	RollupSubmissionActionFailed          = "FAILED"
)

type rollupSubmissionStore interface {
	ListSubmissions(ctx context.Context) ([]rollup.StoredSubmission, error)
	MaterializeAcceptedSubmissions(ctx context.Context) ([]rollup.AcceptedSubmissionMaterialization, error)
	MaterializeAcceptedSubmission(ctx context.Context, submissionID string) (rollup.AcceptedSubmissionMaterialization, error)
	NextEscapeCollateralRootForAnchor(ctx context.Context) (rollup.AcceptedEscapeCollateralRootRecord, bool, error)
	MarkEscapeCollateralRootSubmitted(ctx context.Context, batchID int64, txHash string) (rollup.AcceptedEscapeCollateralRootRecord, error)
	MarkEscapeCollateralRootAnchored(ctx context.Context, batchID int64) (rollup.AcceptedEscapeCollateralRootRecord, error)
	MarkEscapeCollateralRootFailed(ctx context.Context, batchID int64, errMsg string) (rollup.AcceptedEscapeCollateralRootRecord, error)
	RollupFrozen(ctx context.Context) (bool, error)
	PrepareNextSubmission(ctx context.Context, limit int) (rollup.PreparedShadowSubmission, error)
	MarkSubmissionRecordSubmitted(ctx context.Context, submissionID, txHash string) (rollup.StoredSubmission, error)
	MarkSubmissionPublishSubmitted(ctx context.Context, submissionID, txHash string) (rollup.StoredSubmission, error)
	MarkSubmissionDataPublished(ctx context.Context, submissionID string) (rollup.StoredSubmission, error)
	MarkSubmissionAcceptSubmitted(ctx context.Context, submissionID, txHash string) (rollup.StoredSubmission, error)
	MarkSubmissionAccepted(ctx context.Context, submissionID string) (rollup.StoredSubmission, error)
	MarkSubmissionFailed(ctx context.Context, submissionID, errMsg string) (rollup.StoredSubmission, error)
	RecordSubmissionError(ctx context.Context, submissionID, errMsg string) (rollup.StoredSubmission, error)
}

type rollupTxSender interface {
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error)
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
	SendTransaction(ctx context.Context, tx *types.Transaction) error
	ChainID(ctx context.Context) (*big.Int, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

type RollupSubmissionProgress struct {
	Action     string                  `json:"action"`
	Prepared   bool                    `json:"prepared"`
	Submission rollup.StoredSubmission `json:"submission"`
	EscapeRoot rollup.AcceptedEscapeCollateralRootRecord `json:"escape_root"`
	TxHash     string                  `json:"tx_hash,omitempty"`
	Note       string                  `json:"note,omitempty"`
}

type RollupSubmissionRun struct {
	Steps []RollupSubmissionProgress `json:"steps"`
}

type RollupSubmissionProcessor struct {
	logger       *slog.Logger
	cfg          config.ServiceConfig
	store        rollupSubmissionStore
	sender       rollupTxSender
	privateKey   *ecdsa.PrivateKey
	fromAddress  common.Address
	rollupCore   common.Address
	batchLimit   int
	pollInterval time.Duration
}

func NewRollupSubmissionProcessor(
	logger *slog.Logger,
	cfg config.ServiceConfig,
	store rollupSubmissionStore,
	sender rollupTxSender,
) (*RollupSubmissionProcessor, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if store == nil {
		return nil, fmt.Errorf("rollup submission store is required")
	}
	if sender == nil {
		return nil, fmt.Errorf("rollup submission sender is required")
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
	rollupCore, err := validateClaimAddress("rollup_core_address", cfg.RollupCoreAddress)
	if err != nil {
		return nil, err
	}
	batchLimit := cfg.RollupBatchLimit
	if batchLimit <= 0 {
		batchLimit = 256
	}
	pollInterval := cfg.RollupPollInterval
	if pollInterval <= 0 {
		pollInterval = 10 * time.Second
	}

	return &RollupSubmissionProcessor{
		logger:       logger,
		cfg:          cfg,
		store:        store,
		sender:       sender,
		privateKey:   privateKey,
		fromAddress:  crypto.PubkeyToAddress(privateKey.PublicKey),
		rollupCore:   rollupCore,
		batchLimit:   batchLimit,
		pollInterval: pollInterval,
	}, nil
}

func RunRollupSubmissionOnce(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.ServiceConfig,
) (RollupSubmissionProgress, error) {
	dbConn, err := shareddb.OpenPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		return RollupSubmissionProgress{}, err
	}
	defer dbConn.Close()

	rpcPool, err := newRPCPool(ctx, cfg)
	if err != nil {
		return RollupSubmissionProgress{}, err
	}
	defer rpcPool.Close()

	processor, err := NewRollupSubmissionProcessor(logger, cfg, rollup.NewStore(dbConn), rpcPool)
	if err != nil {
		return RollupSubmissionProgress{}, err
	}
	return processor.PollOnce(ctx)
}

func RunRollupSubmissionUntilIdle(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.ServiceConfig,
) (RollupSubmissionRun, error) {
	dbConn, err := shareddb.OpenPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		return RollupSubmissionRun{}, err
	}
	defer dbConn.Close()

	rpcPool, err := newRPCPool(ctx, cfg)
	if err != nil {
		return RollupSubmissionRun{}, err
	}
	defer rpcPool.Close()

	processor, err := NewRollupSubmissionProcessor(logger, cfg, rollup.NewStore(dbConn), rpcPool)
	if err != nil {
		return RollupSubmissionRun{}, err
	}
	return processor.RunUntilIdle(ctx)
}

func (p *RollupSubmissionProcessor) Start(ctx context.Context) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	if _, err := p.PollOnce(ctx); err != nil {
		p.logger.Error("initial rollup submission poll failed", "err", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := p.PollOnce(ctx); err != nil {
				p.logger.Error("rollup submission poll failed", "err", err)
			}
		}
	}
}

func (p *RollupSubmissionProcessor) RunUntilIdle(ctx context.Context) (RollupSubmissionRun, error) {
	run := RollupSubmissionRun{Steps: make([]RollupSubmissionProgress, 0, 8)}
	for {
		progress, err := p.PollOnce(ctx)
		if err != nil {
			return run, err
		}
		run.Steps = append(run.Steps, progress)
		switch progress.Action {
		case RollupSubmissionActionNoop,
			RollupSubmissionActionFrozen,
			RollupSubmissionActionEscapeRootFailed,
			RollupSubmissionActionBlockedAuth,
			RollupSubmissionActionFailedBlocked,
			RollupSubmissionActionFailed:
			return run, nil
		case RollupSubmissionActionAccepted, RollupSubmissionActionEscapeRootAnchored:
			continue
		case RollupSubmissionActionRecordSubmitted,
			RollupSubmissionActionRecordPending,
			RollupSubmissionActionPublishSubmitted,
			RollupSubmissionActionPublishPending,
			RollupSubmissionActionEscapeRootSubmitted,
			RollupSubmissionActionEscapeRootPending,
			RollupSubmissionActionAcceptSubmitted,
			RollupSubmissionActionAcceptPending:
			if err := sleepWithContext(ctx, p.pollInterval); err != nil {
				return run, err
			}
		default:
			return run, fmt.Errorf("unsupported rollup submission action %q", progress.Action)
		}
	}
}

func (p *RollupSubmissionProcessor) PollOnce(ctx context.Context) (RollupSubmissionProgress, error) {
	if _, err := p.store.MaterializeAcceptedSubmissions(ctx); err != nil {
		return RollupSubmissionProgress{}, err
	}
	frozen, err := p.store.RollupFrozen(ctx)
	if err != nil {
		return RollupSubmissionProgress{}, err
	}
	if frozen {
		return RollupSubmissionProgress{
			Action: RollupSubmissionActionFrozen,
			Note:   "rollup core is frozen; submission runtime is idle",
		}, nil
	}
	escapeRoot, hasEscapeRoot, err := p.store.NextEscapeCollateralRootForAnchor(ctx)
	if err != nil {
		return RollupSubmissionProgress{}, err
	}
	if hasEscapeRoot {
		switch escapeRoot.AnchorStatus {
		case rollup.EscapeCollateralAnchorStatusReady:
			return p.submitEscapeRootAnchor(ctx, escapeRoot)
		case rollup.EscapeCollateralAnchorStatusSubmitted:
			return p.advanceAfterEscapeRootReceipt(ctx, escapeRoot)
		case rollup.EscapeCollateralAnchorStatusFailed:
			return RollupSubmissionProgress{
				Action:     RollupSubmissionActionEscapeRootFailed,
				EscapeRoot: escapeRoot,
				Note:       "earliest accepted escape root is in FAILED state and blocks later roots",
			}, nil
		}
	}
	submission, prepared, err := p.nextSubmission(ctx)
	if err != nil {
		if errors.Is(err, rollup.ErrNoPendingSubmission) {
			return RollupSubmissionProgress{
				Action: RollupSubmissionActionNoop,
				Note:   "no pending rollup submission",
			}, nil
		}
		return RollupSubmissionProgress{}, err
	}

	switch submission.Status {
	case rollup.SubmissionStatusBlockedAuth:
		return RollupSubmissionProgress{
			Action:     RollupSubmissionActionBlockedAuth,
			Prepared:   prepared,
			Submission: submission,
			Note:       "earliest submission is blocked on auth join status",
		}, nil
	case rollup.SubmissionStatusFailed:
		return RollupSubmissionProgress{
			Action:     RollupSubmissionActionFailedBlocked,
			Prepared:   prepared,
			Submission: submission,
			Note:       "earliest submission is in FAILED state and blocks later batches",
		}, nil
	case rollup.SubmissionStatusReady:
		return p.submitRecordLeg(ctx, submission, prepared)
	case rollup.SubmissionStatusRecordSubmitted:
		return p.advanceAfterRecordReceipt(ctx, submission)
	case rollup.SubmissionStatusDataPublished:
		return p.submitAcceptLeg(ctx, submission)
	case rollup.SubmissionStatusPublishSubmitted:
		return p.advanceAfterPublishReceipt(ctx, submission)
	case rollup.SubmissionStatusAcceptSubmitted:
		return p.advanceAfterAcceptReceipt(ctx, submission)
	case rollup.SubmissionStatusAccepted:
		return RollupSubmissionProgress{
			Action:     RollupSubmissionActionNoop,
			Prepared:   prepared,
			Submission: submission,
			Note:       "submission is already accepted",
		}, nil
	default:
		return RollupSubmissionProgress{}, fmt.Errorf("unsupported submission status %q for %s", submission.Status, submission.SubmissionID)
	}
}

func (p *RollupSubmissionProcessor) nextSubmission(ctx context.Context) (rollup.StoredSubmission, bool, error) {
	submissions, err := p.store.ListSubmissions(ctx)
	if err != nil {
		return rollup.StoredSubmission{}, false, err
	}
	for _, submission := range submissions {
		switch submission.Status {
		case rollup.SubmissionStatusAccepted:
			continue
		case rollup.SubmissionStatusBlockedAuth,
			rollup.SubmissionStatusFailed,
			rollup.SubmissionStatusReady,
			rollup.SubmissionStatusRecordSubmitted,
			rollup.SubmissionStatusPublishSubmitted,
			rollup.SubmissionStatusDataPublished,
			rollup.SubmissionStatusAcceptSubmitted:
			return submission, false, nil
		default:
			return rollup.StoredSubmission{}, false, fmt.Errorf("unsupported persisted submission status %q for %s", submission.Status, submission.SubmissionID)
		}
	}

	prepared, err := p.store.PrepareNextSubmission(ctx, p.batchLimit)
	if err != nil {
		return rollup.StoredSubmission{}, false, err
	}
	return prepared.StoredSubmission, true, nil
}

func (p *RollupSubmissionProcessor) submitEscapeRootAnchor(
	ctx context.Context,
	root rollup.AcceptedEscapeCollateralRootRecord,
) (RollupSubmissionProgress, error) {
	txHash, err := p.submitCalldata(ctx, mustPackEscapeCollateralRootCalldata(root))
	if err != nil {
		_, _ = p.store.MarkEscapeCollateralRootFailed(ctx, root.BatchID, err.Error())
		return RollupSubmissionProgress{}, err
	}
	updated, err := p.store.MarkEscapeCollateralRootSubmitted(ctx, root.BatchID, txHash)
	if err != nil {
		return RollupSubmissionProgress{}, err
	}
	return RollupSubmissionProgress{
		Action:     RollupSubmissionActionEscapeRootSubmitted,
		EscapeRoot: updated,
		TxHash:     updated.AnchorTxHash,
	}, nil
}

func (p *RollupSubmissionProcessor) advanceAfterEscapeRootReceipt(
	ctx context.Context,
	root rollup.AcceptedEscapeCollateralRootRecord,
) (RollupSubmissionProgress, error) {
	receipt, err := p.lookupReceipt(ctx, root.AnchorTxHash)
	if err != nil {
		_, _ = p.store.MarkEscapeCollateralRootFailed(ctx, root.BatchID, err.Error())
		return RollupSubmissionProgress{}, err
	}
	if receipt == nil {
		return RollupSubmissionProgress{
			Action:     RollupSubmissionActionEscapeRootPending,
			EscapeRoot: root,
			TxHash:     root.AnchorTxHash,
			Note:       "recordEscapeCollateralRoot tx is still pending",
		}, nil
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		updated, updateErr := p.store.MarkEscapeCollateralRootFailed(ctx, root.BatchID, "recordEscapeCollateralRoot tx reverted onchain")
		if updateErr != nil {
			return RollupSubmissionProgress{}, updateErr
		}
		return RollupSubmissionProgress{
			Action:     RollupSubmissionActionEscapeRootFailed,
			EscapeRoot: updated,
			TxHash:     updated.AnchorTxHash,
		}, nil
	}
	observed, err := p.loadEscapeCollateralRootState(ctx, resolveReceiptBlockNumber(receipt), uint64(root.BatchID))
	if err != nil {
		return RollupSubmissionProgress{}, err
	}
	if observed.LatestEscapeCollateralBatchID != uint64(root.BatchID) ||
		observed.MerkleRoot != common.HexToHash(root.MerkleRoot) {
		return RollupSubmissionProgress{
			Action:     RollupSubmissionActionEscapeRootPending,
			EscapeRoot: root,
			TxHash:     root.AnchorTxHash,
			Note:       "escape collateral root receipt succeeded but onchain anchor state has not reconciled yet",
		}, nil
	}
	updated, err := p.store.MarkEscapeCollateralRootAnchored(ctx, root.BatchID)
	if err != nil {
		return RollupSubmissionProgress{}, err
	}
	return RollupSubmissionProgress{
		Action:     RollupSubmissionActionEscapeRootAnchored,
		EscapeRoot: updated,
		TxHash:     updated.AnchorTxHash,
	}, nil
}

func (p *RollupSubmissionProcessor) submitRecordLeg(
	ctx context.Context,
	submission rollup.StoredSubmission,
	prepared bool,
) (RollupSubmissionProgress, error) {
	txHash, err := p.submitCalldata(ctx, submission.RecordCalldata)
	if err != nil {
		_, _ = p.store.RecordSubmissionError(ctx, submission.SubmissionID, err.Error())
		return RollupSubmissionProgress{}, err
	}
	updated, err := p.store.MarkSubmissionRecordSubmitted(ctx, submission.SubmissionID, txHash)
	if err != nil {
		return RollupSubmissionProgress{}, err
	}
	return RollupSubmissionProgress{
		Action:     RollupSubmissionActionRecordSubmitted,
		Prepared:   prepared,
		Submission: updated,
		TxHash:     updated.RecordTxHash,
	}, nil
}

func (p *RollupSubmissionProcessor) advanceAfterRecordReceipt(
	ctx context.Context,
	submission rollup.StoredSubmission,
) (RollupSubmissionProgress, error) {
	receipt, err := p.lookupReceipt(ctx, submission.RecordTxHash)
	if err != nil {
		_, _ = p.store.RecordSubmissionError(ctx, submission.SubmissionID, err.Error())
		return RollupSubmissionProgress{}, err
	}
	if receipt == nil {
		return RollupSubmissionProgress{
			Action:     RollupSubmissionActionRecordPending,
			Submission: submission,
			TxHash:     submission.RecordTxHash,
			Note:       "recordBatchMetadata tx is still pending",
		}, nil
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		updated, updateErr := p.store.MarkSubmissionFailed(
			ctx,
			submission.SubmissionID,
			fmt.Sprintf("recordBatchMetadata tx reverted: %s", submission.RecordTxHash),
		)
		if updateErr != nil {
			return RollupSubmissionProgress{}, updateErr
		}
		return RollupSubmissionProgress{
			Action:     RollupSubmissionActionFailed,
			Submission: updated,
			TxHash:     submission.RecordTxHash,
		}, nil
	}
	reconciled, note, err := p.reconcileRecordSubmissionState(ctx, submission, receipt)
	if err != nil {
		_, _ = p.store.RecordSubmissionError(ctx, submission.SubmissionID, err.Error())
		return RollupSubmissionProgress{}, err
	}
	if !reconciled {
		return RollupSubmissionProgress{
			Action:     RollupSubmissionActionRecordPending,
			Submission: submission,
			TxHash:     submission.RecordTxHash,
			Note:       note,
		}, nil
	}

	return p.submitPublishDataLeg(ctx, submission)
}

func (p *RollupSubmissionProcessor) submitPublishDataLeg(
	ctx context.Context,
	submission rollup.StoredSubmission,
) (RollupSubmissionProgress, error) {
	txHash, err := p.submitCalldata(ctx, submission.PublishCalldata)
	if err != nil {
		_, _ = p.store.RecordSubmissionError(ctx, submission.SubmissionID, err.Error())
		return RollupSubmissionProgress{}, err
	}
	updated, err := p.store.MarkSubmissionPublishSubmitted(ctx, submission.SubmissionID, txHash)
	if err != nil {
		return RollupSubmissionProgress{}, err
	}
	return RollupSubmissionProgress{
		Action:     RollupSubmissionActionPublishSubmitted,
		Submission: updated,
		TxHash:     updated.PublishTxHash,
	}, nil
}

func (p *RollupSubmissionProcessor) advanceAfterPublishReceipt(
	ctx context.Context,
	submission rollup.StoredSubmission,
) (RollupSubmissionProgress, error) {
	receipt, err := p.lookupReceipt(ctx, submission.PublishTxHash)
	if err != nil {
		_, _ = p.store.RecordSubmissionError(ctx, submission.SubmissionID, err.Error())
		return RollupSubmissionProgress{}, err
	}
	if receipt == nil {
		return RollupSubmissionProgress{
			Action:     RollupSubmissionActionPublishPending,
			Submission: submission,
			TxHash:     submission.PublishTxHash,
			Note:       "publishBatchData tx is still pending",
		}, nil
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		updated, updateErr := p.store.MarkSubmissionFailed(
			ctx,
			submission.SubmissionID,
			fmt.Sprintf("publishBatchData tx reverted: %s", submission.PublishTxHash),
		)
		if updateErr != nil {
			return RollupSubmissionProgress{}, updateErr
		}
		return RollupSubmissionProgress{
			Action:     RollupSubmissionActionFailed,
			Submission: updated,
			TxHash:     submission.PublishTxHash,
		}, nil
	}

	reconciled, err := p.reconcilePublishDataState(ctx, submission, receipt)
	if err != nil {
		_, _ = p.store.RecordSubmissionError(ctx, submission.SubmissionID, err.Error())
		return RollupSubmissionProgress{}, err
	}
	if !reconciled {
		return RollupSubmissionProgress{
			Action:     RollupSubmissionActionPublishPending,
			Submission: submission,
			TxHash:     submission.PublishTxHash,
			Note:       "publishBatchData receipt succeeded; waiting for visible onchain DA reconciliation",
		}, nil
	}

	updated, err := p.store.MarkSubmissionDataPublished(ctx, submission.SubmissionID)
	if err != nil {
		return RollupSubmissionProgress{}, err
	}
	return p.submitAcceptLeg(ctx, updated)
}

func (p *RollupSubmissionProcessor) submitAcceptLeg(
	ctx context.Context,
	submission rollup.StoredSubmission,
) (RollupSubmissionProgress, error) {
	txHash, err := p.submitCalldata(ctx, submission.AcceptCalldata)
	if err != nil {
		_, _ = p.store.RecordSubmissionError(ctx, submission.SubmissionID, err.Error())
		return RollupSubmissionProgress{}, err
	}
	updated, err := p.store.MarkSubmissionAcceptSubmitted(ctx, submission.SubmissionID, txHash)
	if err != nil {
		return RollupSubmissionProgress{}, err
	}
	return RollupSubmissionProgress{
		Action:     RollupSubmissionActionAcceptSubmitted,
		Submission: updated,
		TxHash:     updated.AcceptTxHash,
	}, nil
}

func (p *RollupSubmissionProcessor) advanceAfterAcceptReceipt(
	ctx context.Context,
	submission rollup.StoredSubmission,
) (RollupSubmissionProgress, error) {
	receipt, err := p.lookupReceipt(ctx, submission.AcceptTxHash)
	if err != nil {
		_, _ = p.store.RecordSubmissionError(ctx, submission.SubmissionID, err.Error())
		return RollupSubmissionProgress{}, err
	}
	if receipt == nil {
		return RollupSubmissionProgress{
			Action:     RollupSubmissionActionAcceptPending,
			Submission: submission,
			TxHash:     submission.AcceptTxHash,
			Note:       "acceptVerifiedBatch tx is still pending",
		}, nil
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		updated, updateErr := p.store.MarkSubmissionFailed(
			ctx,
			submission.SubmissionID,
			fmt.Sprintf("acceptVerifiedBatch tx reverted: %s", submission.AcceptTxHash),
		)
		if updateErr != nil {
			return RollupSubmissionProgress{}, updateErr
		}
		return RollupSubmissionProgress{
			Action:     RollupSubmissionActionFailed,
			Submission: updated,
			TxHash:     submission.AcceptTxHash,
		}, nil
	}
	reconciled, note, err := p.reconcileAcceptedSubmissionState(ctx, submission, receipt)
	if err != nil {
		_, _ = p.store.RecordSubmissionError(ctx, submission.SubmissionID, err.Error())
		return RollupSubmissionProgress{}, err
	}
	if !reconciled {
		return RollupSubmissionProgress{
			Action:     RollupSubmissionActionAcceptPending,
			Submission: submission,
			TxHash:     submission.AcceptTxHash,
			Note:       note,
		}, nil
	}
	updated, err := p.store.MarkSubmissionAccepted(ctx, submission.SubmissionID)
	if err != nil {
		return RollupSubmissionProgress{}, err
	}
	if _, err := p.store.MaterializeAcceptedSubmission(ctx, updated.SubmissionID); err != nil {
		return RollupSubmissionProgress{}, err
	}
	return RollupSubmissionProgress{
		Action:     RollupSubmissionActionAccepted,
		Submission: updated,
		TxHash:     updated.AcceptTxHash,
	}, nil
}

func (p *RollupSubmissionProcessor) lookupReceipt(ctx context.Context, txHash string) (*types.Receipt, error) {
	normalized := normalizeChainTxHash(txHash)
	if normalized == "" {
		return nil, fmt.Errorf("submission tx hash is required")
	}
	receipt, err := p.sender.TransactionReceipt(ctx, common.HexToHash("0x"+normalized))
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return nil, nil
		}
		return nil, err
	}
	return receipt, nil
}

func (p *RollupSubmissionProcessor) submitCalldata(ctx context.Context, calldata string) (string, error) {
	data := common.FromHex(strings.TrimSpace(calldata))
	if len(data) == 0 {
		return "", fmt.Errorf("submission calldata is empty")
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
		return "", err
	}
	if err := p.sender.SendTransaction(ctx, signedTx); err != nil {
		return "", err
	}
	return normalizeChainTxHash(signedTx.Hash().Hex()), nil
}

func mustEscapeCollateralRootABI(raw string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(raw))
	if err != nil {
		panic(err)
	}
	return parsed
}

func mustPackEscapeCollateralRootCalldata(root rollup.AcceptedEscapeCollateralRootRecord) string {
	method := rollupEscapeCollateralRootABI.Methods["recordEscapeCollateralRoot"]
	data, err := method.Inputs.Pack(
		uint64(root.BatchID),
		common.HexToHash(root.MerkleRoot),
		uint64(root.LeafCount),
		big.NewInt(root.TotalAmount),
	)
	if err != nil {
		panic(err)
	}
	return "0x" + hexEncodeWithSelector(method.ID, data)
}

func hexEncodeWithSelector(selector []byte, args []byte) string {
	return common.Bytes2Hex(append(append([]byte(nil), selector...), args...))
}

type expectedRollupSubmissionState struct {
	BatchID              uint64
	FirstSequenceNo      uint64
	LastSequenceNo       uint64
	EntryCount           uint64
	BatchDataHash        common.Hash
	PrevStateRoot        common.Hash
	BalancesRoot         common.Hash
	OrdersRoot           common.Hash
	PositionsFundingRoot common.Hash
	WithdrawalsRoot      common.Hash
	NextStateRoot        common.Hash
	AuthProofHash        common.Hash
	VerifierGateHash     common.Hash
}

func (p *RollupSubmissionProcessor) reconcileRecordSubmissionState(
	ctx context.Context,
	submission rollup.StoredSubmission,
	receipt *types.Receipt,
) (bool, string, error) {
	expected, err := buildExpectedSubmissionState(submission)
	if err != nil {
		return false, "", err
	}
	observed, err := p.loadRecordedBatchState(ctx, resolveReceiptBlockNumber(receipt), expected.BatchID)
	if err != nil {
		return false, "", err
	}
	if observed.LatestBatchID != expected.BatchID ||
		observed.LatestStateRoot != expected.NextStateRoot ||
		observed.BatchDataHash != expected.BatchDataHash ||
		observed.PrevStateRoot != expected.PrevStateRoot ||
		observed.NextStateRoot != expected.NextStateRoot {
		return false, "recordBatchMetadata receipt succeeded; waiting for visible onchain metadata reconciliation", nil
	}
	return true, "", nil
}

func (p *RollupSubmissionProcessor) reconcileAcceptedSubmissionState(
	ctx context.Context,
	submission rollup.StoredSubmission,
	receipt *types.Receipt,
) (bool, string, error) {
	expected, err := buildExpectedSubmissionState(submission)
	if err != nil {
		return false, "", err
	}
	observed, err := p.loadAcceptedBatchState(ctx, resolveReceiptBlockNumber(receipt), expected.BatchID)
	if err != nil {
		return false, "", err
	}
	if observed.LatestAcceptedBatchID != expected.BatchID ||
		observed.LatestAcceptedStateRoot != expected.NextStateRoot ||
		observed.FirstSequenceNo != expected.FirstSequenceNo ||
		observed.LastSequenceNo != expected.LastSequenceNo ||
		observed.EntryCount != expected.EntryCount ||
		observed.BatchDataHash != expected.BatchDataHash ||
		observed.PrevStateRoot != expected.PrevStateRoot ||
		observed.BalancesRoot != expected.BalancesRoot ||
		observed.OrdersRoot != expected.OrdersRoot ||
		observed.PositionsFundingRoot != expected.PositionsFundingRoot ||
		observed.WithdrawalsRoot != expected.WithdrawalsRoot ||
		observed.NextStateRoot != expected.NextStateRoot ||
		observed.AuthProofHash != expected.AuthProofHash ||
		observed.VerifierGateHash != expected.VerifierGateHash {
		return false, "acceptVerifiedBatch receipt succeeded; waiting for visible onchain acceptance reconciliation", nil
	}
	return true, "", nil
}

func buildExpectedSubmissionState(submission rollup.StoredSubmission) (expectedRollupSubmissionState, error) {
	bundle, err := rollup.DecodeStoredSubmissionBundle(submission)
	if err != nil {
		return expectedRollupSubmissionState{}, err
	}
	batchDataHash, err := parseExpectedHash(bundle.Batch.InputHash, "bundle.batch.input_hash")
	if err != nil {
		return expectedRollupSubmissionState{}, err
	}
	prevStateRoot, err := parseExpectedHash(bundle.Batch.PrevStateRoot, "bundle.batch.prev_state_root")
	if err != nil {
		return expectedRollupSubmissionState{}, err
	}
	balancesRoot, err := parseExpectedHash(bundle.Batch.BalancesRoot, "bundle.batch.balances_root")
	if err != nil {
		return expectedRollupSubmissionState{}, err
	}
	ordersRoot, err := parseExpectedHash(bundle.Batch.OrdersRoot, "bundle.batch.orders_root")
	if err != nil {
		return expectedRollupSubmissionState{}, err
	}
	positionsFundingRoot, err := parseExpectedHash(bundle.Batch.PositionsFundingRoot, "bundle.batch.positions_funding_root")
	if err != nil {
		return expectedRollupSubmissionState{}, err
	}
	withdrawalsRoot, err := parseExpectedHash(bundle.Batch.WithdrawalsRoot, "bundle.batch.withdrawals_root")
	if err != nil {
		return expectedRollupSubmissionState{}, err
	}
	nextStateRoot, err := parseExpectedHash(bundle.Batch.NextStateRoot, "bundle.batch.next_state_root")
	if err != nil {
		return expectedRollupSubmissionState{}, err
	}
	authProofHash, err := parseExpectedHash(submission.AuthProofHash, "submission.auth_proof_hash")
	if err != nil {
		return expectedRollupSubmissionState{}, err
	}
	verifierGateHash, err := parseExpectedHash(submission.VerifierGateHash, "submission.verifier_gate_hash")
	if err != nil {
		return expectedRollupSubmissionState{}, err
	}
	if submission.BatchID <= 0 {
		return expectedRollupSubmissionState{}, fmt.Errorf("submission batch_id must be positive")
	}
	return expectedRollupSubmissionState{
		BatchID:              uint64(submission.BatchID),
		FirstSequenceNo:      uint64(bundle.Batch.FirstSequence),
		LastSequenceNo:       uint64(bundle.Batch.LastSequence),
		EntryCount:           uint64(bundle.Batch.EntryCount),
		BatchDataHash:        batchDataHash,
		PrevStateRoot:        prevStateRoot,
		BalancesRoot:         balancesRoot,
		OrdersRoot:           ordersRoot,
		PositionsFundingRoot: positionsFundingRoot,
		WithdrawalsRoot:      withdrawalsRoot,
		NextStateRoot:        nextStateRoot,
		AuthProofHash:        authProofHash,
		VerifierGateHash:     verifierGateHash,
	}, nil
}

func resolveReceiptBlockNumber(_ *types.Receipt) *big.Int {
	// Always query at latest block; archive state is not available
	// on many public testnet RPCs (BSC testnet, etc.).
	return nil
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
