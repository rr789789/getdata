package model

type ProtocolPointMapping struct {
	Source   string  `json:"source"`
	Property string  `json:"property"`
	Scale    float64 `json:"scale,omitempty"`
	Offset   float64 `json:"offset,omitempty"`
	Unit     string  `json:"unit,omitempty"`
}

type ProductAccessProfile struct {
	Transport      string                 `json:"transport,omitempty"`
	Protocol       string                 `json:"protocol,omitempty"`
	IngestMode     string                 `json:"ingest_mode,omitempty"`
	PayloadFormat  string                 `json:"payload_format,omitempty"`
	AuthMode       string                 `json:"auth_mode,omitempty"`
	SensorTemplate string                 `json:"sensor_template,omitempty"`
	Topic          string                 `json:"topic,omitempty"`
	Notes          string                 `json:"notes,omitempty"`
	Metadata       map[string]string      `json:"metadata,omitempty"`
	PointMappings  []ProtocolPointMapping `json:"point_mappings,omitempty"`
}

type ProtocolCatalogEntry struct {
	ID             string               `json:"id"`
	Name           string               `json:"name"`
	Description    string               `json:"description"`
	Transport      string               `json:"transport"`
	Protocol       string               `json:"protocol"`
	IngestMode     string               `json:"ingest_mode"`
	PayloadFormat  string               `json:"payload_format"`
	SensorTemplate string               `json:"sensor_template,omitempty"`
	ExampleTopic   string               `json:"example_topic,omitempty"`
	CommonSensors  []string             `json:"common_sensors,omitempty"`
	ThingModel     ThingModel           `json:"thing_model"`
	AccessProfile  ProductAccessProfile `json:"access_profile"`
	ExamplePayload map[string]any       `json:"example_payload,omitempty"`
}

func DefaultProtocolCatalog() []ProtocolCatalogEntry {
	return []ProtocolCatalogEntry{
		{
			ID:             "tcp-json-direct",
			Name:           "Direct TCP JSON",
			Description:    "Directly connected sensors over the built-in TCP gateway with JSON Lines payloads.",
			Transport:      "tcp",
			Protocol:       "tcp_json",
			IngestMode:     "gateway_tcp",
			PayloadFormat:  "json_values",
			SensorTemplate: "generic",
			CommonSensors:  []string{"temperature", "humidity", "pressure", "switch"},
			ThingModel: ThingModel{
				Properties: []ThingModelProperty{
					{Identifier: "temperature", Name: "Temperature", DataType: "float", AccessMode: "rw"},
					{Identifier: "humidity", Name: "Humidity", DataType: "float", AccessMode: "r"},
				},
				Services: []ThingModelService{
					{Identifier: "reboot", Name: "Reboot"},
				},
			},
			AccessProfile: ProductAccessProfile{
				Transport:      "tcp",
				Protocol:       "tcp_json",
				IngestMode:     "gateway_tcp",
				PayloadFormat:  "json_values",
				AuthMode:       "token",
				SensorTemplate: "generic",
			},
			ExamplePayload: map[string]any{
				"type":   "telemetry",
				"values": map[string]any{"temperature": 24.2, "humidity": 58},
			},
		},
		{
			ID:             "http-json-push",
			Name:           "HTTP JSON Push",
			Description:    "Sensors or edge gateways push telemetry to the built-in HTTP ingest endpoint.",
			Transport:      "http",
			Protocol:       "http_json",
			IngestMode:     "http_push",
			PayloadFormat:  "json_values",
			SensorTemplate: "environment",
			CommonSensors:  []string{"temperature", "humidity", "battery", "co2"},
			ThingModel: ThingModel{
				Properties: []ThingModelProperty{
					{Identifier: "temperature", Name: "Temperature", DataType: "float", AccessMode: "rw"},
					{Identifier: "humidity", Name: "Humidity", DataType: "float", AccessMode: "r"},
					{Identifier: "battery", Name: "Battery", DataType: "float", AccessMode: "r"},
				},
			},
			AccessProfile: ProductAccessProfile{
				Transport:      "http",
				Protocol:       "http_json",
				IngestMode:     "http_push",
				PayloadFormat:  "json_values",
				AuthMode:       "token",
				SensorTemplate: "environment",
			},
			ExamplePayload: map[string]any{
				"token":  "device_token",
				"values": map[string]any{"temperature": 23.8, "humidity": 54, "battery": 96},
			},
		},
		{
			ID:             "mqtt-json-bridge",
			Name:           "MQTT JSON Bridge",
			Description:    "MQTT broker data forwarded by an edge bridge into the HTTP ingest endpoint.",
			Transport:      "mqtt",
			Protocol:       "mqtt_json",
			IngestMode:     "bridge_http",
			PayloadFormat:  "json_values",
			SensorTemplate: "power-meter",
			ExampleTopic:   "factory/line-a/meter-01/up",
			CommonSensors:  []string{"power_meter", "energy_meter", "gateway"},
			ThingModel: ThingModel{
				Properties: []ThingModelProperty{
					{Identifier: "voltage", Name: "Voltage", DataType: "float", AccessMode: "r"},
					{Identifier: "current", Name: "Current", DataType: "float", AccessMode: "r"},
					{Identifier: "power", Name: "Power", DataType: "float", AccessMode: "r"},
					{Identifier: "energy", Name: "Energy", DataType: "float", AccessMode: "r"},
				},
			},
			AccessProfile: ProductAccessProfile{
				Transport:      "mqtt",
				Protocol:       "mqtt_json",
				IngestMode:     "bridge_http",
				PayloadFormat:  "json_values",
				AuthMode:       "token",
				SensorTemplate: "power-meter",
				Topic:          "factory/+/+/up",
			},
			ExamplePayload: map[string]any{
				"token":  "device_token",
				"values": map[string]any{"voltage": 220.6, "current": 4.1, "power": 905.2, "energy": 10234.7},
			},
		},
		{
			ID:             "modbus-register-map",
			Name:           "Modbus Register Map",
			Description:    "Common RS485 and Modbus TCP sensors mapped from holding/input registers into thing model properties.",
			Transport:      "rs485",
			Protocol:       "modbus_rtu",
			IngestMode:     "http_push",
			PayloadFormat:  "register_map",
			SensorTemplate: "temp-humidity-rs485",
			CommonSensors:  []string{"temp_humidity", "pressure", "level", "flow_meter"},
			ThingModel: ThingModel{
				Properties: []ThingModelProperty{
					{Identifier: "temperature", Name: "Temperature", DataType: "float", AccessMode: "r"},
					{Identifier: "humidity", Name: "Humidity", DataType: "float", AccessMode: "r"},
				},
			},
			AccessProfile: ProductAccessProfile{
				Transport:      "rs485",
				Protocol:       "modbus_rtu",
				IngestMode:     "http_push",
				PayloadFormat:  "register_map",
				AuthMode:       "token",
				SensorTemplate: "temp-humidity-rs485",
				PointMappings: []ProtocolPointMapping{
					{Source: "register:40001", Property: "temperature", Scale: 0.1},
					{Source: "register:40002", Property: "humidity", Scale: 0.1},
				},
			},
			ExamplePayload: map[string]any{
				"token": "device_token",
				"registers": map[string]any{
					"40001": 231,
					"40002": 550,
				},
			},
		},
		{
			ID:             "modbus-tcp-register-map",
			Name:           "Modbus TCP Register Map",
			Description:    "Ethernet-connected power meters and industrial sensors mapped from Modbus TCP registers.",
			Transport:      "ethernet",
			Protocol:       "modbus_tcp",
			IngestMode:     "http_push",
			PayloadFormat:  "register_map",
			SensorTemplate: "power-meter",
			CommonSensors:  []string{"power_meter", "energy_meter", "pressure", "flow_meter"},
			ThingModel: ThingModel{
				Properties: []ThingModelProperty{
					{Identifier: "voltage", Name: "Voltage", DataType: "float", AccessMode: "r"},
					{Identifier: "current", Name: "Current", DataType: "float", AccessMode: "r"},
					{Identifier: "power", Name: "Power", DataType: "float", AccessMode: "r"},
					{Identifier: "energy", Name: "Energy", DataType: "float", AccessMode: "r"},
				},
			},
			AccessProfile: ProductAccessProfile{
				Transport:      "ethernet",
				Protocol:       "modbus_tcp",
				IngestMode:     "http_push",
				PayloadFormat:  "register_map",
				AuthMode:       "token",
				SensorTemplate: "power-meter",
				PointMappings: []ProtocolPointMapping{
					{Source: "register:30001", Property: "voltage", Scale: 0.1},
					{Source: "register:30002", Property: "current", Scale: 0.01},
					{Source: "register:30003", Property: "power", Scale: 0.1},
					{Source: "register:30004", Property: "energy", Scale: 0.1},
				},
			},
			ExamplePayload: map[string]any{
				"token": "device_token",
				"registers": map[string]any{
					"30001": 2214,
					"30002": 418,
					"30003": 9231,
					"30004": 120450,
				},
			},
		},
		{
			ID:             "opcua-node-map",
			Name:           "OPC UA Node Map",
			Description:    "Industrial PLC and SCADA values mapped from OPC UA nodes through a bridge.",
			Transport:      "opcua",
			Protocol:       "opcua_json",
			IngestMode:     "bridge_http",
			PayloadFormat:  "mapped_json",
			SensorTemplate: "industrial-plc",
			CommonSensors:  []string{"plc", "energy_meter", "vibration", "temperature"},
			ThingModel: ThingModel{
				Properties: []ThingModelProperty{
					{Identifier: "temperature", Name: "Temperature", DataType: "float", AccessMode: "r"},
					{Identifier: "running", Name: "Running", DataType: "bool", AccessMode: "r"},
				},
			},
			AccessProfile: ProductAccessProfile{
				Transport:      "opcua",
				Protocol:       "opcua_json",
				IngestMode:     "bridge_http",
				PayloadFormat:  "mapped_json",
				AuthMode:       "token",
				SensorTemplate: "industrial-plc",
				PointMappings: []ProtocolPointMapping{
					{Source: "nodes.ns=2;s=Line1.Temperature", Property: "temperature"},
					{Source: "nodes.ns=2;s=Line1.Running", Property: "running"},
				},
			},
			ExamplePayload: map[string]any{
				"token": "device_token",
				"nodes": map[string]any{
					"ns=2;s=Line1.Temperature": 48.6,
					"ns=2;s=Line1.Running":     true,
				},
			},
		},
		{
			ID:             "bacnet-object-map",
			Name:           "BACnet Object Map",
			Description:    "Building automation values mapped from BACnet objects through a bridge or gateway.",
			Transport:      "bacnet",
			Protocol:       "bacnet_ip",
			IngestMode:     "bridge_http",
			PayloadFormat:  "mapped_json",
			SensorTemplate: "building-hvac",
			CommonSensors:  []string{"hvac", "temperature", "humidity", "co2"},
			ThingModel: ThingModel{
				Properties: []ThingModelProperty{
					{Identifier: "temperature", Name: "Temperature", DataType: "float", AccessMode: "r"},
					{Identifier: "humidity", Name: "Humidity", DataType: "float", AccessMode: "r"},
					{Identifier: "co2", Name: "CO2", DataType: "float", AccessMode: "r"},
				},
			},
			AccessProfile: ProductAccessProfile{
				Transport:      "bacnet",
				Protocol:       "bacnet_ip",
				IngestMode:     "bridge_http",
				PayloadFormat:  "mapped_json",
				AuthMode:       "token",
				SensorTemplate: "building-hvac",
				PointMappings: []ProtocolPointMapping{
					{Source: "objects.analogInput:1.presentValue", Property: "temperature"},
					{Source: "objects.analogInput:2.presentValue", Property: "humidity"},
					{Source: "objects.analogInput:3.presentValue", Property: "co2"},
				},
			},
			ExamplePayload: map[string]any{
				"token": "device_token",
				"objects": map[string]any{
					"analogInput:1.presentValue": 25.4,
					"analogInput:2.presentValue": 55.8,
					"analogInput:3.presentValue": 741,
				},
			},
		},
		{
			ID:             "lorawan-uplink",
			Name:           "LoRaWAN Uplink",
			Description:    "LoRaWAN sensors forwarded by network server webhook or bridge into the platform.",
			Transport:      "lorawan",
			Protocol:       "lorawan_uplink",
			IngestMode:     "bridge_http",
			PayloadFormat:  "json_values",
			SensorTemplate: "battery-env",
			CommonSensors:  []string{"environment", "soil", "water-level", "tracking"},
			ThingModel: ThingModel{
				Properties: []ThingModelProperty{
					{Identifier: "temperature", Name: "Temperature", DataType: "float", AccessMode: "r"},
					{Identifier: "humidity", Name: "Humidity", DataType: "float", AccessMode: "r"},
					{Identifier: "battery", Name: "Battery", DataType: "float", AccessMode: "r"},
				},
			},
			AccessProfile: ProductAccessProfile{
				Transport:      "lorawan",
				Protocol:       "lorawan_uplink",
				IngestMode:     "bridge_http",
				PayloadFormat:  "json_values",
				AuthMode:       "token",
				SensorTemplate: "battery-env",
			},
			ExamplePayload: map[string]any{
				"token":  "device_token",
				"values": map[string]any{"temperature": 19.4, "humidity": 67, "battery": 88},
			},
		},
	}
}
