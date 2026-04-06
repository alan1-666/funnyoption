package service

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"funnyoption/internal/shared/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type txClient interface {
	logReader
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error)
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
	SendTransaction(ctx context.Context, tx *types.Transaction) error
	ChainID(ctx context.Context) (*big.Int, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

type rpcPool struct {
	clients []txClient
}

func newRPCPool(ctx context.Context, cfg config.ServiceConfig) (*rpcPool, error) {
	urls := make([]string, 0, 1+len(cfg.ChainRPCFallbackURLs))
	if primary := strings.TrimSpace(cfg.ChainRPCURL); primary != "" {
		urls = append(urls, primary)
	}
	for _, item := range cfg.ChainRPCFallbackURLs {
		if trimmed := strings.TrimSpace(item); trimmed != "" && trimmed != cfg.ChainRPCURL {
			urls = append(urls, trimmed)
		}
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("chain rpc url is required")
	}

	clients := make([]txClient, 0, len(urls))
	for _, url := range urls {
		client, err := ethclient.DialContext(ctx, url)
		if err != nil {
			for _, opened := range clients {
				opened.Close()
			}
			return nil, err
		}
		clients = append(clients, &ethRPCClient{client: client})
	}
	return &rpcPool{clients: clients}, nil
}

func (p *rpcPool) Close() {
	for _, client := range p.clients {
		client.Close()
	}
}

func (p *rpcPool) withClient(fn func(client txClient) error) error {
	var lastErr error
	for _, client := range p.clients {
		if err := fn(client); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("rpc pool has no clients")
}

func (p *rpcPool) BlockNumber(ctx context.Context) (uint64, error) {
	var head uint64
	err := p.withClient(func(client txClient) error {
		value, err := client.BlockNumber(ctx)
		if err != nil {
			return err
		}
		head = value
		return nil
	})
	return head, err
}

func (p *rpcPool) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	var logs []types.Log
	err := p.withClient(func(client txClient) error {
		value, err := client.FilterLogs(ctx, q)
		if err != nil {
			return err
		}
		logs = value
		return nil
	})
	return logs, err
}

func (p *rpcPool) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	var nonce uint64
	err := p.withClient(func(client txClient) error {
		value, err := client.PendingNonceAt(ctx, account)
		if err != nil {
			return err
		}
		nonce = value
		return nil
	})
	return nonce, err
}

func (p *rpcPool) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	var price *big.Int
	err := p.withClient(func(client txClient) error {
		value, err := client.SuggestGasPrice(ctx)
		if err != nil {
			return err
		}
		price = value
		return nil
	})
	return price, err
}

func (p *rpcPool) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	var gas uint64
	err := p.withClient(func(client txClient) error {
		value, err := client.EstimateGas(ctx, call)
		if err != nil {
			return err
		}
		gas = value
		return nil
	})
	return gas, err
}

func (p *rpcPool) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	var result []byte
	err := p.withClient(func(client txClient) error {
		value, err := client.CallContract(ctx, call, blockNumber)
		if err != nil {
			return err
		}
		result = value
		return nil
	})
	return result, err
}

func (p *rpcPool) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return p.withClient(func(client txClient) error {
		return client.SendTransaction(ctx, tx)
	})
}

func (p *rpcPool) ChainID(ctx context.Context) (*big.Int, error) {
	var chainID *big.Int
	err := p.withClient(func(client txClient) error {
		value, err := client.ChainID(ctx)
		if err != nil {
			return err
		}
		chainID = value
		return nil
	})
	return chainID, err
}

func (p *rpcPool) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	var receipt *types.Receipt
	err := p.withClient(func(client txClient) error {
		value, err := client.TransactionReceipt(ctx, txHash)
		if err != nil {
			return err
		}
		receipt = value
		return nil
	})
	return receipt, err
}

type ethRPCClient struct {
	client *ethclient.Client
}

func (c *ethRPCClient) BlockNumber(ctx context.Context) (uint64, error) {
	return c.client.BlockNumber(ctx)
}

func (c *ethRPCClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	return c.client.FilterLogs(ctx, q)
}

func (c *ethRPCClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return c.client.PendingNonceAt(ctx, account)
}

func (c *ethRPCClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return c.client.SuggestGasPrice(ctx)
}

func (c *ethRPCClient) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	return c.client.EstimateGas(ctx, call)
}

func (c *ethRPCClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return c.client.CallContract(ctx, call, blockNumber)
}

func (c *ethRPCClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return c.client.SendTransaction(ctx, tx)
}

func (c *ethRPCClient) ChainID(ctx context.Context) (*big.Int, error) {
	return c.client.ChainID(ctx)
}

func (c *ethRPCClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return c.client.TransactionReceipt(ctx, txHash)
}

func (c *ethRPCClient) Close() {
	if c.client != nil {
		c.client.Close()
	}
}
