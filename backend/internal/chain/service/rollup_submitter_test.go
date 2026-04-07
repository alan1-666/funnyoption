package service

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"strings"
	"testing"
	"time"

	"funnyoption/internal/rollup"
	"funnyoption/internal/shared/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type fakeRollupSubmissionStore struct {
	submissions          []rollup.StoredSubmission
	prepared             rollup.PreparedShadowSubmission
	prepareErr           error
	materialized         []rollup.AcceptedSubmissionMaterialization
	materializeErr       error
	materializedByIDCall []string
	frozen               bool
	escapeRoots          []rollup.AcceptedEscapeCollateralRootRecord
}

func (f *fakeRollupSubmissionStore) ListSubmissions(ctx context.Context) ([]rollup.StoredSubmission, error) {
	_ = ctx
	result := make([]rollup.StoredSubmission, len(f.submissions))
	copy(result, f.submissions)
	return result, nil
}

func (f *fakeRollupSubmissionStore) PrepareNextSubmission(ctx context.Context, limit int) (rollup.PreparedShadowSubmission, error) {
	_ = ctx
	_ = limit
	if f.prepareErr != nil {
		return rollup.PreparedShadowSubmission{}, f.prepareErr
	}
	if f.prepared.StoredSubmission.SubmissionID == "" {
		return rollup.PreparedShadowSubmission{}, rollup.ErrNoPendingSubmission
	}
	f.upsert(f.prepared.StoredSubmission)
	return f.prepared, nil
}

func (f *fakeRollupSubmissionStore) MaterializeAcceptedSubmissions(ctx context.Context) ([]rollup.AcceptedSubmissionMaterialization, error) {
	_ = ctx
	if f.materializeErr != nil {
		return nil, f.materializeErr
	}
	result := make([]rollup.AcceptedSubmissionMaterialization, len(f.materialized))
	copy(result, f.materialized)
	return result, nil
}

func (f *fakeRollupSubmissionStore) MaterializeAcceptedSubmission(ctx context.Context, submissionID string) (rollup.AcceptedSubmissionMaterialization, error) {
	_ = ctx
	f.materializedByIDCall = append(f.materializedByIDCall, submissionID)
	if f.materializeErr != nil {
		return rollup.AcceptedSubmissionMaterialization{}, f.materializeErr
	}
	item := rollup.AcceptedSubmissionMaterialization{}
	for _, submission := range f.submissions {
		if submission.SubmissionID == submissionID {
			item.Batch.SubmissionID = submissionID
			item.Batch.BatchID = submission.BatchID
			break
		}
	}
	f.materialized = append(f.materialized, item)
	return item, nil
}

func (f *fakeRollupSubmissionStore) RollupFrozen(ctx context.Context) (bool, error) {
	_ = ctx
	return f.frozen, nil
}

func (f *fakeRollupSubmissionStore) NextEscapeCollateralRootForAnchor(ctx context.Context) (rollup.AcceptedEscapeCollateralRootRecord, bool, error) {
	_ = ctx
	for _, root := range f.escapeRoots {
		switch root.AnchorStatus {
		case rollup.EscapeCollateralAnchorStatusReady, rollup.EscapeCollateralAnchorStatusSubmitted, rollup.EscapeCollateralAnchorStatusFailed:
			return root, true, nil
		}
	}
	return rollup.AcceptedEscapeCollateralRootRecord{}, false, nil
}

func (f *fakeRollupSubmissionStore) MarkEscapeCollateralRootSubmitted(ctx context.Context, batchID int64, txHash string) (rollup.AcceptedEscapeCollateralRootRecord, error) {
	_ = ctx
	for i := range f.escapeRoots {
		if f.escapeRoots[i].BatchID == batchID {
			f.escapeRoots[i].AnchorStatus = rollup.EscapeCollateralAnchorStatusSubmitted
			f.escapeRoots[i].AnchorTxHash = txHash
			f.escapeRoots[i].LastError = ""
			f.escapeRoots[i].LastErrorAt = 0
			return f.escapeRoots[i], nil
		}
	}
	return rollup.AcceptedEscapeCollateralRootRecord{}, errors.New("escape root not found")
}

func (f *fakeRollupSubmissionStore) MarkEscapeCollateralRootAnchored(ctx context.Context, batchID int64) (rollup.AcceptedEscapeCollateralRootRecord, error) {
	_ = ctx
	for i := range f.escapeRoots {
		if f.escapeRoots[i].BatchID == batchID {
			f.escapeRoots[i].AnchorStatus = rollup.EscapeCollateralAnchorStatusAnchored
			return f.escapeRoots[i], nil
		}
	}
	return rollup.AcceptedEscapeCollateralRootRecord{}, errors.New("escape root not found")
}

func (f *fakeRollupSubmissionStore) MarkEscapeCollateralRootFailed(ctx context.Context, batchID int64, errMsg string) (rollup.AcceptedEscapeCollateralRootRecord, error) {
	_ = ctx
	for i := range f.escapeRoots {
		if f.escapeRoots[i].BatchID == batchID {
			f.escapeRoots[i].AnchorStatus = rollup.EscapeCollateralAnchorStatusFailed
			f.escapeRoots[i].LastError = errMsg
			return f.escapeRoots[i], nil
		}
	}
	return rollup.AcceptedEscapeCollateralRootRecord{}, errors.New("escape root not found")
}

func (f *fakeRollupSubmissionStore) MarkSubmissionPublishSubmitted(ctx context.Context, submissionID, txHash string) (rollup.StoredSubmission, error) {
	_ = ctx
	return f.update(submissionID, func(item *rollup.StoredSubmission) {
		item.Status = rollup.SubmissionStatusPublishSubmitted
		item.PublishTxHash = txHash
		item.LastError = ""
		item.LastErrorAt = 0
	})
}

func (f *fakeRollupSubmissionStore) MarkSubmissionDataPublished(ctx context.Context, submissionID string) (rollup.StoredSubmission, error) {
	_ = ctx
	return f.update(submissionID, func(item *rollup.StoredSubmission) {
		item.Status = rollup.SubmissionStatusDataPublished
		item.LastError = ""
		item.LastErrorAt = 0
	})
}

func (f *fakeRollupSubmissionStore) MarkSubmissionRecordSubmitted(ctx context.Context, submissionID, txHash string) (rollup.StoredSubmission, error) {
	_ = ctx
	return f.update(submissionID, func(item *rollup.StoredSubmission) {
		item.Status = rollup.SubmissionStatusRecordSubmitted
		item.RecordTxHash = txHash
		item.LastError = ""
		item.LastErrorAt = 0
	})
}

func (f *fakeRollupSubmissionStore) MarkSubmissionAcceptSubmitted(ctx context.Context, submissionID, txHash string) (rollup.StoredSubmission, error) {
	_ = ctx
	return f.update(submissionID, func(item *rollup.StoredSubmission) {
		item.Status = rollup.SubmissionStatusAcceptSubmitted
		item.AcceptTxHash = txHash
		item.LastError = ""
		item.LastErrorAt = 0
	})
}

func (f *fakeRollupSubmissionStore) MarkSubmissionAccepted(ctx context.Context, submissionID string) (rollup.StoredSubmission, error) {
	_ = ctx
	return f.update(submissionID, func(item *rollup.StoredSubmission) {
		item.Status = rollup.SubmissionStatusAccepted
		item.LastError = ""
		item.LastErrorAt = 0
	})
}

func (f *fakeRollupSubmissionStore) MarkSubmissionFailed(ctx context.Context, submissionID, errMsg string) (rollup.StoredSubmission, error) {
	_ = ctx
	return f.update(submissionID, func(item *rollup.StoredSubmission) {
		item.Status = rollup.SubmissionStatusFailed
		item.LastError = errMsg
	})
}

func (f *fakeRollupSubmissionStore) RecordSubmissionError(ctx context.Context, submissionID, errMsg string) (rollup.StoredSubmission, error) {
	_ = ctx
	return f.update(submissionID, func(item *rollup.StoredSubmission) {
		item.LastError = errMsg
	})
}

func (f *fakeRollupSubmissionStore) upsert(submission rollup.StoredSubmission) {
	for index := range f.submissions {
		if f.submissions[index].SubmissionID == submission.SubmissionID {
			f.submissions[index] = submission
			return
		}
	}
	f.submissions = append(f.submissions, submission)
}

func (f *fakeRollupSubmissionStore) update(
	submissionID string,
	mutator func(item *rollup.StoredSubmission),
) (rollup.StoredSubmission, error) {
	for index := range f.submissions {
		if f.submissions[index].SubmissionID == submissionID {
			mutator(&f.submissions[index])
			return f.submissions[index], nil
		}
	}
	return rollup.StoredSubmission{}, errors.New("submission not found")
}

type fakeRollupTxSender struct {
	nonce       uint64
	chainID     *big.Int
	gasPrice    *big.Int
	estimate    uint64
	sendErr     error
	sentTxs     []*types.Transaction
	receipts    map[string]*types.Receipt
	receiptErr  map[string]error
	receiptFn   func(txHash common.Hash) (*types.Receipt, error)
	callResults map[string][]byte
	callErr     map[string]error
	callFn      func(call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

func (f *fakeRollupTxSender) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	_ = ctx
	_ = account
	value := f.nonce
	f.nonce++
	return value, nil
}

func (f *fakeRollupTxSender) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	_ = ctx
	return f.gasPrice, nil
}

func (f *fakeRollupTxSender) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	_ = ctx
	_ = call
	return f.estimate, nil
}

func (f *fakeRollupTxSender) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	_ = ctx
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sentTxs = append(f.sentTxs, tx)
	return nil
}

func (f *fakeRollupTxSender) ChainID(ctx context.Context) (*big.Int, error) {
	_ = ctx
	return f.chainID, nil
}

func (f *fakeRollupTxSender) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	_ = ctx
	if f.callFn != nil {
		return f.callFn(call, blockNumber)
	}
	key := common.Bytes2Hex(call.Data)
	if err, ok := f.callErr[key]; ok {
		return nil, err
	}
	if result, ok := f.callResults[key]; ok {
		return result, nil
	}
	return nil, errors.New("unexpected call contract")
}

func (f *fakeRollupTxSender) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	_ = ctx
	if f.receiptFn != nil {
		return f.receiptFn(txHash)
	}
	key := normalizeChainTxHash(txHash.Hex())
	if err, ok := f.receiptErr[key]; ok {
		return nil, err
	}
	if receipt, ok := f.receipts[key]; ok {
		return receipt, nil
	}
	return nil, ethereum.NotFound
}

func TestRollupSubmissionProcessorPollOnceSubmitsRecord(t *testing.T) {
	key := mustGenerateKey(t)
	submission := mustTestStoredSubmission(t, "rsub_1", 1, rollup.SubmissionStatusReady)
	store := &fakeRollupSubmissionStore{
		submissions: []rollup.StoredSubmission{submission},
	}
	sender := &fakeRollupTxSender{
		nonce:    7,
		chainID:  big.NewInt(97),
		gasPrice: big.NewInt(1_000_000_000),
		estimate: 120000,
		receipts: map[string]*types.Receipt{},
	}
	processor := newTestRollupSubmissionProcessor(t, key, store, sender)

	progress, err := processor.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("PollOnce returned error: %v", err)
	}
	if progress.Action != RollupSubmissionActionRecordSubmitted {
		t.Fatalf("action = %s, want %s", progress.Action, RollupSubmissionActionRecordSubmitted)
	}
	if progress.Submission.Status != rollup.SubmissionStatusRecordSubmitted {
		t.Fatalf("status = %s, want %s", progress.Submission.Status, rollup.SubmissionStatusRecordSubmitted)
	}
	if progress.Submission.RecordTxHash == "" {
		t.Fatalf("expected record tx hash")
	}
	if len(sender.sentTxs) != 1 {
		t.Fatalf("sent tx count = %d, want 1", len(sender.sentTxs))
	}
	if got := common.Bytes2Hex(sender.sentTxs[0].Data()); got != "1111" {
		t.Fatalf("record calldata = %s, want 1111", got)
	}
}

func TestRollupSubmissionProcessorPollOnceSkipsWhenFrozen(t *testing.T) {
	key := mustGenerateKey(t)
	store := &fakeRollupSubmissionStore{
		frozen: true,
		submissions: []rollup.StoredSubmission{
			mustTestStoredSubmission(t, "rsub_frozen", 1, rollup.SubmissionStatusReady),
		},
	}
	sender := &fakeRollupTxSender{
		nonce:    7,
		chainID:  big.NewInt(97),
		gasPrice: big.NewInt(1_000_000_000),
		estimate: 120000,
		receipts: map[string]*types.Receipt{},
	}
	processor := newTestRollupSubmissionProcessor(t, key, store, sender)

	progress, err := processor.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("PollOnce returned error: %v", err)
	}
	if progress.Action != RollupSubmissionActionFrozen {
		t.Fatalf("action = %s, want %s", progress.Action, RollupSubmissionActionFrozen)
	}
	if len(sender.sentTxs) != 0 {
		t.Fatalf("expected no txs to be sent while frozen, got %d", len(sender.sentTxs))
	}
}

func TestRollupSubmissionProcessorPollOnceWaitsForRecordReconciliation(t *testing.T) {
	key := mustGenerateKey(t)
	submission := mustTestStoredSubmission(t, "rsub_2", 2, rollup.SubmissionStatusRecordSubmitted)
	expected := mustExpectedSubmissionState(t, submission)
	recordTxHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	store := &fakeRollupSubmissionStore{
		submissions: []rollup.StoredSubmission{
			withStoredRecordTxHash(submission, recordTxHash),
		},
	}
	sender := &fakeRollupTxSender{
		nonce:    9,
		chainID:  big.NewInt(97),
		gasPrice: big.NewInt(1_000_000_000),
		estimate: 120000,
		receipts: map[string]*types.Receipt{
			recordTxHash: {Status: types.ReceiptStatusSuccessful, BlockNumber: big.NewInt(9)},
		},
		callResults: map[string][]byte{
			mustPackRollupCoreCallData(t, "latestBatchId"):   mustPackRollupCoreCallOutput(t, "latestBatchId", uint64(1)),
			mustPackRollupCoreCallData(t, "latestStateRoot"): mustPackRollupCoreCallOutput(t, "latestStateRoot", expected.PrevStateRoot),
			mustPackRollupCoreCallData(t, "batchMetadata", expected.BatchID): mustPackRollupCoreCallOutput(
				t, "batchMetadata", expected.BatchDataHash, expected.PrevStateRoot, common.Hash{},
			),
		},
	}
	processor := newTestRollupSubmissionProcessor(t, key, store, sender)

	progress, err := processor.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("PollOnce returned error: %v", err)
	}
	if progress.Action != RollupSubmissionActionRecordPending {
		t.Fatalf("action = %s, want %s", progress.Action, RollupSubmissionActionRecordPending)
	}
	if progress.Submission.Status != rollup.SubmissionStatusRecordSubmitted {
		t.Fatalf("status = %s, want %s", progress.Submission.Status, rollup.SubmissionStatusRecordSubmitted)
	}
	if len(sender.sentTxs) != 0 {
		t.Fatalf("expected no accept tx to be sent before record reconciliation")
	}
}

func TestRollupSubmissionProcessorPollOnceAdvancesPublish(t *testing.T) {
	key := mustGenerateKey(t)
	submission := mustTestStoredSubmission(t, "rsub_3a", 3, rollup.SubmissionStatusRecordSubmitted)
	expected := mustExpectedSubmissionState(t, submission)
	recordTxHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	store := &fakeRollupSubmissionStore{
		submissions: []rollup.StoredSubmission{
			withStoredRecordTxHash(submission, recordTxHash),
		},
	}
	sender := &fakeRollupTxSender{
		nonce:    9,
		chainID:  big.NewInt(97),
		gasPrice: big.NewInt(1_000_000_000),
		estimate: 120000,
		receipts: map[string]*types.Receipt{
			recordTxHash: {Status: types.ReceiptStatusSuccessful, BlockNumber: big.NewInt(9)},
		},
		callResults: map[string][]byte{
			mustPackRollupCoreCallData(t, "latestBatchId"):   mustPackRollupCoreCallOutput(t, "latestBatchId", expected.BatchID),
			mustPackRollupCoreCallData(t, "latestStateRoot"): mustPackRollupCoreCallOutput(t, "latestStateRoot", expected.NextStateRoot),
			mustPackRollupCoreCallData(t, "batchMetadata", expected.BatchID): mustPackRollupCoreCallOutput(
				t, "batchMetadata", expected.BatchDataHash, expected.PrevStateRoot, expected.NextStateRoot,
			),
		},
	}
	processor := newTestRollupSubmissionProcessor(t, key, store, sender)

	progress, err := processor.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("PollOnce returned error: %v", err)
	}
	if progress.Action != RollupSubmissionActionPublishSubmitted {
		t.Fatalf("action = %s, want %s", progress.Action, RollupSubmissionActionPublishSubmitted)
	}
	if progress.Submission.Status != rollup.SubmissionStatusPublishSubmitted {
		t.Fatalf("status = %s, want %s", progress.Submission.Status, rollup.SubmissionStatusPublishSubmitted)
	}
	if progress.Submission.PublishTxHash == "" {
		t.Fatalf("expected publish tx hash")
	}
	if len(sender.sentTxs) != 1 {
		t.Fatalf("sent tx count = %d, want 1", len(sender.sentTxs))
	}
}

func TestRollupSubmissionProcessorPollOnceAdvancesAccept(t *testing.T) {
	key := mustGenerateKey(t)
	submission := mustTestStoredSubmission(t, "rsub_3", 3, rollup.SubmissionStatusDataPublished)
	expected := mustExpectedSubmissionState(t, submission)
	store := &fakeRollupSubmissionStore{
		submissions: []rollup.StoredSubmission{submission},
	}
	sender := &fakeRollupTxSender{
		nonce:    9,
		chainID:  big.NewInt(97),
		gasPrice: big.NewInt(1_000_000_000),
		estimate: 120000,
		callResults: map[string][]byte{
			mustPackRollupCoreCallData(t, "latestBatchId"):   mustPackRollupCoreCallOutput(t, "latestBatchId", expected.BatchID),
			mustPackRollupCoreCallData(t, "latestStateRoot"): mustPackRollupCoreCallOutput(t, "latestStateRoot", expected.NextStateRoot),
			mustPackRollupCoreCallData(t, "batchMetadata", expected.BatchID): mustPackRollupCoreCallOutput(
				t, "batchMetadata", expected.BatchDataHash, expected.PrevStateRoot, expected.NextStateRoot,
			),
		},
	}
	processor := newTestRollupSubmissionProcessor(t, key, store, sender)

	progress, err := processor.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("PollOnce returned error: %v", err)
	}
	if progress.Action != RollupSubmissionActionAcceptSubmitted {
		t.Fatalf("action = %s, want %s", progress.Action, RollupSubmissionActionAcceptSubmitted)
	}
	if progress.Submission.Status != rollup.SubmissionStatusAcceptSubmitted {
		t.Fatalf("status = %s, want %s", progress.Submission.Status, rollup.SubmissionStatusAcceptSubmitted)
	}
	if progress.Submission.AcceptTxHash == "" {
		t.Fatalf("expected accept tx hash")
	}
	if len(sender.sentTxs) != 1 {
		t.Fatalf("sent tx count = %d, want 1", len(sender.sentTxs))
	}
}

func TestRollupSubmissionProcessorPollOnceWaitsForAcceptanceReconciliation(t *testing.T) {
	key := mustGenerateKey(t)
	submission := mustTestStoredSubmission(t, "rsub_4", 4, rollup.SubmissionStatusAcceptSubmitted)
	expected := mustExpectedSubmissionState(t, submission)
	acceptTxHash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	store := &fakeRollupSubmissionStore{
		submissions: []rollup.StoredSubmission{
			withStoredAcceptTxHash(submission, acceptTxHash),
		},
	}
	sender := &fakeRollupTxSender{
		nonce:    11,
		chainID:  big.NewInt(97),
		gasPrice: big.NewInt(1_000_000_000),
		estimate: 120000,
		receipts: map[string]*types.Receipt{
			acceptTxHash: {Status: types.ReceiptStatusSuccessful, BlockNumber: big.NewInt(11)},
		},
		callResults: map[string][]byte{
			mustPackRollupCoreCallData(t, "latestAcceptedBatchId"):   mustPackRollupCoreCallOutput(t, "latestAcceptedBatchId", expected.BatchID-1),
			mustPackRollupCoreCallData(t, "latestAcceptedStateRoot"): mustPackRollupCoreCallOutput(t, "latestAcceptedStateRoot", expected.PrevStateRoot),
			mustPackRollupCoreCallData(t, "acceptedBatches", expected.BatchID): mustPackRollupCoreCallOutput(
				t,
				"acceptedBatches",
				uint64(0),
				uint64(0),
				uint64(0),
				common.Hash{},
				common.Hash{},
				common.Hash{},
				common.Hash{},
				common.Hash{},
				common.Hash{},
				common.Hash{},
				common.Hash{},
				common.Hash{},
				common.Hash{},
			),
		},
	}
	processor := newTestRollupSubmissionProcessor(t, key, store, sender)

	progress, err := processor.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("PollOnce returned error: %v", err)
	}
	if progress.Action != RollupSubmissionActionAcceptPending {
		t.Fatalf("action = %s, want %s", progress.Action, RollupSubmissionActionAcceptPending)
	}
	if progress.Submission.Status != rollup.SubmissionStatusAcceptSubmitted {
		t.Fatalf("status = %s, want %s", progress.Submission.Status, rollup.SubmissionStatusAcceptSubmitted)
	}
}

func TestRollupSubmissionProcessorPollOnceMarksAccepted(t *testing.T) {
	key := mustGenerateKey(t)
	submission := mustTestStoredSubmission(t, "rsub_5", 5, rollup.SubmissionStatusAcceptSubmitted)
	expected := mustExpectedSubmissionState(t, submission)
	acceptTxHash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	store := &fakeRollupSubmissionStore{
		submissions: []rollup.StoredSubmission{
			withStoredAcceptTxHash(submission, acceptTxHash),
		},
	}
	sender := &fakeRollupTxSender{
		nonce:    11,
		chainID:  big.NewInt(97),
		gasPrice: big.NewInt(1_000_000_000),
		estimate: 120000,
		receipts: map[string]*types.Receipt{
			acceptTxHash: {Status: types.ReceiptStatusSuccessful, BlockNumber: big.NewInt(11)},
		},
		callResults: map[string][]byte{
			mustPackRollupCoreCallData(t, "latestAcceptedBatchId"):   mustPackRollupCoreCallOutput(t, "latestAcceptedBatchId", expected.BatchID),
			mustPackRollupCoreCallData(t, "latestAcceptedStateRoot"): mustPackRollupCoreCallOutput(t, "latestAcceptedStateRoot", expected.NextStateRoot),
			mustPackRollupCoreCallData(t, "acceptedBatches", expected.BatchID): mustPackRollupCoreCallOutput(
				t,
				"acceptedBatches",
				expected.FirstSequenceNo,
				expected.LastSequenceNo,
				expected.EntryCount,
				expected.BatchDataHash,
				expected.PrevStateRoot,
				expected.BalancesRoot,
				expected.OrdersRoot,
				expected.PositionsFundingRoot,
				expected.WithdrawalsRoot,
				expected.NextStateRoot,
				expected.ConservationHash,
				expected.AuthProofHash,
				expected.VerifierGateHash,
			),
		},
	}
	processor := newTestRollupSubmissionProcessor(t, key, store, sender)

	progress, err := processor.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("PollOnce returned error: %v", err)
	}
	if progress.Action != RollupSubmissionActionAccepted {
		t.Fatalf("action = %s, want %s", progress.Action, RollupSubmissionActionAccepted)
	}
	if progress.Submission.Status != rollup.SubmissionStatusAccepted {
		t.Fatalf("status = %s, want %s", progress.Submission.Status, rollup.SubmissionStatusAccepted)
	}
}

func TestRollupSubmissionProcessorPollOnceBlocksOnEarlierAuthGap(t *testing.T) {
	key := mustGenerateKey(t)
	store := &fakeRollupSubmissionStore{
		submissions: []rollup.StoredSubmission{
			{
				SubmissionID: "rsub_6",
				BatchID:      6,
				Status:       rollup.SubmissionStatusBlockedAuth,
			},
			mustTestStoredSubmission(t, "rsub_7", 7, rollup.SubmissionStatusReady),
		},
	}
	sender := &fakeRollupTxSender{
		nonce:    7,
		chainID:  big.NewInt(97),
		gasPrice: big.NewInt(1_000_000_000),
		estimate: 120000,
		receipts: map[string]*types.Receipt{},
	}
	processor := newTestRollupSubmissionProcessor(t, key, store, sender)

	progress, err := processor.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("PollOnce returned error: %v", err)
	}
	if progress.Action != RollupSubmissionActionBlockedAuth {
		t.Fatalf("action = %s, want %s", progress.Action, RollupSubmissionActionBlockedAuth)
	}
	if len(sender.sentTxs) != 0 {
		t.Fatalf("expected no tx to be sent while earlier batch is blocked")
	}
}

func TestRollupSubmissionProcessorRunUntilIdle(t *testing.T) {
	key := mustGenerateKey(t)
	submission := mustTestStoredSubmission(t, "rsub_8", 8, rollup.SubmissionStatusReady)
	expected := mustExpectedSubmissionState(t, submission)
	store := &fakeRollupSubmissionStore{
		submissions: []rollup.StoredSubmission{submission},
	}
	sender := &fakeRollupTxSender{
		nonce:    20,
		chainID:  big.NewInt(97),
		gasPrice: big.NewInt(1_000_000_000),
		estimate: 120000,
		receipts: map[string]*types.Receipt{},
	}
	sender.receiptFn = func(txHash common.Hash) (*types.Receipt, error) {
		normalized := normalizeChainTxHash(txHash.Hex())
		if len(store.submissions) > 0 {
			if store.submissions[0].RecordTxHash == normalized {
				return &types.Receipt{Status: types.ReceiptStatusSuccessful, BlockNumber: big.NewInt(21)}, nil
			}
			if store.submissions[0].PublishTxHash == normalized {
				return &types.Receipt{Status: types.ReceiptStatusSuccessful, BlockNumber: big.NewInt(21)}, nil
			}
			if store.submissions[0].AcceptTxHash == normalized {
				return &types.Receipt{Status: types.ReceiptStatusSuccessful, BlockNumber: big.NewInt(22)}, nil
			}
		}
		return nil, ethereum.NotFound
	}
	sender.callFn = func(call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
		recordTxHash := ""
		acceptTxHash := ""
		if len(store.submissions) > 0 {
			recordTxHash = store.submissions[0].RecordTxHash
			acceptTxHash = store.submissions[0].AcceptTxHash
		}
		switch common.Bytes2Hex(call.Data) {
		case mustPackRollupCoreCallData(t, "latestBatchId"):
			if recordTxHash != "" {
				return mustPackRollupCoreCallOutput(t, "latestBatchId", expected.BatchID), nil
			}
			return mustPackRollupCoreCallOutput(t, "latestBatchId", uint64(0)), nil
		case mustPackRollupCoreCallData(t, "latestStateRoot"):
			if recordTxHash != "" {
				return mustPackRollupCoreCallOutput(t, "latestStateRoot", expected.NextStateRoot), nil
			}
			return mustPackRollupCoreCallOutput(t, "latestStateRoot", expected.PrevStateRoot), nil
		case mustPackRollupCoreCallData(t, "batchMetadata", expected.BatchID):
			if recordTxHash != "" {
				return mustPackRollupCoreCallOutput(t, "batchMetadata", expected.BatchDataHash, expected.PrevStateRoot, expected.NextStateRoot), nil
			}
			return mustPackRollupCoreCallOutput(t, "batchMetadata", common.Hash{}, common.Hash{}, common.Hash{}), nil
		case mustPackRollupCoreCallData(t, "batchDataPublished", expected.BatchID):
			publishTxHash := ""
			if len(store.submissions) > 0 {
				publishTxHash = store.submissions[0].PublishTxHash
			}
			if publishTxHash != "" {
				return mustPackRollupCoreCallOutput(t, "batchDataPublished", true), nil
			}
			return mustPackRollupCoreCallOutput(t, "batchDataPublished", false), nil
		case mustPackRollupCoreCallData(t, "latestAcceptedBatchId"):
			if acceptTxHash != "" {
				return mustPackRollupCoreCallOutput(t, "latestAcceptedBatchId", expected.BatchID), nil
			}
			return mustPackRollupCoreCallOutput(t, "latestAcceptedBatchId", uint64(0)), nil
		case mustPackRollupCoreCallData(t, "latestAcceptedStateRoot"):
			if acceptTxHash != "" {
				return mustPackRollupCoreCallOutput(t, "latestAcceptedStateRoot", expected.NextStateRoot), nil
			}
			return mustPackRollupCoreCallOutput(t, "latestAcceptedStateRoot", expected.PrevStateRoot), nil
		case mustPackRollupCoreCallData(t, "acceptedBatches", expected.BatchID):
			if acceptTxHash != "" {
				return mustPackRollupCoreCallOutput(
					t,
					"acceptedBatches",
					expected.FirstSequenceNo,
					expected.LastSequenceNo,
					expected.EntryCount,
					expected.BatchDataHash,
					expected.PrevStateRoot,
					expected.BalancesRoot,
					expected.OrdersRoot,
					expected.PositionsFundingRoot,
					expected.WithdrawalsRoot,
					expected.NextStateRoot,
					expected.ConservationHash,
					expected.AuthProofHash,
					expected.VerifierGateHash,
				), nil
			}
			return mustPackRollupCoreCallOutput(
				t,
				"acceptedBatches",
				uint64(0),
				uint64(0),
				uint64(0),
				common.Hash{},
				common.Hash{},
				common.Hash{},
				common.Hash{},
				common.Hash{},
				common.Hash{},
				common.Hash{},
				common.Hash{},
				common.Hash{},
				common.Hash{},
			), nil
		}
		return nil, errors.New("unexpected call contract")
	}
	processor := newTestRollupSubmissionProcessor(t, key, store, sender)
	processor.pollInterval = time.Millisecond

	run, err := processor.RunUntilIdle(context.Background())
	if err != nil {
		t.Fatalf("RunUntilIdle returned error: %v", err)
	}
	if len(run.Steps) < 5 {
		t.Fatalf("expected multiple progress steps (record+publish+accept+reconcile), got %d", len(run.Steps))
	}
	last := run.Steps[len(run.Steps)-1]
	if last.Action != RollupSubmissionActionNoop {
		t.Fatalf("last action = %s, want %s", last.Action, RollupSubmissionActionNoop)
	}
	if store.submissions[0].Status != rollup.SubmissionStatusAccepted {
		t.Fatalf("final stored status = %s, want %s", store.submissions[0].Status, rollup.SubmissionStatusAccepted)
	}
	if len(store.materializedByIDCall) == 0 || store.materializedByIDCall[len(store.materializedByIDCall)-1] != submission.SubmissionID {
		t.Fatalf("expected accepted submission to be materialized, calls=%v", store.materializedByIDCall)
	}
}

func newTestRollupSubmissionProcessor(
	t *testing.T,
	key *ecdsa.PrivateKey,
	store rollupSubmissionStore,
	sender rollupTxSender,
) *RollupSubmissionProcessor {
	t.Helper()
	cfg := config.ServiceConfig{
		ChainOperatorPrivateKey: privateKeyHex(key),
		RollupCoreAddress:       "0x00000000000000000000000000000000000000cc",
		RollupBatchLimit:        256,
		RollupPollInterval:      time.Second,
		ChainGasLimit:           250000,
	}
	processor, err := NewRollupSubmissionProcessor(slog.Default(), cfg, store, sender)
	if err != nil {
		t.Fatalf("NewRollupSubmissionProcessor returned error: %v", err)
	}
	return processor
}

func mustGenerateKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	return key
}

func mustTestStoredSubmission(
	t *testing.T,
	submissionID string,
	batchID int64,
	status string,
) rollup.StoredSubmission {
	t.Helper()
	bundle := rollup.ShadowBatchSubmissionBundle{
		SubmissionVersion: rollup.SubmissionEncodingVersion,
		Status:            status,
		Batch: rollup.SubmissionBatchSummary{
			BatchID:              batchID,
			EncodingVersion:      rollup.BatchEncodingVersion,
			FirstSequence:        10,
			LastSequence:         12,
			EntryCount:           3,
			InputHash:            strings.Repeat("11", 32),
			BatchDataHash:        strings.Repeat("12", 32),
			PrevStateRoot:        strings.Repeat("22", 32),
			BalancesRoot:         strings.Repeat("33", 32),
			OrdersRoot:           strings.Repeat("44", 32),
			PositionsFundingRoot: strings.Repeat("55", 32),
			WithdrawalsRoot:      strings.Repeat("66", 32),
			NextStateRoot:        strings.Repeat("77", 32),
			ConservationHash:     strings.Repeat("aa", 32),
		},
	}
	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("marshal test submission bundle: %v", err)
	}
	return rollup.StoredSubmission{
		SubmissionID:     submissionID,
		BatchID:          batchID,
		Status:           status,
		RecordCalldata:   "0x1111",
		PublishCalldata:  "0x3333",
		AcceptCalldata:   "0x2222",
		BatchDataHash:    "0x" + strings.Repeat("12", 32),
		NextStateRoot:    "0x" + strings.Repeat("77", 32),
		AuthProofHash:    "0x" + strings.Repeat("88", 32),
		VerifierGateHash: "0x" + strings.Repeat("99", 32),
		SubmissionData:   string(raw),
	}
}

func mustExpectedSubmissionState(t *testing.T, submission rollup.StoredSubmission) expectedRollupSubmissionState {
	t.Helper()
	expected, err := buildExpectedSubmissionState(submission)
	if err != nil {
		t.Fatalf("buildExpectedSubmissionState returned error: %v", err)
	}
	return expected
}

func withStoredRecordTxHash(submission rollup.StoredSubmission, txHash string) rollup.StoredSubmission {
	submission.RecordTxHash = txHash
	return submission
}

func withStoredAcceptTxHash(submission rollup.StoredSubmission, txHash string) rollup.StoredSubmission {
	submission.AcceptTxHash = txHash
	return submission
}

func mustPackRollupCoreCallData(t *testing.T, method string, args ...any) string {
	t.Helper()
	packed, err := funnyRollupCoreReadABI.Pack(method, args...)
	if err != nil {
		t.Fatalf("pack %s calldata: %v", method, err)
	}
	return common.Bytes2Hex(packed)
}

func mustPackRollupCoreCallOutput(t *testing.T, method string, values ...any) []byte {
	t.Helper()
	methodABI, ok := funnyRollupCoreReadABI.Methods[method]
	if !ok {
		t.Fatalf("unknown FunnyRollupCore method %s", method)
	}
	normalized := make([]any, 0, len(values))
	for _, value := range values {
		switch typed := value.(type) {
		case uint64:
			normalized = append(normalized, typed)
		case common.Hash:
			normalized = append(normalized, typed)
		default:
			normalized = append(normalized, value)
		}
	}
	packed, err := methodABI.Outputs.Pack(normalized...)
	if err != nil {
		t.Fatalf("pack %s output: %v", method, err)
	}
	return packed
}
