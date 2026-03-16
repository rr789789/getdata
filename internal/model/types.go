package model

import "time"

type Tenant struct {
	ID          string            `json:"id"`
	Slug        string            `json:"slug"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type TenantSummary struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type TenantView struct {
	Tenant             Tenant `json:"tenant"`
	ProductCount       int    `json:"product_count"`
	DeviceCount        int    `json:"device_count"`
	GroupCount         int    `json:"group_count"`
	RuleCount          int    `json:"rule_count"`
	ConfigProfileCount int    `json:"config_profile_count"`
	FirmwareCount      int    `json:"firmware_count"`
	OTACampaignCount   int    `json:"ota_campaign_count"`
}

type Device struct {
	ID         string            `json:"id"`
	TenantID   string            `json:"tenant_id,omitempty"`
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
	ID         string         `json:"id"`
	DeviceID   string         `json:"device_id"`
	CampaignID string         `json:"campaign_id,omitempty"`
	Name       string         `json:"name"`
	Params     map[string]any `json:"params,omitempty"`
	Status     CommandStatus  `json:"status"`
	Result     string         `json:"result,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
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
	ID            string               `json:"id"`
	TenantID      string               `json:"tenant_id,omitempty"`
	Key           string               `json:"key"`
	Name          string               `json:"name"`
	Description   string               `json:"description,omitempty"`
	Metadata      map[string]string    `json:"metadata,omitempty"`
	AccessProfile ProductAccessProfile `json:"access_profile,omitempty"`
	ThingModel    ThingModel            `json:"thing_model"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

type ProductSummary struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type ProductView struct {
	Product     Product        `json:"product"`
	Tenant      *TenantSummary `json:"tenant,omitempty"`
	DeviceCount int            `json:"device_count"`
	OnlineCount int            `json:"online_count"`
}

type DeviceGroup struct {
	ID          string            `json:"id"`
	TenantID    string            `json:"tenant_id,omitempty"`
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
	Tenant      *TenantSummary  `json:"tenant,omitempty"`
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
	Tenant      *TenantSummary  `json:"tenant,omitempty"`
	Product     *ProductSummary `json:"product,omitempty"`
	Groups      []GroupSummary  `json:"groups,omitempty"`
	Online      bool            `json:"online"`
	ConnectedAt *time.Time      `json:"connected_at,omitempty"`
	LastSeen    *time.Time      `json:"last_seen,omitempty"`
}

type ConfigProfile struct {
	ID            string         `json:"id"`
	TenantID      string         `json:"tenant_id,omitempty"`
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
	Tenant  *TenantSummary `json:"tenant,omitempty"`
	Product *ProductSummary `json:"product,omitempty"`
}

type RuleCondition struct {
	Property string `json:"property"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

type RuleActionType string

const (
	RuleActionAlert       RuleActionType = "alert"
	RuleActionSendCommand RuleActionType = "send_command"
	RuleActionApplyConfig RuleActionType = "apply_config_profile"
)

type RuleAction struct {
	Type            RuleActionType `json:"type"`
	Name            string         `json:"name,omitempty"`
	Params          map[string]any `json:"params,omitempty"`
	ConfigProfileID string         `json:"config_profile_id,omitempty"`
	Severity        AlertSeverity  `json:"severity,omitempty"`
	Message         string         `json:"message,omitempty"`
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
	TenantID        string        `json:"tenant_id,omitempty"`
	Name            string        `json:"name"`
	Description     string        `json:"description,omitempty"`
	ProductID       string        `json:"product_id,omitempty"`
	GroupID         string        `json:"group_id,omitempty"`
	DeviceID        string        `json:"device_id,omitempty"`
	Enabled         bool          `json:"enabled"`
	Severity        AlertSeverity `json:"severity"`
	CooldownSeconds int           `json:"cooldown_seconds,omitempty"`
	Condition       RuleCondition `json:"condition"`
	Actions         []RuleAction  `json:"actions,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type RuleView struct {
	Rule            Rule            `json:"rule"`
	Tenant          *TenantSummary  `json:"tenant,omitempty"`
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
	TenantID    string        `json:"tenant_id,omitempty"`
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
	StartedAt           time.Time      `json:"started_at"`
	UptimeSeconds       int64          `json:"uptime_seconds"`
	Runtime             RuntimeStats   `json:"runtime"`
	Storage             StorageStats   `json:"storage"`
	Ingress             IngressStats   `json:"ingress"`
	Transport           TransportStats `json:"transport"`
}

type RuntimeStats struct {
	Goroutines      int    `json:"goroutines"`
	HeapAllocBytes  uint64 `json:"heap_alloc_bytes"`
	HeapInuseBytes  uint64 `json:"heap_inuse_bytes"`
	StackInuseBytes uint64 `json:"stack_inuse_bytes"`
	SysBytes        uint64 `json:"sys_bytes"`
	NumGC           uint32 `json:"num_gc"`
}

type StorageStats struct {
	Backend          string     `json:"backend"`
	PersistencePath  string     `json:"persistence_path,omitempty"`
	Tenants          int64      `json:"tenants"`
	Products         int64      `json:"products"`
	Devices          int64      `json:"devices"`
	Groups           int64      `json:"groups"`
	Rules            int64      `json:"rules"`
	ConfigProfiles   int64      `json:"config_profiles"`
	FirmwareArtifacts int64     `json:"firmware_artifacts"`
	OTACampaigns     int64      `json:"ota_campaigns"`
	Shadows          int64      `json:"shadows"`
	Commands         int64      `json:"commands"`
	Alerts           int64      `json:"alerts"`
	TelemetrySeries  int64      `json:"telemetry_series"`
	TelemetrySamples int64      `json:"telemetry_samples"`
	LastPersistedAt  *time.Time `json:"last_persisted_at,omitempty"`
	PersistErrors    int64      `json:"persist_errors"`
}

type IngressStats struct {
	HTTPRequests          int64 `json:"http_requests"`
	HTTPErrors            int64 `json:"http_errors"`
	HTTPIngestAccepted    int64 `json:"http_ingest_accepted"`
	HTTPIngestRejected    int64 `json:"http_ingest_rejected"`
	TCPTelemetryAccepted  int64 `json:"tcp_telemetry_accepted"`
	TCPCommandAcks        int64 `json:"tcp_command_acks"`
	MQTTMessagesReceived  int64 `json:"mqtt_messages_received"`
	MQTTTelemetryAccepted int64 `json:"mqtt_telemetry_accepted"`
	MQTTCommandAcks       int64 `json:"mqtt_command_acks"`
	BytesIngested         int64 `json:"bytes_ingested"`
	TelemetryValues       int64 `json:"telemetry_values"`
}

type TransportStats struct {
	TCPOnlineDevices         int64 `json:"tcp_online_devices"`
	MQTTOnlineDevices        int64 `json:"mqtt_online_devices"`
	TCPCommandsPublished     int64 `json:"tcp_commands_published"`
	MQTTCommandsPublished    int64 `json:"mqtt_commands_published"`
	MQTTConnectionsAccepted  int64 `json:"mqtt_connections_accepted"`
	MQTTConnectionsRejected  int64 `json:"mqtt_connections_rejected"`
}

type SimulatorLogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

type FirmwareArtifact struct {
	ID           string            `json:"id"`
	TenantID     string            `json:"tenant_id,omitempty"`
	ProductID    string            `json:"product_id,omitempty"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	FileName     string            `json:"file_name,omitempty"`
	URL          string            `json:"url"`
	Checksum     string            `json:"checksum,omitempty"`
	ChecksumType string            `json:"checksum_type,omitempty"`
	SizeBytes    int64             `json:"size_bytes,omitempty"`
	Notes        string            `json:"notes,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type FirmwareArtifactSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type FirmwareArtifactView struct {
	Artifact FirmwareArtifact `json:"artifact"`
	Tenant   *TenantSummary   `json:"tenant,omitempty"`
	Product  *ProductSummary  `json:"product,omitempty"`
}

type OTACampaignStatus string

const (
	OTACampaignStatusPending   OTACampaignStatus = "pending"
	OTACampaignStatusRunning   OTACampaignStatus = "running"
	OTACampaignStatusCompleted OTACampaignStatus = "completed"
	OTACampaignStatusPartial   OTACampaignStatus = "partial"
)

type OTACampaign struct {
	ID               string            `json:"id"`
	TenantID         string            `json:"tenant_id,omitempty"`
	Name             string            `json:"name"`
	FirmwareID       string            `json:"firmware_id"`
	ProductID        string            `json:"product_id,omitempty"`
	GroupID          string            `json:"group_id,omitempty"`
	DeviceID         string            `json:"device_id,omitempty"`
	Status           OTACampaignStatus `json:"status"`
	TotalDevices     int               `json:"total_devices"`
	DispatchedCount  int               `json:"dispatched_count"`
	AckedCount       int               `json:"acked_count"`
	FailedCount      int               `json:"failed_count"`
	LastDispatchedAt *time.Time        `json:"last_dispatched_at,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type OTACampaignView struct {
	Campaign OTACampaign             `json:"campaign"`
	Tenant   *TenantSummary          `json:"tenant,omitempty"`
	Product  *ProductSummary         `json:"product,omitempty"`
	Group    *GroupSummary           `json:"group,omitempty"`
	Device   *DeviceSummary          `json:"device,omitempty"`
	Firmware *FirmwareArtifactSummary `json:"firmware,omitempty"`
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
