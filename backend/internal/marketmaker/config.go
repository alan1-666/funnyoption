package marketmaker

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	APIURL             string
	OperatorPrivateKey string
	OperatorWallet     string
	BotUserID          int64

	DefaultSpread   int64
	DefaultQuantity int64
	DefaultMidPrice int64
	Levels          int
	InventorySkew   float64

	RefreshInterval time.Duration
	WriteInterval   time.Duration
	SeedOnNewMarket bool
}

func LoadConfig() Config {
	cfg := Config{
		APIURL:             envOrDefault("MM_API_URL", "http://localhost:8080"),
		OperatorPrivateKey: os.Getenv("MM_OPERATOR_PRIVATE_KEY"),
		OperatorWallet:     os.Getenv("MM_OPERATOR_WALLET"),
		BotUserID:          envInt64("MM_BOT_USER_ID", 1001),
		DefaultSpread:      envInt64("MM_DEFAULT_SPREAD", 6),
		DefaultQuantity:    envInt64("MM_DEFAULT_QUANTITY", 50),
		DefaultMidPrice:    envInt64("MM_DEFAULT_MID_PRICE", 50),
		Levels:             int(envInt64("MM_LEVELS", 3)),
		InventorySkew:      envFloat("MM_INVENTORY_SKEW", 0.5),
		RefreshInterval:    envDuration("MM_REFRESH_INTERVAL", 5*time.Second),
		WriteInterval:      envDuration("MM_WRITE_INTERVAL", 0),
		SeedOnNewMarket:    envBool("MM_SEED_ON_NEW_MARKET", true),
	}
	return cfg
}

func (c Config) Validate() error {
	if c.APIURL == "" {
		return fmt.Errorf("MM_API_URL is required")
	}
	if c.OperatorPrivateKey == "" {
		return fmt.Errorf("MM_OPERATOR_PRIVATE_KEY is required")
	}
	if c.OperatorWallet == "" {
		return fmt.Errorf("MM_OPERATOR_WALLET is required")
	}
	if c.BotUserID <= 0 {
		return fmt.Errorf("MM_BOT_USER_ID must be positive")
	}
	if c.DefaultSpread < 2 {
		return fmt.Errorf("MM_DEFAULT_SPREAD must be >= 2")
	}
	if c.DefaultQuantity <= 0 {
		return fmt.Errorf("MM_DEFAULT_QUANTITY must be positive")
	}
	if c.Levels < 1 {
		return fmt.Errorf("MM_LEVELS must be >= 1")
	}
	if c.WriteInterval < 0 {
		return fmt.Errorf("MM_WRITE_INTERVAL must not be negative")
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func envFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch v {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	default:
		return fallback
	}
}
