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
	processor := NewProcessor(logger, store, accountRPC, publisher, cfg.KafkaTopics).WithRollup(rollup.NewStore(dbConn))

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
		"processor_ready", processor != nil,
	)
	return grpcx.Run(ctx, logger, cfg.Name, cfg.GRPCAddr, nil)
}
