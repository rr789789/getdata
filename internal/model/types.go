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
