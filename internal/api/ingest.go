package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

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
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	token := extractIngressToken(r, payload)
	if strings.TrimSpace(token) == "" {
		writeError(w, http.StatusUnauthorized, "device token is required")
		return
	}

	device, err := s.service.AuthenticateDevice(r.Context(), deviceID, token)
	if err != nil {
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
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err == nil {
			accessProfile = product.Product.AccessProfile
		}
	}

	values, err := buildIngressValues(payload, accessProfile)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(values) == 0 {
		writeError(w, http.StatusBadRequest, "no telemetry values resolved from payload")
		return
	}

	at := extractIngressTimestamp(payload, time.Now().UTC())
	if err := s.service.HandleTelemetry(r.Context(), device.ID, at, values); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

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

func extractIngressTimestamp(payload map[string]any, fallback time.Time) time.Time {
	for _, key := range []string{"ts", "timestamp", "time"} {
		value, exists := payload[key]
		if !exists {
			continue
		}
		switch typed := value.(type) {
		case float64:
			if typed > 1e12 {
				return time.UnixMilli(int64(typed)).UTC()
			}
			return time.Unix(int64(typed), 0).UTC()
		case string:
			raw := strings.TrimSpace(typed)
			if raw == "" {
				continue
			}
			if unixValue, err := strconv.ParseInt(raw, 10, 64); err == nil {
				if unixValue > 1e12 {
					return time.UnixMilli(unixValue).UTC()
				}
				return time.Unix(unixValue, 0).UTC()
			}
			if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
				return parsed.UTC()
			}
		}
	}
	return fallback.UTC()
}

func buildIngressValues(payload map[string]any, accessProfile model.ProductAccessProfile) (map[string]any, error) {
	if values, ok := extractFirstMap(payload, "values", "telemetry", "properties", "data"); ok && len(values) > 0 {
		return values, nil
	}

	if mapped := resolveMappedValues(payload, accessProfile); len(mapped) > 0 {
		return mapped, nil
	}

	if accessProfile.PayloadFormat == "flat_json" || accessProfile.PayloadFormat == "" || accessProfile.Protocol == "http_json" {
		flat := filterReservedPayloadKeys(payload)
		if len(flat) > 0 {
			return flat, nil
		}
	}

	return nil, errors.New("unsupported payload shape")
}

func extractFirstMap(payload map[string]any, keys ...string) (map[string]any, bool) {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			if mapped, ok := value.(map[string]any); ok {
				return mapped, true
			}
		}
	}
	return nil, false
}

func filterReservedPayloadKeys(payload map[string]any) map[string]any {
	result := make(map[string]any)
	for key, value := range payload {
		switch key {
		case "token", "ts", "timestamp", "time", "protocol", "values", "telemetry", "properties", "data", "registers", "nodes", "objects", "device_id":
			continue
		default:
			result[key] = value
		}
	}
	return result
}

func resolveMappedValues(payload map[string]any, accessProfile model.ProductAccessProfile) map[string]any {
	if len(accessProfile.PointMappings) == 0 {
		return nil
	}

	result := make(map[string]any)
	for _, item := range accessProfile.PointMappings {
		value, ok := lookupIngressSourceValue(payload, item.Source)
		if !ok {
			continue
		}
		result[item.Property] = applyIngressMapping(value, item)
	}
	return result
}

func lookupIngressSourceValue(payload map[string]any, source string) (any, bool) {
	source = strings.TrimSpace(source)
	switch {
	case strings.HasPrefix(source, "register:"):
		return lookupNestedValue(payload, "registers", strings.TrimPrefix(source, "register:"))
	case strings.HasPrefix(source, "nodes."):
		return lookupNestedValue(payload, "nodes", strings.TrimPrefix(source, "nodes."))
	case strings.HasPrefix(source, "objects."):
		return lookupNestedValue(payload, "objects", strings.TrimPrefix(source, "objects."))
	case strings.HasPrefix(source, "values."):
		return lookupNestedValue(payload, "values", strings.TrimPrefix(source, "values."))
	case strings.Contains(source, "."):
		prefix, suffix, ok := strings.Cut(source, ".")
		if !ok {
			return nil, false
		}
		return lookupNestedValue(payload, prefix, suffix)
	default:
		value, exists := payload[source]
		if exists {
			return value, true
		}
		for _, containerKey := range []string{"registers", "nodes", "objects", "values", "properties", "data"} {
			if value, ok := lookupNestedValue(payload, containerKey, source); ok {
				return value, true
			}
		}
		return nil, false
	}
}

func lookupNestedValue(payload map[string]any, containerKey, itemKey string) (any, bool) {
	container, exists := payload[containerKey]
	if !exists {
		return nil, false
	}
	mapped, ok := container.(map[string]any)
	if !ok {
		return nil, false
	}
	value, exists := mapped[itemKey]
	return value, exists
}

func applyIngressMapping(value any, item model.ProtocolPointMapping) any {
	scale := item.Scale
	if scale == 0 {
		scale = 1
	}
	if numeric, ok := ingressFloat64(value); ok {
		return numeric*scale + item.Offset
	}
	return value
}

func ingressFloat64(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}
