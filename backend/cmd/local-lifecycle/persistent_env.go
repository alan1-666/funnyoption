package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"funnyoption/internal/shared/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const erc20ApproveABIJSON = `[
  {
    "inputs": [
      { "internalType": "address", "name": "spender", "type": "address" },
      { "internalType": "uint256", "name": "amount", "type": "uint256" }
    ],
    "name": "approve",
    "outputs": [{ "internalType": "bool", "name": "", "type": "bool" }],
    "stateMutability": "nonpayable",
    "type": "function"
  }
]`

const vaultDepositABIJSON = `[
  {
    "inputs": [
      { "internalType": "uint256", "name": "amount", "type": "uint256" }
    ],
    "name": "deposit",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  }
]`

type lifecycleDepositEnvironment interface {
	Close()
	depositChainID() int64
	summary() proofEnvironmentSummary
	submitDeposit(ctx context.Context, buyer walletIdentity, amount int64) (string, error)
}

type persistentLocalChainEnvironment struct {
	client                *ethclient.Client
	chainID               int64
	chainName             string
	networkName           string
	vaultAddress          common.Address
	tokenAddress          common.Address
	listenerStartBlock    uint64
	listenerConfirmations uint64
}

func buildDepositEnvironment(ctx context.Context, cfg config.ServiceConfig, buyer walletIdentity) (lifecycleDepositEnvironment, error) {
	if shouldUsePersistentLocalChain(cfg) {
		return newPersistentLocalChainEnvironment(ctx, cfg)
	}
	return newListenerProofEnvironment(ctx, buyer)
}

func shouldUsePersistentLocalChain(cfg config.ServiceConfig) bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("FUNNYOPTION_LOCAL_CHAIN_MODE")), "anvil") &&
		strings.TrimSpace(cfg.ChainRPCURL) != "" &&
		strings.TrimSpace(cfg.VaultAddress) != "" &&
		strings.TrimSpace(os.Getenv("FUNNYOPTION_COLLATERAL_TOKEN_ADDRESS")) != ""
}

func newPersistentLocalChainEnvironment(ctx context.Context, cfg config.ServiceConfig) (*persistentLocalChainEnvironment, error) {
	tokenAddress := strings.TrimSpace(os.Getenv("FUNNYOPTION_COLLATERAL_TOKEN_ADDRESS"))
	if tokenAddress == "" {
		return nil, fmt.Errorf("FUNNYOPTION_COLLATERAL_TOKEN_ADDRESS is required for persistent local chain mode")
	}
	client, err := ethclient.DialContext(ctx, cfg.ChainRPCURL)
	if err != nil {
		return nil, fmt.Errorf("dial local chain rpc: %w", err)
	}

	chainID, err := client.ChainID(ctx)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("read local chain id: %w", err)
	}
	if cfg.ChainID > 0 && chainID.Int64() != cfg.ChainID {
		client.Close()
		return nil, fmt.Errorf("chain id mismatch: rpc=%d cfg=%d", chainID.Int64(), cfg.ChainID)
	}

	startBlock := uint64(0)
	if cfg.StartBlock > 0 {
		startBlock = uint64(cfg.StartBlock)
	}
	confirmations := uint64(0)
	if cfg.Confirmations > 0 {
		confirmations = uint64(cfg.Confirmations)
	}

	return &persistentLocalChainEnvironment{
		client:                client,
		chainID:               chainID.Int64(),
		chainName:             cfg.ChainName,
		networkName:           cfg.NetworkName,
		vaultAddress:          common.HexToAddress(cfg.VaultAddress),
		tokenAddress:          common.HexToAddress(tokenAddress),
		listenerStartBlock:    startBlock,
		listenerConfirmations: confirmations,
	}, nil
}

func (e *persistentLocalChainEnvironment) Close() {
	if e != nil && e.client != nil {
		e.client.Close()
	}
}

func (e *persistentLocalChainEnvironment) depositChainID() int64 {
	return e.chainID
}

func (e *persistentLocalChainEnvironment) submitDeposit(ctx context.Context, buyer walletIdentity, amount int64) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("deposit amount must be positive")
	}

	tokenABI, err := abi.JSON(strings.NewReader(erc20ApproveABIJSON))
	if err != nil {
		return "", fmt.Errorf("parse erc20 approve abi: %w", err)
	}
	vaultABI, err := abi.JSON(strings.NewReader(vaultDepositABIJSON))
	if err != nil {
		return "", fmt.Errorf("parse vault deposit abi: %w", err)
	}

	approveAuth, err := bind.NewKeyedTransactorWithChainID(buyer.PrivateKey, big.NewInt(e.chainID))
	if err != nil {
		return "", fmt.Errorf("create approve auth: %w", err)
	}
	approveAuth.Context = ctx
	approveAuth.GasLimit = 200_000

	tokenContract := bind.NewBoundContract(e.tokenAddress, tokenABI, e.client, e.client, e.client)
	approveTx, err := tokenContract.Transact(approveAuth, "approve", e.vaultAddress, big.NewInt(amount))
	if err != nil {
		return "", fmt.Errorf("approve collateral token: %w", err)
	}
	if err := waitForLifecycleReceipt(ctx, e.client, approveTx.Hash()); err != nil {
		return "", fmt.Errorf("wait approve receipt: %w", err)
	}

	depositAuth, err := bind.NewKeyedTransactorWithChainID(buyer.PrivateKey, big.NewInt(e.chainID))
	if err != nil {
		return "", fmt.Errorf("create deposit auth: %w", err)
	}
	depositAuth.Context = ctx
	depositAuth.GasLimit = 250_000

	vaultContract := bind.NewBoundContract(e.vaultAddress, vaultABI, e.client, e.client, e.client)
	depositTx, err := vaultContract.Transact(depositAuth, "deposit", big.NewInt(amount))
	if err != nil {
		return "", fmt.Errorf("submit deposit tx: %w", err)
	}
	if err := waitForLifecycleReceipt(ctx, e.client, depositTx.Hash()); err != nil {
		return "", fmt.Errorf("wait deposit receipt: %w", err)
	}
	return depositTx.Hash().Hex(), nil
}

func (e *persistentLocalChainEnvironment) summary() proofEnvironmentSummary {
	return proofEnvironmentSummary{
		Mode:                  "persistent-local-anvil",
		ChainID:               e.chainID,
		ChainName:             e.chainName,
		NetworkName:           e.networkName,
		VaultAddress:          strings.ToLower(e.vaultAddress.Hex()),
		ListenerStartBlock:    e.listenerStartBlock,
		ListenerConfirmations: e.listenerConfirmations,
	}
}

func waitForLifecycleReceipt(ctx context.Context, client *ethclient.Client, txHash common.Hash) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err == nil {
			if receipt.Status != types.ReceiptStatusSuccessful {
				return fmt.Errorf("transaction %s reverted", txHash.Hex())
			}
			return nil
		}
		if err != nil && err != ethereum.NotFound {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
