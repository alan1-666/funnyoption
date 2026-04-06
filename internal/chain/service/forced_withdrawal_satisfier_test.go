package service

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	chainmodel "funnyoption/internal/chain/model"
	"funnyoption/internal/shared/config"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type fakeForcedWithdrawalSatisfierStore struct {
	requests map[int64]chainmodel.RollupForcedWithdrawalRequest
	matches  map[int64][]chainmodel.ForcedWithdrawalClaimMatch
}

func (f *fakeForcedWithdrawalSatisfierStore) ListPendingRollupForcedWithdrawalRequests(
	ctx context.Context,
	limit int,
) ([]chainmodel.RollupForcedWithdrawalRequest, error) {
	_ = ctx
	items := make([]chainmodel.RollupForcedWithdrawalRequest, 0, limit)
	for requestID := int64(1); requestID <= int64(len(f.requests)); requestID++ {
		if request, ok := f.requests[requestID]; ok && request.Status == forcedWithdrawalStatusRequested {
			items = append(items, request)
		}
	}
	return items, nil
}

func (f *fakeForcedWithdrawalSatisfierStore) ListForcedWithdrawalClaimMatches(
	ctx context.Context,
	requestID int64,
	limit int,
) ([]chainmodel.ForcedWithdrawalClaimMatch, error) {
	_ = ctx
	items := append([]chainmodel.ForcedWithdrawalClaimMatch(nil), f.matches[requestID]...)
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (f *fakeForcedWithdrawalSatisfierStore) UpdateRollupForcedWithdrawalMatch(
	ctx context.Context,
	requestID int64,
	withdrawalID string,
	claimID string,
	status string,
	errMsg string,
) error {
	_ = ctx
	request := f.requests[requestID]
	request.MatchedWithdrawalID = withdrawalID
	request.MatchedClaimID = claimID
	request.SatisfactionStatus = status
	request.SatisfactionLastError = errMsg
	if status == chainmodel.ForcedWithdrawalSatisfactionStatusNone || status == chainmodel.ForcedWithdrawalSatisfactionStatusReady || status == chainmodel.ForcedWithdrawalSatisfactionStatusAmbiguous {
		request.SatisfactionTxHash = ""
		request.SatisfactionSubmittedAt = 0
	}
	f.requests[requestID] = request
	return nil
}

func (f *fakeForcedWithdrawalSatisfierStore) MarkRollupForcedWithdrawalSatisfactionSubmitted(
	ctx context.Context,
	requestID int64,
	txHash string,
) error {
	_ = ctx
	request := f.requests[requestID]
	request.SatisfactionStatus = chainmodel.ForcedWithdrawalSatisfactionStatusSubmitted
	request.SatisfactionTxHash = txHash
	request.SatisfactionSubmittedAt = 123
	request.SatisfactionLastError = ""
	f.requests[requestID] = request
	return nil
}

func (f *fakeForcedWithdrawalSatisfierStore) MarkRollupForcedWithdrawalSatisfactionFailed(
	ctx context.Context,
	requestID int64,
	errMsg string,
) error {
	_ = ctx
	request := f.requests[requestID]
	request.SatisfactionStatus = chainmodel.ForcedWithdrawalSatisfactionStatusFailed
	request.SatisfactionLastError = errMsg
	f.requests[requestID] = request
	return nil
}

func (f *fakeForcedWithdrawalSatisfierStore) UpsertRollupForcedWithdrawalRequest(
	ctx context.Context,
	request chainmodel.RollupForcedWithdrawalRequest,
) error {
	_ = ctx
	current := f.requests[request.RequestID]
	if request.MatchedWithdrawalID == "" {
		request.MatchedWithdrawalID = current.MatchedWithdrawalID
	}
	if request.MatchedClaimID == "" {
		request.MatchedClaimID = current.MatchedClaimID
	}
	if request.SatisfactionTxHash == "" {
		request.SatisfactionTxHash = current.SatisfactionTxHash
	}
	if request.SatisfactionSubmittedAt == 0 {
		request.SatisfactionSubmittedAt = current.SatisfactionSubmittedAt
	}
	if request.SatisfactionLastError == "" {
		request.SatisfactionLastError = current.SatisfactionLastError
	}
	if request.SatisfactionStatus == "" {
		request.SatisfactionStatus = current.SatisfactionStatus
	}
	if request.Status == forcedWithdrawalStatusSatisfied {
		request.SatisfactionStatus = chainmodel.ForcedWithdrawalSatisfactionStatusSatisfied
		if request.SatisfiedClaimID != "" {
			request.MatchedClaimID = request.SatisfiedClaimID
		}
	}
	f.requests[request.RequestID] = request
	return nil
}

func TestForcedWithdrawalSatisfierPollOnceSubmitsMatch(t *testing.T) {
	key := mustGenerateForcedWithdrawalSatisfierKey(t)
	store := &fakeForcedWithdrawalSatisfierStore{
		requests: map[int64]chainmodel.RollupForcedWithdrawalRequest{
			1: {
				RequestID:          1,
				WalletAddress:      "0xabc",
				RecipientAddress:   "0xabc",
				Amount:             123,
				Status:             forcedWithdrawalStatusRequested,
				SatisfactionStatus: chainmodel.ForcedWithdrawalSatisfactionStatusNone,
			},
		},
		matches: map[int64][]chainmodel.ForcedWithdrawalClaimMatch{
			1: {{
				WithdrawalID: "wd_1",
				ClaimID:      "0x00000000000000000000000000000000000000000000000000000000000000ab",
				Amount:       123,
				ClaimedAt:    100,
			}},
		},
	}
	sender := &fakeRollupTxSender{
		nonce:    9,
		chainID:  big.NewInt(31337),
		gasPrice: big.NewInt(1),
		estimate: 90000,
		callResults: map[string][]byte{
			mustPackRollupCoreCallData(t, "frozen"):          mustPackRollupCoreCallOutput(t, "frozen", false),
			mustPackRollupCoreCallData(t, "frozenAt"):        mustPackRollupCoreCallOutput(t, "frozenAt", uint64(0)),
			mustPackRollupCoreCallData(t, "freezeRequestId"): mustPackRollupCoreCallOutput(t, "freezeRequestId", uint64(0)),
		},
		receipts: map[string]*types.Receipt{},
	}
	processor := newTestForcedWithdrawalSatisfier(t, key, store, sender)

	progress, err := processor.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("PollOnce returned error: %v", err)
	}
	if progress.Action != ForcedWithdrawalSatisfyActionSubmitted {
		t.Fatalf("action = %s, want %s", progress.Action, ForcedWithdrawalSatisfyActionSubmitted)
	}
	request := store.requests[1]
	if request.SatisfactionStatus != chainmodel.ForcedWithdrawalSatisfactionStatusSubmitted {
		t.Fatalf("satisfaction_status = %s, want SUBMITTED", request.SatisfactionStatus)
	}
	if request.MatchedClaimID != "0x00000000000000000000000000000000000000000000000000000000000000ab" {
		t.Fatalf("matched_claim_id = %s", request.MatchedClaimID)
	}
	if len(sender.sentTxs) != 1 {
		t.Fatalf("sent tx count = %d, want 1", len(sender.sentTxs))
	}
}

func TestForcedWithdrawalSatisfierPollOnceMarksSatisfiedAfterReceipt(t *testing.T) {
	key := mustGenerateForcedWithdrawalSatisfierKey(t)
	store := &fakeForcedWithdrawalSatisfierStore{
		requests: map[int64]chainmodel.RollupForcedWithdrawalRequest{
			1: {
				RequestID:               1,
				WalletAddress:           "0x1532d37232c783c531bf0ce9860cb15f5f68aeb3",
				RecipientAddress:        "0x1532d37232c783c531bf0ce9860cb15f5f68aeb3",
				Amount:                  123456,
				Status:                  forcedWithdrawalStatusRequested,
				MatchedWithdrawalID:     "wd_1",
				MatchedClaimID:          "0x00000000000000000000000000000000000000000000000000000000000000cd",
				SatisfactionStatus:      chainmodel.ForcedWithdrawalSatisfactionStatusSubmitted,
				SatisfactionTxHash:      "0x1111111111111111111111111111111111111111111111111111111111111111",
				SatisfactionSubmittedAt: 100,
			},
		},
	}
	sender := &fakeRollupTxSender{
		nonce:    9,
		chainID:  big.NewInt(31337),
		gasPrice: big.NewInt(1),
		estimate: 90000,
		callResults: map[string][]byte{
			mustPackRollupCoreCallData(t, "frozen"):          mustPackRollupCoreCallOutput(t, "frozen", false),
			mustPackRollupCoreCallData(t, "frozenAt"):        mustPackRollupCoreCallOutput(t, "frozenAt", uint64(0)),
			mustPackRollupCoreCallData(t, "freezeRequestId"): mustPackRollupCoreCallOutput(t, "freezeRequestId", uint64(0)),
			mustPackRollupCoreCallData(t, "forcedWithdrawalRequests", uint64(1)): mustPackRollupCoreCallOutput(
				t,
				"forcedWithdrawalRequests",
				common.HexToAddress("0x1532d37232c783c531bf0ce9860cb15f5f68aeb3"),
				common.HexToAddress("0x1532d37232c783c531bf0ce9860cb15f5f68aeb3"),
				big.NewInt(123456),
				uint64(10),
				uint64(20),
				common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000cd"),
				uint64(30),
				uint64(0),
				uint8(2),
			),
		},
		receiptFn: func(txHash common.Hash) (*types.Receipt, error) {
			_ = txHash
			return &types.Receipt{
				Status:      types.ReceiptStatusSuccessful,
				BlockNumber: big.NewInt(99),
			}, nil
		},
	}
	processor := newTestForcedWithdrawalSatisfier(t, key, store, sender)

	progress, err := processor.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("PollOnce returned error: %v", err)
	}
	if progress.Action != ForcedWithdrawalSatisfyActionSatisfied {
		t.Fatalf("action = %s, want %s note=%s", progress.Action, ForcedWithdrawalSatisfyActionSatisfied, progress.Note)
	}
	if store.requests[1].Status != forcedWithdrawalStatusSatisfied {
		t.Fatalf("status = %s, want SATISFIED", store.requests[1].Status)
	}
	if store.requests[1].SatisfactionStatus != chainmodel.ForcedWithdrawalSatisfactionStatusSatisfied {
		t.Fatalf("satisfaction_status = %s, want SATISFIED", store.requests[1].SatisfactionStatus)
	}
}

func TestForcedWithdrawalSatisfierPollOnceMarksAmbiguous(t *testing.T) {
	key := mustGenerateForcedWithdrawalSatisfierKey(t)
	store := &fakeForcedWithdrawalSatisfierStore{
		requests: map[int64]chainmodel.RollupForcedWithdrawalRequest{
			1: {
				RequestID:          1,
				Amount:             1,
				Status:             forcedWithdrawalStatusRequested,
				SatisfactionStatus: chainmodel.ForcedWithdrawalSatisfactionStatusNone,
			},
		},
		matches: map[int64][]chainmodel.ForcedWithdrawalClaimMatch{
			1: {
				{WithdrawalID: "wd_1", ClaimID: "0x1", Amount: 1},
				{WithdrawalID: "wd_2", ClaimID: "0x2", Amount: 1},
			},
		},
	}
	sender := &fakeRollupTxSender{
		callResults: map[string][]byte{
			mustPackRollupCoreCallData(t, "frozen"):          mustPackRollupCoreCallOutput(t, "frozen", false),
			mustPackRollupCoreCallData(t, "frozenAt"):        mustPackRollupCoreCallOutput(t, "frozenAt", uint64(0)),
			mustPackRollupCoreCallData(t, "freezeRequestId"): mustPackRollupCoreCallOutput(t, "freezeRequestId", uint64(0)),
		},
	}
	processor := newTestForcedWithdrawalSatisfier(t, key, store, sender)

	progress, err := processor.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("PollOnce returned error: %v", err)
	}
	if progress.Action != ForcedWithdrawalSatisfyActionAmbiguous {
		t.Fatalf("action = %s, want %s", progress.Action, ForcedWithdrawalSatisfyActionAmbiguous)
	}
	if store.requests[1].SatisfactionStatus != chainmodel.ForcedWithdrawalSatisfactionStatusAmbiguous {
		t.Fatalf("satisfaction_status = %s, want AMBIGUOUS", store.requests[1].SatisfactionStatus)
	}
}

func newTestForcedWithdrawalSatisfier(
	t *testing.T,
	key *ecdsa.PrivateKey,
	store forcedWithdrawalSatisfierStore,
	sender *fakeRollupTxSender,
) *ForcedWithdrawalSatisfier {
	t.Helper()
	processor, err := NewForcedWithdrawalSatisfier(nil, config.ServiceConfig{
		ChainOperatorPrivateKey: common.Bytes2Hex(crypto.FromECDSA(key)),
		RollupCoreAddress:       "0x00000000000000000000000000000000000000cc",
		RollupPollInterval:      time.Second,
		ChainGasLimit:           250000,
	}, store, sender)
	if err != nil {
		t.Fatalf("NewForcedWithdrawalSatisfier returned error: %v", err)
	}
	return processor
}

func mustGenerateForcedWithdrawalSatisfierKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	return key
}
