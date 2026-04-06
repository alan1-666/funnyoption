package service

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"funnyoption/internal/rollup"
	"funnyoption/internal/shared/assets"
	"funnyoption/internal/shared/config"
	shareddb "funnyoption/internal/shared/db"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const rollupEscapeHatchABIJSON = `[
  {
    "type":"function",
    "name":"requestForcedWithdrawal",
    "stateMutability":"nonpayable",
    "inputs":[
      {"name":"recipient","type":"address"},
      {"name":"amount","type":"uint256"}
    ],
    "outputs":[{"name":"requestId","type":"uint64"}]
  },
  {
    "type":"function",
    "name":"freezeForMissedForcedWithdrawal",
    "stateMutability":"nonpayable",
    "inputs":[{"name":"requestId","type":"uint64"}],
    "outputs":[]
  },
  {
    "type":"function",
    "name":"claimEscapeCollateral",
    "stateMutability":"nonpayable",
    "inputs":[
      {"name":"batchId","type":"uint64"},
      {"name":"leafIndex","type":"uint64"},
      {"name":"amount","type":"uint256"},
      {"name":"recipient","type":"address"},
      {"name":"proof","type":"bytes32[]"}
    ],
    "outputs":[]
  }
]`

type EscapeHatchProgress struct {
	Action    string                                    `json:"action"`
	RequestID uint64                                    `json:"request_id,omitempty"`
	Wallet    string                                    `json:"wallet,omitempty"`
	Recipient string                                    `json:"recipient,omitempty"`
	Amount    int64                                     `json:"amount,omitempty"`
	TxHash    string                                    `json:"tx_hash,omitempty"`
	Claim     rollup.AcceptedEscapeCollateralLeafRecord `json:"claim,omitempty"`
	Root      rollup.AcceptedEscapeCollateralRootRecord `json:"root,omitempty"`
	Frozen    bool                                      `json:"frozen,omitempty"`
	FrozenAt  uint64                                    `json:"frozen_at,omitempty"`
	Note      string                                    `json:"note,omitempty"`
}

type escapeHatchRunner struct {
	logger      *slog.Logger
	cfg         config.ServiceConfig
	store       *rollup.Store
	sender      rollupTxSender
	contractABI abi.ABI
	rollupCore  common.Address
}

func RunRequestForcedWithdrawalOnce(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.ServiceConfig,
	walletPrivateKey string,
	amountAccounting int64,
	recipient string,
) (EscapeHatchProgress, error) {
	return withEscapeHatchRunner(ctx, logger, cfg, func(ctx context.Context, runner *escapeHatchRunner) (EscapeHatchProgress, error) {
		privateKey, fromAddress, err := parseEscapeWalletKey(walletPrivateKey)
		if err != nil {
			return EscapeHatchProgress{}, err
		}
		recipientAddress, err := normalizeEscapeRecipient(fromAddress, recipient)
		if err != nil {
			return EscapeHatchProgress{}, err
		}
		if amountAccounting <= 0 {
			return EscapeHatchProgress{}, fmt.Errorf("amount must be positive")
		}
		chainAmount, err := assets.AccountingToChainAmount(amountAccounting, cfg.CollateralDecimals, cfg.CollateralDisplayDigits)
		if err != nil {
			return EscapeHatchProgress{}, err
		}
		if chainAmount <= 0 {
			return EscapeHatchProgress{}, fmt.Errorf("amount resolves to non-positive chain amount")
		}

		txHash, err := runner.sendEscapeTx(
			ctx,
			privateKey,
			fromAddress,
			"requestForcedWithdrawal",
			recipientAddress,
			big.NewInt(chainAmount),
		)
		if err != nil {
			return EscapeHatchProgress{}, err
		}
		if err := runner.waitForSuccessfulReceipt(ctx, txHash); err != nil {
			return EscapeHatchProgress{}, err
		}

		requestCount, err := (&RollupSubmissionProcessor{
			sender:      runner.sender,
			fromAddress: fromAddress,
			rollupCore:  runner.rollupCore,
		}).loadForcedWithdrawalRequestCount(ctx, nil)
		if err != nil {
			return EscapeHatchProgress{}, err
		}

		return EscapeHatchProgress{
			Action:    "FORCED_WITHDRAWAL_REQUESTED",
			RequestID: requestCount,
			Wallet:    fromAddress.Hex(),
			Recipient: recipientAddress.Hex(),
			Amount:    chainAmount,
			TxHash:    txHash,
		}, nil
	})
}

func RunFreezeForcedWithdrawalOnce(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.ServiceConfig,
	requestID uint64,
) (EscapeHatchProgress, error) {
	return withEscapeHatchRunner(ctx, logger, cfg, func(ctx context.Context, runner *escapeHatchRunner) (EscapeHatchProgress, error) {
		if requestID == 0 {
			return EscapeHatchProgress{}, fmt.Errorf("request_id must be positive")
		}
		privateKey, fromAddress, err := parseEscapeWalletKey(cfg.ChainOperatorPrivateKey)
		if err != nil {
			return EscapeHatchProgress{}, err
		}

		txHash, err := runner.sendEscapeTx(ctx, privateKey, fromAddress, "freezeForMissedForcedWithdrawal", requestID)
		if err != nil {
			return EscapeHatchProgress{}, err
		}
		if err := runner.waitForSuccessfulReceipt(ctx, txHash); err != nil {
			return EscapeHatchProgress{}, err
		}

		freezeState, err := (&RollupSubmissionProcessor{
			sender:      runner.sender,
			fromAddress: fromAddress,
			rollupCore:  runner.rollupCore,
		}).loadFreezeState(ctx, nil)
		if err != nil {
			return EscapeHatchProgress{}, err
		}

		return EscapeHatchProgress{
			Action:    "ROLLUP_FROZEN",
			RequestID: requestID,
			TxHash:    txHash,
			Frozen:    freezeState.Frozen,
			FrozenAt:  freezeState.FrozenAt,
		}, nil
	})
}

func RunClaimEscapeCollateralOnce(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.ServiceConfig,
	walletPrivateKey string,
	accountID int64,
	claimID string,
	recipient string,
) (EscapeHatchProgress, error) {
	return withEscapeHatchRunner(ctx, logger, cfg, func(ctx context.Context, runner *escapeHatchRunner) (EscapeHatchProgress, error) {
		privateKey, fromAddress, err := parseEscapeWalletKey(walletPrivateKey)
		if err != nil {
			return EscapeHatchProgress{}, err
		}
		recipientAddress, err := normalizeEscapeRecipient(fromAddress, recipient)
		if err != nil {
			return EscapeHatchProgress{}, err
		}

		root, leaf, ok, err := runner.store.GetLatestAnchoredEscapeCollateralClaim(ctx, accountID, fromAddress.Hex(), claimID)
		if err != nil {
			return EscapeHatchProgress{}, err
		}
		if !ok {
			return EscapeHatchProgress{}, fmt.Errorf("no anchored escape collateral claim found for wallet %s", strings.ToLower(fromAddress.Hex()))
		}
		if !rollup.VerifyAcceptedEscapeCollateralProof(root.MerkleRoot, leaf.LeafHash, leaf.ProofHashes, leaf.LeafIndex) {
			return EscapeHatchProgress{}, fmt.Errorf("escape collateral proof does not verify for claim %s", leaf.ClaimID)
		}

		switch leaf.ClaimStatus {
		case rollup.EscapeCollateralClaimStatusClaimed:
			return EscapeHatchProgress{
				Action:    "ESCAPE_COLLATERAL_ALREADY_CLAIMED",
				Wallet:    fromAddress.Hex(),
				Recipient: recipientAddress.Hex(),
				Amount:    leaf.ClaimAmount,
				Claim:     leaf,
				Root:      root,
			}, nil
		case rollup.EscapeCollateralClaimStatusSubmitted:
			progress, err := runner.reconcileSubmittedEscapeClaim(ctx, leaf)
			if err != nil {
				return EscapeHatchProgress{}, err
			}
			progress.Wallet = fromAddress.Hex()
			progress.Recipient = recipientAddress.Hex()
			progress.Root = root
			return progress, nil
		}

		txHash, err := runner.submitEscapeCollateralClaim(ctx, privateKey, fromAddress, recipientAddress, root, leaf)
		if err != nil {
			updated, markErr := runner.store.MarkEscapeCollateralClaimFailed(ctx, leaf.ClaimID, err.Error())
			if markErr == nil {
				leaf = updated
			}
			return EscapeHatchProgress{
				Action:    "ESCAPE_COLLATERAL_CLAIM_FAILED",
				Wallet:    fromAddress.Hex(),
				Recipient: recipientAddress.Hex(),
				Amount:    leaf.ClaimAmount,
				TxHash:    "",
				Claim:     leaf,
				Root:      root,
				Note:      err.Error(),
			}, nil
		}
		leaf, err = runner.store.MarkEscapeCollateralClaimSubmitted(ctx, leaf.ClaimID, txHash)
		if err != nil {
			return EscapeHatchProgress{}, err
		}
		progress, err := runner.reconcileSubmittedEscapeClaim(ctx, leaf)
		if err != nil {
			return EscapeHatchProgress{}, err
		}
		progress.Wallet = fromAddress.Hex()
		progress.Recipient = recipientAddress.Hex()
		progress.Root = root
		return progress, nil
	})
}

func withEscapeHatchRunner(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.ServiceConfig,
	fn func(context.Context, *escapeHatchRunner) (EscapeHatchProgress, error),
) (EscapeHatchProgress, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if strings.TrimSpace(cfg.RollupCoreAddress) == "" {
		return EscapeHatchProgress{}, fmt.Errorf("rollup core address is required")
	}

	dbConn, err := shareddb.OpenPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		return EscapeHatchProgress{}, err
	}
	defer dbConn.Close()

	rpcPool, err := newRPCPool(ctx, cfg)
	if err != nil {
		return EscapeHatchProgress{}, err
	}
	defer rpcPool.Close()

	contractABI, err := abi.JSON(strings.NewReader(rollupEscapeHatchABIJSON))
	if err != nil {
		return EscapeHatchProgress{}, err
	}
	rollupCore, err := validateClaimAddress("rollup_core_address", cfg.RollupCoreAddress)
	if err != nil {
		return EscapeHatchProgress{}, err
	}

	return fn(ctx, &escapeHatchRunner{
		logger:      logger,
		cfg:         cfg,
		store:       rollup.NewStore(dbConn),
		sender:      rpcPool,
		contractABI: contractABI,
		rollupCore:  rollupCore,
	})
}

func parseEscapeWalletKey(privateKeyHex string) (*ecdsa.PrivateKey, common.Address, error) {
	if strings.TrimSpace(privateKeyHex) == "" {
		return nil, common.Address{}, fmt.Errorf("wallet private key is required")
	}
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x"))
	if err != nil {
		return nil, common.Address{}, err
	}
	return privateKey, crypto.PubkeyToAddress(privateKey.PublicKey), nil
}

func normalizeEscapeRecipient(defaultAddress common.Address, recipient string) (common.Address, error) {
	if strings.TrimSpace(recipient) == "" {
		return defaultAddress, nil
	}
	return validateClaimAddress("recipient", recipient)
}

func (r *escapeHatchRunner) submitEscapeCollateralClaim(
	ctx context.Context,
	privateKey *ecdsa.PrivateKey,
	fromAddress common.Address,
	recipientAddress common.Address,
	root rollup.AcceptedEscapeCollateralRootRecord,
	leaf rollup.AcceptedEscapeCollateralLeafRecord,
) (string, error) {
	proof := make([]common.Hash, 0, len(leaf.ProofHashes))
	for _, item := range leaf.ProofHashes {
		proof = append(proof, common.HexToHash(item))
	}
	return r.sendEscapeTx(
		ctx,
		privateKey,
		fromAddress,
		"claimEscapeCollateral",
		uint64(root.BatchID),
		uint64(leaf.LeafIndex),
		big.NewInt(leaf.ClaimAmount),
		recipientAddress,
		proof,
	)
}

func (r *escapeHatchRunner) reconcileSubmittedEscapeClaim(
	ctx context.Context,
	leaf rollup.AcceptedEscapeCollateralLeafRecord,
) (EscapeHatchProgress, error) {
	txHash := normalizeChainTxHash(leaf.ClaimTxHash)
	if txHash == "" {
		updated, err := r.store.MarkEscapeCollateralClaimFailed(ctx, leaf.ClaimID, "submitted escape claim is missing tx hash")
		if err != nil {
			return EscapeHatchProgress{}, err
		}
		return EscapeHatchProgress{
			Action: "ESCAPE_COLLATERAL_CLAIM_FAILED",
			Amount: updated.ClaimAmount,
			Claim:  updated,
			Note:   updated.LastError,
		}, nil
	}

	receipt, err := r.sender.TransactionReceipt(ctx, common.HexToHash("0x"+txHash))
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return EscapeHatchProgress{
				Action: "ESCAPE_COLLATERAL_CLAIM_PENDING",
				Amount: leaf.ClaimAmount,
				TxHash: "0x" + txHash,
				Claim:  leaf,
				Note:   "escape collateral claim transaction is still pending",
			}, nil
		}
		return EscapeHatchProgress{}, err
	}
	if receipt == nil {
		return EscapeHatchProgress{
			Action: "ESCAPE_COLLATERAL_CLAIM_PENDING",
			Amount: leaf.ClaimAmount,
			TxHash: "0x" + txHash,
			Claim:  leaf,
			Note:   "escape collateral claim transaction is still pending",
		}, nil
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		updated, err := r.store.MarkEscapeCollateralClaimFailed(ctx, leaf.ClaimID, "escape collateral claim reverted onchain")
		if err != nil {
			return EscapeHatchProgress{}, err
		}
		return EscapeHatchProgress{
			Action: "ESCAPE_COLLATERAL_CLAIM_FAILED",
			Amount: updated.ClaimAmount,
			TxHash: "0x" + txHash,
			Claim:  updated,
			Note:   updated.LastError,
		}, nil
	}
	updated, err := r.store.MarkEscapeCollateralClaimClaimed(ctx, leaf.ClaimID, txHash)
	if err != nil {
		return EscapeHatchProgress{}, err
	}
	return EscapeHatchProgress{
		Action: "ESCAPE_COLLATERAL_CLAIMED",
		Amount: updated.ClaimAmount,
		TxHash: "0x" + txHash,
		Claim:  updated,
	}, nil
}

func (r *escapeHatchRunner) sendEscapeTx(
	ctx context.Context,
	privateKey *ecdsa.PrivateKey,
	fromAddress common.Address,
	method string,
	args ...any,
) (string, error) {
	data, err := r.contractABI.Pack(method, args...)
	if err != nil {
		return "", err
	}

	chainID, err := r.sender.ChainID(ctx)
	if err != nil {
		return "", err
	}
	nonce, err := r.sender.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return "", err
	}
	gasPrice, err := r.sender.SuggestGasPrice(ctx)
	if err != nil {
		return "", err
	}

	gasLimit := r.cfg.ChainGasLimit
	if gasLimit == 0 {
		gasLimit = 250000
	}
	estimatedGas, err := r.sender.EstimateGas(ctx, ethereum.CallMsg{
		From:     fromAddress,
		To:       &r.rollupCore,
		GasPrice: gasPrice,
		Data:     data,
	})
	if err == nil && estimatedGas > 0 {
		gasLimit = estimatedGas + estimatedGas/5
	}

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &r.rollupCore,
		Value:    big.NewInt(0),
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", err
	}
	if err := r.sender.SendTransaction(ctx, signedTx); err != nil {
		return "", err
	}
	return normalizeChainTxHash(signedTx.Hash().Hex()), nil
}

func (r *escapeHatchRunner) waitForSuccessfulReceipt(ctx context.Context, txHash string) error {
	hash := common.HexToHash("0x" + normalizeChainTxHash(txHash))
	for {
		receipt, err := r.sender.TransactionReceipt(ctx, hash)
		if err != nil {
			if errors.Is(err, ethereum.NotFound) {
				if err := sleepWithContext(ctx, 500*time.Millisecond); err != nil {
					return err
				}
				continue
			}
			return err
		}
		if receipt == nil {
			if err := sleepWithContext(ctx, 500*time.Millisecond); err != nil {
				return err
			}
			continue
		}
		if receipt.Status != types.ReceiptStatusSuccessful {
			return fmt.Errorf("transaction reverted onchain")
		}
		return nil
	}
}
