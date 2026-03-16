package model

import "time"

type Device struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	ProductID  string            `json:"product_id,omitempty"`
	ProductKey string            `json:"product_key,omitempty"`
	Token      string            `json:"token,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

type DeviceSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

type ThingModelParameter struct {
	Identifier  string `json:"identifier"`
	Name        string `json:"name"`
	DataType    string `json:"data_type"`
	Description string `json:"description,omitempty"`
}

type ThingModelProperty struct {
	Identifier  string `json:"identifier"`
	Name        string `json:"name"`
	DataType    string `json:"data_type"`
	AccessMode  string `json:"access_mode,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
}

type ThingModelEvent struct {
	Identifier  string                `json:"identifier"`
	Name        string                `json:"name"`
	Output      []ThingModelParameter `json:"output,omitempty"`
	Description string                `json:"description,omitempty"`
}

type ThingModelService struct {
	Identifier  string                `json:"identifier"`
	Name        string                `json:"name"`
	Input       []ThingModelParameter `json:"input,omitempty"`
	Output      []ThingModelParameter `json:"output,omitempty"`
	Description string                `json:"description,omitempty"`
}

type ThingModel struct {
	Properties []ThingModelProperty `json:"properties,omitempty"`
	Events     []ThingModelEvent    `json:"events,omitempty"`
	Services   []ThingModelService  `json:"services,omitempty"`
	Version    int64                `json:"version"`
	UpdatedAt  time.Time            `json:"updated_at,omitempty"`
}

type Product struct {
	ID          string            `json:"id"`
	Key         string            `json:"key"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	AccessProfile ProductAccessProfile `json:"access_profile,omitempty"`
	ThingModel  ThingModel        `json:"thing_model"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type ProductSummary struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type ProductView struct {
	Product     Product `json:"product"`
	DeviceCount int     `json:"device_count"`
	OnlineCount int     `json:"online_count"`
}

type DeviceGroup struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	ProductID   string            `json:"product_id,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type GroupSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type GroupView struct {
	Group       DeviceGroup     `json:"group"`
	Product     *ProductSummary `json:"product,omitempty"`
	DeviceCount int             `json:"device_count"`
	OnlineCount int             `json:"online_count"`
}

type DeviceShadow struct {
	DeviceID       string         `json:"device_id"`
	ProductID      string         `json:"product_id,omitempty"`
	Reported       map[string]any `json:"reported,omitempty"`
	Desired        map[string]any `json:"desired,omitempty"`
	Version        int64          `json:"version"`
	UpdatedAt      time.Time      `json:"updated_at"`
	LastReportedAt *time.Time     `json:"last_reported_at,omitempty"`
	LastDesiredAt  *time.Time     `json:"last_desired_at,omitempty"`
}

type DeviceView struct {
	Device      Device          `json:"device"`
	Product     *ProductSummary `json:"product,omitempty"`
	Groups      []GroupSummary  `json:"groups,omitempty"`
	Online      bool            `json:"online"`
	ConnectedAt *time.Time      `json:"connected_at,omitempty"`
	LastSeen    *time.Time      `json:"last_seen,omitempty"`
}

type ConfigProfile struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	ProductID     string         `json:"product_id,omitempty"`
	Values        map[string]any `json:"values"`
	AppliedCount  int64          `json:"applied_count"`
	LastAppliedAt *time.Time     `json:"last_applied_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type ConfigProfileView struct {
	Profile ConfigProfile  `json:"profile"`
	Product *ProductSummary `json:"product,omitempty"`
}

type RuleCondition struct {
	Property string `json:"property"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

type AlertStatus string

const (
	AlertStatusNew          AlertStatus = "new"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
)

type Rule struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Description     string        `json:"description,omitempty"`
	ProductID       string        `json:"product_id,omitempty"`
	GroupID         string        `json:"group_id,omitempty"`
	DeviceID        string        `json:"device_id,omitempty"`
	Enabled         bool          `json:"enabled"`
	Severity        AlertSeverity `json:"severity"`
	CooldownSeconds int           `json:"cooldown_seconds,omitempty"`
	Condition       RuleCondition `json:"condition"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type RuleView struct {
	Rule            Rule            `json:"rule"`
	Product         *ProductSummary `json:"product,omitempty"`
	Group           *GroupSummary   `json:"group,omitempty"`
	Device          *DeviceSummary  `json:"device,omitempty"`
	TriggeredCount  int64           `json:"triggered_count"`
	LastTriggeredAt *time.Time      `json:"last_triggered_at,omitempty"`
}

type AlertEvent struct {
	ID          string        `json:"id"`
	RuleID      string        `json:"rule_id"`
	RuleName    string        `json:"rule_name"`
	ProductID   string        `json:"product_id,omitempty"`
	GroupID     string        `json:"group_id,omitempty"`
	DeviceID    string        `json:"device_id"`
	DeviceName  string        `json:"device_name"`
	Property    string        `json:"property"`
	Operator    string        `json:"operator"`
	Threshold   any           `json:"threshold"`
	Value       any           `json:"value"`
	Severity    AlertSeverity `json:"severity"`
	Status      AlertStatus   `json:"status"`
	Message     string        `json:"message"`
	Note        string        `json:"note,omitempty"`
	AckAt       *time.Time    `json:"ack_at,omitempty"`
	ResolvedAt  *time.Time    `json:"resolved_at,omitempty"`
	TriggeredAt time.Time     `json:"triggered_at"`
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
	ID                  string              `json:"id"`
	Device              Device              `json:"device"`
	Connected           bool                `json:"connected"`
	AutoAck             bool                `json:"auto_ack"`
	AutoPing            bool                `json:"auto_ping"`
	AutoTelemetry       bool                `json:"auto_telemetry"`
	TelemetryIntervalMS int                 `json:"telemetry_interval_ms"`
	DefaultValues       map[string]any      `json:"default_values,omitempty"`
	LastConnectAt       *time.Time          `json:"last_connect_at,omitempty"`
	LastDisconnectAt    *time.Time          `json:"last_disconnect_at,omitempty"`
	LastPingAt          *time.Time          `json:"last_ping_at,omitempty"`
	LastTelemetryAt     *time.Time          `json:"last_telemetry_at,omitempty"`
	LastCommandAt       *time.Time          `json:"last_command_at,omitempty"`
	LastError           string              `json:"last_error,omitempty"`
	Logs                []SimulatorLogEntry `json:"logs,omitempty"`
}
