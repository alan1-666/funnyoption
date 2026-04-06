package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
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
      {"name":"authProofHash","type":"bytes32"},
      {"name":"verifierGateHash","type":"bytes32"}
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
	authProofHash, err := decodeABIBytes32Value(acceptedBatchValues[10], "acceptedBatches.authProofHash")
	if err != nil {
		return rollupCoreAcceptedBatchState{}, err
	}
	verifierGateHash, err := decodeABIBytes32Value(acceptedBatchValues[11], "acceptedBatches.verifierGateHash")
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
