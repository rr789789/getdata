package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mvp-platform/internal/api"
	"mvp-platform/internal/config"
	"mvp-platform/internal/core"
	"mvp-platform/internal/model"
	"mvp-platform/internal/setup"
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
		if item.Protocol == "mqtt_json" && item.IngestMode != "broker_mqtt" {
			t.Fatalf("mqtt catalog ingest_mode = %q, want broker_mqtt", item.IngestMode)
		}
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

	advancedResp, err := http.Get(httpServer.URL + "/assets/advanced.js")
	if err != nil {
		t.Fatalf("GET /assets/advanced.js error = %v", err)
	}
	defer advancedResp.Body.Close()

	if advancedResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /assets/advanced.js status = %d, want %d", advancedResp.StatusCode, http.StatusOK)
	}

	configResp, err := http.Get(httpServer.URL + "/runtime-config.js")
	if err != nil {
		t.Fatalf("GET /runtime-config.js error = %v", err)
	}
	defer configResp.Body.Close()

	if configResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /runtime-config.js status = %d, want %d", configResp.StatusCode, http.StatusOK)
	}

	configBody, _ := io.ReadAll(configResp.Body)
	if !strings.Contains(string(configBody), "window.__MVP_RUNTIME_CONFIG__") {
		t.Fatalf("runtime config body missing config bootstrap: %s", string(configBody))
	}
}

func TestCORSAndStandbyGuard(t *testing.T) {
	t.Parallel()

	storage := memory.New(16)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := core.NewService(storage, storage, storage, storage, storage, storage, storage, storage, storage, storage, storage, storage, logger)
	simulators := simulator.NewManager(config.Config{GatewayDialAddr: "127.0.0.1:18830"}, service, logger)
	server := api.NewServer(config.Config{
		GatewayDialAddr:     "127.0.0.1:18830",
		NodeID:              "standby-a",
		NodeRole:            "standby",
		CORSAllowedOrigins:  []string{"http://console.local"},
	}, service, simulators, logger)

	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	req, err := http.NewRequest(http.MethodOptions, httpServer.URL+"/api/v1/devices", nil)
	if err != nil {
		t.Fatalf("NewRequest(OPTIONS) error = %v", err)
	}
	req.Header.Set("Origin", "http://console.local")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /api/v1/devices error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("OPTIONS /api/v1/devices status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://console.local" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "http://console.local")
	}

	readyResp, err := http.Get(httpServer.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz error = %v", err)
	}
	defer readyResp.Body.Close()
	if readyResp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("GET /readyz status = %d, want %d", readyResp.StatusCode, http.StatusServiceUnavailable)
	}

	createResp, err := http.Post(httpServer.URL+"/api/v1/devices", "application/json", bytes.NewReader([]byte(`{"name":"blocked"}`)))
	if err != nil {
		t.Fatalf("POST /api/v1/devices error = %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("POST /api/v1/devices status = %d, want %d", createResp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestReplicaSnapshotEndpoint(t *testing.T) {
	t.Parallel()

	storage := memory.New(16)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := core.NewService(storage, storage, storage, storage, storage, storage, storage, storage, storage, storage, storage, storage, logger)
	recorder := &replicaRecorder{}
	server := api.NewServer(config.Config{
		NodeID:             "standby-b",
		NodeRole:           "standby",
		ReplicaToken:       "secret-token",
		CORSAllowedOrigins: []string{"*"},
	}, service, nil, logger, api.WithReplicaApplier(recorder))

	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	req, err := http.NewRequest(http.MethodPut, httpServer.URL+"/_ha/snapshot", bytes.NewReader([]byte(`{"version":1}`)))
	if err != nil {
		t.Fatalf("NewRequest(replica) error = %v", err)
	}
	req.Header.Set("X-Replica-Token", "secret-token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT /_ha/snapshot error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("PUT /_ha/snapshot status = %d, want %d, body=%s", resp.StatusCode, http.StatusAccepted, string(body))
	}
	if string(recorder.payload) != `{"version":1}` {
		t.Fatalf("replica payload = %q, want %q", string(recorder.payload), `{"version":1}`)
	}
}

func TestReplicaSetupEndpoint(t *testing.T) {
	t.Parallel()

	storage := memory.New(16)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := core.NewService(storage, storage, storage, storage, storage, storage, storage, storage, storage, storage, storage, storage, logger)
	installer, err := setup.NewManager(t.TempDir() + "/instance.json")
	if err != nil {
		t.Fatalf("setup.NewManager() error = %v", err)
	}
	server := api.NewServer(config.Config{
		NodeID:       "standby-setup",
		NodeRole:     "standby",
		ReplicaToken: "secret-token",
	}, service, nil, logger, api.WithInstaller(installer))

	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	req, err := http.NewRequest(http.MethodPut, httpServer.URL+"/_ha/setup", bytes.NewReader([]byte(`{
		"install_lock": true,
		"app_name": "Factory IoT",
		"site_url": "http://primary.local:8080",
		"admin_username": "admin",
		"admin_email": "admin@example.com",
		"default_tenant_name": "Factory East",
		"default_tenant_slug": "factory-east",
		"installed_at": "2026-03-17T00:00:00Z",
		"updated_at": "2026-03-17T00:00:00Z"
	}`)))
	if err != nil {
		t.Fatalf("NewRequest(replica setup) error = %v", err)
	}
	req.Header.Set("X-Replica-Token", "secret-token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT /_ha/setup error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("PUT /_ha/setup status = %d, want %d, body=%s", resp.StatusCode, http.StatusAccepted, string(body))
	}

	statusResp, err := http.Get(httpServer.URL + "/api/v1/install/status")
	if err != nil {
		t.Fatalf("GET /api/v1/install/status error = %v", err)
	}
	defer statusResp.Body.Close()

	var statusPayload map[string]any
	if err := json.NewDecoder(statusResp.Body).Decode(&statusPayload); err != nil {
		t.Fatalf("decode install status error = %v", err)
	}
	if installed, _ := statusPayload["installed"].(bool); !installed {
		t.Fatalf("install status should be true after setup replication: %#v", statusPayload)
	}
	if appName, _ := statusPayload["app_name"].(string); appName != "Factory IoT" {
		t.Fatalf("replicated app_name = %q, want %q", appName, "Factory IoT")
	}
}

func TestInstallBootstrapFlow(t *testing.T) {
	t.Parallel()

	storage := memory.New(16)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := core.NewService(storage, storage, storage, storage, storage, storage, storage, storage, storage, storage, storage, storage, logger)
	installer, err := setup.NewManager(t.TempDir() + "/instance.json")
	if err != nil {
		t.Fatalf("setup.NewManager() error = %v", err)
	}
	server := api.NewServer(config.Config{GatewayDialAddr: "127.0.0.1:18830"}, service, nil, logger, api.WithInstaller(installer))

	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	statusResp, err := http.Get(httpServer.URL + "/api/v1/install/status")
	if err != nil {
		t.Fatalf("GET /api/v1/install/status error = %v", err)
	}
	defer statusResp.Body.Close()

	var statusPayload map[string]any
	if err := json.NewDecoder(statusResp.Body).Decode(&statusPayload); err != nil {
		t.Fatalf("decode install status error = %v", err)
	}
	if installed, _ := statusPayload["installed"].(bool); installed {
		t.Fatal("install status should be false before bootstrap")
	}

	productsResp, err := http.Get(httpServer.URL + "/api/v1/products")
	if err != nil {
		t.Fatalf("GET /api/v1/products error = %v", err)
	}
	defer productsResp.Body.Close()
	if productsResp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("GET /api/v1/products status = %d, want %d", productsResp.StatusCode, http.StatusServiceUnavailable)
	}

	indexResp, err := http.Get(httpServer.URL + "/")
	if err != nil {
		t.Fatalf("GET / error = %v", err)
	}
	defer indexResp.Body.Close()
	indexBody, _ := io.ReadAll(indexResp.Body)
	if !strings.Contains(string(indexBody), "install-form") {
		t.Fatalf("GET / should render install page, body=%s", string(indexBody))
	}

	bootstrapBody := []byte(`{
		"app_name":"Factory IoT",
		"admin_username":"admin",
		"admin_email":"admin@example.com",
		"default_tenant_name":"Factory East",
		"default_tenant_slug":"factory-east"
	}`)
	bootstrapResp, err := http.Post(httpServer.URL+"/api/v1/install/bootstrap", "application/json", bytes.NewReader(bootstrapBody))
	if err != nil {
		t.Fatalf("POST /api/v1/install/bootstrap error = %v", err)
	}
	defer bootstrapResp.Body.Close()
	if bootstrapResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(bootstrapResp.Body)
		t.Fatalf("POST /api/v1/install/bootstrap status = %d, want %d, body=%s", bootstrapResp.StatusCode, http.StatusCreated, string(body))
	}

	tenantsResp, err := http.Get(httpServer.URL + "/api/v1/tenants")
	if err != nil {
		t.Fatalf("GET /api/v1/tenants error = %v", err)
	}
	defer tenantsResp.Body.Close()

	var tenants []model.TenantView
	if err := json.NewDecoder(tenantsResp.Body).Decode(&tenants); err != nil {
		t.Fatalf("decode tenants error = %v", err)
	}
	if len(tenants) != 1 || tenants[0].Tenant.Name != "Factory East" {
		t.Fatalf("unexpected tenants after install bootstrap: %#v", tenants)
	}
}

func TestMetricsEndpoints(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	jsonResp, err := http.Get(httpServer.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics error = %v", err)
	}
	defer jsonResp.Body.Close()

	var stats model.Stats
	if err := json.NewDecoder(jsonResp.Body).Decode(&stats); err != nil {
		t.Fatalf("decode /metrics json error = %v", err)
	}
	if stats.Storage.Backend == "" {
		t.Fatal("storage backend should not be empty")
	}
	if stats.Runtime.Goroutines <= 0 {
		t.Fatalf("runtime goroutines = %d, want > 0", stats.Runtime.Goroutines)
	}

	promReq, err := http.NewRequest(http.MethodGet, httpServer.URL+"/metrics?format=prometheus", nil)
	if err != nil {
		t.Fatalf("NewRequest(prometheus) error = %v", err)
	}
	promResp, err := http.DefaultClient.Do(promReq)
	if err != nil {
		t.Fatalf("GET /metrics?format=prometheus error = %v", err)
	}
	defer promResp.Body.Close()

	body, _ := io.ReadAll(promResp.Body)
	if !strings.Contains(string(body), "mvp_registered_devices") {
		t.Fatalf("prometheus body missing metric, body=%s", string(body))
	}
	if !strings.Contains(string(body), "mvp_runtime_goroutines") {
		t.Fatalf("prometheus body missing runtime metric, body=%s", string(body))
	}
	if !strings.Contains(string(body), "mvp_storage_tenants") {
		t.Fatalf("prometheus body missing tenant metric, body=%s", string(body))
	}
	if !strings.Contains(string(body), "mvp_storage_ota_campaigns") {
		t.Fatalf("prometheus body missing ota metric, body=%s", string(body))
	}
}

func TestTenantFirmwareAndOTAAPIFlow(t *testing.T) {
	t.Parallel()

	server, service := newTestServerWithService()
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	tenantResp, err := http.Post(httpServer.URL+"/api/v1/tenants", "application/json", bytes.NewReader([]byte(`{"name":"Factory East","slug":"factory-east"}`)))
	if err != nil {
		t.Fatalf("POST /api/v1/tenants error = %v", err)
	}
	defer tenantResp.Body.Close()

	if tenantResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(tenantResp.Body)
		t.Fatalf("POST /api/v1/tenants status = %d, want %d, body=%s", tenantResp.StatusCode, http.StatusCreated, string(body))
	}

	var tenant model.Tenant
	if err := json.NewDecoder(tenantResp.Body).Decode(&tenant); err != nil {
		t.Fatalf("decode tenant error = %v", err)
	}

	productBody := []byte(`{
		"tenant_id":"` + tenant.ID + `",
		"name":"edge-sensor",
		"thing_model":{
			"properties":[{"identifier":"temperature","name":"Temperature","data_type":"float"}],
			"services":[
				{"identifier":"reboot","name":"Reboot"},
				{"identifier":"ota_upgrade","name":"OTA Upgrade"}
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

	deviceResp, err := http.Post(httpServer.URL+"/api/v1/devices", "application/json", bytes.NewReader([]byte(`{"tenant_id":"`+tenant.ID+`","name":"edge-01","product_id":"`+product.ID+`"}`)))
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

	service.RegisterSession(device.ID, &testSession{id: "api-tenant-session"})

	firmwareResp, err := http.Post(httpServer.URL+"/api/v1/firmware", "application/json", bytes.NewReader([]byte(`{
		"tenant_id":"`+tenant.ID+`",
		"product_id":"`+product.ID+`",
		"name":"esp8266",
		"version":"1.0.0",
		"file_name":"esp8266.bin",
		"url":"https://example.com/esp8266.bin"
	}`)))
	if err != nil {
		t.Fatalf("POST /api/v1/firmware error = %v", err)
	}
	defer firmwareResp.Body.Close()

	if firmwareResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(firmwareResp.Body)
		t.Fatalf("POST /api/v1/firmware status = %d, want %d, body=%s", firmwareResp.StatusCode, http.StatusCreated, string(body))
	}

	var artifact model.FirmwareArtifact
	if err := json.NewDecoder(firmwareResp.Body).Decode(&artifact); err != nil {
		t.Fatalf("decode firmware error = %v", err)
	}

	otaResp, err := http.Post(httpServer.URL+"/api/v1/ota-campaigns", "application/json", bytes.NewReader([]byte(`{
		"tenant_id":"`+tenant.ID+`",
		"name":"east-rollout",
		"firmware_id":"`+artifact.ID+`",
		"device_id":"`+device.ID+`"
	}`)))
	if err != nil {
		t.Fatalf("POST /api/v1/ota-campaigns error = %v", err)
	}
	defer otaResp.Body.Close()

	if otaResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(otaResp.Body)
		t.Fatalf("POST /api/v1/ota-campaigns status = %d, want %d, body=%s", otaResp.StatusCode, http.StatusCreated, string(body))
	}

	firmwareListResp, err := http.Get(httpServer.URL + "/api/v1/firmware?tenant_id=" + tenant.ID)
	if err != nil {
		t.Fatalf("GET /api/v1/firmware error = %v", err)
	}
	defer firmwareListResp.Body.Close()

	var firmwareViews []model.FirmwareArtifactView
	if err := json.NewDecoder(firmwareListResp.Body).Decode(&firmwareViews); err != nil {
		t.Fatalf("decode firmware list error = %v", err)
	}
	if len(firmwareViews) != 1 || firmwareViews[0].Tenant == nil || firmwareViews[0].Tenant.ID != tenant.ID {
		t.Fatalf("unexpected firmware views: %#v", firmwareViews)
	}

	otaListResp, err := http.Get(httpServer.URL + "/api/v1/ota-campaigns?tenant_id=" + tenant.ID)
	if err != nil {
		t.Fatalf("GET /api/v1/ota-campaigns error = %v", err)
	}
	defer otaListResp.Body.Close()

	var otaViews []model.OTACampaignView
	if err := json.NewDecoder(otaListResp.Body).Decode(&otaViews); err != nil {
		t.Fatalf("decode ota list error = %v", err)
	}
	if len(otaViews) != 1 || otaViews[0].Campaign.DispatchedCount != 1 {
		t.Fatalf("unexpected ota views: %#v", otaViews)
	}
}

type testSession struct {
	id string
}

type replicaRecorder struct {
	payload []byte
}

func (s *testSession) SessionID() string {
	return s.id
}

func (s *testSession) Send(model.ServerMessage) error {
	return nil
}

func (s *testSession) Close() error {
	return nil
}

func (r *replicaRecorder) ApplyReplicaSnapshot(payload []byte) error {
	r.payload = append([]byte(nil), payload...)
	return nil
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
	service := core.NewService(storage, storage, storage, storage, storage, storage, storage, storage, storage, storage, storage, storage, logger)
	simulators := simulator.NewManager(config.Config{GatewayDialAddr: "127.0.0.1:18830"}, service, logger)
	return api.NewServer(config.Config{GatewayDialAddr: "127.0.0.1:18830"}, service, simulators, logger), service
}
