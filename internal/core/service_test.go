package core_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"mvp-platform/internal/core"
	"mvp-platform/internal/model"
	"mvp-platform/internal/store/memory"
)

type mockSession struct {
	id      string
	sendErr error

	mu       sync.Mutex
	messages []model.ServerMessage
	closed   bool
}

func (s *mockSession) SessionID() string {
	return s.id
}

func (s *mockSession) Send(message model.ServerMessage) error {
	if s.sendErr != nil {
		return s.sendErr
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, message)
	return nil
}

func (s *mockSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func TestServiceLifecycle(t *testing.T) {
	t.Parallel()

	service := newTestService()
	ctx := context.Background()

	device, err := service.CreateDevice(ctx, "sensor-a", map[string]string{"region": "cn"})
	if err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	authDevice, err := service.AuthenticateDevice(ctx, device.ID, device.Token)
	if err != nil {
		t.Fatalf("AuthenticateDevice() error = %v", err)
	}
	if authDevice.ID != device.ID {
		t.Fatalf("AuthenticateDevice() got device %q, want %q", authDevice.ID, device.ID)
	}

	session := &mockSession{id: "session-1"}
	service.RegisterSession(device.ID, session)

	command, err := service.SendCommand(ctx, device.ID, "reboot", map[string]any{"delay": 3})
	if err != nil {
		t.Fatalf("SendCommand() error = %v", err)
	}
	if command.Status != model.CommandStatusSent {
		t.Fatalf("SendCommand() status = %q, want %q", command.Status, model.CommandStatusSent)
	}

	session.mu.Lock()
	if len(session.messages) != 1 {
		t.Fatalf("session messages = %d, want 1", len(session.messages))
	}
	if session.messages[0].Type != "command" {
		t.Fatalf("first message type = %q, want command", session.messages[0].Type)
	}
	session.mu.Unlock()

	telemetryTime := time.Now().UTC()
	if err := service.HandleTelemetry(ctx, device.ID, telemetryTime, map[string]any{"temp": 25.5}); err != nil {
		t.Fatalf("HandleTelemetry() error = %v", err)
	}

	if err := service.HandleCommandAck(ctx, device.ID, command.ID, "ok", "accepted"); err != nil {
		t.Fatalf("HandleCommandAck() error = %v", err)
	}

	commands, err := service.ListCommands(ctx, device.ID, 10)
	if err != nil {
		t.Fatalf("ListCommands() error = %v", err)
	}
	if len(commands) != 1 {
		t.Fatalf("commands len = %d, want 1", len(commands))
	}
	if commands[0].Status != model.CommandStatusAcked {
		t.Fatalf("command status = %q, want %q", commands[0].Status, model.CommandStatusAcked)
	}

	telemetry, err := service.ListTelemetry(ctx, device.ID, 10)
	if err != nil {
		t.Fatalf("ListTelemetry() error = %v", err)
	}
	if len(telemetry) != 1 {
		t.Fatalf("telemetry len = %d, want 1", len(telemetry))
	}
	if got := telemetry[0].Values["temp"]; got != 25.5 {
		t.Fatalf("telemetry temp = %#v, want 25.5", got)
	}

	view, err := service.GetDevice(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDevice() error = %v", err)
	}
	if !view.Online {
		t.Fatal("GetDevice() online = false, want true")
	}

	service.UnregisterSession(device.ID, session.SessionID())

	stats := service.Stats()
	if stats.RegisteredDevices != 1 {
		t.Fatalf("RegisteredDevices = %d, want 1", stats.RegisteredDevices)
	}
	if stats.TelemetryReceived != 1 {
		t.Fatalf("TelemetryReceived = %d, want 1", stats.TelemetryReceived)
	}
	if stats.CommandsSent != 1 {
		t.Fatalf("CommandsSent = %d, want 1", stats.CommandsSent)
	}
	if stats.CommandAcks != 1 {
		t.Fatalf("CommandAcks = %d, want 1", stats.CommandAcks)
	}
}

func TestSendCommandOfflineMarksFailed(t *testing.T) {
	t.Parallel()

	service := newTestService()
	ctx := context.Background()

	device, err := service.CreateDevice(ctx, "sensor-b", nil)
	if err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	command, err := service.SendCommand(ctx, device.ID, "sync", nil)
	if !errors.Is(err, core.ErrDeviceOffline) {
		t.Fatalf("SendCommand() error = %v, want %v", err, core.ErrDeviceOffline)
	}
	if command.Status != model.CommandStatusFailed {
		t.Fatalf("command status = %q, want %q", command.Status, model.CommandStatusFailed)
	}
}

func newTestService() *core.Service {
	storage := memory.New(16)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return core.NewService(storage, storage, storage, logger)
}
