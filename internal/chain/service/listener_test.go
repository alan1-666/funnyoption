package service

import (
	"context"
	"errors"
	"log/slog"
	"math/big"
	"testing"
	"time"

	accountclient "funnyoption/internal/account/client"
	chainmodel "funnyoption/internal/chain/model"
	"funnyoption/internal/shared/config"
	sharedkafka "funnyoption/internal/shared/kafka"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type fakeLogReader struct {
	head      uint64
	logs      []types.Log
	filterErr error
	queries   []ethereum.FilterQuery
}

func (f *fakeLogReader) BlockNumber(ctx context.Context) (uint64, error) {
	_ = ctx
	return f.head, nil
}

func (f *fakeLogReader) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	_ = ctx
	f.queries = append(f.queries, q)
	if f.filterErr != nil {
		return nil, f.filterErr
	}
	return f.logs, nil
}

func (f *fakeLogReader) Close() {}

func TestDepositListenerPollOnceCreditsMatchedWallet(t *testing.T) {
	store := &fakeDepositStore{
		walletUsers: map[string]int64{
			"0x00000000000000000000000000000000000000aa": 1001,
		},
		deposit: chainmodel.Deposit{
			DepositID:     "dep_test",
			UserID:        1001,
			WalletAddress: "0x00000000000000000000000000000000000000aa",
			VaultAddress:  "0x00000000000000000000000000000000000000bb",
			Asset:         "USDT",
			Amount:        500,
			ChainName:     "bsc",
			NetworkName:   "testnet",
			TxHash:        "0x1",
			LogIndex:      3,
			BlockNumber:   108,
			Status:        "CONFIRMED",
		},
	}
	account := &fakeChainAccountClient{
		creditResult: accountclient.CreditResult{UserID: 1001, Asset: "USDT", Available: 500, Applied: true},
	}
	publisher := &fakeChainPublisher{}
	processor := NewProcessor(slog.Default(), store, account, publisher, sharedkafka.NewTopics("funnyoption."))
	reader := &fakeLogReader{
		head: 120,
		logs: []types.Log{
			{
				Address:     common.HexToAddress("0x00000000000000000000000000000000000000bb"),
				Topics:      []common.Hash{depositEventTopic, common.BytesToHash(common.HexToAddress("0x00000000000000000000000000000000000000aa").Bytes())},
				Data:        common.LeftPadBytes(big.NewInt(5_000_000).Bytes(), 32),
				BlockNumber: 108,
				TxHash:      common.HexToHash("0x1"),
				Index:       3,
			},
		},
	}
	cfg := config.ServiceConfig{
		ChainName:               "bsc",
		NetworkName:             "testnet",
		VaultAddress:            "0x00000000000000000000000000000000000000bb",
		Confirmations:           6,
		StartBlock:              100,
		PollInterval:            time.Second,
		CollateralSymbol:        "USDT",
		CollateralDecimals:      6,
		CollateralDisplayDigits: 2,
	}

	listener, err := NewDepositListenerWithReader(slog.Default(), cfg, store, processor, reader)
	if err != nil {
		t.Fatalf("NewDepositListenerWithReader returned error: %v", err)
	}

	if err := listener.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce returned error: %v", err)
	}
	if account.calls != 1 {
		t.Fatalf("expected account credit once, got %d", account.calls)
	}
	if publisher.topic != "funnyoption.chain.deposit" {
		t.Fatalf("unexpected publish topic: %s", publisher.topic)
	}
	if len(reader.queries) != 1 {
		t.Fatalf("expected one filter query, got %d", len(reader.queries))
	}
	if reader.queries[0].FromBlock.Uint64() != 100 || reader.queries[0].ToBlock.Uint64() != 114 {
		t.Fatalf("unexpected block range: %d-%d", reader.queries[0].FromBlock.Uint64(), reader.queries[0].ToBlock.Uint64())
	}
}

func TestDepositListenerPollOnceResumesFromSavedCursor(t *testing.T) {
	store := &fakeDepositStore{
		scanCursor:    700,
		hasScanCursor: true,
	}
	account := &fakeChainAccountClient{}
	publisher := &fakeChainPublisher{}
	processor := NewProcessor(slog.Default(), store, account, publisher, sharedkafka.NewTopics("funnyoption."))
	reader := &fakeLogReader{head: 750}
	cfg := config.ServiceConfig{
		ChainName:               "bsc",
		NetworkName:             "testnet",
		VaultAddress:            "0x00000000000000000000000000000000000000bb",
		Confirmations:           6,
		StartBlock:              100,
		PollInterval:            time.Second,
		CollateralSymbol:        "USDT",
		CollateralDecimals:      6,
		CollateralDisplayDigits: 2,
	}

	listener, err := NewDepositListenerWithReader(slog.Default(), cfg, store, processor, reader)
	if err != nil {
		t.Fatalf("NewDepositListenerWithReader returned error: %v", err)
	}

	if err := listener.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce returned error: %v", err)
	}
	if len(reader.queries) != 1 {
		t.Fatalf("expected one filter query, got %d", len(reader.queries))
	}
	if reader.queries[0].FromBlock.Uint64() != 700 || reader.queries[0].ToBlock.Uint64() != 744 {
		t.Fatalf("unexpected block range: %d-%d", reader.queries[0].FromBlock.Uint64(), reader.queries[0].ToBlock.Uint64())
	}
	if listener.nextBlock != 745 {
		t.Fatalf("expected next block 745, got %d", listener.nextBlock)
	}
	if store.scanCursor != 745 {
		t.Fatalf("expected saved scan cursor 745, got %d", store.scanCursor)
	}
}

func TestDepositListenerPollOnceSkipsPrunedHistoryAndPersistsCursor(t *testing.T) {
	store := &fakeDepositStore{}
	account := &fakeChainAccountClient{}
	publisher := &fakeChainPublisher{}
	processor := NewProcessor(slog.Default(), store, account, publisher, sharedkafka.NewTopics("funnyoption."))
	reader := &fakeLogReader{
		head:      1000,
		filterErr: errors.New("History has been pruned for this block"),
	}
	cfg := config.ServiceConfig{
		ChainName:               "bsc",
		NetworkName:             "testnet",
		VaultAddress:            "0x00000000000000000000000000000000000000bb",
		Confirmations:           0,
		StartBlock:              100,
		PollInterval:            time.Second,
		CollateralSymbol:        "USDT",
		CollateralDecimals:      6,
		CollateralDisplayDigits: 2,
	}

	listener, err := NewDepositListenerWithReader(slog.Default(), cfg, store, processor, reader)
	if err != nil {
		t.Fatalf("NewDepositListenerWithReader returned error: %v", err)
	}

	if err := listener.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce returned error: %v", err)
	}
	if len(reader.queries) != 1 {
		t.Fatalf("expected one filter query, got %d", len(reader.queries))
	}
	if reader.queries[0].FromBlock.Uint64() != 100 || reader.queries[0].ToBlock.Uint64() != 599 {
		t.Fatalf("unexpected block range: %d-%d", reader.queries[0].FromBlock.Uint64(), reader.queries[0].ToBlock.Uint64())
	}
	if listener.nextBlock != 1001 {
		t.Fatalf("expected listener to fast-forward to 1001, got %d", listener.nextBlock)
	}
	if store.scanCursor != 1001 {
		t.Fatalf("expected saved scan cursor 1001, got %d", store.scanCursor)
	}
	if account.calls != 0 {
		t.Fatalf("expected no account credit, got %d", account.calls)
	}
}

func TestDepositListenerPollOnceCreditsFreshDepositAfterPrunedFastForwardRestart(t *testing.T) {
	store := &fakeDepositStore{
		walletUsers: map[string]int64{
			"0x00000000000000000000000000000000000000aa": 1001,
		},
		deposit: chainmodel.Deposit{
			DepositID:     "dep_fresh",
			UserID:        1001,
			WalletAddress: "0x00000000000000000000000000000000000000aa",
			VaultAddress:  "0x00000000000000000000000000000000000000bb",
			Asset:         "USDT",
			Amount:        700,
			ChainName:     "bsc",
			NetworkName:   "testnet",
			TxHash:        "0xfresh",
			LogIndex:      5,
			BlockNumber:   1002,
			Status:        "CONFIRMED",
		},
	}
	cfg := config.ServiceConfig{
		ChainName:               "bsc",
		NetworkName:             "testnet",
		VaultAddress:            "0x00000000000000000000000000000000000000bb",
		Confirmations:           0,
		StartBlock:              100,
		PollInterval:            time.Second,
		CollateralSymbol:        "USDT",
		CollateralDecimals:      6,
		CollateralDisplayDigits: 2,
	}

	firstAccount := &fakeChainAccountClient{}
	firstProcessor := NewProcessor(slog.Default(), store, firstAccount, &fakeChainPublisher{}, sharedkafka.NewTopics("funnyoption."))
	firstReader := &fakeLogReader{
		head:      1000,
		filterErr: errors.New("History has been pruned for this block"),
	}
	firstListener, err := NewDepositListenerWithReader(slog.Default(), cfg, store, firstProcessor, firstReader)
	if err != nil {
		t.Fatalf("NewDepositListenerWithReader returned error: %v", err)
	}
	if err := firstListener.pollOnce(context.Background()); err != nil {
		t.Fatalf("first pollOnce returned error: %v", err)
	}
	if store.scanCursor != 1001 {
		t.Fatalf("expected checkpoint 1001 after pruned fast-forward, got %d", store.scanCursor)
	}

	secondAccount := &fakeChainAccountClient{
		creditResult: accountclient.CreditResult{UserID: 1001, Asset: "USDT", Available: 700, Applied: true},
	}
	secondPublisher := &fakeChainPublisher{}
	secondProcessor := NewProcessor(slog.Default(), store, secondAccount, secondPublisher, sharedkafka.NewTopics("funnyoption."))
	secondReader := &fakeLogReader{
		head: 1010,
		logs: []types.Log{
			{
				Address:     common.HexToAddress("0x00000000000000000000000000000000000000bb"),
				Topics:      []common.Hash{depositEventTopic, common.BytesToHash(common.HexToAddress("0x00000000000000000000000000000000000000aa").Bytes())},
				Data:        common.LeftPadBytes(big.NewInt(7_000_000).Bytes(), 32),
				BlockNumber: 1002,
				TxHash:      common.HexToHash("0xfresh"),
				Index:       5,
			},
		},
	}
	secondListener, err := NewDepositListenerWithReader(slog.Default(), cfg, store, secondProcessor, secondReader)
	if err != nil {
		t.Fatalf("NewDepositListenerWithReader returned error: %v", err)
	}
	if err := secondListener.pollOnce(context.Background()); err != nil {
		t.Fatalf("second pollOnce returned error: %v", err)
	}
	if len(secondReader.queries) != 1 {
		t.Fatalf("expected one post-restart filter query, got %d", len(secondReader.queries))
	}
	if secondReader.queries[0].FromBlock.Uint64() != 1001 || secondReader.queries[0].ToBlock.Uint64() != 1010 {
		t.Fatalf("unexpected post-restart block range: %d-%d", secondReader.queries[0].FromBlock.Uint64(), secondReader.queries[0].ToBlock.Uint64())
	}
	if secondAccount.calls != 1 {
		t.Fatalf("expected fresh deposit credit once, got %d", secondAccount.calls)
	}
	if store.markCreditedID != "dep_fresh" {
		t.Fatalf("expected dep_fresh to be marked credited, got %s", store.markCreditedID)
	}
	if secondPublisher.topic != "funnyoption.chain.deposit" {
		t.Fatalf("unexpected publish topic: %s", secondPublisher.topic)
	}
	if store.scanCursor != 1011 {
		t.Fatalf("expected checkpoint 1011 after fresh deposit scan, got %d", store.scanCursor)
	}
}

func TestDepositListenerSkipsWalletWithoutSession(t *testing.T) {
	store := &fakeDepositStore{}
	account := &fakeChainAccountClient{}
	publisher := &fakeChainPublisher{}
	processor := NewProcessor(slog.Default(), store, account, publisher, sharedkafka.NewTopics("funnyoption."))
	reader := &fakeLogReader{
		head: 10,
		logs: []types.Log{
			{
				Address:     common.HexToAddress("0x00000000000000000000000000000000000000bb"),
				Topics:      []common.Hash{depositEventTopic, common.BytesToHash(common.HexToAddress("0x00000000000000000000000000000000000000ff").Bytes())},
				Data:        common.LeftPadBytes(big.NewInt(2_000_000).Bytes(), 32),
				BlockNumber: 4,
				TxHash:      common.HexToHash("0x2"),
				Index:       1,
			},
		},
	}
	cfg := config.ServiceConfig{
		ChainName:               "bsc",
		NetworkName:             "testnet",
		VaultAddress:            "0x00000000000000000000000000000000000000bb",
		Confirmations:           0,
		StartBlock:              1,
		PollInterval:            time.Second,
		CollateralSymbol:        "USDT",
		CollateralDecimals:      6,
		CollateralDisplayDigits: 2,
	}

	listener, err := NewDepositListenerWithReader(slog.Default(), cfg, store, processor, reader)
	if err != nil {
		t.Fatalf("NewDepositListenerWithReader returned error: %v", err)
	}

	if err := listener.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce returned error: %v", err)
	}
	if account.calls != 0 {
		t.Fatalf("expected no account credit, got %d", account.calls)
	}
}

func TestDepositListenerPollOnceDebitsWithdrawalWallet(t *testing.T) {
	store := &fakeDepositStore{
		walletUsers: map[string]int64{
			"0x00000000000000000000000000000000000000aa": 1001,
		},
		withdrawal: chainmodel.Withdrawal{
			WithdrawalID:     "0x00000000000000000000000000000000000000000000000000000000000000ff",
			UserID:           1001,
			WalletAddress:    "0x00000000000000000000000000000000000000aa",
			RecipientAddress: "0x00000000000000000000000000000000000000cc",
			VaultAddress:     "0x00000000000000000000000000000000000000bb",
			Asset:            "USDT",
			Amount:           250,
			ChainName:        "bsc",
			NetworkName:      "testnet",
			TxHash:           "0x3",
			LogIndex:         4,
			BlockNumber:      109,
			Status:           "QUEUED",
		},
	}
	account := &fakeChainAccountClient{
		debitResult: accountclient.DebitResult{UserID: 1001, Asset: "USDT", Available: 250, Applied: true},
	}
	publisher := &fakeChainPublisher{}
	processor := NewProcessor(slog.Default(), store, account, publisher, sharedkafka.NewTopics("funnyoption."))
	withdrawalID := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000ff")
	reader := &fakeLogReader{
		head: 120,
		logs: []types.Log{
			{
				Address: common.HexToAddress("0x00000000000000000000000000000000000000bb"),
				Topics: []common.Hash{
					withdrawalEventTopic,
					withdrawalID,
					common.BytesToHash(common.HexToAddress("0x00000000000000000000000000000000000000aa").Bytes()),
				},
				Data: append(
					common.LeftPadBytes(big.NewInt(2_500_000).Bytes(), 32),
					common.LeftPadBytes(common.HexToAddress("0x00000000000000000000000000000000000000cc").Bytes(), 32)...,
				),
				BlockNumber: 109,
				TxHash:      common.HexToHash("0x3"),
				Index:       4,
			},
		},
	}
	cfg := config.ServiceConfig{
		ChainName:               "bsc",
		NetworkName:             "testnet",
		VaultAddress:            "0x00000000000000000000000000000000000000bb",
		Confirmations:           6,
		StartBlock:              100,
		PollInterval:            time.Second,
		CollateralSymbol:        "USDT",
		CollateralDecimals:      6,
		CollateralDisplayDigits: 2,
	}

	listener, err := NewDepositListenerWithReader(slog.Default(), cfg, store, processor, reader)
	if err != nil {
		t.Fatalf("NewDepositListenerWithReader returned error: %v", err)
	}

	if err := listener.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce returned error: %v", err)
	}
	if account.debitCalls != 1 {
		t.Fatalf("expected account debit once, got %d", account.debitCalls)
	}
	if publisher.topic != "funnyoption.chain.withdrawal" {
		t.Fatalf("unexpected publish topic: %s", publisher.topic)
	}
}
