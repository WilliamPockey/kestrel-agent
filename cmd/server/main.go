package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"kestrel/internal/app"
	"kestrel/internal/config"
	"kestrel/internal/observability"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	logger, err := observability.NewLogger(cfg.Log)
	if err != nil {
		slog.Error("init logger", "err", err)
		os.Exit(1)
	}
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, logger); err != nil {
		logger.Error("server exited", "err", err)
		os.Exit(1)
	}
}
