package main

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"funnyoption/internal/shared/config"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

const mockVaultABIJSON = `[
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

type proofEnvironmentSummary struct {
	Mode                  string `json:"mode"`
	ChainID               int64  `json:"chain_id"`
	ChainName             string `json:"chain_name"`
	NetworkName           string `json:"network_name"`
	VaultAddress          string `json:"vault_address"`
	ListenerStartBlock    uint64 `json:"listener_start_block"`
	ListenerConfirmations uint64 `json:"listener_confirmations"`
}

type listenerProofEnvironment struct {
	backend               *backends.SimulatedBackend
	vaultABI              abi.ABI
	vaultAddress          common.Address
	chainID               int64
	chainName             string
	networkName           string
	listenerStartBlock    uint64
	listenerConfirmations uint64
}

type proofLogReader struct {
	*backends.SimulatedBackend
}

func (r proofLogReader) Close() {
	if r.SimulatedBackend != nil {
		_ = r.SimulatedBackend.Close()
	}
}

func newListenerProofEnvironment(ctx context.Context, buyer walletIdentity) (*listenerProofEnvironment, error) {
	vaultABI, err := abi.JSON(strings.NewReader(mockVaultABIJSON))
	if err != nil {
		return nil, fmt.Errorf("parse mock vault ABI: %w", err)
	}

	buyerAddress := common.HexToAddress(buyer.Address)
	backend := backends.NewSimulatedBackend(types.GenesisAlloc{
		buyerAddress: {Balance: new(big.Int).Mul(big.NewInt(10), big.NewInt(params.Ether))},
	}, 15_000_000)

	chainID, err := backend.ChainID(ctx)
	if err != nil {
		backend.Close()
		return nil, fmt.Errorf("read proof chain id: %w", err)
	}

	deployAuth, err := bind.NewKeyedTransactorWithChainID(buyer.PrivateKey, chainID)
	if err != nil {
		backend.Close()
		return nil, fmt.Errorf("create deploy auth: %w", err)
	}
	deployAuth.Context = ctx
	deployAuth.GasLimit = 200_000

	emptyABI, err := abi.JSON(strings.NewReader("[]"))
	if err != nil {
		backend.Close()
		return nil, fmt.Errorf("parse empty ABI: %w", err)
	}

	vaultAddress, _, _, err := bind.DeployContract(deployAuth, emptyABI, mockVaultInitCode(), backend)
	if err != nil {
		backend.Close()
		return nil, fmt.Errorf("deploy proof vault: %w", err)
	}
	backend.Commit()

	head, err := backend.BlockNumber(ctx)
	if err != nil {
		backend.Close()
		return nil, fmt.Errorf("read proof chain head: %w", err)
	}

	return &listenerProofEnvironment{
		backend:               backend,
		vaultABI:              vaultABI,
		vaultAddress:          vaultAddress,
		chainID:               chainID.Int64(),
		chainName:             "simulated",
		networkName:           "local-proof",
		listenerStartBlock:    head + 1,
		listenerConfirmations: 0,
	}, nil
}

func (e *listenerProofEnvironment) Close() {
	if e != nil && e.backend != nil {
		_ = e.backend.Close()
	}
}

func (e *listenerProofEnvironment) logReader() proofLogReader {
	return proofLogReader{SimulatedBackend: e.backend}
}

func (e *listenerProofEnvironment) listenerConfig(base config.ServiceConfig) config.ServiceConfig {
	cfg := base
	cfg.ChainName = e.chainName
	cfg.NetworkName = e.networkName
	cfg.VaultAddress = e.vaultAddress.Hex()
	cfg.Confirmations = int64(e.listenerConfirmations)
	cfg.StartBlock = int64(e.listenerStartBlock)
	cfg.PollInterval = 100 * time.Millisecond
	return cfg
}

func (e *listenerProofEnvironment) submitDeposit(ctx context.Context, buyer walletIdentity, amount int64) (string, error) {
	depositAuth, err := bind.NewKeyedTransactorWithChainID(buyer.PrivateKey, big.NewInt(e.chainID))
	if err != nil {
		return "", fmt.Errorf("create deposit auth: %w", err)
	}
	depositAuth.Context = ctx
	depositAuth.GasLimit = 150_000

	contract := bind.NewBoundContract(e.vaultAddress, e.vaultABI, e.backend, e.backend, e.backend)
	tx, err := contract.Transact(depositAuth, "deposit", big.NewInt(amount))
	if err != nil {
		return "", fmt.Errorf("submit deposit tx: %w", err)
	}
	e.backend.Commit()
	return tx.Hash().Hex(), nil
}

func (e *listenerProofEnvironment) summary() proofEnvironmentSummary {
	return proofEnvironmentSummary{
		Mode:                  "listener-driven-local-proof",
		ChainID:               e.chainID,
		ChainName:             e.chainName,
		NetworkName:           e.networkName,
		VaultAddress:          strings.ToLower(e.vaultAddress.Hex()),
		ListenerStartBlock:    e.listenerStartBlock,
		ListenerConfirmations: e.listenerConfirmations,
	}
}

func (e *listenerProofEnvironment) depositChainID() int64 {
	return e.chainID
}

func mockVaultInitCode() []byte {
	runtime := mockVaultRuntimeCode()
	initCode := []byte{
		0x60, byte(len(runtime)),
		0x60, 0x0c,
		0x60, 0x00,
		0x39,
		0x60, byte(len(runtime)),
		0x60, 0x00,
		0xf3,
	}
	return append(initCode, runtime...)
}

func mockVaultRuntimeCode() []byte {
	eventTopic := crypto.Keccak256Hash([]byte("Deposited(address,uint256)")).Bytes()
	code := []byte{
		0x60, 0x04,
		0x35,
		0x60, 0x00,
		0x52,
		0x33,
		0x7f,
	}
	code = append(code, eventTopic...)
	code = append(code,
		0x60, 0x20,
		0x60, 0x00,
		0xa2,
		0x00,
	)
	return code
}
