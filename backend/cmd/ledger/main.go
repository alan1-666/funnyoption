package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	ledgerservice "funnyoption/internal/ledger/service"
	"funnyoption/internal/shared/config"
	"funnyoption/internal/shared/logger"
)

func main() {
	cfg := config.Load("ledger")
	appLogger := logger.New(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	appLogger.Info("starting service", "config", cfg.String())
	if err := ledgerservice.Run(ctx, appLogger, cfg); err != nil {
		log.Fatal(err)
	}
}
