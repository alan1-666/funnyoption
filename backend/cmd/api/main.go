package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"funnyoption/internal/api"
	"funnyoption/internal/shared/config"
	"funnyoption/internal/shared/logger"
)

func main() {
	cfg := config.Load("api")
	appLogger := logger.New(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	appLogger.Info("starting service", "config", cfg.String())
	if err := api.Run(ctx, appLogger, cfg); err != nil {
		log.Fatal(err)
	}
}
