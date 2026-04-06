package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"funnyoption/internal/rollup"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const funnyRollupCoreReadABIJSON = `[
  {
    "type":"function",
    "name":"latestBatchId",
    "stateMutability":"view",
    "inputs":[],
    "outputs":[{"name":"","type":"uint64"}]
  },
  {
    "type":"function",
    "name":"latestStateRoot",
    "stateMutability":"view",
    "inputs":[],
    "outputs":[{"name":"","type":"bytes32"}]
  },
  {
    "type":"function",
    "name":"latestAcceptedBatchId",
    "stateMutability":"view",
    "inputs":[],
    "outputs":[{"name":"","type":"uint64"}]
  },
  {
    "type":"function",
    "name":"latestAcceptedStateRoot",
    "stateMutability":"view",
    "inputs":[],
    "outputs":[{"name":"","type":"bytes32"}]
  },
  {
    "type":"function",
    "name":"batchMetadata",
    "stateMutability":"view",
    "inputs":[{"name":"","type":"uint64"}],
    "outputs":[
      {"name":"batchDataHash","type":"bytes32"},
      {"name":"prevStateRoot","type":"bytes32"},
      {"name":"nextStateRoot","type":"bytes32"}
    ]
  },
  {
    "type":"function",
    "name":"acceptedBatches",
    "stateMutability":"view",
    "inputs":[{"name":"","type":"uint64"}],
    "outputs":[
      {"name":"firstSequenceNo","type":"uint64"},
      {"name":"lastSequenceNo","type":"uint64"},
      {"name":"entryCount","type":"uint64"},
      {"name":"batchDataHash","type":"bytes32"},
      {"name":"prevStateRoot","type":"bytes32"},
      {"name":"balancesRoot","type":"bytes32"},
      {"name":"ordersRoot","type":"bytes32"},
      {"name":"positionsFundingRoot","type":"bytes32"},
      {"name":"withdrawalsRoot","type":"bytes32"},
      {"name":"nextStateRoot","type":"bytes32"},
      {"name":"conservationHash","type":"bytes32"},
      {"name":"authProofHash","type":"bytes32"},
      {"name":"verifierGateHash","type":"bytes32"}
    ]
  },
  {
    "type":"function",
    "name":"frozen",
    "stateMutability":"view",
    "inputs":[],
    "outputs":[{"name":"","type":"bool"}]
  },
  {
    "type":"function",
    "name":"frozenAt",
    "stateMutability":"view",
    "inputs":[],
    "outputs":[{"name":"","type":"uint64"}]
  },
  {
    "type":"function",
    "name":"freezeRequestId",
    "stateMutability":"view",
    "inputs":[],
    "outputs":[{"name":"","type":"uint64"}]
  },
  {
    "type":"function",
    "name":"forcedWithdrawalRequestCount",
    "stateMutability":"view",
    "inputs":[],
    "outputs":[{"name":"","type":"uint64"}]
  },
  {
    "type":"function",
    "name":"forcedWithdrawalRequests",
    "stateMutability":"view",
    "inputs":[{"name":"","type":"uint64"}],
    "outputs":[
      {"name":"wallet","type":"address"},
      {"name":"recipient","type":"address"},
      {"name":"amount","type":"uint256"},
      {"name":"requestedAt","type":"uint64"},
      {"name":"deadlineAt","type":"uint64"},
      {"name":"satisfiedClaimId","type":"bytes32"},
      {"name":"satisfiedAt","type":"uint64"},
      {"name":"frozenAt","type":"uint64"},
      {"name":"status","type":"uint8"}
    ]
  },
  {
    "type":"function",
    "name":"latestEscapeCollateralBatchId",
    "stateMutability":"view",
    "inputs":[],
    "outputs":[{"name":"","type":"uint64"}]
  },
  {
    "type":"function",
    "name":"latestEscapeCollateralRoot",
    "stateMutability":"view",
    "inputs":[],
    "outputs":[{"name":"","type":"bytes32"}]
  },
  {
    "type":"function",
    "name":"batchDataPublished",
    "stateMutability":"view",
    "inputs":[{"name":"","type":"uint64"}],
    "outputs":[{"name":"","type":"bool"}]
  },
  {
    "type":"function",
    "name":"escapeCollateralRoots",
    "stateMutability":"view",
    "inputs":[{"name":"","type":"uint64"}],
    "outputs":[
      {"name":"merkleRoot","type":"bytes32"},
      {"name":"leafCount","type":"uint64"},
      {"name":"totalAmount","type":"uint256"},
      {"name":"anchoredAt","type":"uint64"}
    ]
  }
]`

var funnyRollupCoreReadABI = mustRollupCoreReadABI(funnyRollupCoreReadABIJSON)

type rollupCoreBatchMetadataState struct {
	LatestBatchID   uint64
	LatestStateRoot common.Hash
	BatchDataHash   common.Hash
	PrevStateRoot   common.Hash
	NextStateRoot   common.Hash
}

type rollupCoreAcceptedBatchState struct {
	LatestAcceptedBatchID   uint64
	LatestAcceptedStateRoot common.Hash
	FirstSequenceNo         uint64
	LastSequenceNo          uint64
	EntryCount              uint64
	BatchDataHash           common.Hash
	PrevStateRoot           common.Hash
	BalancesRoot            common.Hash
	OrdersRoot              common.Hash
	PositionsFundingRoot    common.Hash
	WithdrawalsRoot         common.Hash
	NextStateRoot           common.Hash
	AuthProofHash           common.Hash
	VerifierGateHash        common.Hash
}

type rollupCoreFreezeState struct {
	Frozen          bool
	FrozenAt        uint64
	FreezeRequestID uint64
}

type rollupCoreForcedWithdrawalRequestState struct {
	RequestID        uint64
	Wallet           common.Address
	Recipient        common.Address
	Amount           int64
	RequestedAt      uint64
	DeadlineAt       uint64
	SatisfiedClaimID common.Hash
	SatisfiedAt      uint64
	FrozenAt         uint64
	Status           uint8
}

type rollupCoreEscapeCollateralRootState struct {
	LatestEscapeCollateralBatchID uint64
	LatestEscapeCollateralRoot    common.Hash
	MerkleRoot                    common.Hash
	LeafCount                     uint64
	TotalAmount                   int64
	AnchoredAt                    uint64
}

func mustRollupCoreReadABI(raw string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(raw))
	if err != nil {
		panic(err)
	}
	return parsed
}

func (p *RollupSubmissionProcessor) loadRecordedBatchState(
	ctx context.Context,
	blockNumber *big.Int,
	batchID uint64,
) (rollupCoreBatchMetadataState, error) {
	latestBatchIDValues, err := p.callRollupCoreMethod(ctx, blockNumber, "latestBatchId")
	if err != nil {
		return rollupCoreBatchMetadataState{}, err
	}
	latestBatchID, err := decodeABIUint64Value(latestBatchIDValues[0], "latestBatchId")
	if err != nil {
		return rollupCoreBatchMetadataState{}, err
	}

	latestStateRootValues, err := p.callRollupCoreMethod(ctx, blockNumber, "latestStateRoot")
	if err != nil {
		return rollupCoreBatchMetadataState{}, err
	}
	latestStateRoot, err := decodeABIBytes32Value(latestStateRootValues[0], "latestStateRoot")
	if err != nil {
		return rollupCoreBatchMetadataState{}, err
	}

	batchMetadataValues, err := p.callRollupCoreMethod(ctx, blockNumber, "batchMetadata", batchID)
	if err != nil {
		return rollupCoreBatchMetadataState{}, err
	}
	batchDataHash, err := decodeABIBytes32Value(batchMetadataValues[0], "batchMetadata.batchDataHash")
	if err != nil {
		return rollupCoreBatchMetadataState{}, err
	}
	prevStateRoot, err := decodeABIBytes32Value(batchMetadataValues[1], "batchMetadata.prevStateRoot")
	if err != nil {
		return rollupCoreBatchMetadataState{}, err
	}
	nextStateRoot, err := decodeABIBytes32Value(batchMetadataValues[2], "batchMetadata.nextStateRoot")
	if err != nil {
		return rollupCoreBatchMetadataState{}, err
	}

	return rollupCoreBatchMetadataState{
		LatestBatchID:   latestBatchID,
		LatestStateRoot: latestStateRoot,
		BatchDataHash:   batchDataHash,
		PrevStateRoot:   prevStateRoot,
		NextStateRoot:   nextStateRoot,
	}, nil
}

func (p *RollupSubmissionProcessor) loadBatchDataPublishedState(
	ctx context.Context,
	blockNumber *big.Int,
	batchID uint64,
) (bool, error) {
	values, err := p.callRollupCoreMethod(ctx, blockNumber, "batchDataPublished", batchID)
	if err != nil {
		return false, err
	}
	return decodeABIBoolValue(values[0], "batchDataPublished")
}

func (p *RollupSubmissionProcessor) reconcilePublishDataState(
	ctx context.Context,
	submission rollup.StoredSubmission,
	receipt *types.Receipt,
) (bool, error) {
	if submission.BatchID <= 0 {
		return false, fmt.Errorf("submission batch_id must be positive")
	}
	published, err := p.loadBatchDataPublishedState(ctx, resolveReceiptBlockNumber(receipt), uint64(submission.BatchID))
	if err != nil {
		return false, err
	}
	return published, nil
}

func (p *RollupSubmissionProcessor) loadAcceptedBatchState(
	ctx context.Context,
	blockNumber *big.Int,
	batchID uint64,
) (rollupCoreAcceptedBatchState, error) {
	latestAcceptedBatchIDValues, err := p.callRollupCoreMethod(ctx, blockNumber, "latestAcceptedBatchId")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}
	latestAcceptedBatchID, err := decodeABIUint64Value(latestAcceptedBatchIDValues[0], "latestAcceptedBatchId")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}

	latestAcceptedStateRootValues, err := p.callRollupCoreMethod(ctx, blockNumber, "latestAcceptedStateRoot")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}
	latestAcceptedStateRoot, err := decodeABIBytes32Value(latestAcceptedStateRootValues[0], "latestAcceptedStateRoot")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}

	acceptedBatchValues, err := p.callRollupCoreMethod(ctx, blockNumber, "acceptedBatches", batchID)
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}

	firstSequenceNo, err := decodeABIUint64Value(acceptedBatchValues[0], "acceptedBatches.firstSequenceNo")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}
	lastSequenceNo, err := decodeABIUint64Value(acceptedBatchValues[1], "acceptedBatches.lastSequenceNo")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}
	entryCount, err := decodeABIUint64Value(acceptedBatchValues[2], "acceptedBatches.entryCount")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}
	batchDataHash, err := decodeABIBytes32Value(acceptedBatchValues[3], "acceptedBatches.batchDataHash")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}
	prevStateRoot, err := decodeABIBytes32Value(acceptedBatchValues[4], "acceptedBatches.prevStateRoot")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}
	balancesRoot, err := decodeABIBytes32Value(acceptedBatchValues[5], "acceptedBatches.balancesRoot")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}
	ordersRoot, err := decodeABIBytes32Value(acceptedBatchValues[6], "acceptedBatches.ordersRoot")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}
	positionsFundingRoot, err := decodeABIBytes32Value(acceptedBatchValues[7], "acceptedBatches.positionsFundingRoot")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}
	withdrawalsRoot, err := decodeABIBytes32Value(acceptedBatchValues[8], "acceptedBatches.withdrawalsRoot")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}
	nextStateRoot, err := decodeABIBytes32Value(acceptedBatchValues[9], "acceptedBatches.nextStateRoot")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}
	// index 10 is conservationHash (not used in reconciliation)
	authProofHash, err := decodeABIBytes32Value(acceptedBatchValues[11], "acceptedBatches.authProofHash")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}
	verifierGateHash, err := decodeABIBytes32Value(acceptedBatchValues[12], "acceptedBatches.verifierGateHash")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}

	return rollupCoreAcceptedBatchState{
		LatestAcceptedBatchID:   latestAcceptedBatchID,
		LatestAcceptedStateRoot: latestAcceptedStateRoot,
		FirstSequenceNo:         firstSequenceNo,
		LastSequenceNo:          lastSequenceNo,
		EntryCount:              entryCount,
		BatchDataHash:           batchDataHash,
		PrevStateRoot:           prevStateRoot,
		BalancesRoot:            balancesRoot,
		OrdersRoot:              ordersRoot,
		PositionsFundingRoot:    positionsFundingRoot,
		WithdrawalsRoot:         withdrawalsRoot,
		NextStateRoot:           nextStateRoot,
		AuthProofHash:           authProofHash,
		VerifierGateHash:        verifierGateHash,
	}, nil
}

func (p *RollupSubmissionProcessor) loadFreezeState(
	ctx context.Context,
	blockNumber *big.Int,
) (rollupCoreFreezeState, error) {
	frozenValues, err := p.callRollupCoreMethod(ctx, blockNumber, "frozen")
	if err != nil {
		return rollupCoreFreezeState{}, err
	}
	frozen, err := decodeABIBoolValue(frozenValues[0], "frozen")
	if err != nil {
		return rollupCoreFreezeState{}, err
	}

	frozenAtValues, err := p.callRollupCoreMethod(ctx, blockNumber, "frozenAt")
	if err != nil {
		return rollupCoreFreezeState{}, err
	}
	frozenAt, err := decodeABIUint64Value(frozenAtValues[0], "frozenAt")
	if err != nil {
		return rollupCoreFreezeState{}, err
	}

	freezeRequestIDValues, err := p.callRollupCoreMethod(ctx, blockNumber, "freezeRequestId")
	if err != nil {
		return rollupCoreFreezeState{}, err
	}
	freezeRequestID, err := decodeABIUint64Value(freezeRequestIDValues[0], "freezeRequestId")
	if err != nil {
		return rollupCoreFreezeState{}, err
	}

	return rollupCoreFreezeState{
		Frozen:          frozen,
		FrozenAt:        frozenAt,
		FreezeRequestID: freezeRequestID,
	}, nil
}

func (p *RollupSubmissionProcessor) loadEscapeCollateralRootState(
	ctx context.Context,
	blockNumber *big.Int,
	batchID uint64,
) (rollupCoreEscapeCollateralRootState, error) {
	latestBatchValues, err := p.callRollupCoreMethod(ctx, blockNumber, "latestEscapeCollateralBatchId")
	if err != nil {
		return rollupCoreEscapeCollateralRootState{}, err
	}
	latestBatchID, err := decodeABIUint64Value(latestBatchValues[0], "latestEscapeCollateralBatchId")
	if err != nil {
		return rollupCoreEscapeCollateralRootState{}, err
	}

	latestRootValues, err := p.callRollupCoreMethod(ctx, blockNumber, "latestEscapeCollateralRoot")
	if err != nil {
		return rollupCoreEscapeCollateralRootState{}, err
	}
	latestRoot, err := decodeABIBytes32Value(latestRootValues[0], "latestEscapeCollateralRoot")
	if err != nil {
		return rollupCoreEscapeCollateralRootState{}, err
	}

	rootValues, err := p.callRollupCoreMethod(ctx, blockNumber, "escapeCollateralRoots", batchID)
	if err != nil {
		return rollupCoreEscapeCollateralRootState{}, err
	}
	merkleRoot, err := decodeABIBytes32Value(rootValues[0], "escapeCollateralRoots.merkleRoot")
	if err != nil {
		return rollupCoreEscapeCollateralRootState{}, err
	}
	leafCount, err := decodeABIUint64Value(rootValues[1], "escapeCollateralRoots.leafCount")
	if err != nil {
		return rollupCoreEscapeCollateralRootState{}, err
	}
	totalAmount, err := decodeABIInt64Value(rootValues[2], "escapeCollateralRoots.totalAmount")
	if err != nil {
		return rollupCoreEscapeCollateralRootState{}, err
	}
	anchoredAt, err := decodeABIUint64Value(rootValues[3], "escapeCollateralRoots.anchoredAt")
	if err != nil {
		return rollupCoreEscapeCollateralRootState{}, err
	}

	return rollupCoreEscapeCollateralRootState{
		LatestEscapeCollateralBatchID: latestBatchID,
		LatestEscapeCollateralRoot:    latestRoot,
		MerkleRoot:                    merkleRoot,
		LeafCount:                     leafCount,
		TotalAmount:                   totalAmount,
		AnchoredAt:                    anchoredAt,
	}, nil
}

func (p *RollupSubmissionProcessor) loadForcedWithdrawalRequestCount(
	ctx context.Context,
	blockNumber *big.Int,
) (uint64, error) {
	values, err := p.callRollupCoreMethod(ctx, blockNumber, "forcedWithdrawalRequestCount")
	if err != nil {
		return 0, err
	}
	return decodeABIUint64Value(values[0], "forcedWithdrawalRequestCount")
}

func (p *RollupSubmissionProcessor) loadForcedWithdrawalRequest(
	ctx context.Context,
	blockNumber *big.Int,
	requestID uint64,
) (rollupCoreForcedWithdrawalRequestState, error) {
	values, err := p.callRollupCoreMethod(ctx, blockNumber, "forcedWithdrawalRequests", requestID)
	if err != nil {
		return rollupCoreForcedWithdrawalRequestState{}, err
	}
	wallet, err := decodeABIAddressValue(values[0], "forcedWithdrawalRequests.wallet")
	if err != nil {
		return rollupCoreForcedWithdrawalRequestState{}, err
	}
	recipient, err := decodeABIAddressValue(values[1], "forcedWithdrawalRequests.recipient")
	if err != nil {
		return rollupCoreForcedWithdrawalRequestState{}, err
	}
	amount, err := decodeABIInt64FromUintValue(values[2], "forcedWithdrawalRequests.amount")
	if err != nil {
		return rollupCoreForcedWithdrawalRequestState{}, err
	}
	requestedAt, err := decodeABIUint64Value(values[3], "forcedWithdrawalRequests.requestedAt")
	if err != nil {
		return rollupCoreForcedWithdrawalRequestState{}, err
	}
	deadlineAt, err := decodeABIUint64Value(values[4], "forcedWithdrawalRequests.deadlineAt")
	if err != nil {
		return rollupCoreForcedWithdrawalRequestState{}, err
	}
	satisfiedClaimID, err := decodeABIBytes32Value(values[5], "forcedWithdrawalRequests.satisfiedClaimId")
	if err != nil {
		return rollupCoreForcedWithdrawalRequestState{}, err
	}
	satisfiedAt, err := decodeABIUint64Value(values[6], "forcedWithdrawalRequests.satisfiedAt")
	if err != nil {
		return rollupCoreForcedWithdrawalRequestState{}, err
	}
	frozenAt, err := decodeABIUint64Value(values[7], "forcedWithdrawalRequests.frozenAt")
	if err != nil {
		return rollupCoreForcedWithdrawalRequestState{}, err
	}
	status, err := decodeABIUint8Value(values[8], "forcedWithdrawalRequests.status")
	if err != nil {
		return rollupCoreForcedWithdrawalRequestState{}, err
	}

	return rollupCoreForcedWithdrawalRequestState{
		RequestID:        requestID,
		Wallet:           wallet,
		Recipient:        recipient,
		Amount:           amount,
		RequestedAt:      requestedAt,
		DeadlineAt:       deadlineAt,
		SatisfiedClaimID: satisfiedClaimID,
		SatisfiedAt:      satisfiedAt,
		FrozenAt:         frozenAt,
		Status:           status,
	}, nil
}

func (p *RollupSubmissionProcessor) callRollupCoreMethod(
	ctx context.Context,
	blockNumber *big.Int,
	method string,
	args ...any,
) ([]any, error) {
	encoded, err := funnyRollupCoreReadABI.Pack(method, args...)
	if err != nil {
		return nil, fmt.Errorf("pack FunnyRollupCore.%s call: %w", method, err)
	}
	raw, err := p.sender.CallContract(ctx, ethereum.CallMsg{
		From: p.fromAddress,
		To:   &p.rollupCore,
		Data: encoded,
	}, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("call FunnyRollupCore.%s: %w", method, err)
	}
	values, err := funnyRollupCoreReadABI.Unpack(method, raw)
	if err != nil {
		return nil, fmt.Errorf("unpack FunnyRollupCore.%s: %w", method, err)
	}
	return values, nil
}

func decodeABIUint64Value(value any, field string) (uint64, error) {
	switch typed := value.(type) {
	case uint8:
		return uint64(typed), nil
	case uint16:
		return uint64(typed), nil
	case uint32:
		return uint64(typed), nil
	case uint64:
		return typed, nil
	case *big.Int:
		if typed.Sign() < 0 || !typed.IsUint64() {
			return 0, fmt.Errorf("%s must fit uint64", field)
		}
		return typed.Uint64(), nil
	case big.Int:
		if typed.Sign() < 0 || !typed.IsUint64() {
			return 0, fmt.Errorf("%s must fit uint64", field)
		}
		return typed.Uint64(), nil
	default:
		return 0, fmt.Errorf("%s returned unsupported uint type %T", field, value)
	}
}

func decodeABIUint8Value(value any, field string) (uint8, error) {
	switch typed := value.(type) {
	case uint8:
		return typed, nil
	case uint16:
		if typed > 0xff {
			return 0, fmt.Errorf("%s must fit uint8", field)
		}
		return uint8(typed), nil
	case uint32:
		if typed > 0xff {
			return 0, fmt.Errorf("%s must fit uint8", field)
		}
		return uint8(typed), nil
	case uint64:
		if typed > 0xff {
			return 0, fmt.Errorf("%s must fit uint8", field)
		}
		return uint8(typed), nil
	case *big.Int:
		if typed.Sign() < 0 || !typed.IsUint64() || typed.Uint64() > 0xff {
			return 0, fmt.Errorf("%s must fit uint8", field)
		}
		return uint8(typed.Uint64()), nil
	case big.Int:
		if typed.Sign() < 0 || !typed.IsUint64() || typed.Uint64() > 0xff {
			return 0, fmt.Errorf("%s must fit uint8", field)
		}
		return uint8(typed.Uint64()), nil
	default:
		return 0, fmt.Errorf("%s returned unsupported uint8 type %T", field, value)
	}
}

func decodeABIInt64FromUintValue(value any, field string) (int64, error) {
	switch typed := value.(type) {
	case *big.Int:
		if typed.Sign() < 0 || !typed.IsInt64() {
			return 0, fmt.Errorf("%s must fit int64", field)
		}
		return typed.Int64(), nil
	case big.Int:
		if typed.Sign() < 0 || !typed.IsInt64() {
			return 0, fmt.Errorf("%s must fit int64", field)
		}
		return typed.Int64(), nil
	case uint64:
		if typed > uint64(^uint64(0)>>1) {
			return 0, fmt.Errorf("%s must fit int64", field)
		}
		return int64(typed), nil
	default:
		return 0, fmt.Errorf("%s returned unsupported uint256 type %T", field, value)
	}
}

func decodeABIInt64Value(value any, field string) (int64, error) {
	return decodeABIInt64FromUintValue(value, field)
}

func decodeABIBoolValue(value any, field string) (bool, error) {
	typed, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%s returned unsupported bool type %T", field, value)
	}
	return typed, nil
}

func decodeABIAddressValue(value any, field string) (common.Address, error) {
	switch typed := value.(type) {
	case common.Address:
		return typed, nil
	case [20]byte:
		return common.BytesToAddress(typed[:]), nil
	case []byte:
		if len(typed) != 20 {
			return common.Address{}, fmt.Errorf("%s must be 20 bytes, got %d", field, len(typed))
		}
		return common.BytesToAddress(typed), nil
	default:
		return common.Address{}, fmt.Errorf("%s returned unsupported address type %T", field, value)
	}
}

func decodeABIBytes32Value(value any, field string) (common.Hash, error) {
	switch typed := value.(type) {
	case common.Hash:
		return typed, nil
	case [32]byte:
		return common.BytesToHash(typed[:]), nil
	case []byte:
		if len(typed) != 32 {
			return common.Hash{}, fmt.Errorf("%s must be 32 bytes, got %d", field, len(typed))
		}
		return common.BytesToHash(typed), nil
	default:
		return common.Hash{}, fmt.Errorf("%s returned unsupported bytes32 type %T", field, value)
	}
}

func parseExpectedHash(value, field string) (common.Hash, error) {
	normalized := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "0x")
	if len(normalized) != 64 {
		return common.Hash{}, fmt.Errorf("%s must be 32-byte hex", field)
	}
	if _, err := hex.DecodeString(normalized); err != nil {
		return common.Hash{}, fmt.Errorf("%s must be 32-byte hex: %w", field, err)
	}
	return common.HexToHash("0x" + normalized), nil
}
