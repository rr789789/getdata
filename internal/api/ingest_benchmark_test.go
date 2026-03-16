package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkHTTPIngest(b *testing.B) {
	server := newTestServer()
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	createResp, err := http.Post(httpServer.URL+"/api/v1/devices", "application/json", bytes.NewReader([]byte(`{"name":"bench-http-device"}`)))
	if err != nil {
		b.Fatalf("POST /devices error = %v", err)
	}
	defer createResp.Body.Close()

	var device struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&device); err != nil {
		b.Fatalf("decode device error = %v", err)
	}

	client := &http.Client{}
	bodyTemplate := `{"token":"%s","values":{"temperature":23.4,"humidity":58,"pressure":1006.2}}`

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest(http.MethodPost, httpServer.URL+"/api/v1/ingest/http/"+device.ID, bytes.NewReader([]byte(
			fmt.Sprintf(bodyTemplate, device.Token),
		)))
		if err != nil {
			b.Fatalf("NewRequest() error = %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			b.Fatalf("POST /ingest/http error = %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusAccepted {
			b.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusAccepted)
		}
	}
}
