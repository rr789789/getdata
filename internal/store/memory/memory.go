package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"mvp-platform/internal/model"
	"mvp-platform/internal/store"
)

type Store struct {
	mu                 sync.RWMutex
	devices            map[string]model.Device
	telemetryByDevice  map[string][]model.Telemetry
	commandByID        map[string]model.Command
	commandIDsByDevice map[string][]string
	telemetryRetention int
}

func New(telemetryRetention int) *Store {
	if telemetryRetention <= 0 {
		telemetryRetention = 200
	}

	return &Store{
		devices:            make(map[string]model.Device),
		telemetryByDevice:  make(map[string][]model.Telemetry),
		commandByID:        make(map[string]model.Command),
		commandIDsByDevice: make(map[string][]string),
		telemetryRetention: telemetryRetention,
	}
}

func (s *Store) CreateDevice(_ context.Context, device model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.devices[device.ID]; exists {
		return store.ErrDeviceExists
	}

	s.devices[device.ID] = cloneDevice(device)
	return nil
}

func (s *Store) GetDevice(_ context.Context, deviceID string) (model.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	device, exists := s.devices[deviceID]
	if !exists {
		return model.Device{}, store.ErrDeviceNotFound
	}
	return cloneDevice(device), nil
}

func (s *Store) ListDevices(_ context.Context) ([]model.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.Device, 0, len(s.devices))
	for _, device := range s.devices {
		result = append(result, cloneDevice(device))
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (s *Store) AppendTelemetry(_ context.Context, telemetry model.Telemetry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := append(s.telemetryByDevice[telemetry.DeviceID], cloneTelemetry(telemetry))
	if len(items) > s.telemetryRetention {
		items = items[len(items)-s.telemetryRetention:]
	}
	s.telemetryByDevice[telemetry.DeviceID] = items
	return nil
}

func (s *Store) ListTelemetryByDevice(_ context.Context, deviceID string, limit int) ([]model.Telemetry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.telemetryByDevice[deviceID]
	if len(items) == 0 {
		return []model.Telemetry{}, nil
	}

	start := 0
	if limit > 0 && len(items) > limit {
		start = len(items) - limit
	}

	selected := items[start:]
	result := make([]model.Telemetry, 0, len(selected))
	for i := len(selected) - 1; i >= 0; i-- {
		result = append(result, cloneTelemetry(selected[i]))
	}
	return result, nil
}

func (s *Store) SaveCommand(_ context.Context, command model.Command) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.commandByID[command.ID] = cloneCommand(command)
	s.commandIDsByDevice[command.DeviceID] = append(s.commandIDsByDevice[command.DeviceID], command.ID)
	return nil
}

func (s *Store) GetCommand(_ context.Context, commandID string) (model.Command, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	command, exists := s.commandByID[commandID]
	if !exists {
		return model.Command{}, store.ErrCommandNotFound
	}
	return cloneCommand(command), nil
}

func (s *Store) UpdateCommandStatus(_ context.Context, commandID string, status model.CommandStatus, result string) (model.Command, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	command, exists := s.commandByID[commandID]
	if !exists {
		return model.Command{}, store.ErrCommandNotFound
	}

	command.Status = status
	command.Result = result
	command.UpdatedAt = time.Now().UTC()
	s.commandByID[commandID] = cloneCommand(command)
	return cloneCommand(command), nil
}

func (s *Store) ListCommandsByDevice(_ context.Context, deviceID string, limit int) ([]model.Command, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.commandIDsByDevice[deviceID]
	if len(ids) == 0 {
		return []model.Command{}, nil
	}

	start := 0
	if limit > 0 && len(ids) > limit {
		start = len(ids) - limit
	}

	selected := ids[start:]
	result := make([]model.Command, 0, len(selected))
	for i := len(selected) - 1; i >= 0; i-- {
		command := s.commandByID[selected[i]]
		result = append(result, cloneCommand(command))
	}
	return result, nil
}

func cloneDevice(device model.Device) model.Device {
	return model.Device{
		ID:        device.ID,
		Name:      device.Name,
		Token:     device.Token,
		Metadata:  cloneStringMap(device.Metadata),
		CreatedAt: device.CreatedAt,
	}
}

func cloneTelemetry(telemetry model.Telemetry) model.Telemetry {
	return model.Telemetry{
		DeviceID:  telemetry.DeviceID,
		Timestamp: telemetry.Timestamp,
		Values:    cloneAnyMap(telemetry.Values),
	}
}

func cloneCommand(command model.Command) model.Command {
	return model.Command{
		ID:        command.ID,
		DeviceID:  command.DeviceID,
		Name:      command.Name,
		Params:    cloneAnyMap(command.Params),
		Status:    command.Status,
		Result:    command.Result,
		CreatedAt: command.CreatedAt,
		UpdatedAt: command.UpdatedAt,
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
