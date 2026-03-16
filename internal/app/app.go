package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"mvp-platform/internal/api"
	"mvp-platform/internal/config"
	"mvp-platform/internal/core"
	"mvp-platform/internal/gateway"
	"mvp-platform/internal/store/memory"
)

type App struct {
	cfg     config.Config
	logger  *slog.Logger
	api     *api.Server
	gateway *gateway.Server
}

func New(cfg config.Config, logger *slog.Logger) *App {
	if logger == nil {
		logger = slog.Default()
	}

	storage := memory.New(cfg.TelemetryRetention)
	service := core.NewService(storage, storage, storage, logger.With("component", "core"))

	return &App{
		cfg:     cfg,
		logger:  logger,
		api:     api.NewServer(cfg, service, logger.With("component", "api")),
		gateway: gateway.NewServer(cfg, service, logger.With("component", "gateway")),
	}
}

func (a *App) Run(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)
	var wg sync.WaitGroup

	start := func(name string, run func(context.Context) error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
				select {
				case errCh <- fmt.Errorf("%s: %w", name, err):
				default:
				}
				cancel()
			}
		}()
	}

	start("http-api", a.api.Run)
	start("device-gateway", a.gateway.Run)

	var runErr error
	select {
	case runErr = <-errCh:
	case <-ctx.Done():
	}

	cancel()
	wg.Wait()
	return runErr
}
