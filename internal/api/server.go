package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mvp-platform/internal/config"
	"mvp-platform/internal/core"
	"mvp-platform/internal/model"
	"mvp-platform/internal/simulator"
	"mvp-platform/internal/store"
)

type Server struct {
	cfg        config.Config
	service    *core.Service
	simulators *simulator.Manager
	logger     *slog.Logger
}

func NewServer(cfg config.Config, service *core.Service, simulators *simulator.Manager, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	return &Server{
		cfg:        cfg,
		service:    service,
		simulators: simulators,
		logger:     logger,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/api/v1/protocol-catalog", s.handleProtocolCatalog)
	mux.HandleFunc("/api/v1/tenants", s.handleTenants)
	mux.HandleFunc("/api/v1/ingest/http/", s.handleHTTPIngestRoutes)
	mux.HandleFunc("/api/v1/products", s.handleProducts)
	mux.HandleFunc("/api/v1/products/", s.handleProductRoutes)
	mux.HandleFunc("/api/v1/groups", s.handleGroups)
	mux.HandleFunc("/api/v1/groups/", s.handleGroupRoutes)
	mux.HandleFunc("/api/v1/config-profiles", s.handleConfigProfiles)
	mux.HandleFunc("/api/v1/config-profiles/", s.handleConfigProfileRoutes)
	mux.HandleFunc("/api/v1/firmware", s.handleFirmwareArtifacts)
	mux.HandleFunc("/api/v1/ota-campaigns", s.handleOTACampaigns)
	mux.HandleFunc("/api/v1/devices", s.handleDevices)
	mux.HandleFunc("/api/v1/devices/", s.handleDeviceRoutes)
	mux.HandleFunc("/api/v1/rules", s.handleRules)
	mux.HandleFunc("/api/v1/alerts", s.handleAlerts)
	mux.HandleFunc("/api/v1/alerts/", s.handleAlertRoutes)
	mux.HandleFunc("/api/v1/simulators", s.handleSimulators)
	mux.HandleFunc("/api/v1/simulators/", s.handleSimulatorRoutes)
	mux.Handle("/assets/", s.staticHandler())
	mux.HandleFunc("/", s.handleIndex)
	return s.instrument(mux)
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

	stats := s.service.Stats()
	if wantsPrometheus(r) {
		writePrometheusMetrics(w, stats)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/devices" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleListDevices(w, r)
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
	case "tags":
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleUpdateDeviceTags(w, r, deviceID)
	case "shadow":
		switch r.Method {
		case http.MethodGet:
			s.handleGetShadow(w, r, deviceID)
		case http.MethodPut:
			s.handleUpdateShadow(w, r, deviceID)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
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
		TenantID  string            `json:"tenant_id"`
		Name      string            `json:"name"`
		ProductID string            `json:"product_id"`
		Tags      map[string]string `json:"tags"`
		Metadata  map[string]string `json:"metadata"`
	}

	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	device, err := s.service.CreateDeviceWithTenant(r.Context(), request.TenantID, request.Name, request.ProductID, request.Tags, request.Metadata)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrTenantNotFound) || errors.Is(err, store.ErrProductNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, device)
}

func (s *Server) handleUpdateDeviceTags(w http.ResponseWriter, r *http.Request, deviceID string) {
	var request struct {
		Tags map[string]string `json:"tags"`
	}

	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	device, err := s.service.UpdateDeviceTags(r.Context(), deviceID, request.Tags)
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

func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	productID := strings.TrimSpace(r.URL.Query().Get("product_id"))
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	devices, err := s.service.ListDevicesByTenant(r.Context(), tenantID, productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, devices)
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

func (s *Server) handleGetShadow(w http.ResponseWriter, r *http.Request, deviceID string) {
	shadow, err := s.service.GetShadow(r.Context(), deviceID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrDeviceNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, shadow)
}

func (s *Server) handleUpdateShadow(w http.ResponseWriter, r *http.Request, deviceID string) {
	var request struct {
		Desired map[string]any `json:"desired"`
	}

	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	shadow, err := s.service.UpdateDesiredShadow(r.Context(), deviceID, request.Desired)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrDeviceNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, shadow)
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

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(payload []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(payload)
}

func (s *Server) instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		s.service.RecordHTTPRequest(status)
	})
}

func wantsPrometheus(r *http.Request) bool {
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("format")), "prometheus") {
		return true
	}
	return strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/plain")
}

func writePrometheusMetrics(w http.ResponseWriter, stats model.Stats) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	lines := []string{
		fmt.Sprintf("mvp_registered_devices %d", stats.RegisteredDevices),
		fmt.Sprintf("mvp_online_devices %d", stats.OnlineDevices),
		fmt.Sprintf("mvp_total_connections %d", stats.TotalConnections),
		fmt.Sprintf("mvp_rejected_connections %d", stats.RejectedConnections),
		fmt.Sprintf("mvp_telemetry_received_total %d", stats.TelemetryReceived),
		fmt.Sprintf("mvp_commands_sent_total %d", stats.CommandsSent),
		fmt.Sprintf("mvp_command_acks_total %d", stats.CommandAcks),
		fmt.Sprintf("mvp_uptime_seconds %d", stats.UptimeSeconds),
		fmt.Sprintf("mvp_runtime_goroutines %d", stats.Runtime.Goroutines),
		fmt.Sprintf("mvp_runtime_heap_alloc_bytes %d", stats.Runtime.HeapAllocBytes),
		fmt.Sprintf("mvp_runtime_heap_inuse_bytes %d", stats.Runtime.HeapInuseBytes),
		fmt.Sprintf("mvp_runtime_stack_inuse_bytes %d", stats.Runtime.StackInuseBytes),
		fmt.Sprintf("mvp_runtime_sys_bytes %d", stats.Runtime.SysBytes),
		fmt.Sprintf("mvp_runtime_gc_cycles_total %d", stats.Runtime.NumGC),
		fmt.Sprintf("mvp_http_requests_total %d", stats.Ingress.HTTPRequests),
		fmt.Sprintf("mvp_http_errors_total %d", stats.Ingress.HTTPErrors),
		fmt.Sprintf("mvp_http_ingest_accepted_total %d", stats.Ingress.HTTPIngestAccepted),
		fmt.Sprintf("mvp_http_ingest_rejected_total %d", stats.Ingress.HTTPIngestRejected),
		fmt.Sprintf("mvp_tcp_telemetry_accepted_total %d", stats.Ingress.TCPTelemetryAccepted),
		fmt.Sprintf("mvp_tcp_command_acks_total %d", stats.Ingress.TCPCommandAcks),
		fmt.Sprintf("mvp_mqtt_messages_received_total %d", stats.Ingress.MQTTMessagesReceived),
		fmt.Sprintf("mvp_mqtt_telemetry_accepted_total %d", stats.Ingress.MQTTTelemetryAccepted),
		fmt.Sprintf("mvp_mqtt_command_acks_total %d", stats.Ingress.MQTTCommandAcks),
		fmt.Sprintf("mvp_bytes_ingested_total %d", stats.Ingress.BytesIngested),
		fmt.Sprintf("mvp_telemetry_values_total %d", stats.Ingress.TelemetryValues),
		fmt.Sprintf("mvp_storage_tenants %d", stats.Storage.Tenants),
		fmt.Sprintf("mvp_storage_products %d", stats.Storage.Products),
		fmt.Sprintf("mvp_storage_devices %d", stats.Storage.Devices),
		fmt.Sprintf("mvp_storage_groups %d", stats.Storage.Groups),
		fmt.Sprintf("mvp_storage_rules %d", stats.Storage.Rules),
		fmt.Sprintf("mvp_storage_config_profiles %d", stats.Storage.ConfigProfiles),
		fmt.Sprintf("mvp_storage_firmware_artifacts %d", stats.Storage.FirmwareArtifacts),
		fmt.Sprintf("mvp_storage_ota_campaigns %d", stats.Storage.OTACampaigns),
		fmt.Sprintf("mvp_storage_shadows %d", stats.Storage.Shadows),
		fmt.Sprintf("mvp_storage_commands %d", stats.Storage.Commands),
		fmt.Sprintf("mvp_storage_alerts %d", stats.Storage.Alerts),
		fmt.Sprintf("mvp_storage_telemetry_series %d", stats.Storage.TelemetrySeries),
		fmt.Sprintf("mvp_storage_telemetry_samples %d", stats.Storage.TelemetrySamples),
		fmt.Sprintf("mvp_storage_persist_errors_total %d", stats.Storage.PersistErrors),
		fmt.Sprintf("mvp_transport_tcp_online_devices %d", stats.Transport.TCPOnlineDevices),
		fmt.Sprintf("mvp_transport_mqtt_online_devices %d", stats.Transport.MQTTOnlineDevices),
		fmt.Sprintf("mvp_transport_tcp_commands_published_total %d", stats.Transport.TCPCommandsPublished),
		fmt.Sprintf("mvp_transport_mqtt_commands_published_total %d", stats.Transport.MQTTCommandsPublished),
		fmt.Sprintf("mvp_transport_mqtt_connections_accepted_total %d", stats.Transport.MQTTConnectionsAccepted),
		fmt.Sprintf("mvp_transport_mqtt_connections_rejected_total %d", stats.Transport.MQTTConnectionsRejected),
	}
	if stats.Storage.LastPersistedAt != nil {
		lines = append(lines, fmt.Sprintf("mvp_storage_last_persisted_unix %d", stats.Storage.LastPersistedAt.Unix()))
	}
	_, _ = io.WriteString(w, strings.Join(lines, "\n")+"\n")
}

func approximatePayloadBytes(payload any) int {
	data, err := json.Marshal(payload)
	if err != nil {
		return 0
	}
	return len(data)
}
