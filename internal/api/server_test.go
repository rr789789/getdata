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

func newTestServer() *api.Server {
	storage := memory.New(16)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := core.NewService(storage, storage, storage, logger)
	return api.NewServer(config.Config{}, service, logger)
}
