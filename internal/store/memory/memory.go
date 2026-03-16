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
	products           map[string]model.Product
	devices            map[string]model.Device
	groups             map[string]model.DeviceGroup
	rules              map[string]model.Rule
	configProfiles     map[string]model.ConfigProfile
	shadows            map[string]model.DeviceShadow
	telemetryByDevice  map[string][]model.Telemetry
	commandByID        map[string]model.Command
	commandIDsByDevice map[string][]string
	groupIDsByDevice   map[string][]string
	deviceIDsByGroup   map[string][]string
	alerts             []model.AlertEvent
	telemetryRetention int
	alertRetention     int
}

func New(telemetryRetention int) *Store {
	if telemetryRetention <= 0 {
		telemetryRetention = 200
	}

	return &Store{
		products:           make(map[string]model.Product),
		devices:            make(map[string]model.Device),
		groups:             make(map[string]model.DeviceGroup),
		rules:              make(map[string]model.Rule),
		configProfiles:     make(map[string]model.ConfigProfile),
		shadows:            make(map[string]model.DeviceShadow),
		telemetryByDevice:  make(map[string][]model.Telemetry),
		commandByID:        make(map[string]model.Command),
		commandIDsByDevice: make(map[string][]string),
		groupIDsByDevice:   make(map[string][]string),
		deviceIDsByGroup:   make(map[string][]string),
		telemetryRetention: telemetryRetention,
		alertRetention:     max(telemetryRetention*2, 200),
	}
}

func (s *Store) CreateProduct(_ context.Context, product model.Product) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.products[product.ID]; exists {
		return store.ErrProductExists
	}

	s.products[product.ID] = cloneProduct(product)
	return nil
}

func (s *Store) GetProduct(_ context.Context, productID string) (model.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	product, exists := s.products[productID]
	if !exists {
		return model.Product{}, store.ErrProductNotFound
	}
	return cloneProduct(product), nil
}

func (s *Store) ListProducts(_ context.Context) ([]model.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.Product, 0, len(s.products))
	for _, product := range s.products {
		result = append(result, cloneProduct(product))
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (s *Store) SaveProduct(_ context.Context, product model.Product) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.products[product.ID]; !exists {
		return store.ErrProductNotFound
	}

	s.products[product.ID] = cloneProduct(product)
	return nil
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

func (s *Store) SaveDevice(_ context.Context, device model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.devices[device.ID]; !exists {
		return store.ErrDeviceNotFound
	}

	s.devices[device.ID] = cloneDevice(device)
	return nil
}

func (s *Store) CreateGroup(_ context.Context, group model.DeviceGroup) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.groups[group.ID]; exists {
		return store.ErrGroupExists
	}

	s.groups[group.ID] = cloneGroup(group)
	return nil
}

func (s *Store) GetGroup(_ context.Context, groupID string) (model.DeviceGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	group, exists := s.groups[groupID]
	if !exists {
		return model.DeviceGroup{}, store.ErrGroupNotFound
	}
	return cloneGroup(group), nil
}

func (s *Store) ListGroups(_ context.Context) ([]model.DeviceGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.DeviceGroup, 0, len(s.groups))
	for _, group := range s.groups {
		result = append(result, cloneGroup(group))
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (s *Store) SaveGroup(_ context.Context, group model.DeviceGroup) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.groups[group.ID]; !exists {
		return store.ErrGroupNotFound
	}

	s.groups[group.ID] = cloneGroup(group)
	return nil
}

func (s *Store) AddDeviceToGroup(_ context.Context, groupID, deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.groups[groupID]; !exists {
		return store.ErrGroupNotFound
	}
	if _, exists := s.devices[deviceID]; !exists {
		return store.ErrDeviceNotFound
	}

	s.groupIDsByDevice[deviceID] = addUniqueString(s.groupIDsByDevice[deviceID], groupID)
	s.deviceIDsByGroup[groupID] = addUniqueString(s.deviceIDsByGroup[groupID], deviceID)
	return nil
}

func (s *Store) RemoveDeviceFromGroup(_ context.Context, groupID, deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.groups[groupID]; !exists {
		return store.ErrGroupNotFound
	}
	if _, exists := s.devices[deviceID]; !exists {
		return store.ErrDeviceNotFound
	}

	s.groupIDsByDevice[deviceID] = removeString(s.groupIDsByDevice[deviceID], groupID)
	s.deviceIDsByGroup[groupID] = removeString(s.deviceIDsByGroup[groupID], deviceID)
	return nil
}

func (s *Store) ListGroupIDsByDevice(_ context.Context, deviceID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.devices[deviceID]; !exists {
		return nil, store.ErrDeviceNotFound
	}
	return cloneStringSliceSorted(s.groupIDsByDevice[deviceID]), nil
}

func (s *Store) ListDeviceIDsByGroup(_ context.Context, groupID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.groups[groupID]; !exists {
		return nil, store.ErrGroupNotFound
	}
	return cloneStringSliceSorted(s.deviceIDsByGroup[groupID]), nil
}

func (s *Store) CreateRule(_ context.Context, rule model.Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.rules[rule.ID]; exists {
		return store.ErrRuleExists
	}

	s.rules[rule.ID] = cloneRule(rule)
	return nil
}

func (s *Store) GetRule(_ context.Context, ruleID string) (model.Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rule, exists := s.rules[ruleID]
	if !exists {
		return model.Rule{}, store.ErrRuleNotFound
	}
	return cloneRule(rule), nil
}

func (s *Store) ListRules(_ context.Context) ([]model.Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.Rule, 0, len(s.rules))
	for _, rule := range s.rules {
		result = append(result, cloneRule(rule))
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (s *Store) SaveRule(_ context.Context, rule model.Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.rules[rule.ID]; !exists {
		return store.ErrRuleNotFound
	}

	s.rules[rule.ID] = cloneRule(rule)
	return nil
}

func (s *Store) CreateConfigProfile(_ context.Context, profile model.ConfigProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.configProfiles[profile.ID]; exists {
		return store.ErrConfigExists
	}

	s.configProfiles[profile.ID] = cloneConfigProfile(profile)
	return nil
}

func (s *Store) GetConfigProfile(_ context.Context, profileID string) (model.ConfigProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profile, exists := s.configProfiles[profileID]
	if !exists {
		return model.ConfigProfile{}, store.ErrConfigNotFound
	}
	return cloneConfigProfile(profile), nil
}

func (s *Store) ListConfigProfiles(_ context.Context) ([]model.ConfigProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.ConfigProfile, 0, len(s.configProfiles))
	for _, profile := range s.configProfiles {
		result = append(result, cloneConfigProfile(profile))
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (s *Store) SaveConfigProfile(_ context.Context, profile model.ConfigProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.configProfiles[profile.ID]; !exists {
		return store.ErrConfigNotFound
	}

	s.configProfiles[profile.ID] = cloneConfigProfile(profile)
	return nil
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

func (s *Store) SaveShadow(_ context.Context, shadow model.DeviceShadow) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.shadows[shadow.DeviceID] = cloneShadow(shadow)
	return nil
}

func (s *Store) GetShadow(_ context.Context, deviceID string) (model.DeviceShadow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	shadow, exists := s.shadows[deviceID]
	if !exists {
		return model.DeviceShadow{}, store.ErrShadowNotFound
	}
	return cloneShadow(shadow), nil
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

func (s *Store) AppendAlert(_ context.Context, alert model.AlertEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.alerts = append(s.alerts, cloneAlert(alert))
	if len(s.alerts) > s.alertRetention {
		s.alerts = s.alerts[len(s.alerts)-s.alertRetention:]
	}
	return nil
}

func (s *Store) GetAlert(_ context.Context, alertID string) (model.AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := len(s.alerts) - 1; i >= 0; i-- {
		if s.alerts[i].ID == alertID {
			return cloneAlert(s.alerts[i]), nil
		}
	}
	return model.AlertEvent{}, store.ErrAlertNotFound
}

func (s *Store) SaveAlert(_ context.Context, alert model.AlertEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.alerts {
		if s.alerts[i].ID == alert.ID {
			s.alerts[i] = cloneAlert(alert)
			return nil
		}
	}
	return store.ErrAlertNotFound
}

func (s *Store) ListAlerts(_ context.Context, limit int) ([]model.AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.alerts) == 0 {
		return []model.AlertEvent{}, nil
	}

	start := 0
	if limit > 0 && len(s.alerts) > limit {
		start = len(s.alerts) - limit
	}

	selected := s.alerts[start:]
	result := make([]model.AlertEvent, 0, len(selected))
	for i := len(selected) - 1; i >= 0; i-- {
		result = append(result, cloneAlert(selected[i]))
	}
	return result, nil
}

func cloneDevice(device model.Device) model.Device {
	return model.Device{
		ID:         device.ID,
		Name:       device.Name,
		ProductID:  device.ProductID,
		ProductKey: device.ProductKey,
		Token:      device.Token,
		Tags:       cloneStringMap(device.Tags),
		Metadata:   cloneStringMap(device.Metadata),
		CreatedAt:  device.CreatedAt,
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

func cloneProduct(product model.Product) model.Product {
	return model.Product{
		ID:          product.ID,
		Key:         product.Key,
		Name:        product.Name,
		Description: product.Description,
		Metadata:    cloneStringMap(product.Metadata),
		ThingModel:  cloneThingModel(product.ThingModel),
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}
}

func cloneGroup(group model.DeviceGroup) model.DeviceGroup {
	return model.DeviceGroup{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
		ProductID:   group.ProductID,
		Tags:        cloneStringMap(group.Tags),
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
	}
}

func cloneRule(rule model.Rule) model.Rule {
	return model.Rule{
		ID:              rule.ID,
		Name:            rule.Name,
		Description:     rule.Description,
		ProductID:       rule.ProductID,
		GroupID:         rule.GroupID,
		DeviceID:        rule.DeviceID,
		Enabled:         rule.Enabled,
		Severity:        rule.Severity,
		CooldownSeconds: rule.CooldownSeconds,
		Condition: model.RuleCondition{
			Property: rule.Condition.Property,
			Operator: rule.Condition.Operator,
			Value:    rule.Condition.Value,
		},
		CreatedAt: rule.CreatedAt,
		UpdatedAt: rule.UpdatedAt,
	}
}

func cloneConfigProfile(profile model.ConfigProfile) model.ConfigProfile {
	result := model.ConfigProfile{
		ID:           profile.ID,
		Name:         profile.Name,
		Description:  profile.Description,
		ProductID:    profile.ProductID,
		Values:       cloneAnyMap(profile.Values),
		AppliedCount: profile.AppliedCount,
		CreatedAt:    profile.CreatedAt,
		UpdatedAt:    profile.UpdatedAt,
	}
	if profile.LastAppliedAt != nil {
		value := *profile.LastAppliedAt
		result.LastAppliedAt = &value
	}
	return result
}

func cloneThingModel(modelValue model.ThingModel) model.ThingModel {
	properties := make([]model.ThingModelProperty, 0, len(modelValue.Properties))
	for _, property := range modelValue.Properties {
		properties = append(properties, property)
	}

	events := make([]model.ThingModelEvent, 0, len(modelValue.Events))
	for _, event := range modelValue.Events {
		eventCopy := event
		eventCopy.Output = cloneThingModelParameters(event.Output)
		events = append(events, eventCopy)
	}

	services := make([]model.ThingModelService, 0, len(modelValue.Services))
	for _, service := range modelValue.Services {
		serviceCopy := service
		serviceCopy.Input = cloneThingModelParameters(service.Input)
		serviceCopy.Output = cloneThingModelParameters(service.Output)
		services = append(services, serviceCopy)
	}

	return model.ThingModel{
		Properties: properties,
		Events:     events,
		Services:   services,
		Version:    modelValue.Version,
		UpdatedAt:  modelValue.UpdatedAt,
	}
}

func cloneThingModelParameters(values []model.ThingModelParameter) []model.ThingModelParameter {
	if len(values) == 0 {
		return nil
	}

	result := make([]model.ThingModelParameter, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

func cloneShadow(shadow model.DeviceShadow) model.DeviceShadow {
	result := model.DeviceShadow{
		DeviceID:  shadow.DeviceID,
		ProductID: shadow.ProductID,
		Reported:  cloneAnyMap(shadow.Reported),
		Desired:   cloneAnyMap(shadow.Desired),
		Version:   shadow.Version,
		UpdatedAt: shadow.UpdatedAt,
	}
	if shadow.LastReportedAt != nil {
		value := *shadow.LastReportedAt
		result.LastReportedAt = &value
	}
	if shadow.LastDesiredAt != nil {
		value := *shadow.LastDesiredAt
		result.LastDesiredAt = &value
	}
	return result
}

func cloneAlert(alert model.AlertEvent) model.AlertEvent {
	result := model.AlertEvent{
		ID:          alert.ID,
		RuleID:      alert.RuleID,
		RuleName:    alert.RuleName,
		ProductID:   alert.ProductID,
		GroupID:     alert.GroupID,
		DeviceID:    alert.DeviceID,
		DeviceName:  alert.DeviceName,
		Property:    alert.Property,
		Operator:    alert.Operator,
		Threshold:   alert.Threshold,
		Value:       alert.Value,
		Severity:    alert.Severity,
		Status:      alert.Status,
		Message:     alert.Message,
		Note:        alert.Note,
		TriggeredAt: alert.TriggeredAt,
	}
	if alert.AckAt != nil {
		value := *alert.AckAt
		result.AckAt = &value
	}
	if alert.ResolvedAt != nil {
		value := *alert.ResolvedAt
		result.ResolvedAt = &value
	}
	return result
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

func cloneStringSliceSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func addUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func removeString(values []string, value string) []string {
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	for _, existing := range values {
		if existing != value {
			result = append(result, existing)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
