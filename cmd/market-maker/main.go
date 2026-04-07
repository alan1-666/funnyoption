package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"funnyoption/internal/marketmaker"
	"funnyoption/internal/shared/config"
	sharedkafka "funnyoption/internal/shared/kafka"
	"funnyoption/internal/shared/logger"
)

func main() {
	svcCfg := config.Load("market-maker")
	appLogger := logger.New(svcCfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mmCfg := marketmaker.LoadConfig()
	if err := mmCfg.Validate(); err != nil {
		log.Fatalf("invalid market-maker config: %v", err)
	}

	svc := marketmaker.NewService(appLogger, mmCfg)

	marketConsumer := sharedkafka.NewJSONConsumer(
		appLogger,
		svcCfg.KafkaBrokers,
		svcCfg.KafkaTopics.MarketEvent,
		"funnyoption-market-maker",
		svc.HandleMarketEvent,
	)
	marketConsumer.Start(ctx)
	defer marketConsumer.Close()

	tradeConsumer := sharedkafka.NewJSONConsumer(
		appLogger,
		svcCfg.KafkaBrokers,
		svcCfg.KafkaTopics.TradeMatched,
		"funnyoption-market-maker",
		svc.HandleTradeMatched,
	)
	tradeConsumer.Start(ctx)
	defer tradeConsumer.Close()

	orderConsumer := sharedkafka.NewJSONConsumer(
		appLogger,
		svcCfg.KafkaBrokers,
		svcCfg.KafkaTopics.OrderEvent,
		"funnyoption-market-maker",
		svc.HandleOrderEvent,
	)
	orderConsumer.Start(ctx)
	defer orderConsumer.Close()

	appLogger.Info("market-maker service started",
		"api_url", mmCfg.APIURL,
		"bot_user_id", mmCfg.BotUserID,
		"spread", mmCfg.DefaultSpread,
		"quantity", mmCfg.DefaultQuantity,
		"levels", mmCfg.Levels,
		"refresh_interval", mmCfg.RefreshInterval,
	)

	if err := svc.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}
