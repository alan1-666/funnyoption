package service

import (
	"context"
	"log/slog"

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

	store := NewSQLStore(dbConn)
	publisher := sharedkafka.NewJSONPublisher(logger, cfg.KafkaBrokers)
	defer publisher.Close()

	processor := NewProcessor(store, publisher, cfg.KafkaTopics)
	positionConsumer := sharedkafka.NewJSONConsumer(
		logger,
		cfg.KafkaBrokers,
		cfg.KafkaTopics.PositionChange,
		"funnyoption-settlement",
		processor.HandlePositionChanged,
	)
	positionConsumer.Start(ctx)
	defer positionConsumer.Close()

	marketConsumer := sharedkafka.NewJSONConsumer(
		logger,
		cfg.KafkaBrokers,
		cfg.KafkaTopics.MarketEvent,
		"funnyoption-settlement",
		processor.HandleMarketEvent,
	)
	marketConsumer.Start(ctx)
	defer marketConsumer.Close()

	logger.Info(
		"settlement service bootstrapped",
		"position_topic", cfg.KafkaTopics.PositionChange,
		"market_topic", cfg.KafkaTopics.MarketEvent,
		"settlement_topic", cfg.KafkaTopics.SettlementDone,
	)
	return grpcx.Run(ctx, logger, cfg.Name, cfg.GRPCAddr, nil)
}
