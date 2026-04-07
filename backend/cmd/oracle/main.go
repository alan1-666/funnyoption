package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	oracleservice "funnyoption/internal/oracle/service"
	"funnyoption/internal/shared/config"
	"funnyoption/internal/shared/logger"
)

func main() {
	cfg := config.Load("oracle")
	appLogger := logger.New(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	appLogger.Info("starting service", "config", cfg.String())
	if err := oracleservice.Run(ctx, appLogger, cfg); err != nil {
		log.Fatal(err)
	}
}
