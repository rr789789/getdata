package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"mvp-platform/internal/ingest"
	"mvp-platform/internal/model"
	"mvp-platform/internal/store"
)

func (s *Server) handleProtocolCatalog(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/protocol-catalog" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, s.service.ProtocolCatalog())
}

func (s *Server) handleHTTPIngestRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/ingest/http/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	s.handleHTTPIngest(w, r, path)
}

func (s *Server) handleHTTPIngest(w http.ResponseWriter, r *http.Request, deviceID string) {
	var payload map[string]any
	if err := decodeJSON(r, &payload); err != nil {
		s.service.RecordHTTPIngestResult(false, 0, 0)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	token := extractIngressToken(r, payload)
	if strings.TrimSpace(token) == "" {
		s.service.RecordHTTPIngestResult(false, 0, 0)
		writeError(w, http.StatusUnauthorized, "device token is required")
		return
	}

	device, err := s.service.AuthenticateDevice(r.Context(), deviceID, token)
	if err != nil {
		s.service.RecordHTTPIngestResult(false, 0, 0)
		switch {
		case errors.Is(err, store.ErrDeviceNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, store.ErrInvalidCredential):
			writeError(w, http.StatusUnauthorized, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	var accessProfile model.ProductAccessProfile
	if device.ProductID != "" {
		product, err := s.service.GetProduct(r.Context(), device.ProductID)
		if err != nil && !errors.Is(err, store.ErrProductNotFound) {
			s.service.RecordHTTPIngestResult(false, 0, 0)
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err == nil {
			accessProfile = product.Product.AccessProfile
		}
	}

	values, err := ingest.BuildValues(payload, accessProfile)
	if err != nil {
		s.service.RecordHTTPIngestResult(false, 0, 0)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(values) == 0 {
		s.service.RecordHTTPIngestResult(false, 0, 0)
		writeError(w, http.StatusBadRequest, "no telemetry values resolved from payload")
		return
	}

	at := ingest.ExtractTimestamp(payload, time.Now().UTC())
	if err := s.service.HandleTelemetry(r.Context(), device.ID, at, values); err != nil {
		s.service.RecordHTTPIngestResult(false, 0, len(values))
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.service.RecordHTTPIngestResult(true, approximatePayloadBytes(payload), len(values))

	shadow, err := s.service.GetShadow(r.Context(), device.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"device_id":      device.ID,
		"protocol":       accessProfile.Protocol,
		"ingest_mode":    accessProfile.IngestMode,
		"accepted_at":    time.Now().UTC(),
		"resolved_values": values,
		"shadow":         shadow,
	})
}

func extractIngressToken(r *http.Request, payload map[string]any) string {
	if raw := strings.TrimSpace(r.URL.Query().Get("token")); raw != "" {
		return raw
	}
	if raw := strings.TrimSpace(r.Header.Get("X-Device-Token")); raw != "" {
		return raw
	}
	if raw := strings.TrimSpace(r.Header.Get("Authorization")); raw != "" {
		if strings.HasPrefix(strings.ToLower(raw), "bearer ") {
			return strings.TrimSpace(raw[7:])
		}
		return raw
	}
	if value, ok := payload["token"].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}
