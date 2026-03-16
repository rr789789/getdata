package file_test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	storefile "mvp-platform/internal/store/file"
	"mvp-platform/internal/model"
)

func TestFileStorePersistsAndReloads(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.json")

	first, err := storefile.New(path, 16)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	product := model.Product{
		ID:        "prd-1",
		Key:       "pk-1",
		Name:      "persist-product",
		ThingModel: model.ThingModel{Version: 1},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := first.CreateProduct(ctx, product); err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}

	device := model.Device{
		ID:        "dev-1",
		Name:      "persist-device",
		ProductID: product.ID,
		Token:     "token-1",
		CreatedAt: time.Now().UTC(),
	}
	if err := first.CreateDevice(ctx, device); err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	shadow := model.DeviceShadow{
		DeviceID:  device.ID,
		ProductID: product.ID,
		Reported:  map[string]any{"temperature": 24.5},
		Desired:   map[string]any{"temperature": 25.0},
		Version:   2,
		UpdatedAt: time.Now().UTC(),
	}
	if err := first.SaveShadow(ctx, shadow); err != nil {
		t.Fatalf("SaveShadow() error = %v", err)
	}

	telemetry := model.Telemetry{
		DeviceID:  device.ID,
		Timestamp: time.Now().UTC(),
		Values:    map[string]any{"temperature": 24.5},
	}
	if err := first.AppendTelemetry(ctx, telemetry); err != nil {
		t.Fatalf("AppendTelemetry() error = %v", err)
	}

	reloaded, err := storefile.New(path, 16)
	if err != nil {
		t.Fatalf("New(reload) error = %v", err)
	}

	gotDevice, err := reloaded.GetDevice(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDevice() error = %v", err)
	}
	if gotDevice.Token != device.Token {
		t.Fatalf("reloaded token = %q, want %q", gotDevice.Token, device.Token)
	}

	gotShadow, err := reloaded.GetShadow(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetShadow() error = %v", err)
	}
	if got := gotShadow.Reported["temperature"]; got != 24.5 {
		t.Fatalf("reloaded shadow temperature = %#v, want 24.5", got)
	}

	items, err := reloaded.ListTelemetryByDevice(ctx, device.ID, 10)
	if err != nil {
		t.Fatalf("ListTelemetryByDevice() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("telemetry len = %d, want 1", len(items))
	}

	stats, err := reloaded.StorageStats(ctx)
	if err != nil {
		t.Fatalf("StorageStats() error = %v", err)
	}
	if stats.Backend != "file" {
		t.Fatalf("backend = %q, want file", stats.Backend)
	}
	if stats.Devices != 1 || stats.TelemetrySamples != 1 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

func TestFileStoreConcurrentMutations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.json")

	store, err := storefile.New(path, 16)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	const deviceCount = 24
	var wg sync.WaitGroup
	for index := 0; index < deviceCount; index++ {
		index := index
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := store.CreateDevice(ctx, model.Device{
				ID:        fmt.Sprintf("dev-%02d", index),
				Name:      fmt.Sprintf("device-%02d", index),
				Token:     fmt.Sprintf("token-%02d", index),
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			})
			if err != nil {
				t.Errorf("CreateDevice(%d) error = %v", index, err)
			}
		}()
	}
	wg.Wait()

	devices, err := store.ListDevices(ctx)
	if err != nil {
		t.Fatalf("ListDevices() error = %v", err)
	}
	if len(devices) != deviceCount {
		t.Fatalf("device count = %d, want %d", len(devices), deviceCount)
	}

	reloaded, err := storefile.New(path, 16)
	if err != nil {
		t.Fatalf("New(reload) error = %v", err)
	}
	devices, err = reloaded.ListDevices(ctx)
	if err != nil {
		t.Fatalf("reloaded ListDevices() error = %v", err)
	}
	if len(devices) != deviceCount {
		t.Fatalf("reloaded device count = %d, want %d", len(devices), deviceCount)
	}
}
