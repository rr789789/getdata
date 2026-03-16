package memory

import (
	"context"
	"sort"

	"mvp-platform/internal/model"
)

type Snapshot struct {
	Products           map[string]model.Product        `json:"products"`
	Devices            map[string]model.Device         `json:"devices"`
	Groups             map[string]model.DeviceGroup    `json:"groups"`
	Rules              map[string]model.Rule           `json:"rules"`
	ConfigProfiles     map[string]model.ConfigProfile  `json:"config_profiles"`
	Shadows            map[string]model.DeviceShadow   `json:"shadows"`
	TelemetryByDevice  map[string][]model.Telemetry    `json:"telemetry_by_device"`
	CommandByID        map[string]model.Command        `json:"command_by_id"`
	CommandIDsByDevice map[string][]string             `json:"command_ids_by_device"`
	GroupIDsByDevice   map[string][]string             `json:"group_ids_by_device"`
	DeviceIDsByGroup   map[string][]string             `json:"device_ids_by_group"`
	Alerts             []model.AlertEvent              `json:"alerts"`
	TelemetryRetention int                             `json:"telemetry_retention"`
	AlertRetention     int                             `json:"alert_retention"`
}

func (s *Store) BackendName() string {
	return "memory"
}

func (s *Store) PersistencePath() string {
	return ""
}

func (s *Store) StorageStats(_ context.Context) (model.StorageStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var telemetrySamples int64
	for _, items := range s.telemetryByDevice {
		telemetrySamples += int64(len(items))
	}

	return model.StorageStats{
		Backend:          s.BackendName(),
		Products:         int64(len(s.products)),
		Devices:          int64(len(s.devices)),
		Groups:           int64(len(s.groups)),
		Rules:            int64(len(s.rules)),
		ConfigProfiles:   int64(len(s.configProfiles)),
		Shadows:          int64(len(s.shadows)),
		Commands:         int64(len(s.commandByID)),
		Alerts:           int64(len(s.alerts)),
		TelemetrySeries:  int64(len(s.telemetryByDevice)),
		TelemetrySamples: telemetrySamples,
	}, nil
}

func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := Snapshot{
		Products:           make(map[string]model.Product, len(s.products)),
		Devices:            make(map[string]model.Device, len(s.devices)),
		Groups:             make(map[string]model.DeviceGroup, len(s.groups)),
		Rules:              make(map[string]model.Rule, len(s.rules)),
		ConfigProfiles:     make(map[string]model.ConfigProfile, len(s.configProfiles)),
		Shadows:            make(map[string]model.DeviceShadow, len(s.shadows)),
		TelemetryByDevice:  make(map[string][]model.Telemetry, len(s.telemetryByDevice)),
		CommandByID:        make(map[string]model.Command, len(s.commandByID)),
		CommandIDsByDevice: make(map[string][]string, len(s.commandIDsByDevice)),
		GroupIDsByDevice:   make(map[string][]string, len(s.groupIDsByDevice)),
		DeviceIDsByGroup:   make(map[string][]string, len(s.deviceIDsByGroup)),
		Alerts:             make([]model.AlertEvent, 0, len(s.alerts)),
		TelemetryRetention: s.telemetryRetention,
		AlertRetention:     s.alertRetention,
	}

	for key, value := range s.products {
		snapshot.Products[key] = cloneProduct(value)
	}
	for key, value := range s.devices {
		snapshot.Devices[key] = cloneDevice(value)
	}
	for key, value := range s.groups {
		snapshot.Groups[key] = cloneGroup(value)
	}
	for key, value := range s.rules {
		snapshot.Rules[key] = cloneRule(value)
	}
	for key, value := range s.configProfiles {
		snapshot.ConfigProfiles[key] = cloneConfigProfile(value)
	}
	for key, value := range s.shadows {
		snapshot.Shadows[key] = cloneShadow(value)
	}
	for key, items := range s.telemetryByDevice {
		cloned := make([]model.Telemetry, 0, len(items))
		for _, item := range items {
			cloned = append(cloned, cloneTelemetry(item))
		}
		snapshot.TelemetryByDevice[key] = cloned
	}
	for key, value := range s.commandByID {
		snapshot.CommandByID[key] = cloneCommand(value)
	}
	for key, values := range s.commandIDsByDevice {
		snapshot.CommandIDsByDevice[key] = append([]string(nil), values...)
	}
	for key, values := range s.groupIDsByDevice {
		snapshot.GroupIDsByDevice[key] = append([]string(nil), values...)
	}
	for key, values := range s.deviceIDsByGroup {
		snapshot.DeviceIDsByGroup[key] = append([]string(nil), values...)
	}
	for _, alert := range s.alerts {
		snapshot.Alerts = append(snapshot.Alerts, cloneAlert(alert))
	}

	return snapshot
}

func (s *Store) Restore(snapshot Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.products = make(map[string]model.Product, len(snapshot.Products))
	for key, value := range snapshot.Products {
		s.products[key] = cloneProduct(value)
	}

	s.devices = make(map[string]model.Device, len(snapshot.Devices))
	for key, value := range snapshot.Devices {
		s.devices[key] = cloneDevice(value)
	}

	s.groups = make(map[string]model.DeviceGroup, len(snapshot.Groups))
	for key, value := range snapshot.Groups {
		s.groups[key] = cloneGroup(value)
	}

	s.rules = make(map[string]model.Rule, len(snapshot.Rules))
	for key, value := range snapshot.Rules {
		s.rules[key] = cloneRule(value)
	}

	s.configProfiles = make(map[string]model.ConfigProfile, len(snapshot.ConfigProfiles))
	for key, value := range snapshot.ConfigProfiles {
		s.configProfiles[key] = cloneConfigProfile(value)
	}

	s.shadows = make(map[string]model.DeviceShadow, len(snapshot.Shadows))
	for key, value := range snapshot.Shadows {
		s.shadows[key] = cloneShadow(value)
	}

	s.telemetryByDevice = make(map[string][]model.Telemetry, len(snapshot.TelemetryByDevice))
	for key, items := range snapshot.TelemetryByDevice {
		cloned := make([]model.Telemetry, 0, len(items))
		for _, item := range items {
			cloned = append(cloned, cloneTelemetry(item))
		}
		s.telemetryByDevice[key] = cloned
	}

	s.commandByID = make(map[string]model.Command, len(snapshot.CommandByID))
	for key, value := range snapshot.CommandByID {
		s.commandByID[key] = cloneCommand(value)
	}

	s.commandIDsByDevice = make(map[string][]string, len(snapshot.CommandIDsByDevice))
	for key, values := range snapshot.CommandIDsByDevice {
		s.commandIDsByDevice[key] = append([]string(nil), values...)
	}

	s.groupIDsByDevice = make(map[string][]string, len(snapshot.GroupIDsByDevice))
	for key, values := range snapshot.GroupIDsByDevice {
		s.groupIDsByDevice[key] = append([]string(nil), values...)
	}

	s.deviceIDsByGroup = make(map[string][]string, len(snapshot.DeviceIDsByGroup))
	for key, values := range snapshot.DeviceIDsByGroup {
		s.deviceIDsByGroup[key] = append([]string(nil), values...)
		sort.Strings(s.deviceIDsByGroup[key])
	}

	s.alerts = make([]model.AlertEvent, 0, len(snapshot.Alerts))
	for _, alert := range snapshot.Alerts {
		s.alerts = append(s.alerts, cloneAlert(alert))
	}

	if snapshot.TelemetryRetention > 0 {
		s.telemetryRetention = snapshot.TelemetryRetention
	}
	if snapshot.AlertRetention > 0 {
		s.alertRetention = snapshot.AlertRetention
	}
}
