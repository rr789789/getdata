package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"mvp-platform/internal/api"
	"mvp-platform/internal/config"
	"mvp-platform/internal/core"
	"mvp-platform/internal/gateway"
	"mvp-platform/internal/mqtt"
	"mvp-platform/internal/simulator"
	"mvp-platform/internal/store"
	storefile "mvp-platform/internal/store/file"
	"mvp-platform/internal/store/memory"
)

type App struct {
	cfg        config.Config
	logger     *slog.Logger
	api        *api.Server
	gateway    *gateway.Server
	mqtt       *mqtt.Server
	simulators *simulator.Manager
}

type appStore interface {
	store.ProductStore
	store.DeviceStore
	store.GroupStore
	store.RuleStore
	store.ConfigStore
	store.TelemetryStore
	store.ShadowStore
	store.CommandStore
	store.AlertStore
	store.Inspector
}

func New(cfg config.Config, logger *slog.Logger) (*App, error) {
	if logger == nil {
		logger = slog.Default()
	}

	storage, err := newStorage(cfg)
	if err != nil {
		return nil, err
	}
	service := core.NewService(storage, storage, storage, storage, storage, storage, storage, storage, storage, logger.With("component", "core"))
	simulators := simulator.NewManager(cfg, service, logger.With("component", "simulator"))

	return &App{
		cfg:        cfg,
		logger:     logger,
		api:        api.NewServer(cfg, service, simulators, logger.With("component", "api")),
		gateway:    gateway.NewServer(cfg, service, logger.With("component", "gateway")),
		mqtt:       mqtt.NewServer(cfg, service, logger.With("component", "mqtt")),
		simulators: simulators,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 3)
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
	start("mqtt-broker", a.mqtt.Run)

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

func newStorage(cfg config.Config) (appStore, error) {
	backend := strings.ToLower(strings.TrimSpace(cfg.StoreBackend))
	switch backend {
	case "", "memory":
		return memory.New(cfg.TelemetryRetention), nil
	case "file", "json":
		store, err := storefile.New(cfg.StorePath, cfg.TelemetryRetention)
		if err != nil {
			return nil, err
		}
		return store, nil
	default:
		return nil, fmt.Errorf("unsupported store backend %q", cfg.StoreBackend)
	}
}
