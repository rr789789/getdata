package model

import "time"

type Device struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Token     string            `json:"token,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

type Telemetry struct {
	DeviceID  string         `json:"device_id"`
	Timestamp time.Time      `json:"timestamp"`
	Values    map[string]any `json:"values"`
}

type CommandStatus string

const (
	CommandStatusPending CommandStatus = "pending"
	CommandStatusSent    CommandStatus = "sent"
	CommandStatusAcked   CommandStatus = "acked"
	CommandStatusFailed  CommandStatus = "failed"
)

type Command struct {
	ID        string         `json:"id"`
	DeviceID  string         `json:"device_id"`
	Name      string         `json:"name"`
	Params    map[string]any `json:"params,omitempty"`
	Status    CommandStatus  `json:"status"`
	Result    string         `json:"result,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type DeviceView struct {
	Device      Device     `json:"device"`
	Online      bool       `json:"online"`
	ConnectedAt *time.Time `json:"connected_at,omitempty"`
	LastSeen    *time.Time `json:"last_seen,omitempty"`
}

type Stats struct {
	RegisteredDevices   int64 `json:"registered_devices"`
	OnlineDevices       int64 `json:"online_devices"`
	TotalConnections    int64 `json:"total_connections"`
	RejectedConnections int64 `json:"rejected_connections"`
	TelemetryReceived   int64 `json:"telemetry_received"`
	CommandsSent        int64 `json:"commands_sent"`
	CommandAcks         int64 `json:"command_acks"`
}

type SimulatorLogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

type SimulatorView struct {
	ID                  string            `json:"id"`
	Device              Device            `json:"device"`
	Connected           bool              `json:"connected"`
	AutoAck             bool              `json:"auto_ack"`
	AutoPing            bool              `json:"auto_ping"`
	AutoTelemetry       bool              `json:"auto_telemetry"`
	TelemetryIntervalMS int               `json:"telemetry_interval_ms"`
	DefaultValues       map[string]any    `json:"default_values,omitempty"`
	LastConnectAt       *time.Time        `json:"last_connect_at,omitempty"`
	LastDisconnectAt    *time.Time        `json:"last_disconnect_at,omitempty"`
	LastPingAt          *time.Time        `json:"last_ping_at,omitempty"`
	LastTelemetryAt     *time.Time        `json:"last_telemetry_at,omitempty"`
	LastCommandAt       *time.Time        `json:"last_command_at,omitempty"`
	LastError           string            `json:"last_error,omitempty"`
	Logs                []SimulatorLogEntry `json:"logs,omitempty"`
}
