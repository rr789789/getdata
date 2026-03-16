package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mvp-platform/internal/config"
	"mvp-platform/internal/core"
	"mvp-platform/internal/store"
)

type Server struct {
	cfg     config.Config
	service *core.Service
	logger  *slog.Logger
}

func NewServer(cfg config.Config, service *core.Service, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	return &Server{
		cfg:     cfg,
		service: service,
		logger:  logger,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/api/v1/devices", s.handleDevices)
	mux.HandleFunc("/api/v1/devices/", s.handleDeviceRoutes)
	return mux
}

func (s *Server) Run(ctx context.Context) error {
	server := &http.Server{
		Addr:              s.cfg.HTTPAddr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	s.logger.Info("http api listening", "addr", s.cfg.HTTPAddr)
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, s.service.Stats())
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/devices" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	switch r.Method {
	case http.MethodPost:
		s.handleCreateDevice(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleDeviceRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/devices/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	parts := strings.Split(path, "/")
	deviceID := parts[0]

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleGetDevice(w, r, deviceID)
		return
	}

	switch parts[1] {
	case "telemetry":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleListTelemetry(w, r, deviceID)
	case "commands":
		switch r.Method {
		case http.MethodGet:
			s.handleListCommands(w, r, deviceID)
		case http.MethodPost:
			s.handleSendCommand(w, r, deviceID)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *Server) handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name     string            `json:"name"`
		Metadata map[string]string `json:"metadata"`
	}

	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	device, err := s.service.CreateDevice(r.Context(), request.Name, request.Metadata)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, device)
}

func (s *Server) handleGetDevice(w http.ResponseWriter, r *http.Request, deviceID string) {
	device, err := s.service.GetDevice(r.Context(), deviceID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrDeviceNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, device)
}

func (s *Server) handleListTelemetry(w http.ResponseWriter, r *http.Request, deviceID string) {
	limit := parseLimit(r, 50, 500)
	telemetry, err := s.service.ListTelemetry(r.Context(), deviceID, limit)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrDeviceNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, telemetry)
}

func (s *Server) handleListCommands(w http.ResponseWriter, r *http.Request, deviceID string) {
	limit := parseLimit(r, 50, 500)
	commands, err := s.service.ListCommands(r.Context(), deviceID, limit)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrDeviceNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, commands)
}

func (s *Server) handleSendCommand(w http.ResponseWriter, r *http.Request, deviceID string) {
	var request struct {
		Name   string         `json:"name"`
		Params map[string]any `json:"params"`
	}

	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	command, err := s.service.SendCommand(r.Context(), deviceID, request.Name, request.Params)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrDeviceNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, core.ErrDeviceOffline):
			writeJSON(w, http.StatusConflict, command)
		default:
			writeError(w, http.StatusBadGateway, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusAccepted, command)
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func parseLimit(r *http.Request, fallback, max int) int {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}
