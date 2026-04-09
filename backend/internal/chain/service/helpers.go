package service

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/net/context"
)

type logReader interface {
	BlockNumber(ctx context.Context) (uint64, error)
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
	Close()
}

func normalizeChainName(chainName string) string {
	trimmed := strings.ToLower(strings.TrimSpace(chainName))
	if trimmed == "" {
		return "bsc"
	}
	return trimmed
}

func normalizeNetworkName(networkName string) string {
	trimmed := strings.ToLower(strings.TrimSpace(networkName))
	if trimmed == "" {
		return "testnet"
	}
	return trimmed
}

func normalizeChainTxHash(txHash string) string {
	trimmed := strings.ToLower(strings.TrimSpace(txHash))
	return strings.TrimPrefix(trimmed, "0x")
}

func validateClaimAddress(field, value string) (common.Address, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return common.Address{}, fmt.Errorf("%s is required", field)
	}
	if !common.IsHexAddress(trimmed) {
		return common.Address{}, fmt.Errorf("%s must be a valid EVM address", field)
	}
	address := common.HexToAddress(trimmed)
	if address == (common.Address{}) {
		return common.Address{}, fmt.Errorf("%s must not be zero address", field)
	}
	return address, nil
}
