package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"mvp-platform/internal/app"
	"mvp-platform/internal/config"
)

func main() {
	cfg := config.Load()
	cfg.DisableEmbeddedUI = true

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	})).With("node_id", cfg.NodeID, "role", cfg.NodeRole)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application, err := app.New(cfg, logger)
	if err != nil {
		logger.Error("failed to initialize backend application", "error", err)
		os.Exit(1)
	}
	if err := application.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("backend application stopped with error", "error", err)
		os.Exit(1)
	}
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
