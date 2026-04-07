package main

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestListenerProofEnvironmentEmitsCanonicalDepositLog(t *testing.T) {
	key, err := crypto.HexToECDSA("59c6995e998f97a5a004497e5daef0d4f7dcd0cfd5401397dbeed52b21965b1d")
	if err != nil {
		t.Fatalf("parse buyer key: %v", err)
	}
	buyer := walletIdentity{
		Label:      "buyer",
		UserID:     1001,
		PrivateKey: key,
		Address:    strings.ToLower(crypto.PubkeyToAddress(key.PublicKey).Hex()),
	}

	ctx := context.Background()
	env, err := newListenerProofEnvironment(ctx, buyer)
	if err != nil {
		t.Fatalf("newListenerProofEnvironment returned error: %v", err)
	}
	defer env.Close()

	txHash, err := env.submitDeposit(ctx, buyer, 5000)
	if err != nil {
		t.Fatalf("submitDeposit returned error: %v", err)
	}

	receipt, err := env.backend.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		t.Fatalf("TransactionReceipt returned error: %v", err)
	}
	if receipt.Status != 1 {
		t.Fatalf("expected successful receipt, got status %d", receipt.Status)
	}
	if len(receipt.Logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(receipt.Logs))
	}

	logEntry := receipt.Logs[0]
	expectedTopic := crypto.Keccak256Hash([]byte("Deposited(address,uint256)"))
	if logEntry.Address != env.vaultAddress {
		t.Fatalf("expected log address %s, got %s", env.vaultAddress.Hex(), logEntry.Address.Hex())
	}
	if len(logEntry.Topics) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(logEntry.Topics))
	}
	if logEntry.Topics[0] != expectedTopic {
		t.Fatalf("expected signature topic %s, got %s", expectedTopic.Hex(), logEntry.Topics[0].Hex())
	}
	if common.BytesToAddress(logEntry.Topics[1].Bytes()) != common.HexToAddress(buyer.Address) {
		t.Fatalf("expected wallet topic %s, got %s", buyer.Address, common.BytesToAddress(logEntry.Topics[1].Bytes()).Hex())
	}
	if amount := new(big.Int).SetBytes(logEntry.Data).Int64(); amount != 5000 {
		t.Fatalf("expected amount 5000, got %d", amount)
	}
}
