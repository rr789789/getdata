package api

import (
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := resolveAllowedOrigin(s.cfg.CORSAllowedOrigins, strings.TrimSpace(r.Header.Get("Origin")))
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Request-ID, X-Replica-Token")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withStandbyGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.IsStandby() && isMutatingMethod(r.Method) && r.URL.Path != "/_ha/snapshot" {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"error":   "node is in standby mode",
				"node_id": s.cfg.NodeID,
				"role":    normalizedNodeRole(s.cfg.NodeRole),
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	payload := map[string]any{
		"status":  "ready",
		"time":    time.Now().UTC(),
		"node_id": s.cfg.NodeID,
		"role":    normalizedNodeRole(s.cfg.NodeRole),
	}
	if s.cfg.IsStandby() {
		payload["status"] = "standby"
		writeJSON(w, http.StatusServiceUnavailable, payload)
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	stats := s.service.Stats()
	writeJSON(w, http.StatusOK, map[string]any{
		"node_id":                 s.cfg.NodeID,
		"role":                    normalizedNodeRole(s.cfg.NodeRole),
		"standby":                 s.cfg.IsStandby(),
		"embedded_ui":             !s.cfg.DisableEmbeddedUI,
		"cors_allowed_origins":    s.cfg.CORSAllowedOrigins,
		"replica_endpoint":        s.replica != nil,
		"store_backend":           stats.Storage.Backend,
		"store_persistence_path":  stats.Storage.PersistencePath,
		"metrics_started_at":      stats.StartedAt,
		"metrics_uptime_seconds":  stats.UptimeSeconds,
		"gateway_enabled":         !s.cfg.IsStandby(),
		"mqtt_enabled":            !s.cfg.IsStandby(),
	})
}

func (s *Server) handleReplicaSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.replica == nil {
		writeError(w, http.StatusServiceUnavailable, "replica snapshot endpoint unavailable")
		return
	}
	if strings.TrimSpace(s.cfg.ReplicaToken) == "" {
		writeError(w, http.StatusServiceUnavailable, "replica token not configured")
		return
	}
	if subtleConstantCompare(strings.TrimSpace(r.Header.Get("X-Replica-Token")), s.cfg.ReplicaToken) == false {
		writeError(w, http.StatusForbidden, "replica token rejected")
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "unable to read snapshot payload")
		return
	}
	if err := s.replica.ApplyReplicaSnapshot(body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":    "replicated",
		"time":      time.Now().UTC(),
		"node_id":   s.cfg.NodeID,
		"node_role": normalizedNodeRole(s.cfg.NodeRole),
	})
}

func resolveAllowedOrigin(allowed []string, origin string) string {
	if origin == "" {
		return ""
	}
	if len(allowed) == 0 {
		return ""
	}
	for _, item := range allowed {
		item = strings.TrimSpace(item)
		switch {
		case item == "*":
			return "*"
		case strings.EqualFold(item, origin):
			return origin
		}
	}
	return ""
}

func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func clientIP(r *http.Request) string {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		value := strings.TrimSpace(r.Header.Get(header))
		if value == "" {
			continue
		}
		if header == "X-Forwarded-For" && strings.Contains(value, ",") {
			value = strings.TrimSpace(strings.Split(value, ",")[0])
		}
		if value != "" {
			return value
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		if errors.Is(err, net.ErrClosed) {
			return ""
		}
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}

func normalizedNodeRole(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "standby") {
		return "standby"
	}
	return "primary"
}

func subtleConstantCompare(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	var diff byte
	for index := 0; index < len(left); index++ {
		diff |= left[index] ^ right[index]
	}
	return diff == 0
}
