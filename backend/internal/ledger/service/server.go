package service

import (
	"context"
	"log/slog"

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

	journal := NewPersistentJournal(NewSQLStore(dbConn))
	processor := NewTradeProcessor(journal)
	settlementProcessor := NewSettlementProcessor(journal)
	depositProcessor := NewDepositProcessor(journal)
	withdrawalProcessor := NewWithdrawalProcessor(journal)
	consumer := sharedkafka.NewJSONConsumer(
		logger,
		cfg.KafkaBrokers,
		cfg.KafkaTopics.TradeMatched,
		"funnyoption-ledger",
		processor.HandleTradeMatched,
	)
	consumer.Start(ctx)
	defer consumer.Close()

	settlementConsumer := sharedkafka.NewJSONConsumer(
		logger,
		cfg.KafkaBrokers,
		cfg.KafkaTopics.SettlementDone,
		"funnyoption-ledger",
		settlementProcessor.HandleSettlementCompleted,
	)
	settlementConsumer.Start(ctx)
	defer settlementConsumer.Close()

	depositConsumer := sharedkafka.NewJSONConsumer(
		logger,
		cfg.KafkaBrokers,
		cfg.KafkaTopics.ChainDeposit,
		"funnyoption-ledger",
		depositProcessor.HandleChainDeposit,
	)
	depositConsumer.Start(ctx)
	defer depositConsumer.Close()

	withdrawalConsumer := sharedkafka.NewJSONConsumer(
		logger,
		cfg.KafkaBrokers,
		cfg.KafkaTopics.ChainWithdraw,
		"funnyoption-ledger",
		withdrawalProcessor.HandleChainWithdrawal,
	)
	withdrawalConsumer.Start(ctx)
	defer withdrawalConsumer.Close()

	health.ListenAndServe(ctx, logger, cfg.HTTPAddr, cfg.Name, cfg.Env)

	logger.Info(
		"ledger service bootstrapped",
		"grpc_addr", cfg.GRPCAddr,
		"postgres_dsn", cfg.PostgresDSN,
		"redis_addr", cfg.RedisAddr,
		"trade_topic", cfg.KafkaTopics.TradeMatched,
		"settlement_topic", cfg.KafkaTopics.SettlementDone,
		"chain_deposit_topic", cfg.KafkaTopics.ChainDeposit,
		"chain_withdraw_topic", cfg.KafkaTopics.ChainWithdraw,
		"journal_ready", journal != nil,
	)
	return grpcx.Run(ctx, logger, cfg.Name, cfg.GRPCAddr, nil)
}
