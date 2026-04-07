package service

import (
	"context"
	"log/slog"
	"time"

	"funnyoption/internal/matching/pipeline"
	"funnyoption/internal/rollup"
	"funnyoption/internal/shared/config"
	shareddb "funnyoption/internal/shared/db"
	"funnyoption/internal/shared/fee"
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
	cachedStore := NewCachedCommandStore(store)

	sequence, err := store.MaxTradeSequence(ctx)
	if err != nil {
		return err
	}
	restingOrders, err := store.LoadRestingOrders(ctx)
	if err != nil {
		return err
	}

	publisher := sharedkafka.NewJSONPublisher(logger, cfg.KafkaBrokers)
	defer publisher.Close()

	feeSched := fee.Schedule{TakerFeeBps: cfg.TakerFeeBps, MakerFeeBps: cfg.MakerFeeBps}

	candles := NewCandleBook(defaultCandleIntervalMillis, defaultCandleHistoryLimit)

	pipe := pipeline.New(
		logger,
		cfg.KafkaBrokers,
		cfg.KafkaTopics.OrderCommand,
		"funnyoption-matching",
		cachedStore,
		cachedStore,
		publisher,
		cfg.KafkaTopics,
		feeSched,
		candles,
		pipeline.Config{
			InputRBSize:  8192,
			OutputRBSize: 8192,
		},
	)

	if err := pipe.Restore(sequence, restingOrders); err != nil {
		return err
	}

	pipe.Start(ctx)
	pipe.StartStatsReporter(ctx, logger, 10*time.Second)
	defer pipe.Close()

	expirySweeper := newOrderExpirySweeper(logger, pipe, store, publisher, cfg.KafkaTopics)
	expirySweeper.Start(ctx)

	logger.Info(
		"matching service bootstrapped (pipeline mode)",
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
		"input_rb_size", 8192,
		"output_rb_size", 8192,
	)

	return grpcx.Run(ctx, logger, cfg.Name, cfg.GRPCAddr, nil)
}
