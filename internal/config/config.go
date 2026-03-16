package config

import (
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr           string
	GatewayAddr        string
	GatewayDialAddr    string
	MQTTAddr           string
	LogLevel           string
	ShutdownTimeout    time.Duration
	DeviceAuthTimeout  time.Duration
	DeviceIdleTimeout  time.Duration
	DeviceWriteTimeout time.Duration
	DeviceQueueSize    int
	TelemetryRetention int
	MaxMessageBytes    int
	StoreBackend       string
	StorePath          string
}

func Load() Config {
	gatewayAddr := getEnv("MVP_GATEWAY_ADDR", ":18830")

	return Config{
		HTTPAddr:           getEnv("MVP_HTTP_ADDR", ":8080"),
		GatewayAddr:        gatewayAddr,
		GatewayDialAddr:    getEnv("MVP_GATEWAY_DIAL_ADDR", defaultGatewayDialAddr(gatewayAddr)),
		MQTTAddr:           getEnv("MVP_MQTT_ADDR", ":1883"),
		LogLevel:           getEnv("MVP_LOG_LEVEL", "info"),
		ShutdownTimeout:    getEnvDuration("MVP_SHUTDOWN_TIMEOUT", 10*time.Second),
		DeviceAuthTimeout:  getEnvDuration("MVP_DEVICE_AUTH_TIMEOUT", 15*time.Second),
		DeviceIdleTimeout:  getEnvDuration("MVP_DEVICE_IDLE_TIMEOUT", 90*time.Second),
		DeviceWriteTimeout: getEnvDuration("MVP_DEVICE_WRITE_TIMEOUT", 5*time.Second),
		DeviceQueueSize:    getEnvInt("MVP_DEVICE_QUEUE_SIZE", 128),
		TelemetryRetention: getEnvInt("MVP_TELEMETRY_RETENTION", 200),
		MaxMessageBytes:    getEnvInt("MVP_MAX_MESSAGE_BYTES", 1024*1024),
		StoreBackend:       getEnv("MVP_STORE_BACKEND", "file"),
		StorePath:          getEnv("MVP_STORE_PATH", "./data/mvp-platform-state.json"),
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

func defaultGatewayDialAddr(listenAddr string) string {
	listenAddr = strings.TrimSpace(listenAddr)
	if listenAddr == "" {
		return "127.0.0.1:18830"
	}

	if strings.HasPrefix(listenAddr, ":") {
		return "127.0.0.1" + listenAddr
	}

	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return listenAddr
	}

	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}
