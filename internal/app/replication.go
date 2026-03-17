package app

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"mvp-platform/internal/config"
)

type replicaPersistHookSetter interface {
	SetAfterPersistHook(func(context.Context, []byte))
}

type setupPersistHookSetter interface {
	SetAfterPersistHook(func(context.Context, []byte))
}

func configureReplication(cfg config.Config, storage appStore, installer interface{}, logger *slog.Logger) []func(*appOptions) {
	var options []func(*appOptions)

	if cfg.IsStandby() && strings.TrimSpace(cfg.ReplicaToken) != "" {
		if applier, ok := storage.(interface{ ApplyReplicaSnapshot([]byte) error }); ok {
			options = append(options, func(target *appOptions) {
				target.enableReplicaApplier = applier
			})
		}
	}

	if cfg.IsStandby() {
		return options
	}

	if len(cfg.ReplicaPeers) == 0 || strings.TrimSpace(cfg.ReplicaToken) == "" {
		return options
	}

	hookSetter, ok := storage.(replicaPersistHookSetter)
	if !ok {
		logger.Warn("replica peers configured but store backend does not support snapshot replication", "backend", cfg.StoreBackend)
		return options
	}

	hookSetter.SetAfterPersistHook(newReplicaPersistHook(cfg, logger.With("component", "replica")))
	if hookSetter, ok := installer.(setupPersistHookSetter); ok {
		hookSetter.SetAfterPersistHook(newReplicaSetupHook(cfg, logger.With("component", "replica-setup")))
	}
	return options
}

type appOptions struct {
	enableReplicaApplier interface{ ApplyReplicaSnapshot([]byte) error }
}

func newReplicaPersistHook(cfg config.Config, logger *slog.Logger) func(context.Context, []byte) {
	peers := normalizeReplicaPeers(cfg.ReplicaPeers)
	client := &http.Client{Timeout: cfg.ReplicaTimeout}

	return func(_ context.Context, snapshot []byte) {
		for _, peer := range peers {
			replicatePayload(client, cfg, logger, peer, "/_ha/snapshot", snapshot)
		}
	}
}

func newReplicaSetupHook(cfg config.Config, logger *slog.Logger) func(context.Context, []byte) {
	peers := normalizeReplicaPeers(cfg.ReplicaPeers)
	client := &http.Client{Timeout: cfg.ReplicaTimeout}

	return func(_ context.Context, snapshot []byte) {
		for _, peer := range peers {
			replicatePayload(client, cfg, logger, peer, "/_ha/setup", snapshot)
		}
	}
}

func replicatePayload(client *http.Client, cfg config.Config, logger *slog.Logger, peer, endpoint string, snapshot []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ReplicaTimeout)
	defer cancel()

	url := strings.TrimRight(peer, "/") + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(snapshot))
	if err != nil {
		logger.Warn("unable to build replica request", "peer", peer, "endpoint", endpoint, "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Replica-Token", cfg.ReplicaToken)

	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("replica push failed", "peer", peer, "endpoint", endpoint, "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 == 2 {
		logger.Debug("replica payload pushed", "peer", peer, "endpoint", endpoint, "status", resp.StatusCode)
		_, _ = io.Copy(io.Discard, resp.Body)
		return
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	logger.Warn(
		"replica push rejected",
		"peer", peer,
		"endpoint", endpoint,
		"status", resp.StatusCode,
		"body", strings.TrimSpace(string(body)),
	)
}

func normalizeReplicaPeers(peers []string) []string {
	result := make([]string, 0, len(peers))
	seen := make(map[string]struct{}, len(peers))
	for _, peer := range peers {
		peer = strings.TrimSpace(peer)
		if peer == "" {
			continue
		}
		peer = strings.TrimRight(peer, "/")
		if _, exists := seen[peer]; exists {
			continue
		}
		seen[peer] = struct{}{}
		result = append(result, peer)
	}
	return result
}
