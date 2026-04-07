package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	notificationservice "funnyoption/internal/notification"
	"funnyoption/internal/shared/config"
	"funnyoption/internal/shared/logger"
)

func main() {
	cfg := config.Load("notification")
	appLogger := logger.New(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	appLogger.Info("starting service", "config", cfg.String())
	if err := notificationservice.Run(ctx, appLogger, cfg); err != nil {
		log.Fatal(err)
	}
}
