package core_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"mvp-platform/internal/model"
)

func BenchmarkHandleTelemetry(b *testing.B) {
	service := newTestService()
	ctx := context.Background()

	product, err := service.CreateProduct(ctx, "bench-product", "benchmark", nil, model.ProductAccessProfile{}, model.ThingModel{
		Properties: []model.ThingModelProperty{
			{Identifier: "temperature", Name: "Temperature", DataType: "float"},
			{Identifier: "humidity", Name: "Humidity", DataType: "float"},
		},
	})
	if err != nil {
		b.Fatalf("CreateProduct() error = %v", err)
	}
	device, err := service.CreateDevice(ctx, "bench-device", product.ID, nil, nil)
	if err != nil {
		b.Fatalf("CreateDevice() error = %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		values := map[string]any{
			"temperature": 20.0 + float64(i%15),
			"humidity":    45.0 + float64(i%20),
		}
		if err := service.HandleTelemetry(ctx, device.ID, time.Now().UTC(), values); err != nil {
			b.Fatalf("HandleTelemetry() error = %v", err)
		}
	}
}

func BenchmarkHandleTelemetryParallel(b *testing.B) {
	service := newTestService()
	ctx := context.Background()

	product, err := service.CreateProduct(ctx, "parallel-product", "benchmark", nil, model.ProductAccessProfile{}, model.ThingModel{
		Properties: []model.ThingModelProperty{
			{Identifier: "temperature", Name: "Temperature", DataType: "float"},
		},
	})
	if err != nil {
		b.Fatalf("CreateProduct() error = %v", err)
	}

	deviceIDs := make([]string, 0, 64)
	for i := 0; i < 64; i++ {
		device, createErr := service.CreateDevice(ctx, fmt.Sprintf("device-%d", i), product.ID, nil, nil)
		if createErr != nil {
			b.Fatalf("CreateDevice(%d) error = %v", i, createErr)
		}
		deviceIDs = append(deviceIDs, device.ID)
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		index := 0
		for pb.Next() {
			deviceID := deviceIDs[index%len(deviceIDs)]
			index++
			if err := service.HandleTelemetry(ctx, deviceID, time.Now().UTC(), map[string]any{"temperature": 21.7}); err != nil {
				b.Fatalf("HandleTelemetry() error = %v", err)
			}
		}
	})
}

func BenchmarkSendCommand(b *testing.B) {
	service := newTestService()
	ctx := context.Background()

	product, err := service.CreateProduct(ctx, "command-product", "benchmark", nil, model.ProductAccessProfile{}, model.ThingModel{
		Services: []model.ThingModelService{
			{Identifier: "reboot", Name: "Reboot"},
		},
	})
	if err != nil {
		b.Fatalf("CreateProduct() error = %v", err)
	}
	device, err := service.CreateDevice(ctx, "command-device", product.ID, nil, nil)
	if err != nil {
		b.Fatalf("CreateDevice() error = %v", err)
	}
	service.RegisterSession(device.ID, &mockSession{id: "bench-session"})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := service.SendCommand(ctx, device.ID, "reboot", map[string]any{"delay": i % 3}); err != nil {
			b.Fatalf("SendCommand() error = %v", err)
		}
	}
}
