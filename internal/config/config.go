package config

import (
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	NodeID             string
	NodeRole           string
	HTTPAddr           string
	GatewayAddr        string
	GatewayDialAddr    string
	MQTTAddr           string
	LogLevel           string
	CORSAllowedOrigins []string
	DisableEmbeddedUI  bool
	ShutdownTimeout    time.Duration
	DeviceAuthTimeout  time.Duration
	DeviceIdleTimeout  time.Duration
	DeviceWriteTimeout time.Duration
	DeviceQueueSize    int
	TelemetryRetention int
	MaxMessageBytes    int
	StoreBackend       string
	StorePath          string
	ReplicaPeers       []string
	ReplicaToken       string
	ReplicaTimeout     time.Duration
}

func Load() Config {
	gatewayAddr := getEnv("MVP_GATEWAY_ADDR", ":18830")

	return Config{
		NodeID:             getEnv("MVP_NODE_ID", defaultNodeID()),
		NodeRole:           normalizeNodeRole(getEnv("MVP_NODE_ROLE", "primary")),
		HTTPAddr:           getEnv("MVP_HTTP_ADDR", ":8080"),
		GatewayAddr:        gatewayAddr,
		GatewayDialAddr:    getEnv("MVP_GATEWAY_DIAL_ADDR", defaultGatewayDialAddr(gatewayAddr)),
		MQTTAddr:           getEnv("MVP_MQTT_ADDR", ":1883"),
		LogLevel:           getEnv("MVP_LOG_LEVEL", "info"),
		CORSAllowedOrigins: getEnvList("MVP_CORS_ALLOWED_ORIGINS", []string{"*"}),
		DisableEmbeddedUI:  getEnvBool("MVP_DISABLE_EMBEDDED_UI", false),
		ShutdownTimeout:    getEnvDuration("MVP_SHUTDOWN_TIMEOUT", 10*time.Second),
		DeviceAuthTimeout:  getEnvDuration("MVP_DEVICE_AUTH_TIMEOUT", 15*time.Second),
		DeviceIdleTimeout:  getEnvDuration("MVP_DEVICE_IDLE_TIMEOUT", 90*time.Second),
		DeviceWriteTimeout: getEnvDuration("MVP_DEVICE_WRITE_TIMEOUT", 5*time.Second),
		DeviceQueueSize:    getEnvInt("MVP_DEVICE_QUEUE_SIZE", 128),
		TelemetryRetention: getEnvInt("MVP_TELEMETRY_RETENTION", 200),
		MaxMessageBytes:    getEnvInt("MVP_MAX_MESSAGE_BYTES", 1024*1024),
		StoreBackend:       getEnv("MVP_STORE_BACKEND", "file"),
		StorePath:          getEnv("MVP_STORE_PATH", "./data/mvp-platform-state.json"),
		ReplicaPeers:       getEnvList("MVP_HA_REPLICA_PEERS", nil),
		ReplicaToken:       getEnv("MVP_HA_REPLICA_TOKEN", ""),
		ReplicaTimeout:     getEnvDuration("MVP_HA_REPLICA_TIMEOUT", 3*time.Second),
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

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}

	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func getEnvList(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return append([]string(nil), fallback...)
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	if len(result) == 0 {
		return append([]string(nil), fallback...)
	}
	return result
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

func defaultNodeID() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		return "mvp-node"
	}
	return host
}

func normalizeNodeRole(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "backup", "secondary", "replica", "standby":
		return "standby"
	default:
		return "primary"
	}
}

func (c Config) IsStandby() bool {
	return normalizeNodeRole(c.NodeRole) == "standby"
}
