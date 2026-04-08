package service

import (
	"context"
	"log/slog"

	accountv1 "funnyoption/internal/gen/accountv1"
	"funnyoption/internal/shared/config"
	shareddb "funnyoption/internal/shared/db"
	"funnyoption/internal/shared/grpcx"
	"funnyoption/internal/shared/health"
	sharedkafka "funnyoption/internal/shared/kafka"

	"google.golang.org/grpc"
)

func Run(ctx context.Context, logger *slog.Logger, cfg config.ServiceConfig) error {
	dbConn, err := shareddb.OpenPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	store := NewSQLStore(dbConn)
	book := NewBalanceBookWithStore(store)
	if balances, err := store.LoadBalances(ctx); err != nil {
		return err
	} else if freezes, err := store.LoadFreezes(ctx); err != nil {
		return err
	} else {
		book.Hydrate(balances, freezes)
	}
	seedDemoBalances(cfg.Env, book)
	registry := NewPersistentOrderRegistry(store)
	processor := NewEventProcessor(book, registry, store)

	orderConsumer := sharedkafka.NewJSONConsumer(
		logger,
		cfg.KafkaBrokers,
		cfg.KafkaTopics.OrderEvent,
		"funnyoption-account",
		processor.HandleOrderEvent,
	)
	orderConsumer.Start(ctx)
	defer orderConsumer.Close()

	tradeConsumer := sharedkafka.NewJSONConsumer(
		logger,
		cfg.KafkaBrokers,
		cfg.KafkaTopics.TradeMatched,
		"funnyoption-account",
		processor.HandleTradeMatched,
	)
	tradeConsumer.Start(ctx)
	defer tradeConsumer.Close()

	settlementConsumer := sharedkafka.NewJSONConsumer(
		logger,
		cfg.KafkaBrokers,
		cfg.KafkaTopics.SettlementDone,
		"funnyoption-account",
		processor.HandleSettlementCompleted,
	)
	settlementConsumer.Start(ctx)
	defer settlementConsumer.Close()

	health.ListenAndServe(ctx, logger, cfg.HTTPAddr, cfg.Name, cfg.Env)

	logger.Info(
		"account service bootstrapped",
		"redis_addr", cfg.RedisAddr,
		"postgres_dsn", cfg.PostgresDSN,
		"balance_count", book.BalanceCount(),
		"balance_book_ready", book != nil,
		"order_event_topic", cfg.KafkaTopics.OrderEvent,
		"trade_topic", cfg.KafkaTopics.TradeMatched,
		"settlement_topic", cfg.KafkaTopics.SettlementDone,
	)
	return grpcx.Run(ctx, logger, cfg.Name, cfg.GRPCAddr, func(server *grpc.Server) {
		accountv1.RegisterAccountServiceServer(server, NewGRPCServer(book))
	})
}

func seedDemoBalances(env string, book *BalanceBook) {
	if env != "local" || book.BalanceCount() > 0 {
		return
	}
	for _, userID := range []int64{1001, 1002, 1003} {
		book.SeedBalance(userID, "USDT", 1_000_000)
	}
}
