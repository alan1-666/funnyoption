package main

import (
	"context"
	"log"

	"funnyoption/internal/chain/service"
	"funnyoption/internal/shared/config"
	"funnyoption/internal/shared/logger"
)

func main() {
	cfg := config.Load("chain")
	logr := logger.New(cfg.LogLevel)
	if err := service.Run(context.Background(), logr, cfg); err != nil {
		log.Fatal(err)
	}
}
