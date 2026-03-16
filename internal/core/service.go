package core

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mvp-platform/internal/model"
	"mvp-platform/internal/store"
	"mvp-platform/internal/util"
)

var ErrDeviceOffline = errors.New("device offline")

type Session interface {
	SessionID() string
	Send(message model.ServerMessage) error
	Close() error
}

type Service struct {
	devices   store.DeviceStore
	telemetry store.TelemetryStore
	commands  store.CommandStore
	logger    *slog.Logger

	mu     sync.RWMutex
	states map[string]deviceState

	registeredDevices   atomic.Int64
	onlineDevices       atomic.Int64
	totalConnections    atomic.Int64
	rejectedConnections atomic.Int64
	telemetryReceived   atomic.Int64
	commandsSent        atomic.Int64
	commandAcks         atomic.Int64
}

type deviceState struct {
	session     Session
	sessionID   string
	connectedAt time.Time
	lastSeen    time.Time
}

func NewService(devices store.DeviceStore, telemetry store.TelemetryStore, commands store.CommandStore, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		devices:   devices,
		telemetry: telemetry,
		commands:  commands,
		logger:    logger,
		states:    make(map[string]deviceState),
	}
}

func (s *Service) CreateDevice(ctx context.Context, name string, metadata map[string]string) (model.Device, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "device"
	}

	device := model.Device{
		ID:        util.NewID("dev"),
		Name:      name,
		Token:     util.NewToken(),
		Metadata:  cloneStringMap(metadata),
		CreatedAt: time.Now().UTC(),
	}

	if err := s.devices.CreateDevice(ctx, device); err != nil {
		return model.Device{}, err
	}

	s.registeredDevices.Add(1)
	return device, nil
}

func (s *Service) GetDevice(ctx context.Context, deviceID string) (model.DeviceView, error) {
	device, err := s.devices.GetDevice(ctx, deviceID)
	if err != nil {
		return model.DeviceView{}, err
	}

	s.mu.RLock()
	state := s.states[deviceID]
	s.mu.RUnlock()

	view := model.DeviceView{
		Device: device,
		Online: state.session != nil,
	}
	if !state.connectedAt.IsZero() {
		connectedAt := state.connectedAt
		view.ConnectedAt = &connectedAt
	}
	if !state.lastSeen.IsZero() {
		lastSeen := state.lastSeen
		view.LastSeen = &lastSeen
	}
	return view, nil
}

func (s *Service) ListDevices(ctx context.Context) ([]model.DeviceView, error) {
	devices, err := s.devices.ListDevices(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]model.DeviceView, 0, len(devices))
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, device := range devices {
		state := s.states[device.ID]
		view := model.DeviceView{
			Device: device,
			Online: state.session != nil,
		}
		if !state.connectedAt.IsZero() {
			connectedAt := state.connectedAt
			view.ConnectedAt = &connectedAt
		}
		if !state.lastSeen.IsZero() {
			lastSeen := state.lastSeen
			view.LastSeen = &lastSeen
		}
		result = append(result, view)
	}

	return result, nil
}

func (s *Service) AuthenticateDevice(ctx context.Context, deviceID, token string) (model.Device, error) {
	device, err := s.devices.GetDevice(ctx, strings.TrimSpace(deviceID))
	if err != nil {
		return model.Device{}, err
	}
	if device.Token != strings.TrimSpace(token) {
		return model.Device{}, store.ErrInvalidCredential
	}
	return device, nil
}

func (s *Service) RegisterSession(deviceID string, session Session) {
	now := time.Now().UTC()

	var previous Session
	var shouldIncrement bool

	s.mu.Lock()
	state := s.states[deviceID]
	if state.session == nil {
		shouldIncrement = true
	} else if state.sessionID != session.SessionID() {
		previous = state.session
	}

	s.states[deviceID] = deviceState{
		session:     session,
		sessionID:   session.SessionID(),
		connectedAt: now,
		lastSeen:    now,
	}
	s.mu.Unlock()

	if shouldIncrement {
		s.onlineDevices.Add(1)
	}
	if previous != nil {
		s.logger.Warn("replacing existing session", "device_id", deviceID, "session_id", previous.SessionID())
		_ = previous.Close()
	}
}

func (s *Service) UnregisterSession(deviceID, sessionID string) {
	now := time.Now().UTC()
	var shouldDecrement bool

	s.mu.Lock()
	state := s.states[deviceID]
	if state.session != nil && state.sessionID == sessionID {
		state.session = nil
		state.sessionID = ""
		state.lastSeen = now
		s.states[deviceID] = state
		shouldDecrement = true
	}
	s.mu.Unlock()

	if shouldDecrement {
		s.onlineDevices.Add(-1)
	}
}

func (s *Service) TouchDevice(deviceID string, at time.Time) {
	at = at.UTC()

	s.mu.Lock()
	state := s.states[deviceID]
	if state.lastSeen.Before(at) {
		state.lastSeen = at
	}
	if state.connectedAt.IsZero() && state.session != nil {
		state.connectedAt = at
	}
	s.states[deviceID] = state
	s.mu.Unlock()
}

func (s *Service) HandleTelemetry(ctx context.Context, deviceID string, at time.Time, values map[string]any) error {
	if _, err := s.devices.GetDevice(ctx, deviceID); err != nil {
		return err
	}

	telemetry := model.Telemetry{
		DeviceID:  deviceID,
		Timestamp: at.UTC(),
		Values:    cloneAnyMap(values),
	}
	if telemetry.Timestamp.IsZero() {
		telemetry.Timestamp = time.Now().UTC()
	}

	if err := s.telemetry.AppendTelemetry(ctx, telemetry); err != nil {
		return err
	}

	s.telemetryReceived.Add(1)
	s.TouchDevice(deviceID, telemetry.Timestamp)
	return nil
}

func (s *Service) SendCommand(ctx context.Context, deviceID, name string, params map[string]any) (model.Command, error) {
	if _, err := s.devices.GetDevice(ctx, deviceID); err != nil {
		return model.Command{}, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return model.Command{}, errors.New("command name is required")
	}

	now := time.Now().UTC()
	command := model.Command{
		ID:        util.NewID("cmd"),
		DeviceID:  deviceID,
		Name:      name,
		Params:    cloneAnyMap(params),
		Status:    model.CommandStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.commands.SaveCommand(ctx, command); err != nil {
		return model.Command{}, err
	}

	s.mu.RLock()
	state := s.states[deviceID]
	session := state.session
	s.mu.RUnlock()

	if session == nil {
		updated, updateErr := s.commands.UpdateCommandStatus(ctx, command.ID, model.CommandStatusFailed, ErrDeviceOffline.Error())
		if updateErr == nil {
			command = updated
		}
		return command, ErrDeviceOffline
	}

	message := model.ServerMessage{
		Type:       "command",
		ServerTime: now.UnixMilli(),
		CommandID:  command.ID,
		Name:       command.Name,
		Params:     cloneAnyMap(command.Params),
	}

	if err := session.Send(message); err != nil {
		updated, updateErr := s.commands.UpdateCommandStatus(ctx, command.ID, model.CommandStatusFailed, err.Error())
		if updateErr == nil {
			command = updated
		}
		return command, err
	}

	updated, err := s.commands.UpdateCommandStatus(ctx, command.ID, model.CommandStatusSent, "sent")
	if err != nil {
		return model.Command{}, err
	}

	s.commandsSent.Add(1)
	s.TouchDevice(deviceID, now)
	return updated, nil
}

func (s *Service) HandleCommandAck(ctx context.Context, deviceID, commandID, status, result string) error {
	command, err := s.commands.GetCommand(ctx, commandID)
	if err != nil {
		return err
	}
	if command.DeviceID != deviceID {
		return store.ErrCommandNotFound
	}

	nextStatus := model.CommandStatusAcked
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "error", "timeout":
		nextStatus = model.CommandStatusFailed
	}

	if _, err := s.commands.UpdateCommandStatus(ctx, commandID, nextStatus, strings.TrimSpace(result)); err != nil {
		return err
	}

	s.commandAcks.Add(1)
	s.TouchDevice(deviceID, time.Now().UTC())
	return nil
}

func (s *Service) ListTelemetry(ctx context.Context, deviceID string, limit int) ([]model.Telemetry, error) {
	if _, err := s.devices.GetDevice(ctx, deviceID); err != nil {
		return nil, err
	}
	return s.telemetry.ListTelemetryByDevice(ctx, deviceID, limit)
}

func (s *Service) ListCommands(ctx context.Context, deviceID string, limit int) ([]model.Command, error) {
	if _, err := s.devices.GetDevice(ctx, deviceID); err != nil {
		return nil, err
	}
	return s.commands.ListCommandsByDevice(ctx, deviceID, limit)
}

func (s *Service) RecordConnectionAccepted() {
	s.totalConnections.Add(1)
}

func (s *Service) RecordConnectionRejected() {
	s.rejectedConnections.Add(1)
}

func (s *Service) Stats() model.Stats {
	return model.Stats{
		RegisteredDevices:   s.registeredDevices.Load(),
		OnlineDevices:       s.onlineDevices.Load(),
		TotalConnections:    s.totalConnections.Load(),
		RejectedConnections: s.rejectedConnections.Load(),
		TelemetryReceived:   s.telemetryReceived.Load(),
		CommandsSent:        s.commandsSent.Load(),
		CommandAcks:         s.commandAcks.Load(),
	}
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}

	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func cloneAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}

	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
