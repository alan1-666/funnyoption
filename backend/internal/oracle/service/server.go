package service

import (
	"context"
	"crypto/ecdsa"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

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
	stats := &OracleStats{}
	worker := NewWorker(logger, store, provider, publisher, cfg.KafkaTopics, oraclePollInterval(cfg))
	worker.SetStats(stats)

	startObservabilityHTTP(ctx, logger, observabilityHTTPAddr(), cfg, cfg.KafkaTopics, stats)

	if signerKey := loadOracleSignerKey(logger); signerKey != nil {
		worker.SetSignerKey(signerKey)
		signerAddr := crypto.PubkeyToAddress(signerKey.PublicKey)
		logger.Info("oracle attestation signer configured", "address", signerAddr.Hex())
	}
	if trustedSigners := loadTrustedSigners(logger); len(trustedSigners) > 0 {
		worker.SetTrustedSigners(trustedSigners)
		logger.Info("oracle trusted signers configured", "count", len(trustedSigners))
	}

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

func loadOracleSignerKey(logger *slog.Logger) *ecdsa.PrivateKey {
	raw := strings.TrimSpace(os.Getenv("FUNNYOPTION_ORACLE_SIGNER_KEY"))
	if raw == "" {
		return nil
	}
	raw = strings.TrimPrefix(raw, "0x")
	key, err := crypto.HexToECDSA(raw)
	if err != nil {
		logger.Warn("invalid FUNNYOPTION_ORACLE_SIGNER_KEY, attestation signing disabled", "err", err)
		return nil
	}
	return key
}

func loadTrustedSigners(logger *slog.Logger) []common.Address {
	raw := strings.TrimSpace(os.Getenv("FUNNYOPTION_ORACLE_TRUSTED_SIGNERS"))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var signers []common.Address
	for _, part := range parts {
		addr := strings.TrimSpace(part)
		if addr == "" {
			continue
		}
		if !common.IsHexAddress(addr) {
			logger.Warn("skipping invalid trusted signer address", "address", addr)
			continue
		}
		signers = append(signers, common.HexToAddress(addr))
	}
	return signers
}
