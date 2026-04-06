package service

import (
	"context"
	"strings"
	"testing"

	accountclient "funnyoption/internal/account/client"
	chainmodel "funnyoption/internal/chain/model"
	claimmodel "funnyoption/internal/chain/model"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type fakeDepositStore struct {
	deposit                chainmodel.Deposit
	withdrawal             chainmodel.Withdrawal
	lastDeposit            chainmodel.Deposit
	lastWithdrawal         chainmodel.Withdrawal
	upsertErr              error
	markCreditedID         string
	markDebitedID          string
	walletUsers            map[string]int64
	scanCursor             uint64
	hasScanCursor          bool
	savedScanCursors       []uint64
	claimTasks             []claimmodel.ClaimTask
	confirmedClaimTx       string
	confirmedEscapeClaimID string
}

func (f *fakeDepositStore) UpsertDeposit(ctx context.Context, deposit chainmodel.Deposit) (chainmodel.Deposit, error) {
	_ = ctx
	f.lastDeposit = deposit
	if f.upsertErr != nil {
		return chainmodel.Deposit{}, f.upsertErr
	}
	if f.deposit.DepositID == "" {
		f.deposit = deposit
	}
	return f.deposit, nil
}

func (f *fakeDepositStore) MarkDepositCredited(ctx context.Context, depositID string, creditedAt int64) error {
	_ = ctx
	_ = creditedAt
	f.markCreditedID = depositID
	return nil
}

func (f *fakeDepositStore) UpsertWithdrawal(ctx context.Context, withdrawal chainmodel.Withdrawal) (chainmodel.Withdrawal, error) {
	_ = ctx
	f.lastWithdrawal = withdrawal
	if f.upsertErr != nil {
		return chainmodel.Withdrawal{}, f.upsertErr
	}
	if f.withdrawal.WithdrawalID == "" {
		f.withdrawal = withdrawal
	}
	return f.withdrawal, nil
}

func (f *fakeDepositStore) MarkWithdrawalDebited(ctx context.Context, withdrawalID string, debitedAt int64) error {
	_ = ctx
	_ = debitedAt
	f.markDebitedID = withdrawalID
	return nil
}

func (f *fakeDepositStore) LookupActiveUserByWallet(ctx context.Context, walletAddress string) (int64, error) {
	_ = ctx
	if userID, ok := f.walletUsers[strings.ToLower(walletAddress)]; ok {
		return userID, nil
	}
	return 0, ErrWalletSessionNotFound
}

func (f *fakeDepositStore) LoadVaultScanCursor(ctx context.Context, chainName string, networkName string, vaultAddress string) (uint64, bool, error) {
	_ = ctx
	_ = chainName
	_ = networkName
	_ = vaultAddress
	return f.scanCursor, f.hasScanCursor, nil
}

func (f *fakeDepositStore) SaveVaultScanCursor(ctx context.Context, chainName string, networkName string, vaultAddress string, nextBlock uint64) error {
	_ = ctx
	_ = chainName
	_ = networkName
	_ = vaultAddress
	f.scanCursor = nextBlock
	f.hasScanCursor = true
	f.savedScanCursors = append(f.savedScanCursors, nextBlock)
	return nil
}

func (f *fakeDepositStore) ListPendingClaims(ctx context.Context, limit int) ([]claimmodel.ClaimTask, error) {
	_ = ctx
	_ = limit
	return f.claimTasks, nil
}

func (f *fakeDepositStore) MarkClaimSubmitted(ctx context.Context, id int64, txHash string) error {
	_ = ctx
	_ = id
	_ = txHash
	return nil
}

func (f *fakeDepositStore) MarkClaimFailed(ctx context.Context, id int64, errMsg string) error {
	_ = ctx
	_ = id
	_ = errMsg
	return nil
}

func (f *fakeDepositStore) MarkClaimConfirmedByTxHash(ctx context.Context, txHash string) error {
	_ = ctx
	f.confirmedClaimTx = txHash
	return nil
}

func (f *fakeDepositStore) MarkAcceptedEscapeClaimConfirmed(ctx context.Context, claimID, txHash string) error {
	_ = ctx
	_ = txHash
	f.confirmedEscapeClaimID = claimID
	return nil
}

type fakeChainAccountClient struct {
	creditResult accountclient.CreditResult
	debitResult  accountclient.DebitResult
	err          error
	calls        int
	debitCalls   int
}

func (f *fakeChainAccountClient) PreFreeze(ctx context.Context, req accountclient.FreezeRequest) (accountclient.FreezeRecord, error) {
	_ = ctx
	_ = req
	return accountclient.FreezeRecord{}, nil
}

func (f *fakeChainAccountClient) ReleaseFreeze(ctx context.Context, freezeID string) error {
	_ = ctx
	_ = freezeID
	return nil
}

func (f *fakeChainAccountClient) GetBalance(ctx context.Context, userID int64, asset string) (accountclient.Balance, error) {
	_ = ctx
	_ = userID
	_ = asset
	return accountclient.Balance{}, nil
}

func (f *fakeChainAccountClient) CreditBalance(ctx context.Context, userID int64, asset string, amount int64, refType, refID string) (accountclient.CreditResult, error) {
	_ = ctx
	_ = userID
	_ = asset
	_ = amount
	_ = refType
	_ = refID
	f.calls++
	return f.creditResult, f.err
}

func (f *fakeChainAccountClient) DebitBalance(ctx context.Context, userID int64, asset string, amount int64, refType, refID string) (accountclient.DebitResult, error) {
	_ = ctx
	_ = userID
	_ = asset
	_ = amount
	_ = refType
	_ = refID
	f.debitCalls++
	return f.debitResult, f.err
}

func (f *fakeChainAccountClient) Close() error { return nil }

type fakeChainPublisher struct {
	topic   string
	key     string
	payload any
}

func (f *fakeChainPublisher) PublishJSON(ctx context.Context, topic, key string, payload any) error {
	_ = ctx
	f.topic = topic
	f.key = key
	f.payload = payload
	return nil
}

func (f *fakeChainPublisher) Close() error { return nil }

func TestApplyConfirmedDepositCreditsOnce(t *testing.T) {
	store := &fakeDepositStore{
		deposit: chainmodel.Deposit{
			DepositID:     "dep_1",
			UserID:        1001,
			WalletAddress: "0xabc",
			VaultAddress:  "0xvault",
			Asset:         "USDT",
			Amount:        1_000,
			ChainName:     "bsc",
			NetworkName:   "testnet",
			TxHash:        "0xtx",
			LogIndex:      1,
			BlockNumber:   2,
			Status:        "CONFIRMED",
		},
	}
	account := &fakeChainAccountClient{
		creditResult: accountclient.CreditResult{UserID: 1001, Asset: "USDT", Available: 1000, Applied: true},
	}
	publisher := &fakeChainPublisher{}
	processor := NewProcessor(nil, store, account, publisher, sharedkafka.NewTopics("funnyoption."))

	err := processor.ApplyConfirmedDeposit(context.Background(), chainmodel.Deposit{
		DepositID:     "dep_1",
		UserID:        1001,
		WalletAddress: "0xabc",
		VaultAddress:  "0xvault",
		Asset:         "USDT",
		Amount:        1_000,
		TxHash:        "0xtx",
		LogIndex:      1,
	})
	if err != nil {
		t.Fatalf("ApplyConfirmedDeposit returned error: %v", err)
	}
	if account.calls != 1 {
		t.Fatalf("expected account credit once, got %d", account.calls)
	}
	if store.markCreditedID != "dep_1" {
		t.Fatalf("expected marked deposit dep_1, got %s", store.markCreditedID)
	}
	if publisher.topic != "funnyoption.chain.deposit" {
		t.Fatalf("unexpected publish topic: %s", publisher.topic)
	}
}

func TestApplyConfirmedDepositSkipsAlreadyCredited(t *testing.T) {
	store := &fakeDepositStore{
		deposit: chainmodel.Deposit{
			DepositID:  "dep_credited",
			UserID:     1001,
			Asset:      "USDT",
			Amount:     100,
			CreditedAt: 123,
		},
	}
	account := &fakeChainAccountClient{}
	publisher := &fakeChainPublisher{}
	processor := NewProcessor(nil, store, account, publisher, sharedkafka.NewTopics("funnyoption."))

	if err := processor.ApplyConfirmedDeposit(context.Background(), chainmodel.Deposit{
		DepositID: "dep_credited",
		UserID:    1001,
		Asset:     "USDT",
		Amount:    100,
	}); err != nil {
		t.Fatalf("ApplyConfirmedDeposit returned error: %v", err)
	}
	if account.calls != 0 {
		t.Fatalf("expected no credit call, got %d", account.calls)
	}
}

func TestApplyConfirmedDepositNormalizesTxHash(t *testing.T) {
	store := &fakeDepositStore{}
	account := &fakeChainAccountClient{
		creditResult: accountclient.CreditResult{UserID: 1001, Asset: "USDT", Available: 1000, Applied: true},
	}
	publisher := &fakeChainPublisher{}
	processor := NewProcessor(nil, store, account, publisher, sharedkafka.NewTopics("funnyoption."))

	if err := processor.ApplyConfirmedDeposit(context.Background(), chainmodel.Deposit{
		DepositID: "dep_norm",
		UserID:    1001,
		Asset:     "USDT",
		Amount:    1000,
		TxHash:    "0xABC123",
		LogIndex:  1,
	}); err != nil {
		t.Fatalf("ApplyConfirmedDeposit returned error: %v", err)
	}
	if store.lastDeposit.TxHash != "abc123" {
		t.Fatalf("expected normalized tx hash abc123, got %s", store.lastDeposit.TxHash)
	}
}

func TestApplyConfirmedWithdrawalDebitsOnce(t *testing.T) {
	store := &fakeDepositStore{
		withdrawal: chainmodel.Withdrawal{
			WithdrawalID:     "wdq_1",
			UserID:           1001,
			WalletAddress:    "0xabc",
			RecipientAddress: "0xabc",
			VaultAddress:     "0xvault",
			Asset:            "USDT",
			Amount:           600,
			ChainName:        "bsc",
			NetworkName:      "testnet",
			TxHash:           "0xtx",
			LogIndex:         2,
			BlockNumber:      3,
			Status:           "QUEUED",
		},
	}
	account := &fakeChainAccountClient{
		debitResult: accountclient.DebitResult{UserID: 1001, Asset: "USDT", Available: 400, Applied: true},
	}
	publisher := &fakeChainPublisher{}
	processor := NewProcessor(nil, store, account, publisher, sharedkafka.NewTopics("funnyoption."))

	err := processor.ApplyConfirmedWithdrawal(context.Background(), chainmodel.Withdrawal{
		WithdrawalID:     "wdq_1",
		UserID:           1001,
		WalletAddress:    "0xabc",
		RecipientAddress: "0xabc",
		VaultAddress:     "0xvault",
		Asset:            "USDT",
		Amount:           600,
		TxHash:           "0xtx",
		LogIndex:         2,
	})
	if err != nil {
		t.Fatalf("ApplyConfirmedWithdrawal returned error: %v", err)
	}
	if account.debitCalls != 1 {
		t.Fatalf("expected account debit once, got %d", account.debitCalls)
	}
	if store.markDebitedID != "wdq_1" {
		t.Fatalf("expected marked withdrawal wdq_1, got %s", store.markDebitedID)
	}
	if publisher.topic != "funnyoption.chain.withdrawal" {
		t.Fatalf("unexpected publish topic: %s", publisher.topic)
	}
}

func TestApplyConfirmedWithdrawalNormalizesTxHash(t *testing.T) {
	store := &fakeDepositStore{}
	account := &fakeChainAccountClient{
		debitResult: accountclient.DebitResult{UserID: 1001, Asset: "USDT", Available: 400, Applied: true},
	}
	publisher := &fakeChainPublisher{}
	processor := NewProcessor(nil, store, account, publisher, sharedkafka.NewTopics("funnyoption."))

	if err := processor.ApplyConfirmedWithdrawal(context.Background(), chainmodel.Withdrawal{
		WithdrawalID:     "wdq_norm",
		UserID:           1001,
		WalletAddress:    "0xabc",
		RecipientAddress: "0xabc",
		VaultAddress:     "0xvault",
		Asset:            "USDT",
		Amount:           600,
		TxHash:           "0xDEF456",
		LogIndex:         2,
	}); err != nil {
		t.Fatalf("ApplyConfirmedWithdrawal returned error: %v", err)
	}
	if store.lastWithdrawal.TxHash != "def456" {
		t.Fatalf("expected normalized tx hash def456, got %s", store.lastWithdrawal.TxHash)
	}
}

func TestBuildChainEventIDFitsDepositIDColumn(t *testing.T) {
	value := buildChainEventID("dep", "0xabcdef", 7)
	if len(value) > 64 {
		t.Fatalf("expected deposit event id to fit varchar(64), got len=%d value=%s", len(value), value)
	}
	if !strings.HasPrefix(value, "dep_") {
		t.Fatalf("expected deposit event id prefix dep_, got %s", value)
	}
}
