package ingest

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"mvp-platform/internal/model"
)

func ExtractTimestamp(payload map[string]any, fallback time.Time) time.Time {
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

func BuildValues(payload map[string]any, accessProfile model.ProductAccessProfile) (map[string]any, error) {
	if values, ok := extractFirstMap(payload, "values", "telemetry", "properties", "data"); ok && len(values) > 0 {
		return values, nil
	}

	if mapped := resolveMappedValues(payload, accessProfile); len(mapped) > 0 {
		return mapped, nil
	}

	if accessProfile.PayloadFormat == "flat_json" || accessProfile.PayloadFormat == "" || accessProfile.Protocol == "http_json" || accessProfile.Protocol == "mqtt_json" {
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
		case "token", "ts", "timestamp", "time", "protocol", "values", "telemetry", "properties", "data", "registers", "nodes", "objects", "device_id", "command_id", "status", "message", "type":
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
		value, ok := lookupSourceValue(payload, item.Source)
		if !ok {
			continue
		}
		result[item.Property] = applyMapping(value, item)
	}
	return result
}

func lookupSourceValue(payload map[string]any, source string) (any, bool) {
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
		if value, exists := payload[source]; exists {
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

func applyMapping(value any, item model.ProtocolPointMapping) any {
	scale := item.Scale
	if scale == 0 {
		scale = 1
	}
	if numeric, ok := float64Value(value); ok {
		return numeric*scale + item.Offset
	}
	return value
}

func float64Value(value any) (float64, bool) {
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
