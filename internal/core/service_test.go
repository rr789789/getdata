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

	product, err := service.CreateProduct(ctx, "meter-product", "demo", nil, model.ThingModel{
		Properties: []model.ThingModelProperty{
			{Identifier: "temp", Name: "Temperature", DataType: "float"},
		},
		Services: []model.ThingModelService{
			{Identifier: "reboot", Name: "Reboot"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}

	device, err := service.CreateDevice(ctx, "sensor-a", product.ID, map[string]string{"role": "meter"}, map[string]string{"region": "cn"})
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
	if view.Product == nil || view.Product.ID != product.ID {
		t.Fatalf("device product = %#v, want product id %q", view.Product, product.ID)
	}
	if got := view.Device.Tags["role"]; got != "meter" {
		t.Fatalf("device tag role = %q, want meter", got)
	}

	shadow, err := service.GetShadow(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetShadow() error = %v", err)
	}
	if got := shadow.Reported["temp"]; got != 25.5 {
		t.Fatalf("shadow reported temp = %#v, want 25.5", got)
	}

	updatedShadow, err := service.UpdateDesiredShadow(ctx, device.ID, map[string]any{"temp": 26.0})
	if err != nil {
		t.Fatalf("UpdateDesiredShadow() error = %v", err)
	}
	if got := updatedShadow.Desired["temp"]; got != 26.0 {
		t.Fatalf("shadow desired temp = %#v, want 26.0", got)
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

	device, err := service.CreateDevice(ctx, "sensor-b", "", nil, nil)
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

func TestGroupRuleAlertFlow(t *testing.T) {
	t.Parallel()

	service := newTestService()
	ctx := context.Background()

	product, err := service.CreateProduct(ctx, "boiler-product", "demo", nil, model.ThingModel{
		Properties: []model.ThingModelProperty{
			{Identifier: "temp", Name: "Temperature", DataType: "float"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}

	device, err := service.CreateDevice(ctx, "boiler-01", product.ID, nil, nil)
	if err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	group, err := service.CreateGroup(ctx, "north-room", "north room devices", product.ID, map[string]string{"floor": "2"})
	if err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}

	groupView, err := service.AssignDeviceToGroup(ctx, group.ID, device.ID)
	if err != nil {
		t.Fatalf("AssignDeviceToGroup() error = %v", err)
	}
	if groupView.DeviceCount != 1 {
		t.Fatalf("group device count = %d, want 1", groupView.DeviceCount)
	}

	rule, err := service.CreateRule(ctx, "high-temp", "temperature threshold", product.ID, group.ID, "", true, model.AlertSeverityCritical, 60, model.RuleCondition{
		Property: "temp",
		Operator: "gt",
		Value:    30.0,
	})
	if err != nil {
		t.Fatalf("CreateRule() error = %v", err)
	}

	baseTime := time.Now().UTC()
	if err := service.HandleTelemetry(ctx, device.ID, baseTime, map[string]any{"temp": 31.5}); err != nil {
		t.Fatalf("HandleTelemetry() first error = %v", err)
	}
	if err := service.HandleTelemetry(ctx, device.ID, baseTime.Add(30*time.Second), map[string]any{"temp": 32.1}); err != nil {
		t.Fatalf("HandleTelemetry() cooldown error = %v", err)
	}
	if err := service.HandleTelemetry(ctx, device.ID, baseTime.Add(2*time.Minute), map[string]any{"temp": 33.7}); err != nil {
		t.Fatalf("HandleTelemetry() second trigger error = %v", err)
	}

	alerts, err := service.ListAlerts(ctx, 10, "", "", device.ID, "")
	if err != nil {
		t.Fatalf("ListAlerts() error = %v", err)
	}
	if len(alerts) != 2 {
		t.Fatalf("alerts len = %d, want 2", len(alerts))
	}
	if alerts[0].RuleID != rule.ID {
		t.Fatalf("latest alert rule id = %q, want %q", alerts[0].RuleID, rule.ID)
	}

	view, err := service.GetDevice(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDevice() error = %v", err)
	}
	if len(view.Groups) != 1 || view.Groups[0].ID != group.ID {
		t.Fatalf("device groups = %#v, want group %q", view.Groups, group.ID)
	}

	rules, err := service.ListRules(ctx)
	if err != nil {
		t.Fatalf("ListRules() error = %v", err)
	}
	if len(rules) != 1 || rules[0].TriggeredCount != 2 {
		t.Fatalf("unexpected rule views: %#v", rules)
	}

	updatedAlert, err := service.UpdateAlert(ctx, alerts[0].ID, model.AlertStatusAcknowledged, "operator checked")
	if err != nil {
		t.Fatalf("UpdateAlert() error = %v", err)
	}
	if updatedAlert.Status != model.AlertStatusAcknowledged || updatedAlert.Note != "operator checked" {
		t.Fatalf("unexpected updated alert: %#v", updatedAlert)
	}
}

func TestConfigProfileFlow(t *testing.T) {
	t.Parallel()

	service := newTestService()
	ctx := context.Background()

	product, err := service.CreateProduct(ctx, "config-product", "demo", nil, model.ThingModel{
		Properties: []model.ThingModelProperty{
			{Identifier: "temperature", Name: "Temperature", DataType: "float"},
			{Identifier: "enabled", Name: "Enabled", DataType: "bool"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}

	device, err := service.CreateDevice(ctx, "cfg-device", product.ID, nil, nil)
	if err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	profile, err := service.CreateConfigProfile(ctx, "night-mode", "config template", product.ID, map[string]any{
		"temperature": 22.5,
		"enabled":     true,
	})
	if err != nil {
		t.Fatalf("CreateConfigProfile() error = %v", err)
	}

	shadow, err := service.ApplyConfigProfile(ctx, profile.ID, device.ID)
	if err != nil {
		t.Fatalf("ApplyConfigProfile() error = %v", err)
	}
	if got := shadow.Desired["temperature"]; got != 22.5 {
		t.Fatalf("desired temperature = %#v, want 22.5", got)
	}

	profiles, err := service.ListConfigProfiles(ctx)
	if err != nil {
		t.Fatalf("ListConfigProfiles() error = %v", err)
	}
	if len(profiles) != 1 || profiles[0].Profile.AppliedCount != 1 {
		t.Fatalf("unexpected config profiles: %#v", profiles)
	}
}

func newTestService() *core.Service {
	storage := memory.New(16)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return core.NewService(storage, storage, storage, storage, storage, storage, storage, storage, storage, logger)
}
