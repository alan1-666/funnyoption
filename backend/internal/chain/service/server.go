package service

import (
	"context"
	"log/slog"
	"strings"

	"funnyoption/internal/rollup"
	"funnyoption/internal/shared/config"
	shareddb "funnyoption/internal/shared/db"
	"funnyoption/internal/shared/grpcx"
	"funnyoption/internal/shared/health"
)

func Run(ctx context.Context, logger *slog.Logger, cfg config.ServiceConfig) error {
	dbConn, err := shareddb.OpenPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	store := NewSQLStore(dbConn)
	rollupStore := rollup.NewStore(dbConn)

	var rpcPool *rpcPool
	if cfg.ChainRPCURL != "" {
		rpcPool, err = newRPCPool(ctx, cfg)
		if err != nil {
			return err
		}
		defer rpcPool.Close()
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
		"chain_name", cfg.ChainName,
		"network_name", cfg.NetworkName,
		"chain_rpc_url", cfg.ChainRPCURL,
		"forced_withdrawal_mirror_ready", forcedWithdrawalMirror != nil,
		"forced_withdrawal_satisfier_ready", forcedWithdrawalSatisfier != nil,
		"rollup_submission_ready", rollupSubmissionProcessor != nil,
	)
	return grpcx.Run(ctx, logger, cfg.Name, cfg.GRPCAddr, nil)
}
