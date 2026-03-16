package model

type DeviceMessage struct {
	Type      string         `json:"type"`
	DeviceID  string         `json:"device_id,omitempty"`
	Token     string         `json:"token,omitempty"`
	TS        int64          `json:"ts,omitempty"`
	Values    map[string]any `json:"values,omitempty"`
	CommandID string         `json:"command_id,omitempty"`
	Status    string         `json:"status,omitempty"`
	Message   string         `json:"message,omitempty"`
}

type ServerMessage struct {
	Type       string         `json:"type"`
	ServerTime int64          `json:"server_time,omitempty"`
	DeviceID   string         `json:"device_id,omitempty"`
	CommandID  string         `json:"command_id,omitempty"`
	Name       string         `json:"name,omitempty"`
	Params     map[string]any `json:"params,omitempty"`
	Status     string         `json:"status,omitempty"`
	Message    string         `json:"message,omitempty"`
	Error      string         `json:"error,omitempty"`
}
