package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"funnyoption/internal/chain/service"
	"funnyoption/internal/shared/config"
	"funnyoption/internal/shared/logger"
)

func main() {
	cfg := config.Load("chain")
	logr := logger.New(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logr.Info("starting service", "config", cfg.String())
	if err := service.Run(ctx, logr, cfg); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}
