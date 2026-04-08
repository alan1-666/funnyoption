package service

import (
	"context"
	"log/slog"
	"strings"

	accountclient "funnyoption/internal/account/client"
	"funnyoption/internal/rollup"
	"funnyoption/internal/shared/config"
	shareddb "funnyoption/internal/shared/db"
	"funnyoption/internal/shared/grpcx"
	"funnyoption/internal/shared/health"
	sharedkafka "funnyoption/internal/shared/kafka"
)

func Run(ctx context.Context, logger *slog.Logger, cfg config.ServiceConfig) error {
	dbConn, err := shareddb.OpenPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	accountRPC, err := accountclient.NewGRPCClient(cfg.AccountGRPCAddr)
	if err != nil {
		return err
	}
	defer accountRPC.Close()

	publisher := sharedkafka.NewJSONPublisher(logger, cfg.KafkaBrokers)
	defer publisher.Close()

	store := NewSQLStore(dbConn)
	rollupStore := rollup.NewStore(dbConn)
	processor := NewProcessor(logger, store, accountRPC, publisher, cfg.KafkaTopics).WithRollup(rollupStore)

	var rpcPool *rpcPool
	if cfg.ChainRPCURL != "" {
		rpcPool, err = newRPCPool(ctx, cfg)
		if err != nil {
			return err
		}
		defer rpcPool.Close()
	}

	var listener *DepositListener
	if rpcPool != nil && cfg.VaultAddress != "" {
		listener, err = NewDepositListenerWithReader(logger, cfg, store, processor, rpcPool)
		if err != nil {
			return err
		}
		go listener.Start(ctx)
	} else {
		logger.Info("skip deposit listener bootstrap", "reason", "chain rpc url or vault address is empty")
	}

	var claimProcessor *ClaimProcessor
	if rpcPool != nil && cfg.VaultAddress != "" && strings.TrimSpace(cfg.ChainOperatorPrivateKey) != "" {
		claimProcessor, err = NewClaimProcessor(logger, cfg, store, rpcPool)
		if err != nil {
			return err
		}
		go claimProcessor.Start(ctx)
	} else {
		logger.Info("skip claim processor bootstrap", "reason", "rpc, vault address, or operator private key is empty")
	}

	var forcedWithdrawalMirror *ForcedWithdrawalMirrorProcessor
	if rpcPool != nil && strings.TrimSpace(cfg.RollupCoreAddress) != "" {
		forcedWithdrawalMirror, err = NewForcedWithdrawalMirrorProcessor(logger, cfg, store, rpcPool)
		if err != nil {
			return err
		}
		go forcedWithdrawalMirror.Start(ctx)
	} else {
		logger.Info("skip forced-withdrawal mirror bootstrap", "reason", "rpc or rollup core address is empty")
	}

	var forcedWithdrawalSatisfier *ForcedWithdrawalSatisfier
	if rpcPool != nil && strings.TrimSpace(cfg.RollupCoreAddress) != "" && strings.TrimSpace(cfg.ChainOperatorPrivateKey) != "" {
		forcedWithdrawalSatisfier, err = NewForcedWithdrawalSatisfier(logger, cfg, store, rpcPool)
		if err != nil {
			return err
		}
		go forcedWithdrawalSatisfier.Start(ctx)
	} else {
		logger.Info("skip forced-withdrawal satisfier bootstrap", "reason", "rpc, rollup core address, or operator private key is empty")
	}

	var rollupSubmissionProcessor *RollupSubmissionProcessor
	if rpcPool != nil && strings.TrimSpace(cfg.RollupCoreAddress) != "" && strings.TrimSpace(cfg.ChainOperatorPrivateKey) != "" {
		rollupSubmissionProcessor, err = NewRollupSubmissionProcessor(logger, cfg, rollupStore, rpcPool)
		if err != nil {
			return err
		}
		go rollupSubmissionProcessor.Start(ctx)
	} else {
		logger.Info("skip rollup submission processor bootstrap", "reason", "rpc, rollup core address, or operator private key is empty")
	}

	health.ListenAndServe(ctx, logger, cfg.HTTPAddr, cfg.Name, cfg.Env)

	logger.Info(
		"chain service bootstrapped",
		"service", cfg.Name,
		"env", cfg.Env,
		"grpc_addr", cfg.GRPCAddr,
		"account_grpc_addr", cfg.AccountGRPCAddr,
		"chain_name", cfg.ChainName,
		"network_name", cfg.NetworkName,
		"chain_rpc_url", cfg.ChainRPCURL,
		"vault_address", cfg.VaultAddress,
		"confirmations", cfg.Confirmations,
		"start_block", cfg.StartBlock,
		"poll_interval", cfg.PollInterval,
		"chain_deposit_topic", cfg.KafkaTopics.ChainDeposit,
		"chain_withdraw_topic", cfg.KafkaTopics.ChainWithdraw,
		"deposit_listener_ready", listener != nil,
		"claim_processor_ready", claimProcessor != nil,
		"forced_withdrawal_mirror_ready", forcedWithdrawalMirror != nil,
		"forced_withdrawal_satisfier_ready", forcedWithdrawalSatisfier != nil,
		"rollup_submission_ready", rollupSubmissionProcessor != nil,
		"processor_ready", processor != nil,
	)
	return grpcx.Run(ctx, logger, cfg.Name, cfg.GRPCAddr, nil)
}
