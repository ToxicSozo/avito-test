package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ToxicSozo/avito-test/internal/app"
	"github.com/ToxicSozo/avito-test/internal/config"
	"github.com/ToxicSozo/avito-test/internal/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logg := logger.New(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, logg); err != nil {
		logg.Error("app run failed", "err", err)
		os.Exit(1)
	}
}
