package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mvp-platform/internal/api"
	"mvp-platform/internal/config"
	"mvp-platform/internal/core"
	"mvp-platform/internal/model"
	"mvp-platform/internal/simulator"
	"mvp-platform/internal/store/memory"
)

func TestDeviceAPIFlow(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	createBody := []byte(`{"name":"api-device","metadata":{"site":"lab"}}`)
	createResp, err := http.Post(httpServer.URL+"/api/v1/devices", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST /api/v1/devices error = %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/v1/devices status = %d, want %d", createResp.StatusCode, http.StatusCreated)
	}

	var device model.Device
	if err := json.NewDecoder(createResp.Body).Decode(&device); err != nil {
		t.Fatalf("decode device error = %v", err)
	}
	if device.ID == "" || device.Token == "" {
		t.Fatalf("device id/token should not be empty: %#v", device)
	}

	listResp, err := http.Get(httpServer.URL + "/api/v1/devices")
	if err != nil {
		t.Fatalf("GET /api/v1/devices error = %v", err)
	}
	defer listResp.Body.Close()

	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/devices status = %d, want %d", listResp.StatusCode, http.StatusOK)
	}

	var devices []model.DeviceView
	if err := json.NewDecoder(listResp.Body).Decode(&devices); err != nil {
		t.Fatalf("decode device list error = %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("device list len = %d, want 1", len(devices))
	}

	getResp, err := http.Get(httpServer.URL + "/api/v1/devices/" + device.ID)
	if err != nil {
		t.Fatalf("GET /api/v1/devices/{id} error = %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/devices/{id} status = %d, want %d", getResp.StatusCode, http.StatusOK)
	}

	var view model.DeviceView
	if err := json.NewDecoder(getResp.Body).Decode(&view); err != nil {
		t.Fatalf("decode device view error = %v", err)
	}
	if view.Device.ID != device.ID {
		t.Fatalf("device view id = %q, want %q", view.Device.ID, device.ID)
	}

	commandBody := []byte(`{"name":"reboot","params":{"delay":1}}`)
	commandResp, err := http.Post(httpServer.URL+"/api/v1/devices/"+device.ID+"/commands", "application/json", bytes.NewReader(commandBody))
	if err != nil {
		t.Fatalf("POST /commands error = %v", err)
	}
	defer commandResp.Body.Close()

	if commandResp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(commandResp.Body)
		t.Fatalf("POST /commands status = %d, want %d, body=%s", commandResp.StatusCode, http.StatusConflict, string(body))
	}

	shadowResp, err := http.Get(httpServer.URL + "/api/v1/devices/" + device.ID + "/shadow")
	if err != nil {
		t.Fatalf("GET /shadow error = %v", err)
	}
	defer shadowResp.Body.Close()

	if shadowResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /shadow status = %d, want %d", shadowResp.StatusCode, http.StatusOK)
	}
}

func TestProductAndShadowAPIFlow(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	productBody := []byte(`{
		"name":"thermostat-product",
		"description":"demo product",
		"thing_model":{
			"properties":[{"identifier":"temperature","name":"Temperature","data_type":"float","access_mode":"rw"}],
			"services":[{"identifier":"reboot","name":"Reboot"}]
		}
	}`)

	productResp, err := http.Post(httpServer.URL+"/api/v1/products", "application/json", bytes.NewReader(productBody))
	if err != nil {
		t.Fatalf("POST /api/v1/products error = %v", err)
	}
	defer productResp.Body.Close()

	if productResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(productResp.Body)
		t.Fatalf("POST /api/v1/products status = %d, want %d, body=%s", productResp.StatusCode, http.StatusCreated, string(body))
	}

	var product model.Product
	if err := json.NewDecoder(productResp.Body).Decode(&product); err != nil {
		t.Fatalf("decode product error = %v", err)
	}

	deviceBody := []byte(`{"name":"thermostat-01","product_id":"` + product.ID + `"}`)
	deviceResp, err := http.Post(httpServer.URL+"/api/v1/devices", "application/json", bytes.NewReader(deviceBody))
	if err != nil {
		t.Fatalf("POST /api/v1/devices error = %v", err)
	}
	defer deviceResp.Body.Close()

	if deviceResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(deviceResp.Body)
		t.Fatalf("POST /api/v1/devices status = %d, want %d, body=%s", deviceResp.StatusCode, http.StatusCreated, string(body))
	}

	var device model.Device
	if err := json.NewDecoder(deviceResp.Body).Decode(&device); err != nil {
		t.Fatalf("decode device error = %v", err)
	}
	if device.ProductID != product.ID {
		t.Fatalf("device.ProductID = %q, want %q", device.ProductID, product.ID)
	}

	updateShadowReq, err := http.NewRequest(http.MethodPut, httpServer.URL+"/api/v1/devices/"+device.ID+"/shadow", bytes.NewReader([]byte(`{"desired":{"temperature":26.3}}`)))
	if err != nil {
		t.Fatalf("new PUT /shadow request error = %v", err)
	}
	updateShadowReq.Header.Set("Content-Type", "application/json")

	updateShadowResp, err := http.DefaultClient.Do(updateShadowReq)
	if err != nil {
		t.Fatalf("PUT /shadow error = %v", err)
	}
	defer updateShadowResp.Body.Close()

	if updateShadowResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updateShadowResp.Body)
		t.Fatalf("PUT /shadow status = %d, want %d, body=%s", updateShadowResp.StatusCode, http.StatusOK, string(body))
	}

	productsResp, err := http.Get(httpServer.URL + "/api/v1/products")
	if err != nil {
		t.Fatalf("GET /api/v1/products error = %v", err)
	}
	defer productsResp.Body.Close()

	if productsResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/products status = %d, want %d", productsResp.StatusCode, http.StatusOK)
	}

	var products []model.ProductView
	if err := json.NewDecoder(productsResp.Body).Decode(&products); err != nil {
		t.Fatalf("decode product list error = %v", err)
	}
	if len(products) != 1 || products[0].DeviceCount != 1 {
		t.Fatalf("unexpected product views: %#v", products)
	}
}

func TestProtocolCatalogAndHTTPIngestFlow(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	catalogResp, err := http.Get(httpServer.URL + "/api/v1/protocol-catalog")
	if err != nil {
		t.Fatalf("GET /api/v1/protocol-catalog error = %v", err)
	}
	defer catalogResp.Body.Close()

	if catalogResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/protocol-catalog status = %d, want %d", catalogResp.StatusCode, http.StatusOK)
	}

	var catalog []model.ProtocolCatalogEntry
	if err := json.NewDecoder(catalogResp.Body).Decode(&catalog); err != nil {
		t.Fatalf("decode protocol catalog error = %v", err)
	}
	if len(catalog) == 0 {
		t.Fatal("protocol catalog should not be empty")
	}
	seenCatalog := make(map[string]bool, len(catalog))
	for _, item := range catalog {
		seenCatalog[item.Protocol] = true
	}
	for _, protocol := range []string{"modbus_tcp", "bacnet_ip"} {
		if !seenCatalog[protocol] {
			t.Fatalf("protocol catalog missing %s template", protocol)
		}
	}

	productBody := []byte(`{
		"name":"rs485-env",
		"description":"modbus mapped sensor",
		"access_profile":{
			"transport":"rs485",
			"protocol":"modbus_rtu",
			"ingest_mode":"http_push",
			"payload_format":"register_map",
			"point_mappings":[
				{"source":"register:40001","property":"temperature","scale":0.1},
				{"source":"register:40002","property":"humidity","scale":0.1}
			]
		},
		"thing_model":{
			"properties":[
				{"identifier":"temperature","name":"Temperature","data_type":"float"},
				{"identifier":"humidity","name":"Humidity","data_type":"float"}
			]
		}
	}`)

	productResp, err := http.Post(httpServer.URL+"/api/v1/products", "application/json", bytes.NewReader(productBody))
	if err != nil {
		t.Fatalf("POST /api/v1/products error = %v", err)
	}
	defer productResp.Body.Close()

	if productResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(productResp.Body)
		t.Fatalf("POST /api/v1/products status = %d, want %d, body=%s", productResp.StatusCode, http.StatusCreated, string(body))
	}

	var product model.Product
	if err := json.NewDecoder(productResp.Body).Decode(&product); err != nil {
		t.Fatalf("decode product error = %v", err)
	}
	if product.AccessProfile.Protocol != "modbus_rtu" {
		t.Fatalf("product access protocol = %q, want modbus_rtu", product.AccessProfile.Protocol)
	}

	deviceResp, err := http.Post(httpServer.URL+"/api/v1/devices", "application/json", bytes.NewReader([]byte(`{"name":"modbus-device","product_id":"`+product.ID+`"}`)))
	if err != nil {
		t.Fatalf("POST /api/v1/devices error = %v", err)
	}
	defer deviceResp.Body.Close()

	var device model.Device
	if err := json.NewDecoder(deviceResp.Body).Decode(&device); err != nil {
		t.Fatalf("decode device error = %v", err)
	}

	ingestResp, err := http.Post(httpServer.URL+"/api/v1/ingest/http/"+device.ID, "application/json", bytes.NewReader([]byte(`{
		"token":"`+device.Token+`",
		"registers":{"40001":231,"40002":556}
	}`)))
	if err != nil {
		t.Fatalf("POST /api/v1/ingest/http/{id} error = %v", err)
	}
	defer ingestResp.Body.Close()

	if ingestResp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(ingestResp.Body)
		t.Fatalf("POST /api/v1/ingest/http/{id} status = %d, want %d, body=%s", ingestResp.StatusCode, http.StatusAccepted, string(body))
	}

	shadowResp, err := http.Get(httpServer.URL + "/api/v1/devices/" + device.ID + "/shadow")
	if err != nil {
		t.Fatalf("GET /shadow error = %v", err)
	}
	defer shadowResp.Body.Close()

	var shadow model.DeviceShadow
	if err := json.NewDecoder(shadowResp.Body).Decode(&shadow); err != nil {
		t.Fatalf("decode shadow error = %v", err)
	}
	if got := shadow.Reported["temperature"]; got != 23.1 {
		t.Fatalf("reported temperature = %#v, want 23.1", got)
	}
	if got := shadow.Reported["humidity"]; got != 55.6 {
		t.Fatalf("reported humidity = %#v, want 55.6", got)
	}
}

func TestGroupAndRuleAPIFlow(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	productResp, err := http.Post(httpServer.URL+"/api/v1/products", "application/json", bytes.NewReader([]byte(`{
		"name":"factory-product",
		"thing_model":{"properties":[{"identifier":"temperature","name":"Temperature","data_type":"float"}]}
	}`)))
	if err != nil {
		t.Fatalf("POST /api/v1/products error = %v", err)
	}
	defer productResp.Body.Close()

	var product model.Product
	if err := json.NewDecoder(productResp.Body).Decode(&product); err != nil {
		t.Fatalf("decode product error = %v", err)
	}

	deviceResp, err := http.Post(httpServer.URL+"/api/v1/devices", "application/json", bytes.NewReader([]byte(`{"name":"factory-01","product_id":"`+product.ID+`"}`)))
	if err != nil {
		t.Fatalf("POST /api/v1/devices error = %v", err)
	}
	defer deviceResp.Body.Close()

	var device model.Device
	if err := json.NewDecoder(deviceResp.Body).Decode(&device); err != nil {
		t.Fatalf("decode device error = %v", err)
	}

	groupResp, err := http.Post(httpServer.URL+"/api/v1/groups", "application/json", bytes.NewReader([]byte(`{
		"name":"line-a",
		"product_id":"`+product.ID+`",
		"tags":{"site":"workshop"}
	}`)))
	if err != nil {
		t.Fatalf("POST /api/v1/groups error = %v", err)
	}
	defer groupResp.Body.Close()

	if groupResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(groupResp.Body)
		t.Fatalf("POST /api/v1/groups status = %d, want %d, body=%s", groupResp.StatusCode, http.StatusCreated, string(body))
	}

	var group model.DeviceGroup
	if err := json.NewDecoder(groupResp.Body).Decode(&group); err != nil {
		t.Fatalf("decode group error = %v", err)
	}

	assignResp, err := http.Post(httpServer.URL+"/api/v1/groups/"+group.ID+"/devices", "application/json", bytes.NewReader([]byte(`{"device_id":"`+device.ID+`"}`)))
	if err != nil {
		t.Fatalf("POST /api/v1/groups/{id}/devices error = %v", err)
	}
	defer assignResp.Body.Close()

	if assignResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(assignResp.Body)
		t.Fatalf("POST /api/v1/groups/{id}/devices status = %d, want %d, body=%s", assignResp.StatusCode, http.StatusOK, string(body))
	}

	ruleResp, err := http.Post(httpServer.URL+"/api/v1/rules", "application/json", bytes.NewReader([]byte(`{
		"name":"temp-high",
		"product_id":"`+product.ID+`",
		"group_id":"`+group.ID+`",
		"severity":"critical",
		"cooldown_seconds":30,
		"condition":{"property":"temperature","operator":"gt","value":30}
	}`)))
	if err != nil {
		t.Fatalf("POST /api/v1/rules error = %v", err)
	}
	defer ruleResp.Body.Close()

	if ruleResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(ruleResp.Body)
		t.Fatalf("POST /api/v1/rules status = %d, want %d, body=%s", ruleResp.StatusCode, http.StatusCreated, string(body))
	}

	groupsResp, err := http.Get(httpServer.URL + "/api/v1/groups")
	if err != nil {
		t.Fatalf("GET /api/v1/groups error = %v", err)
	}
	defer groupsResp.Body.Close()

	var groups []model.GroupView
	if err := json.NewDecoder(groupsResp.Body).Decode(&groups); err != nil {
		t.Fatalf("decode group list error = %v", err)
	}
	if len(groups) != 1 || groups[0].DeviceCount != 1 {
		t.Fatalf("unexpected groups: %#v", groups)
	}

	rulesResp, err := http.Get(httpServer.URL + "/api/v1/rules")
	if err != nil {
		t.Fatalf("GET /api/v1/rules error = %v", err)
	}
	defer rulesResp.Body.Close()

	var rules []model.RuleView
	if err := json.NewDecoder(rulesResp.Body).Decode(&rules); err != nil {
		t.Fatalf("decode rule list error = %v", err)
	}
	if len(rules) != 1 || rules[0].Rule.GroupID != group.ID {
		t.Fatalf("unexpected rules: %#v", rules)
	}
}

func TestConfigProfileAndAlertAPIFlow(t *testing.T) {
	t.Parallel()

	server, service := newTestServerWithService()
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	productResp, err := http.Post(httpServer.URL+"/api/v1/products", "application/json", bytes.NewReader([]byte(`{
		"name":"config-product",
		"thing_model":{"properties":[
			{"identifier":"temperature","name":"Temperature","data_type":"float"},
			{"identifier":"enabled","name":"Enabled","data_type":"bool"}
		]}
	}`)))
	if err != nil {
		t.Fatalf("POST /api/v1/products error = %v", err)
	}
	defer productResp.Body.Close()

	var product model.Product
	if err := json.NewDecoder(productResp.Body).Decode(&product); err != nil {
		t.Fatalf("decode product error = %v", err)
	}

	deviceResp, err := http.Post(httpServer.URL+"/api/v1/devices", "application/json", bytes.NewReader([]byte(`{"name":"cfg-01","product_id":"`+product.ID+`"}`)))
	if err != nil {
		t.Fatalf("POST /api/v1/devices error = %v", err)
	}
	defer deviceResp.Body.Close()

	var device model.Device
	if err := json.NewDecoder(deviceResp.Body).Decode(&device); err != nil {
		t.Fatalf("decode device error = %v", err)
	}

	tagReq, err := http.NewRequest(http.MethodPut, httpServer.URL+"/api/v1/devices/"+device.ID+"/tags", bytes.NewReader([]byte(`{"tags":{"site":"lab","zone":"A"}}`)))
	if err != nil {
		t.Fatalf("new PUT /tags request error = %v", err)
	}
	tagReq.Header.Set("Content-Type", "application/json")
	tagResp, err := http.DefaultClient.Do(tagReq)
	if err != nil {
		t.Fatalf("PUT /tags error = %v", err)
	}
	defer tagResp.Body.Close()
	if tagResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(tagResp.Body)
		t.Fatalf("PUT /tags status = %d, want %d, body=%s", tagResp.StatusCode, http.StatusOK, string(body))
	}

	profileResp, err := http.Post(httpServer.URL+"/api/v1/config-profiles", "application/json", bytes.NewReader([]byte(`{
		"name":"night-mode",
		"product_id":"`+product.ID+`",
		"values":{"temperature":21.5,"enabled":true}
	}`)))
	if err != nil {
		t.Fatalf("POST /api/v1/config-profiles error = %v", err)
	}
	defer profileResp.Body.Close()
	if profileResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(profileResp.Body)
		t.Fatalf("POST /api/v1/config-profiles status = %d, want %d, body=%s", profileResp.StatusCode, http.StatusCreated, string(body))
	}

	var profile model.ConfigProfile
	if err := json.NewDecoder(profileResp.Body).Decode(&profile); err != nil {
		t.Fatalf("decode profile error = %v", err)
	}

	applyResp, err := http.Post(httpServer.URL+"/api/v1/config-profiles/"+profile.ID+"/apply", "application/json", bytes.NewReader([]byte(`{"device_id":"`+device.ID+`"}`)))
	if err != nil {
		t.Fatalf("POST /apply config error = %v", err)
	}
	defer applyResp.Body.Close()
	if applyResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(applyResp.Body)
		t.Fatalf("POST /apply config status = %d, want %d, body=%s", applyResp.StatusCode, http.StatusOK, string(body))
	}

	if _, err := service.CreateRule(context.Background(), "temp-high", "demo", product.ID, "", device.ID, true, model.AlertSeverityCritical, 0, model.RuleCondition{
		Property: "temperature",
		Operator: "gt",
		Value:    30.0,
	}); err != nil {
		t.Fatalf("CreateRule() error = %v", err)
	}
	if err := service.HandleTelemetry(context.Background(), device.ID, time.Now().UTC(), map[string]any{"temperature": 32.2, "enabled": true}); err != nil {
		t.Fatalf("HandleTelemetry() error = %v", err)
	}

	alertsResp, err := http.Get(httpServer.URL + "/api/v1/alerts?device_id=" + device.ID)
	if err != nil {
		t.Fatalf("GET /api/v1/alerts error = %v", err)
	}
	defer alertsResp.Body.Close()

	var alerts []model.AlertEvent
	if err := json.NewDecoder(alertsResp.Body).Decode(&alerts); err != nil {
		t.Fatalf("decode alerts error = %v", err)
	}
	if len(alerts) == 0 {
		t.Fatal("expected at least one alert")
	}

	updateAlertReq, err := http.NewRequest(http.MethodPut, httpServer.URL+"/api/v1/alerts/"+alerts[0].ID, bytes.NewReader([]byte(`{"status":"acknowledged","note":"checked from api test"}`)))
	if err != nil {
		t.Fatalf("new PUT /alerts request error = %v", err)
	}
	updateAlertReq.Header.Set("Content-Type", "application/json")
	updateAlertResp, err := http.DefaultClient.Do(updateAlertReq)
	if err != nil {
		t.Fatalf("PUT /alerts/{id} error = %v", err)
	}
	defer updateAlertResp.Body.Close()
	if updateAlertResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updateAlertResp.Body)
		t.Fatalf("PUT /alerts/{id} status = %d, want %d, body=%s", updateAlertResp.StatusCode, http.StatusOK, string(body))
	}
}

func TestUIEndpoints(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	indexResp, err := http.Get(httpServer.URL + "/")
	if err != nil {
		t.Fatalf("GET / error = %v", err)
	}
	defer indexResp.Body.Close()

	if indexResp.StatusCode != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", indexResp.StatusCode, http.StatusOK)
	}

	assetResp, err := http.Get(httpServer.URL + "/assets/app.js")
	if err != nil {
		t.Fatalf("GET /assets/app.js error = %v", err)
	}
	defer assetResp.Body.Close()

	if assetResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /assets/app.js status = %d, want %d", assetResp.StatusCode, http.StatusOK)
	}
}

func TestSimulatorAPIFlow(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	createBody := []byte(`{
		"name":"ui-simulator",
		"auto_connect":false,
		"auto_ack":true,
		"auto_ping":false,
		"auto_telemetry":false,
		"telemetry_interval_ms":5000,
		"default_values":{"temperature":24.2}
	}`)

	createResp, err := http.Post(httpServer.URL+"/api/v1/simulators", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST /api/v1/simulators error = %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		t.Fatalf("POST /api/v1/simulators status = %d, want %d, body=%s", createResp.StatusCode, http.StatusCreated, string(body))
	}

	var sim model.SimulatorView
	if err := json.NewDecoder(createResp.Body).Decode(&sim); err != nil {
		t.Fatalf("decode simulator error = %v", err)
	}
	if sim.ID == "" || sim.Device.ID == "" {
		t.Fatalf("simulator id/device id should not be empty: %#v", sim)
	}

	listResp, err := http.Get(httpServer.URL + "/api/v1/simulators")
	if err != nil {
		t.Fatalf("GET /api/v1/simulators error = %v", err)
	}
	defer listResp.Body.Close()

	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/simulators status = %d, want %d", listResp.StatusCode, http.StatusOK)
	}

	var simulators []model.SimulatorView
	if err := json.NewDecoder(listResp.Body).Decode(&simulators); err != nil {
		t.Fatalf("decode simulator list error = %v", err)
	}
	if len(simulators) != 1 {
		t.Fatalf("simulator list len = %d, want 1", len(simulators))
	}

	telemetryResp, err := http.Post(httpServer.URL+"/api/v1/simulators/"+sim.ID+"/telemetry", "application/json", bytes.NewReader([]byte(`{"values":{"temperature":26.5}}`)))
	if err != nil {
		t.Fatalf("POST /api/v1/simulators/{id}/telemetry error = %v", err)
	}
	defer telemetryResp.Body.Close()

	if telemetryResp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(telemetryResp.Body)
		t.Fatalf("POST /api/v1/simulators/{id}/telemetry status = %d, want %d, body=%s", telemetryResp.StatusCode, http.StatusConflict, string(body))
	}
}

func newTestServer() *api.Server {
	server, _ := newTestServerWithService()
	return server
}

func newTestServerWithService() (*api.Server, *core.Service) {
	storage := memory.New(16)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := core.NewService(storage, storage, storage, storage, storage, storage, storage, storage, storage, logger)
	simulators := simulator.NewManager(config.Config{GatewayDialAddr: "127.0.0.1:18830"}, service, logger)
	return api.NewServer(config.Config{GatewayDialAddr: "127.0.0.1:18830"}, service, simulators, logger), service
}
