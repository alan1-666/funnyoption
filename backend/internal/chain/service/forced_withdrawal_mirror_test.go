package service

import (
	"context"
	"math/big"
	"testing"

	chainmodel "funnyoption/internal/chain/model"
	"funnyoption/internal/shared/config"

	"github.com/ethereum/go-ethereum/common"
)

type fakeForcedWithdrawalMirrorStore struct {
	requests []chainmodel.RollupForcedWithdrawalRequest
	freeze   chainmodel.RollupFreezeState
}

func (f *fakeForcedWithdrawalMirrorStore) UpsertRollupForcedWithdrawalRequest(
	ctx context.Context,
	request chainmodel.RollupForcedWithdrawalRequest,
) error {
	_ = ctx
	for index := range f.requests {
		if f.requests[index].RequestID == request.RequestID {
			f.requests[index] = request
			return nil
		}
	}
	f.requests = append(f.requests, request)
	return nil
}

func (f *fakeForcedWithdrawalMirrorStore) UpsertRollupFreezeState(
	ctx context.Context,
	state chainmodel.RollupFreezeState,
) error {
	_ = ctx
	f.freeze = state
	return nil
}

func TestForcedWithdrawalMirrorProcessorPollOnceMirrorsCoreState(t *testing.T) {
	store := &fakeForcedWithdrawalMirrorStore{}
	sender := &fakeRollupTxSender{
		callResults: map[string][]byte{
			mustPackRollupCoreCallData(t, "frozen"):                       mustPackRollupCoreCallOutput(t, "frozen", true),
			mustPackRollupCoreCallData(t, "frozenAt"):                     mustPackRollupCoreCallOutput(t, "frozenAt", uint64(1234)),
			mustPackRollupCoreCallData(t, "freezeRequestId"):              mustPackRollupCoreCallOutput(t, "freezeRequestId", uint64(2)),
			mustPackRollupCoreCallData(t, "forcedWithdrawalRequestCount"): mustPackRollupCoreCallOutput(t, "forcedWithdrawalRequestCount", uint64(2)),
			mustPackRollupCoreCallData(t, "forcedWithdrawalRequests", uint64(1)): mustPackRollupCoreCallOutput(
				t,
				"forcedWithdrawalRequests",
				common.HexToAddress("0x00000000000000000000000000000000000000a1"),
				common.HexToAddress("0x00000000000000000000000000000000000000b1"),
				big.NewInt(700),
				uint64(100),
				uint64(200),
				common.Hash{},
				uint64(0),
				uint64(0),
				uint8(1),
			),
			mustPackRollupCoreCallData(t, "forcedWithdrawalRequests", uint64(2)): mustPackRollupCoreCallOutput(
				t,
				"forcedWithdrawalRequests",
				common.HexToAddress("0x00000000000000000000000000000000000000a2"),
				common.HexToAddress("0x00000000000000000000000000000000000000b2"),
				big.NewInt(900),
				uint64(110),
				uint64(210),
				common.HexToHash("0xabc"),
				uint64(220),
				uint64(1234),
				uint8(3),
			),
		},
	}
	processor, err := NewForcedWithdrawalMirrorProcessor(nil, config.ServiceConfig{
		RollupCoreAddress:  "0x00000000000000000000000000000000000000cc",
		RollupPollInterval: 2,
	}, store, sender)
	if err != nil {
		t.Fatalf("NewForcedWithdrawalMirrorProcessor returned error: %v", err)
	}

	progress, err := processor.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("PollOnce returned error: %v", err)
	}
	if progress.RequestCount != 2 {
		t.Fatalf("request_count = %d, want 2", progress.RequestCount)
	}
	if !progress.Frozen {
		t.Fatalf("expected frozen state")
	}
	if progress.FreezeRequestID != 2 {
		t.Fatalf("freeze_request_id = %d, want 2", progress.FreezeRequestID)
	}
	if !store.freeze.Frozen || store.freeze.RequestID != 2 || store.freeze.FrozenAt != 1234 {
		t.Fatalf("unexpected freeze mirror: %+v", store.freeze)
	}
	if len(store.requests) != 2 {
		t.Fatalf("request mirror count = %d, want 2", len(store.requests))
	}
	if store.requests[0].Status != forcedWithdrawalStatusRequested {
		t.Fatalf("request[0].status = %s, want %s", store.requests[0].Status, forcedWithdrawalStatusRequested)
	}
	if store.requests[1].Status != forcedWithdrawalStatusFrozen {
		t.Fatalf("request[1].status = %s, want %s", store.requests[1].Status, forcedWithdrawalStatusFrozen)
	}
	if store.requests[1].SatisfiedClaimID != normalizeChainTxHash(common.HexToHash("0xabc").Hex()) {
		t.Fatalf("request[1].satisfied_claim_id = %s", store.requests[1].SatisfiedClaimID)
	}
}
