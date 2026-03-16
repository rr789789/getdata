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
	"mvp-platform/internal/simulator"
	"mvp-platform/internal/store/memory"
)

type App struct {
	cfg        config.Config
	logger     *slog.Logger
	api        *api.Server
	gateway    *gateway.Server
	simulators *simulator.Manager
}

func New(cfg config.Config, logger *slog.Logger) *App {
	if logger == nil {
		logger = slog.Default()
	}

	storage := memory.New(cfg.TelemetryRetention)
	service := core.NewService(storage, storage, storage, logger.With("component", "core"))
	simulators := simulator.NewManager(cfg, service, logger.With("component", "simulator"))

	return &App{
		cfg:        cfg,
		logger:     logger,
		api:        api.NewServer(cfg, service, simulators, logger.With("component", "api")),
		gateway:    gateway.NewServer(cfg, service, logger.With("component", "gateway")),
		simulators: simulators,
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
	a.simulators.Close()
	wg.Wait()
	return runErr
}
