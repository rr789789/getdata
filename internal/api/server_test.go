package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

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
	storage := memory.New(16)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := core.NewService(storage, storage, storage, logger)
	simulators := simulator.NewManager(config.Config{GatewayDialAddr: "127.0.0.1:18830"}, service, logger)
	return api.NewServer(config.Config{GatewayDialAddr: "127.0.0.1:18830"}, service, simulators, logger)
}
