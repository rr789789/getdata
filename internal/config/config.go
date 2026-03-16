package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr           string
	GatewayAddr        string
	LogLevel           string
	ShutdownTimeout    time.Duration
	DeviceAuthTimeout  time.Duration
	DeviceIdleTimeout  time.Duration
	DeviceWriteTimeout time.Duration
	DeviceQueueSize    int
	TelemetryRetention int
	MaxMessageBytes    int
}

func Load() Config {
	return Config{
		HTTPAddr:           getEnv("MVP_HTTP_ADDR", ":8080"),
		GatewayAddr:        getEnv("MVP_GATEWAY_ADDR", ":18830"),
		LogLevel:           getEnv("MVP_LOG_LEVEL", "info"),
		ShutdownTimeout:    getEnvDuration("MVP_SHUTDOWN_TIMEOUT", 10*time.Second),
		DeviceAuthTimeout:  getEnvDuration("MVP_DEVICE_AUTH_TIMEOUT", 15*time.Second),
		DeviceIdleTimeout:  getEnvDuration("MVP_DEVICE_IDLE_TIMEOUT", 90*time.Second),
		DeviceWriteTimeout: getEnvDuration("MVP_DEVICE_WRITE_TIMEOUT", 5*time.Second),
		DeviceQueueSize:    getEnvInt("MVP_DEVICE_QUEUE_SIZE", 128),
		TelemetryRetention: getEnvInt("MVP_TELEMETRY_RETENTION", 200),
		MaxMessageBytes:    getEnvInt("MVP_MAX_MESSAGE_BYTES", 1024*1024),
	}
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
