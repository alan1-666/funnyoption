package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"funnyoption/internal/shared/kafka"
)

type ServiceConfig struct {
	Name                    string
	Env                     string
	HTTPAddr                string
	GRPCAddr                string
	AccountGRPCAddr         string
	CollateralSymbol        string
	CollateralDecimals      int
	CollateralDisplayDigits int
	DefaultOperatorUserID   int64
	OperatorWallets         []string
	ChainRPCURL             string
	ChainRPCFallbackURLs    []string
	ChainOperatorPrivateKey string
	ChainName               string
	NetworkName             string
	VaultAddress            string
	ChainID                 int64
	Confirmations           int64
	StartBlock              int64
	ClaimPollInterval       time.Duration
	ChainGasLimit           uint64
	LogLevel                string
	ShutdownTimeout         time.Duration
	PollInterval            time.Duration
	PostgresDSN             string
	RedisAddr               string
	KafkaBrokers            []string
	KafkaTopicPrefx         string
	KafkaTopics             kafka.Topics
}

func Load(serviceName string) ServiceConfig {
	serviceKey := normalize(serviceName)
	cfg := ServiceConfig{
		Name:                    serviceName,
		Env:                     getenv("FUNNYOPTION_ENV", "local"),
		HTTPAddr:                getenv(serviceKey+"_HTTP_ADDR", defaultHTTPAddr(serviceName)),
		GRPCAddr:                getenv(serviceKey+"_GRPC_ADDR", defaultGRPCAddr(serviceName)),
		AccountGRPCAddr:         getenv("FUNNYOPTION_ACCOUNT_GRPC_ADDR", defaultGRPCAddr("account")),
		CollateralSymbol:        getenv("FUNNYOPTION_COLLATERAL_SYMBOL", "USDT"),
		CollateralDecimals:      getenvInt("FUNNYOPTION_COLLATERAL_DECIMALS", 6),
		CollateralDisplayDigits: getenvInt("FUNNYOPTION_COLLATERAL_ACCOUNTING_DECIMALS", 2),
		DefaultOperatorUserID:   int64(getenvInt("FUNNYOPTION_DEFAULT_OPERATOR_USER_ID", 1001)),
		OperatorWallets:         splitCSV(getenv("FUNNYOPTION_OPERATOR_WALLETS", "")),
		ChainRPCURL:             getenv("FUNNYOPTION_CHAIN_RPC_URL", ""),
		ChainOperatorPrivateKey: getenv("FUNNYOPTION_CHAIN_OPERATOR_PRIVATE_KEY", ""),
		ChainName:               getenv("FUNNYOPTION_CHAIN_NAME", "bsc"),
		NetworkName:             getenv("FUNNYOPTION_NETWORK_NAME", "testnet"),
		VaultAddress:            getenv("FUNNYOPTION_VAULT_ADDRESS", ""),
		ChainID:                 int64(getenvInt("FUNNYOPTION_CHAIN_ID", 97)),
		Confirmations:           int64(getenvInt("FUNNYOPTION_CHAIN_CONFIRMATIONS", 6)),
		StartBlock:              int64(getenvInt("FUNNYOPTION_CHAIN_START_BLOCK", 0)),
		ClaimPollInterval:       getenvDuration("FUNNYOPTION_CHAIN_CLAIM_POLL_INTERVAL", 10*time.Second),
		ChainGasLimit:           uint64(getenvInt("FUNNYOPTION_CHAIN_GAS_LIMIT", 250000)),
		LogLevel:                getenv("FUNNYOPTION_LOG_LEVEL", "info"),
		ShutdownTimeout:         getenvDuration("FUNNYOPTION_SHUTDOWN_TIMEOUT", 10*time.Second),
		PollInterval:            getenvDuration("FUNNYOPTION_CHAIN_POLL_INTERVAL", 5*time.Second),
		PostgresDSN:             getenv("FUNNYOPTION_POSTGRES_DSN", "postgres://postgres:postgres@127.0.0.1:5432/funnyoption?sslmode=disable"),
		RedisAddr:               getenv("FUNNYOPTION_REDIS_ADDR", "127.0.0.1:6379"),
		KafkaTopicPrefx:         getenv("FUNNYOPTION_KAFKA_TOPIC_PREFIX", "funnyoption."),
	}

	brokers := getenv("FUNNYOPTION_KAFKA_BROKERS", "127.0.0.1:9092")
	cfg.KafkaBrokers = strings.Split(brokers, ",")
	cfg.ChainRPCFallbackURLs = splitCSV(getenv("FUNNYOPTION_CHAIN_RPC_FALLBACK_URLS", ""))
	cfg.KafkaTopics = kafka.NewTopics(cfg.KafkaTopicPrefx)

	return cfg
}

func (c ServiceConfig) String() string {
	return fmt.Sprintf("service=%s env=%s http=%s grpc=%s", c.Name, c.Env, c.HTTPAddr, c.GRPCAddr)
}

func normalize(serviceName string) string {
	return "FUNNYOPTION_" + strings.ToUpper(strings.ReplaceAll(serviceName, "-", "_"))
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err == nil {
		return duration
	}

	seconds, err := strconv.Atoi(value)
	if err == nil {
		return time.Duration(seconds) * time.Second
	}

	return fallback
}

func getenvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	raw := strings.Split(value, ",")
	items := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func defaultHTTPAddr(serviceName string) string {
	switch serviceName {
	case "api":
		return ":8080"
	case "ws":
		return ":8081"
	default:
		return ""
	}
}

func defaultGRPCAddr(serviceName string) string {
	switch serviceName {
	case "matching":
		return ":9090"
	case "account":
		return ":9091"
	case "ledger":
		return ":9095"
	case "settlement":
		return ":9093"
	case "chain":
		return ":9094"
	default:
		return ""
	}
}
