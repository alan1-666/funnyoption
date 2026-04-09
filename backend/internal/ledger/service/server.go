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

	health.ListenAndServe(ctx, logger, cfg.HTTPAddr, cfg.Name, cfg.Env)

	logger.Info(
		"ledger service bootstrapped",
		"grpc_addr", cfg.GRPCAddr,
		"trade_topic", cfg.KafkaTopics.TradeMatched,
		"settlement_topic", cfg.KafkaTopics.SettlementDone,
		"journal_ready", journal != nil,
	)
	return grpcx.Run(ctx, logger, cfg.Name, cfg.GRPCAddr, nil)
}
