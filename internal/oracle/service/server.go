package service

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"funnyoption/internal/shared/config"
	shareddb "funnyoption/internal/shared/db"
	sharedkafka "funnyoption/internal/shared/kafka"
)

func Run(ctx context.Context, logger *slog.Logger, cfg config.ServiceConfig) error {
	dbConn, err := shareddb.OpenPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	publisher := sharedkafka.NewJSONPublisher(logger, cfg.KafkaBrokers)
	defer publisher.Close()

	store := NewSQLStore(dbConn)
	httpClient := &http.Client{Timeout: 5 * time.Second}
	provider := NewBinanceProvider(binanceBaseURL(), httpClient)
	worker := NewWorker(logger, store, provider, publisher, cfg.KafkaTopics, oraclePollInterval(cfg))

	logger.Info(
		"oracle worker bootstrapped",
		"service", cfg.Name,
		"env", cfg.Env,
		"poll_interval", oraclePollInterval(cfg),
		"market_topic", cfg.KafkaTopics.MarketEvent,
		"provider", OracleProviderKeyBinance,
		"base_url", binanceBaseURL(),
	)

	worker.Start(ctx)
	return nil
}

func oraclePollInterval(cfg config.ServiceConfig) time.Duration {
	value := strings.TrimSpace(os.Getenv("FUNNYOPTION_ORACLE_POLL_INTERVAL"))
	if value == "" {
		if cfg.PollInterval > 0 {
			return cfg.PollInterval
		}
		return 5 * time.Second
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 5 * time.Second
	}
	return duration
}

func binanceBaseURL() string {
	value := strings.TrimSpace(os.Getenv("FUNNYOPTION_ORACLE_BINANCE_BASE_URL"))
	if value == "" {
		return "https://api.binance.com"
	}
	return value
}
