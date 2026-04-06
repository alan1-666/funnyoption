package service

import (
	"context"
	"crypto/ecdsa"
	"log/slog"
	"math/big"
	"strings"
	"testing"

	chainmodel "funnyoption/internal/chain/model"
	"funnyoption/internal/shared/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type fakeClaimStore struct {
	tasks                  []chainmodel.ClaimTask
	submittedTasks         []chainmodel.ClaimTask
	submittedID            int64
	submittedTxHash        string
	confirmedTxHash        string
	failedID               int64
	failedError            string
	confirmedEscapeClaimID string
}

func (f *fakeClaimStore) ListPendingClaims(ctx context.Context, limit int) ([]chainmodel.ClaimTask, error) {
	_ = ctx
	_ = limit
	return f.tasks, nil
}

func (f *fakeClaimStore) ListSubmittedClaims(ctx context.Context, limit int) ([]chainmodel.ClaimTask, error) {
	_ = ctx
	_ = limit
	return f.submittedTasks, nil
}

func (f *fakeClaimStore) MarkClaimSubmitted(ctx context.Context, id int64, txHash string) error {
	_ = ctx
	f.submittedID = id
	f.submittedTxHash = txHash
	return nil
}

func (f *fakeClaimStore) MarkClaimFailed(ctx context.Context, id int64, errMsg string) error {
	_ = ctx
	f.failedID = id
	f.failedError = errMsg
	return nil
}

func (f *fakeClaimStore) MarkClaimConfirmedByTxHash(ctx context.Context, txHash string) error {
	_ = ctx
	f.confirmedTxHash = txHash
	return nil
}

func (f *fakeClaimStore) MarkAcceptedEscapeClaimConfirmed(ctx context.Context, claimID, txHash string) error {
	_ = ctx
	_ = txHash
	f.confirmedEscapeClaimID = claimID
	return nil
}

type fakeTxSender struct {
	nonce    uint64
	chainID  *big.Int
	gasPrice *big.Int
	estimate uint64
	sendErr  error
	sentTx   *types.Transaction
	receipts map[string]*types.Receipt
}

func (f *fakeTxSender) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	_ = ctx
	_ = account
	return f.nonce, nil
}

func (f *fakeTxSender) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	_ = ctx
	return f.gasPrice, nil
}

func (f *fakeTxSender) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	_ = ctx
	_ = call
	return f.estimate, nil
}

func (f *fakeTxSender) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	_ = ctx
	f.sentTx = tx
	return f.sendErr
}

func (f *fakeTxSender) ChainID(ctx context.Context) (*big.Int, error) {
	_ = ctx
	return f.chainID, nil
}

func (f *fakeTxSender) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	_ = ctx
	if f.receipts == nil {
		return nil, ethereum.NotFound
	}
	receipt, ok := f.receipts[strings.ToLower(strings.TrimPrefix(txHash.Hex(), "0x"))]
	if !ok {
		return nil, ethereum.NotFound
	}
	return receipt, nil
}

func TestClaimProcessorPollOnceSubmitsClaim(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	store := &fakeClaimStore{
		tasks: []chainmodel.ClaimTask{
			{
				ID:               1,
				RefID:            "evt_settlement_1",
				WalletAddress:    "0x00000000000000000000000000000000000000aa",
				RecipientAddress: "0x00000000000000000000000000000000000000bb",
				PayoutAmount:     1000,
			},
		},
	}
	sender := &fakeTxSender{
		nonce:    7,
		chainID:  big.NewInt(97),
		gasPrice: big.NewInt(1_000_000_000),
		estimate: 120000,
	}
	cfg := config.ServiceConfig{
		VaultAddress:            "0x00000000000000000000000000000000000000cc",
		ChainOperatorPrivateKey: privateKeyHex(key),
		ChainGasLimit:           250000,
		CollateralSymbol:        "USDT",
		CollateralDecimals:      6,
		CollateralDisplayDigits: 2,
	}
	processor, err := NewClaimProcessor(slog.Default(), cfg, store, sender)
	if err != nil {
		t.Fatalf("NewClaimProcessor returned error: %v", err)
	}

	if err := processor.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce returned error: %v", err)
	}
	if store.submittedID != 1 {
		t.Fatalf("expected submitted id 1, got %d", store.submittedID)
	}
	if store.submittedTxHash == "" {
		t.Fatalf("expected submitted tx hash")
	}
	if strings.HasPrefix(store.submittedTxHash, "0x") {
		t.Fatalf("expected normalized submitted tx hash without 0x prefix, got %s", store.submittedTxHash)
	}
	if sender.sentTx == nil {
		t.Fatalf("expected transaction to be sent")
	}

	contractABI, err := abi.JSON(strings.NewReader(vaultClaimABI))
	if err != nil {
		t.Fatalf("abi.JSON returned error: %v", err)
	}
	args, err := contractABI.Methods["processClaim"].Inputs.Unpack(sender.sentTx.Data()[4:])
	if err != nil {
		t.Fatalf("Inputs.Unpack returned error: %v", err)
	}
	if len(args) != 4 {
		t.Fatalf("expected 4 contract args, got %d", len(args))
	}
	amountArg, ok := args[2].(*big.Int)
	if !ok {
		t.Fatalf("expected amount arg to be *big.Int, got %T", args[2])
	}
	if amountArg.Int64() != 10_000_000 {
		t.Fatalf("expected chain payout amount 10000000, got %d", amountArg.Int64())
	}
}

func TestClaimProcessorPollOnceFailsInvalidQueuedClaim(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	store := &fakeClaimStore{
		tasks: []chainmodel.ClaimTask{
			{
				ID:               9,
				RefID:            "evt_settlement_9",
				WalletAddress:    "invalid-address",
				RecipientAddress: "0x00000000000000000000000000000000000000bb",
				PayoutAmount:     1000,
			},
		},
	}
	sender := &fakeTxSender{
		nonce:    7,
		chainID:  big.NewInt(97),
		gasPrice: big.NewInt(1_000_000_000),
		estimate: 120000,
	}
	cfg := config.ServiceConfig{
		VaultAddress:            "0x00000000000000000000000000000000000000cc",
		ChainOperatorPrivateKey: privateKeyHex(key),
		ChainGasLimit:           250000,
		CollateralSymbol:        "USDT",
		CollateralDecimals:      6,
		CollateralDisplayDigits: 2,
	}
	processor, err := NewClaimProcessor(slog.Default(), cfg, store, sender)
	if err != nil {
		t.Fatalf("NewClaimProcessor returned error: %v", err)
	}

	if err := processor.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce returned error: %v", err)
	}
	if store.failedID != 9 {
		t.Fatalf("expected failed id 9, got %d", store.failedID)
	}
	if store.submittedID != 0 {
		t.Fatalf("expected no submitted id, got %d", store.submittedID)
	}
	if sender.sentTx != nil {
		t.Fatalf("expected no transaction to be sent for invalid claim task")
	}
	if !strings.Contains(store.failedError, "wallet_address must be a valid EVM address") {
		t.Fatalf("unexpected failed error: %s", store.failedError)
	}
}

func TestClaimProcessorPollOnceConfirmsSubmittedClaimReceipt(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}
	txHash := "c065a79e5d3c4ceb66a80c554e29ed12b96e4d6ed5e58d575b0a4d2330f5a068"
	store := &fakeClaimStore{
		submittedTasks: []chainmodel.ClaimTask{{
			ID:     42,
			RefID:  "wd_42",
			TxHash: txHash,
			Status: "SUBMITTED",
		}},
	}
	sender := &fakeTxSender{
		nonce:    7,
		chainID:  big.NewInt(97),
		gasPrice: big.NewInt(1_000_000_000),
		estimate: 120000,
		receipts: map[string]*types.Receipt{
			txHash: {Status: types.ReceiptStatusSuccessful},
		},
	}
	cfg := config.ServiceConfig{
		VaultAddress:            "0x00000000000000000000000000000000000000cc",
		ChainOperatorPrivateKey: privateKeyHex(key),
		ChainGasLimit:           250000,
		CollateralSymbol:        "USDT",
		CollateralDecimals:      6,
		CollateralDisplayDigits: 2,
	}
	processor, err := NewClaimProcessor(slog.Default(), cfg, store, sender)
	if err != nil {
		t.Fatalf("NewClaimProcessor returned error: %v", err)
	}

	if err := processor.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce returned error: %v", err)
	}
	if store.confirmedTxHash != txHash {
		t.Fatalf("expected confirmed tx hash %s, got %s", txHash, store.confirmedTxHash)
	}
	if store.submittedID != 0 {
		t.Fatalf("expected no new claim submission, got %d", store.submittedID)
	}
	if sender.sentTx != nil {
		t.Fatalf("expected no new transaction for submitted claim reconciliation")
	}
}

func privateKeyHex(key *ecdsa.PrivateKey) string {
	return common.Bytes2Hex(crypto.FromECDSA(key))
}
