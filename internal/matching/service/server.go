package service

import (
	"context"
	"log/slog"

	"funnyoption/internal/matching/engine"
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

	store := NewSQLStore(dbConn).WithRollup(rollup.NewStore(dbConn))
	matcher := engine.NewAsync(logger, 2048)
	sequence, err := store.MaxTradeSequence(ctx)
	if err != nil {
		return err
	}
	restingOrders, err := store.LoadRestingOrders(ctx)
	if err != nil {
		return err
	}
	if err := matcher.Restore(sequence, restingOrders); err != nil {
		return err
	}
	matcher.Start(ctx)
	publisher := sharedkafka.NewJSONPublisher(logger, cfg.KafkaBrokers)
	defer publisher.Close()

	cachedStore := NewCachedCommandStore(store)
	processor := NewCommandProcessor(logger, matcher, publisher, cfg.KafkaTopics, cachedStore)
	expirySweeper := newOrderExpirySweeper(logger, matcher, store, publisher, cfg.KafkaTopics)
	expirySweeper.Start(ctx)
	consumer := sharedkafka.NewJSONConsumer(
		logger,
		cfg.KafkaBrokers,
		cfg.KafkaTopics.OrderCommand,
		"funnyoption-matching",
		processor.HandleOrderCommand,
	)
	consumer.Start(ctx)
	defer consumer.Close()

	logger.Info(
		"matching service bootstrapped",
		"ingress", "kafka",
		"grpc_addr", cfg.GRPCAddr,
		"kafka_brokers", cfg.KafkaBrokers,
		"order_command_topic", cfg.KafkaTopics.OrderCommand,
		"order_event_topic", cfg.KafkaTopics.OrderEvent,
		"trade_topic", cfg.KafkaTopics.TradeMatched,
		"depth_topic", cfg.KafkaTopics.QuoteDepth,
		"ticker_topic", cfg.KafkaTopics.QuoteTicker,
		"candle_topic", cfg.KafkaTopics.QuoteCandle,
		"postgres_dsn", cfg.PostgresDSN,
		"restored_trade_sequence", sequence,
		"restored_resting_orders", len(restingOrders),
		"book_count", matcher.BookCount(),
	)

	return grpcx.Run(ctx, logger, cfg.Name, cfg.GRPCAddr, nil)
}
