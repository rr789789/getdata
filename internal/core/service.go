package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mvp-platform/internal/model"
	"mvp-platform/internal/store"
	"mvp-platform/internal/util"
)

var (
	ErrDeviceOffline        = errors.New("device offline")
	ErrUnknownThingProperty = errors.New("unknown thing model property")
	ErrInvalidThingModel    = errors.New("invalid thing model")
	ErrInvalidProduct       = errors.New("invalid product")
	ErrInvalidGroup         = errors.New("invalid group")
	ErrInvalidRule          = errors.New("invalid rule")
	ErrInvalidConfig        = errors.New("invalid config profile")
	ErrInvalidAlert         = errors.New("invalid alert")
)

type Session interface {
	SessionID() string
	Send(message model.ServerMessage) error
	Close() error
}

type Service struct {
	products  store.ProductStore
	devices   store.DeviceStore
	groups    store.GroupStore
	rules     store.RuleStore
	configs   store.ConfigStore
	telemetry store.TelemetryStore
	shadows   store.ShadowStore
	commands  store.CommandStore
	alerts    store.AlertStore
	logger    *slog.Logger

	mu         sync.RWMutex
	states     map[string]deviceState
	ruleStates map[string]ruleRuntimeState

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

type ruleRuntimeState struct {
	TriggeredCount  int64
	LastTriggeredAt time.Time
}

func NewService(
	products store.ProductStore,
	devices store.DeviceStore,
	groups store.GroupStore,
	rules store.RuleStore,
	configs store.ConfigStore,
	telemetry store.TelemetryStore,
	shadows store.ShadowStore,
	commands store.CommandStore,
	alerts store.AlertStore,
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		products:   products,
		devices:    devices,
		groups:     groups,
		rules:      rules,
		configs:    configs,
		telemetry:  telemetry,
		shadows:    shadows,
		commands:   commands,
		alerts:     alerts,
		logger:     logger,
		states:     make(map[string]deviceState),
		ruleStates: make(map[string]ruleRuntimeState),
	}
}

func (s *Service) CreateProduct(ctx context.Context, name, description string, metadata map[string]string, accessProfile model.ProductAccessProfile, thingModel model.ThingModel) (model.Product, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.Product{}, fmt.Errorf("%w: product name is required", ErrInvalidProduct)
	}

	now := time.Now().UTC()
	normalizedThingModel, err := normalizeThingModel(thingModel, 1, now)
	if err != nil {
		return model.Product{}, err
	}
	normalizedAccessProfile, err := normalizeAccessProfile(accessProfile)
	if err != nil {
		return model.Product{}, err
	}
	if err := validateAccessMappings(normalizedThingModel, normalizedAccessProfile); err != nil {
		return model.Product{}, err
	}

	product := model.Product{
		ID:            util.NewID("prd"),
		Key:           util.NewID("pk"),
		Name:          name,
		Description:   strings.TrimSpace(description),
		Metadata:      cloneStringMap(metadata),
		AccessProfile: normalizedAccessProfile,
		ThingModel:    normalizedThingModel,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.products.CreateProduct(ctx, product); err != nil {
		return model.Product{}, err
	}
	return product, nil
}

func (s *Service) GetProduct(ctx context.Context, productID string) (model.ProductView, error) {
	product, err := s.products.GetProduct(ctx, strings.TrimSpace(productID))
	if err != nil {
		return model.ProductView{}, err
	}
	return s.buildProductView(ctx, product)
}

func (s *Service) ListProducts(ctx context.Context) ([]model.ProductView, error) {
	products, err := s.products.ListProducts(ctx)
	if err != nil {
		return nil, err
	}

	views := make([]model.ProductView, 0, len(products))
	for _, product := range products {
		view, buildErr := s.buildProductView(ctx, product)
		if buildErr != nil {
			return nil, buildErr
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) UpdateProductThingModel(ctx context.Context, productID string, thingModel model.ThingModel) (model.Product, error) {
	product, err := s.products.GetProduct(ctx, strings.TrimSpace(productID))
	if err != nil {
		return model.Product{}, err
	}

	now := time.Now().UTC()
	normalizedThingModel, err := normalizeThingModel(thingModel, product.ThingModel.Version+1, now)
	if err != nil {
		return model.Product{}, err
	}

	product.ThingModel = normalizedThingModel
	product.UpdatedAt = now
	if err := s.products.SaveProduct(ctx, product); err != nil {
		return model.Product{}, err
	}
	return product, nil
}

func (s *Service) UpdateProductAccessProfile(ctx context.Context, productID string, accessProfile model.ProductAccessProfile) (model.Product, error) {
	product, err := s.products.GetProduct(ctx, strings.TrimSpace(productID))
	if err != nil {
		return model.Product{}, err
	}

	normalizedAccessProfile, err := normalizeAccessProfile(accessProfile)
	if err != nil {
		return model.Product{}, err
	}
	if err := validateAccessMappings(product.ThingModel, normalizedAccessProfile); err != nil {
		return model.Product{}, err
	}

	product.AccessProfile = normalizedAccessProfile
	product.UpdatedAt = time.Now().UTC()
	if err := s.products.SaveProduct(ctx, product); err != nil {
		return model.Product{}, err
	}
	return product, nil
}

func (s *Service) ProtocolCatalog() []model.ProtocolCatalogEntry {
	return model.DefaultProtocolCatalog()
}

func (s *Service) CreateGroup(ctx context.Context, name, description, productID string, tags map[string]string) (model.DeviceGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.DeviceGroup{}, fmt.Errorf("%w: group name is required", ErrInvalidGroup)
	}

	productID = strings.TrimSpace(productID)
	if productID != "" {
		if _, err := s.products.GetProduct(ctx, productID); err != nil {
			return model.DeviceGroup{}, err
		}
	}

	now := time.Now().UTC()
	group := model.DeviceGroup{
		ID:          util.NewID("grp"),
		Name:        name,
		Description: strings.TrimSpace(description),
		ProductID:   productID,
		Tags:        cloneStringMap(tags),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.groups.CreateGroup(ctx, group); err != nil {
		return model.DeviceGroup{}, err
	}
	return group, nil
}

func (s *Service) GetGroup(ctx context.Context, groupID string) (model.GroupView, error) {
	group, err := s.groups.GetGroup(ctx, strings.TrimSpace(groupID))
	if err != nil {
		return model.GroupView{}, err
	}
	return s.buildGroupView(ctx, group)
}

func (s *Service) ListGroups(ctx context.Context) ([]model.GroupView, error) {
	groups, err := s.groups.ListGroups(ctx)
	if err != nil {
		return nil, err
	}

	views := make([]model.GroupView, 0, len(groups))
	for _, group := range groups {
		view, buildErr := s.buildGroupView(ctx, group)
		if buildErr != nil {
			return nil, buildErr
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) AssignDeviceToGroup(ctx context.Context, groupID, deviceID string) (model.GroupView, error) {
	group, err := s.groups.GetGroup(ctx, strings.TrimSpace(groupID))
	if err != nil {
		return model.GroupView{}, err
	}

	device, err := s.devices.GetDevice(ctx, strings.TrimSpace(deviceID))
	if err != nil {
		return model.GroupView{}, err
	}

	if group.ProductID != "" && device.ProductID != group.ProductID {
		return model.GroupView{}, fmt.Errorf("%w: group %s only accepts devices from product %s", ErrInvalidGroup, group.ID, group.ProductID)
	}

	if err := s.groups.AddDeviceToGroup(ctx, group.ID, device.ID); err != nil {
		return model.GroupView{}, err
	}
	return s.buildGroupView(ctx, group)
}

func (s *Service) RemoveDeviceFromGroup(ctx context.Context, groupID, deviceID string) (model.GroupView, error) {
	group, err := s.groups.GetGroup(ctx, strings.TrimSpace(groupID))
	if err != nil {
		return model.GroupView{}, err
	}
	if _, err := s.devices.GetDevice(ctx, strings.TrimSpace(deviceID)); err != nil {
		return model.GroupView{}, err
	}
	if err := s.groups.RemoveDeviceFromGroup(ctx, group.ID, strings.TrimSpace(deviceID)); err != nil {
		return model.GroupView{}, err
	}
	return s.buildGroupView(ctx, group)
}

func (s *Service) CreateDevice(ctx context.Context, name, productID string, tags, metadata map[string]string) (model.Device, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "device"
	}

	productID = strings.TrimSpace(productID)
	var product model.Product
	var err error
	if productID != "" {
		product, err = s.products.GetProduct(ctx, productID)
		if err != nil {
			return model.Device{}, err
		}
	}

	now := time.Now().UTC()
	device := model.Device{
		ID:         util.NewID("dev"),
		Name:       name,
		ProductID:  productID,
		ProductKey: product.Key,
		Token:      util.NewToken(),
		Tags:       cloneStringMap(tags),
		Metadata:   cloneStringMap(metadata),
		CreatedAt:  now,
	}

	if err := s.devices.CreateDevice(ctx, device); err != nil {
		return model.Device{}, err
	}

	shadow := model.DeviceShadow{
		DeviceID:  device.ID,
		ProductID: productID,
		Reported:  map[string]any{},
		Desired:   map[string]any{},
		Version:   1,
		UpdatedAt: now,
	}
	if err := s.shadows.SaveShadow(ctx, shadow); err != nil {
		return model.Device{}, err
	}

	s.registeredDevices.Add(1)
	return device, nil
}

func (s *Service) UpdateDeviceTags(ctx context.Context, deviceID string, tags map[string]string) (model.Device, error) {
	device, err := s.devices.GetDevice(ctx, strings.TrimSpace(deviceID))
	if err != nil {
		return model.Device{}, err
	}

	device.Tags = cloneStringMap(tags)
	if err := s.devices.SaveDevice(ctx, device); err != nil {
		return model.Device{}, err
	}
	return device, nil
}

func (s *Service) CreateConfigProfile(ctx context.Context, name, description, productID string, values map[string]any) (model.ConfigProfile, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.ConfigProfile{}, fmt.Errorf("%w: config profile name is required", ErrInvalidConfig)
	}

	productID = strings.TrimSpace(productID)
	product, hasProduct := s.loadProduct(ctx, productID)
	if productID != "" && !hasProduct {
		return model.ConfigProfile{}, store.ErrProductNotFound
	}
	if hasProduct {
		if err := validateThingValues(product, values); err != nil {
			return model.ConfigProfile{}, err
		}
	}

	now := time.Now().UTC()
	profile := model.ConfigProfile{
		ID:          util.NewID("cfg"),
		Name:        name,
		Description: strings.TrimSpace(description),
		ProductID:   productID,
		Values:      cloneAnyMap(values),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if profile.Values == nil {
		profile.Values = map[string]any{}
	}

	if err := s.configs.CreateConfigProfile(ctx, profile); err != nil {
		return model.ConfigProfile{}, err
	}
	return profile, nil
}

func (s *Service) ListConfigProfiles(ctx context.Context) ([]model.ConfigProfileView, error) {
	profiles, err := s.configs.ListConfigProfiles(ctx)
	if err != nil {
		return nil, err
	}

	views := make([]model.ConfigProfileView, 0, len(profiles))
	for _, profile := range profiles {
		view, buildErr := s.buildConfigProfileView(ctx, profile)
		if buildErr != nil {
			return nil, buildErr
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) ApplyConfigProfile(ctx context.Context, profileID, deviceID string) (model.DeviceShadow, error) {
	profile, err := s.configs.GetConfigProfile(ctx, strings.TrimSpace(profileID))
	if err != nil {
		return model.DeviceShadow{}, err
	}

	device, err := s.devices.GetDevice(ctx, strings.TrimSpace(deviceID))
	if err != nil {
		return model.DeviceShadow{}, err
	}
	if profile.ProductID != "" && device.ProductID != profile.ProductID {
		return model.DeviceShadow{}, fmt.Errorf("%w: config profile product scope mismatch", ErrInvalidConfig)
	}

	shadow, err := s.UpdateDesiredShadow(ctx, device.ID, profile.Values)
	if err != nil {
		return model.DeviceShadow{}, err
	}

	now := time.Now().UTC()
	profile.AppliedCount++
	profile.LastAppliedAt = &now
	profile.UpdatedAt = now
	if err := s.configs.SaveConfigProfile(ctx, profile); err != nil {
		return model.DeviceShadow{}, err
	}
	return shadow, nil
}

func (s *Service) GetDevice(ctx context.Context, deviceID string) (model.DeviceView, error) {
	device, err := s.devices.GetDevice(ctx, strings.TrimSpace(deviceID))
	if err != nil {
		return model.DeviceView{}, err
	}
	return s.buildDeviceView(ctx, device)
}

func (s *Service) ListDevices(ctx context.Context, productID string) ([]model.DeviceView, error) {
	devices, err := s.devices.ListDevices(ctx)
	if err != nil {
		return nil, err
	}

	productID = strings.TrimSpace(productID)
	result := make([]model.DeviceView, 0, len(devices))
	for _, device := range devices {
		if productID != "" && device.ProductID != productID {
			continue
		}

		view, buildErr := s.buildDeviceView(ctx, device)
		if buildErr != nil {
			return nil, buildErr
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *Service) GetShadow(ctx context.Context, deviceID string) (model.DeviceShadow, error) {
	device, err := s.devices.GetDevice(ctx, strings.TrimSpace(deviceID))
	if err != nil {
		return model.DeviceShadow{}, err
	}
	return s.ensureShadow(ctx, device)
}

func (s *Service) UpdateDesiredShadow(ctx context.Context, deviceID string, desired map[string]any) (model.DeviceShadow, error) {
	device, err := s.devices.GetDevice(ctx, strings.TrimSpace(deviceID))
	if err != nil {
		return model.DeviceShadow{}, err
	}

	product, hasProduct := s.loadProduct(ctx, device.ProductID)
	if hasProduct {
		if err := validateThingValues(product, desired); err != nil {
			return model.DeviceShadow{}, err
		}
	}

	shadow, err := s.ensureShadow(ctx, device)
	if err != nil {
		return model.DeviceShadow{}, err
	}

	now := time.Now().UTC()
	shadow.Desired = cloneAnyMap(desired)
	if shadow.Desired == nil {
		shadow.Desired = map[string]any{}
	}
	shadow.Version++
	shadow.UpdatedAt = now
	shadow.LastDesiredAt = &now

	if err := s.shadows.SaveShadow(ctx, shadow); err != nil {
		return model.DeviceShadow{}, err
	}
	return shadow, nil
}

func (s *Service) CreateRule(
	ctx context.Context,
	name, description, productID, groupID, deviceID string,
	enabled bool,
	severity model.AlertSeverity,
	cooldownSeconds int,
	condition model.RuleCondition,
) (model.Rule, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.Rule{}, fmt.Errorf("%w: rule name is required", ErrInvalidRule)
	}

	groupID = strings.TrimSpace(groupID)
	deviceID = strings.TrimSpace(deviceID)
	productID = strings.TrimSpace(productID)
	if productID == "" && groupID == "" && deviceID == "" {
		return model.Rule{}, fmt.Errorf("%w: at least one scope product_id, group_id or device_id is required", ErrInvalidRule)
	}
	if cooldownSeconds < 0 {
		return model.Rule{}, fmt.Errorf("%w: cooldown_seconds must be >= 0", ErrInvalidRule)
	}

	if groupID != "" {
		group, err := s.groups.GetGroup(ctx, groupID)
		if err != nil {
			return model.Rule{}, err
		}
		if group.ProductID != "" {
			if productID == "" {
				productID = group.ProductID
			} else if productID != group.ProductID {
				return model.Rule{}, fmt.Errorf("%w: group product scope mismatch", ErrInvalidRule)
			}
		}
	}

	if deviceID != "" {
		device, err := s.devices.GetDevice(ctx, deviceID)
		if err != nil {
			return model.Rule{}, err
		}
		if device.ProductID != "" {
			if productID == "" {
				productID = device.ProductID
			} else if productID != device.ProductID {
				return model.Rule{}, fmt.Errorf("%w: device product scope mismatch", ErrInvalidRule)
			}
		}
	}

	product, hasProduct := s.loadProduct(ctx, productID)
	normalizedCondition, err := normalizeRuleCondition(product, hasProduct, condition)
	if err != nil {
		return model.Rule{}, err
	}

	now := time.Now().UTC()
	rule := model.Rule{
		ID:              util.NewID("rul"),
		Name:            name,
		Description:     strings.TrimSpace(description),
		ProductID:       productID,
		GroupID:         groupID,
		DeviceID:        deviceID,
		Enabled:         enabled,
		Severity:        normalizeSeverity(severity),
		CooldownSeconds: cooldownSeconds,
		Condition:       normalizedCondition,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.rules.CreateRule(ctx, rule); err != nil {
		return model.Rule{}, err
	}
	return rule, nil
}

func (s *Service) ListRules(ctx context.Context) ([]model.RuleView, error) {
	rules, err := s.rules.ListRules(ctx)
	if err != nil {
		return nil, err
	}

	views := make([]model.RuleView, 0, len(rules))
	for _, rule := range rules {
		view, buildErr := s.buildRuleView(ctx, rule)
		if buildErr != nil {
			return nil, buildErr
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) ListAlerts(ctx context.Context, limit int, productID, groupID, deviceID, ruleID string) ([]model.AlertEvent, error) {
	sourceLimit := limit
	if sourceLimit <= 0 {
		sourceLimit = 100
	}
	if (productID != "" || groupID != "" || deviceID != "" || ruleID != "") && sourceLimit < 200 {
		sourceLimit = 200
	}

	alerts, err := s.alerts.ListAlerts(ctx, sourceLimit)
	if err != nil {
		return nil, err
	}

	productID = strings.TrimSpace(productID)
	groupID = strings.TrimSpace(groupID)
	deviceID = strings.TrimSpace(deviceID)
	ruleID = strings.TrimSpace(ruleID)

	filtered := make([]model.AlertEvent, 0, len(alerts))
	for _, alert := range alerts {
		if productID != "" && alert.ProductID != productID {
			continue
		}
		if groupID != "" && alert.GroupID != groupID {
			continue
		}
		if deviceID != "" && alert.DeviceID != deviceID {
			continue
		}
		if ruleID != "" && alert.RuleID != ruleID {
			continue
		}
		filtered = append(filtered, alert)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered, nil
}

func (s *Service) UpdateAlert(ctx context.Context, alertID string, status model.AlertStatus, note string) (model.AlertEvent, error) {
	alert, err := s.alerts.GetAlert(ctx, strings.TrimSpace(alertID))
	if err != nil {
		return model.AlertEvent{}, err
	}

	status = normalizeAlertStatus(status)
	if status == "" {
		return model.AlertEvent{}, fmt.Errorf("%w: unsupported alert status", ErrInvalidAlert)
	}

	now := time.Now().UTC()
	alert.Status = status
	alert.Note = strings.TrimSpace(note)
	switch status {
	case model.AlertStatusAcknowledged:
		alert.AckAt = &now
	case model.AlertStatusResolved:
		if alert.AckAt == nil {
			alert.AckAt = &now
		}
		alert.ResolvedAt = &now
	}

	if err := s.alerts.SaveAlert(ctx, alert); err != nil {
		return model.AlertEvent{}, err
	}
	return alert, nil
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
	device, err := s.devices.GetDevice(ctx, deviceID)
	if err != nil {
		return err
	}

	product, hasProduct := s.loadProduct(ctx, device.ProductID)
	if hasProduct {
		if err := validateThingValues(product, values); err != nil {
			return err
		}
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

	shadow, err := s.ensureShadow(ctx, device)
	if err != nil {
		return err
	}

	now := telemetry.Timestamp
	for key, value := range telemetry.Values {
		if shadow.Reported == nil {
			shadow.Reported = make(map[string]any)
		}
		shadow.Reported[key] = value

		if desiredValue, exists := shadow.Desired[key]; exists && valuesEqual(desiredValue, value) {
			delete(shadow.Desired, key)
		}
	}
	shadow.Version++
	shadow.UpdatedAt = now
	shadow.LastReportedAt = &now
	if err := s.shadows.SaveShadow(ctx, shadow); err != nil {
		return err
	}

	if err := s.evaluateRules(ctx, device, telemetry); err != nil {
		return err
	}

	s.telemetryReceived.Add(1)
	s.TouchDevice(deviceID, telemetry.Timestamp)
	return nil
}

func (s *Service) SendCommand(ctx context.Context, deviceID, name string, params map[string]any) (model.Command, error) {
	device, err := s.devices.GetDevice(ctx, deviceID)
	if err != nil {
		return model.Command{}, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return model.Command{}, errors.New("command name is required")
	}

	product, hasProduct := s.loadProduct(ctx, device.ProductID)
	if hasProduct {
		if err := validateCommandName(product, name); err != nil {
			return model.Command{}, err
		}
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

func (s *Service) buildProductView(ctx context.Context, product model.Product) (model.ProductView, error) {
	devices, err := s.devices.ListDevices(ctx)
	if err != nil {
		return model.ProductView{}, err
	}

	view := model.ProductView{Product: product}
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, device := range devices {
		if device.ProductID != product.ID {
			continue
		}

		view.DeviceCount++
		if state := s.states[device.ID]; state.session != nil {
			view.OnlineCount++
		}
	}
	return view, nil
}

func (s *Service) buildGroupView(ctx context.Context, group model.DeviceGroup) (model.GroupView, error) {
	view := model.GroupView{Group: group}
	if group.ProductID != "" {
		product, err := s.products.GetProduct(ctx, group.ProductID)
		if err != nil && !errors.Is(err, store.ErrProductNotFound) {
			return model.GroupView{}, err
		}
		if err == nil {
			view.Product = &model.ProductSummary{ID: product.ID, Key: product.Key, Name: product.Name}
		}
	}

	deviceIDs, err := s.groups.ListDeviceIDsByGroup(ctx, group.ID)
	if err != nil {
		return model.GroupView{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, deviceID := range deviceIDs {
		view.DeviceCount++
		if state := s.states[deviceID]; state.session != nil {
			view.OnlineCount++
		}
	}
	return view, nil
}

func (s *Service) buildRuleView(ctx context.Context, rule model.Rule) (model.RuleView, error) {
	view := model.RuleView{Rule: rule}

	if rule.ProductID != "" {
		product, err := s.products.GetProduct(ctx, rule.ProductID)
		if err != nil && !errors.Is(err, store.ErrProductNotFound) {
			return model.RuleView{}, err
		}
		if err == nil {
			view.Product = &model.ProductSummary{ID: product.ID, Key: product.Key, Name: product.Name}
		}
	}
	if rule.GroupID != "" {
		group, err := s.groups.GetGroup(ctx, rule.GroupID)
		if err != nil && !errors.Is(err, store.ErrGroupNotFound) {
			return model.RuleView{}, err
		}
		if err == nil {
			view.Group = &model.GroupSummary{ID: group.ID, Name: group.Name}
		}
	}
	if rule.DeviceID != "" {
		device, err := s.devices.GetDevice(ctx, rule.DeviceID)
		if err != nil && !errors.Is(err, store.ErrDeviceNotFound) {
			return model.RuleView{}, err
		}
		if err == nil {
			view.Device = &model.DeviceSummary{ID: device.ID, Name: device.Name}
		}
	}

	prefix := makeRuleStateKey(rule.ID, "")
	s.mu.RLock()
	for key, runtime := range s.ruleStates {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		view.TriggeredCount += runtime.TriggeredCount
		if runtime.LastTriggeredAt.IsZero() {
			continue
		}
		if view.LastTriggeredAt == nil || runtime.LastTriggeredAt.After(*view.LastTriggeredAt) {
			lastTriggeredAt := runtime.LastTriggeredAt
			view.LastTriggeredAt = &lastTriggeredAt
		}
	}
	s.mu.RUnlock()

	return view, nil
}

func (s *Service) buildConfigProfileView(ctx context.Context, profile model.ConfigProfile) (model.ConfigProfileView, error) {
	view := model.ConfigProfileView{Profile: profile}
	if profile.ProductID == "" {
		return view, nil
	}

	product, err := s.products.GetProduct(ctx, profile.ProductID)
	if err != nil && !errors.Is(err, store.ErrProductNotFound) {
		return model.ConfigProfileView{}, err
	}
	if err == nil {
		view.Product = &model.ProductSummary{ID: product.ID, Key: product.Key, Name: product.Name}
	}
	return view, nil
}

func (s *Service) buildDeviceView(ctx context.Context, device model.Device) (model.DeviceView, error) {
	view := model.DeviceView{Device: device}
	if device.ProductID != "" {
		product, err := s.products.GetProduct(ctx, device.ProductID)
		if err != nil && !errors.Is(err, store.ErrProductNotFound) {
			return model.DeviceView{}, err
		}
		if err == nil {
			view.Product = &model.ProductSummary{
				ID:   product.ID,
				Key:  product.Key,
				Name: product.Name,
			}
		}
	}

	groups, err := s.buildGroupSummaries(ctx, device.ID)
	if err != nil && !errors.Is(err, store.ErrDeviceNotFound) {
		return model.DeviceView{}, err
	}
	view.Groups = groups

	s.mu.RLock()
	state := s.states[device.ID]
	s.mu.RUnlock()

	view.Online = state.session != nil
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

func (s *Service) ensureShadow(ctx context.Context, device model.Device) (model.DeviceShadow, error) {
	shadow, err := s.shadows.GetShadow(ctx, device.ID)
	if err == nil {
		return shadow, nil
	}
	if !errors.Is(err, store.ErrShadowNotFound) {
		return model.DeviceShadow{}, err
	}

	now := time.Now().UTC()
	shadow = model.DeviceShadow{
		DeviceID:  device.ID,
		ProductID: device.ProductID,
		Reported:  map[string]any{},
		Desired:   map[string]any{},
		Version:   1,
		UpdatedAt: now,
	}
	if saveErr := s.shadows.SaveShadow(ctx, shadow); saveErr != nil {
		return model.DeviceShadow{}, saveErr
	}
	return shadow, nil
}

func (s *Service) evaluateRules(ctx context.Context, device model.Device, telemetry model.Telemetry) error {
	rules, err := s.rules.ListRules(ctx)
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		return nil
	}

	var groupMembership map[string]struct{}
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if rule.ProductID != "" && rule.ProductID != device.ProductID {
			continue
		}
		if rule.DeviceID != "" && rule.DeviceID != device.ID {
			continue
		}
		if rule.GroupID != "" {
			if groupMembership == nil {
				groupMembership = make(map[string]struct{})
				groupIDs, err := s.groups.ListGroupIDsByDevice(ctx, device.ID)
				if err != nil && !errors.Is(err, store.ErrDeviceNotFound) {
					return err
				}
				for _, groupID := range groupIDs {
					groupMembership[groupID] = struct{}{}
				}
			}
			if _, exists := groupMembership[rule.GroupID]; !exists {
				continue
			}
		}

		value, exists := telemetry.Values[rule.Condition.Property]
		if !exists {
			continue
		}
		if !matchesRuleCondition(rule.Condition.Operator, value, rule.Condition.Value) {
			continue
		}
		if !s.canTriggerRule(rule.ID, device.ID, rule.CooldownSeconds, telemetry.Timestamp) {
			continue
		}

		alert := model.AlertEvent{
			ID:          util.NewID("alt"),
			RuleID:      rule.ID,
			RuleName:    rule.Name,
			ProductID:   device.ProductID,
			GroupID:     rule.GroupID,
			DeviceID:    device.ID,
			DeviceName:  device.Name,
			Property:    rule.Condition.Property,
			Operator:    rule.Condition.Operator,
			Threshold:   rule.Condition.Value,
			Value:       value,
			Severity:    rule.Severity,
			Status:      model.AlertStatusNew,
			Message:     fmt.Sprintf("%s: %s %s %v, got %v", rule.Name, rule.Condition.Property, rule.Condition.Operator, rule.Condition.Value, value),
			TriggeredAt: telemetry.Timestamp,
		}
		if err := s.alerts.AppendAlert(ctx, alert); err != nil {
			return err
		}
		s.recordRuleTrigger(rule.ID, device.ID, telemetry.Timestamp)
	}
	return nil
}

func (s *Service) canTriggerRule(ruleID, deviceID string, cooldownSeconds int, at time.Time) bool {
	if cooldownSeconds <= 0 {
		return true
	}

	s.mu.RLock()
	state := s.ruleStates[makeRuleStateKey(ruleID, deviceID)]
	s.mu.RUnlock()

	if state.LastTriggeredAt.IsZero() {
		return true
	}
	return at.Sub(state.LastTriggeredAt) >= time.Duration(cooldownSeconds)*time.Second
}

func (s *Service) recordRuleTrigger(ruleID, deviceID string, at time.Time) {
	key := makeRuleStateKey(ruleID, deviceID)

	s.mu.Lock()
	runtime := s.ruleStates[key]
	runtime.TriggeredCount++
	runtime.LastTriggeredAt = at.UTC()
	s.ruleStates[key] = runtime
	s.mu.Unlock()
}

func (s *Service) buildGroupSummaries(ctx context.Context, deviceID string) ([]model.GroupSummary, error) {
	groupIDs, err := s.groups.ListGroupIDsByDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if len(groupIDs) == 0 {
		return nil, nil
	}

	result := make([]model.GroupSummary, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		group, err := s.groups.GetGroup(ctx, groupID)
		if err != nil && !errors.Is(err, store.ErrGroupNotFound) {
			return nil, err
		}
		if err == nil {
			result = append(result, model.GroupSummary{ID: group.ID, Name: group.Name})
		}
	}
	return result, nil
}

func (s *Service) loadProduct(ctx context.Context, productID string) (model.Product, bool) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return model.Product{}, false
	}

	product, err := s.products.GetProduct(ctx, productID)
	if err != nil {
		return model.Product{}, false
	}
	return product, true
}

func normalizeThingModel(thingModel model.ThingModel, version int64, updatedAt time.Time) (model.ThingModel, error) {
	propertySeen := make(map[string]struct{})
	serviceSeen := make(map[string]struct{})
	eventSeen := make(map[string]struct{})

	normalized := model.ThingModel{
		Properties: make([]model.ThingModelProperty, 0, len(thingModel.Properties)),
		Events:     make([]model.ThingModelEvent, 0, len(thingModel.Events)),
		Services:   make([]model.ThingModelService, 0, len(thingModel.Services)),
		Version:    version,
		UpdatedAt:  updatedAt,
	}

	for _, property := range thingModel.Properties {
		property.Identifier = strings.TrimSpace(property.Identifier)
		property.Name = strings.TrimSpace(property.Name)
		property.DataType = normalizeDataType(property.DataType)
		property.AccessMode = normalizeAccessMode(property.AccessMode)
		if property.Identifier == "" || property.Name == "" || property.DataType == "" {
			return model.ThingModel{}, fmt.Errorf("%w: property identifier, name and data_type are required", ErrInvalidThingModel)
		}
		if !isSupportedDataType(property.DataType) {
			return model.ThingModel{}, fmt.Errorf("%w: unsupported property data_type %q", ErrInvalidThingModel, property.DataType)
		}
		if _, exists := propertySeen[property.Identifier]; exists {
			return model.ThingModel{}, fmt.Errorf("%w: duplicate property identifier %q", ErrInvalidThingModel, property.Identifier)
		}
		propertySeen[property.Identifier] = struct{}{}
		normalized.Properties = append(normalized.Properties, property)
	}

	for _, event := range thingModel.Events {
		event.Identifier = strings.TrimSpace(event.Identifier)
		event.Name = strings.TrimSpace(event.Name)
		if event.Identifier == "" || event.Name == "" {
			return model.ThingModel{}, fmt.Errorf("%w: event identifier and name are required", ErrInvalidThingModel)
		}
		if _, exists := eventSeen[event.Identifier]; exists {
			return model.ThingModel{}, fmt.Errorf("%w: duplicate event identifier %q", ErrInvalidThingModel, event.Identifier)
		}
		eventSeen[event.Identifier] = struct{}{}
		event.Output = normalizeParameters(event.Output)
		normalized.Events = append(normalized.Events, event)
	}

	for _, service := range thingModel.Services {
		service.Identifier = strings.TrimSpace(service.Identifier)
		service.Name = strings.TrimSpace(service.Name)
		if service.Identifier == "" || service.Name == "" {
			return model.ThingModel{}, fmt.Errorf("%w: service identifier and name are required", ErrInvalidThingModel)
		}
		if _, exists := serviceSeen[service.Identifier]; exists {
			return model.ThingModel{}, fmt.Errorf("%w: duplicate service identifier %q", ErrInvalidThingModel, service.Identifier)
		}
		serviceSeen[service.Identifier] = struct{}{}
		service.Input = normalizeParameters(service.Input)
		service.Output = normalizeParameters(service.Output)
		normalized.Services = append(normalized.Services, service)
	}

	return normalized, nil
}

func normalizeParameters(values []model.ThingModelParameter) []model.ThingModelParameter {
	if len(values) == 0 {
		return nil
	}

	result := make([]model.ThingModelParameter, 0, len(values))
	for _, value := range values {
		value.Identifier = strings.TrimSpace(value.Identifier)
		value.Name = strings.TrimSpace(value.Name)
		value.DataType = normalizeDataType(value.DataType)
		result = append(result, value)
	}
	return result
}

func normalizeAccessMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "r", "read":
		return "r"
	case "w", "write":
		return "w"
	default:
		return "rw"
	}
}

func normalizeDataType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "string":
		return "text"
	case "integer":
		return "int"
	default:
		return value
	}
}

func normalizeSeverity(value model.AlertSeverity) model.AlertSeverity {
	switch strings.ToLower(strings.TrimSpace(string(value))) {
	case "critical", "fatal", "high":
		return model.AlertSeverityCritical
	case "info", "notice", "low":
		return model.AlertSeverityInfo
	default:
		return model.AlertSeverityWarning
	}
}

func normalizeAlertStatus(value model.AlertStatus) model.AlertStatus {
	switch strings.ToLower(strings.TrimSpace(string(value))) {
	case "new":
		return model.AlertStatusNew
	case "ack", "acked", "acknowledged":
		return model.AlertStatusAcknowledged
	case "resolved", "closed", "done":
		return model.AlertStatusResolved
	default:
		return ""
	}
}

func normalizeAccessProfile(profile model.ProductAccessProfile) (model.ProductAccessProfile, error) {
	profile.Transport = normalizeTransport(profile.Transport)
	profile.Protocol = normalizeProtocol(profile.Protocol)
	profile.IngestMode = normalizeIngestMode(profile.IngestMode)
	profile.PayloadFormat = normalizePayloadFormat(profile.PayloadFormat)
	profile.AuthMode = normalizeAuthMode(profile.AuthMode)
	profile.SensorTemplate = strings.TrimSpace(strings.ToLower(profile.SensorTemplate))
	profile.Topic = strings.TrimSpace(profile.Topic)
	profile.Notes = strings.TrimSpace(profile.Notes)
	profile.Metadata = cloneStringMap(profile.Metadata)

	if profile.Protocol == "" {
		switch profile.IngestMode {
		case "http_push":
			profile.Protocol = "http_json"
		case "bridge_http":
			profile.Protocol = "mqtt_json"
		default:
			profile.Protocol = "tcp_json"
		}
	}
	if profile.Transport == "" {
		switch profile.Protocol {
		case "http_json":
			profile.Transport = "http"
		case "mqtt_json":
			profile.Transport = "mqtt"
		case "modbus_tcp":
			profile.Transport = "ethernet"
		case "modbus_rtu":
			profile.Transport = "rs485"
		case "opcua_json":
			profile.Transport = "opcua"
		case "bacnet_ip":
			profile.Transport = "bacnet"
		case "lorawan_uplink":
			profile.Transport = "lorawan"
		default:
			profile.Transport = "tcp"
		}
	}
	if profile.IngestMode == "" {
		switch profile.Protocol {
		case "tcp_json":
			profile.IngestMode = "gateway_tcp"
		case "http_json", "modbus_tcp", "modbus_rtu":
			profile.IngestMode = "http_push"
		default:
			profile.IngestMode = "bridge_http"
		}
	}
	if profile.PayloadFormat == "" {
		switch profile.Protocol {
		case "modbus_tcp", "modbus_rtu":
			profile.PayloadFormat = "register_map"
		default:
			profile.PayloadFormat = "json_values"
		}
	}
	if profile.AuthMode == "" {
		profile.AuthMode = "token"
	}

	if profile.Protocol == "" || profile.Transport == "" || profile.IngestMode == "" || profile.PayloadFormat == "" || profile.AuthMode == "" {
		return model.ProductAccessProfile{}, fmt.Errorf("%w: invalid access profile", ErrInvalidProduct)
	}

	if len(profile.PointMappings) > 0 {
		normalizedMappings := make([]model.ProtocolPointMapping, 0, len(profile.PointMappings))
		for _, item := range profile.PointMappings {
			item.Source = strings.TrimSpace(item.Source)
			item.Property = strings.TrimSpace(item.Property)
			item.Unit = strings.TrimSpace(item.Unit)
			if item.Source == "" || item.Property == "" {
				return model.ProductAccessProfile{}, fmt.Errorf("%w: point mapping source and property are required", ErrInvalidProduct)
			}
			if item.Scale == 0 {
				item.Scale = 1
			}
			normalizedMappings = append(normalizedMappings, item)
		}
		profile.PointMappings = normalizedMappings
	}

	return profile, nil
}

func normalizeTransport(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "tcp", "socket":
		return "tcp"
	case "http", "https":
		return "http"
	case "mqtt":
		return "mqtt"
	case "rs485", "serial":
		return "rs485"
	case "ethernet":
		return "ethernet"
	case "opcua", "opc_ua":
		return "opcua"
	case "bacnet", "bacnet_ip":
		return "bacnet"
	case "lorawan", "lora":
		return "lorawan"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeProtocol(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "default":
		return ""
	case "tcp", "tcp_json", "json_lines":
		return "tcp_json"
	case "http", "http_json":
		return "http_json"
	case "mqtt", "mqtt_json":
		return "mqtt_json"
	case "modbus", "modbus_tcp":
		return "modbus_tcp"
	case "modbus_rtu":
		return "modbus_rtu"
	case "opcua", "opcua_json":
		return "opcua_json"
	case "bacnet", "bacnet_ip":
		return "bacnet_ip"
	case "lorawan", "lorawan_uplink":
		return "lorawan_uplink"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeIngestMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "default":
		return ""
	case "gateway", "gateway_tcp", "direct":
		return "gateway_tcp"
	case "http", "http_push":
		return "http_push"
	case "bridge", "bridge_http", "webhook":
		return "bridge_http"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizePayloadFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "default":
		return ""
	case "json", "json_values":
		return "json_values"
	case "flat", "flat_json":
		return "flat_json"
	case "registers", "register_map":
		return "register_map"
	case "mapped", "mapped_json":
		return "mapped_json"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeAuthMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "default":
		return ""
	case "token", "device_token":
		return "token"
	case "bearer", "bearer_token":
		return "bearer_token"
	case "bridge_secret", "secret":
		return "bridge_secret"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func validateAccessMappings(thingModel model.ThingModel, accessProfile model.ProductAccessProfile) error {
	if len(accessProfile.PointMappings) == 0 || len(thingModel.Properties) == 0 {
		return nil
	}

	properties := make(map[string]struct{}, len(thingModel.Properties))
	for _, property := range thingModel.Properties {
		properties[property.Identifier] = struct{}{}
	}
	for _, item := range accessProfile.PointMappings {
		if _, exists := properties[item.Property]; !exists {
			return fmt.Errorf("%w: point mapping property %q not found in thing model", ErrInvalidProduct, item.Property)
		}
	}
	return nil
}

func normalizeRuleCondition(product model.Product, hasProduct bool, condition model.RuleCondition) (model.RuleCondition, error) {
	condition.Property = strings.TrimSpace(condition.Property)
	condition.Operator = normalizeRuleOperator(condition.Operator)
	if condition.Property == "" {
		return model.RuleCondition{}, fmt.Errorf("%w: condition.property is required", ErrInvalidRule)
	}
	if condition.Operator == "" {
		return model.RuleCondition{}, fmt.Errorf("%w: unsupported condition.operator", ErrInvalidRule)
	}

	if hasProduct {
		var property *model.ThingModelProperty
		for i := range product.ThingModel.Properties {
			if product.ThingModel.Properties[i].Identifier == condition.Property {
				property = &product.ThingModel.Properties[i]
				break
			}
		}
		if property == nil {
			return model.RuleCondition{}, fmt.Errorf("%w: property %q not found in product thing model", ErrInvalidRule, condition.Property)
		}
		if !matchesThingDataType(property.DataType, condition.Value) {
			return model.RuleCondition{}, fmt.Errorf("%w: rule threshold for %s expects %s", ErrInvalidRule, condition.Property, property.DataType)
		}
		if isNumericRuleOperator(condition.Operator) && !isNumber(condition.Value) {
			return model.RuleCondition{}, fmt.Errorf("%w: numeric operator %s requires a numeric threshold", ErrInvalidRule, condition.Operator)
		}
	}

	return condition, nil
}

func normalizeRuleOperator(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case ">", "gt":
		return "gt"
	case ">=", "gte":
		return "gte"
	case "<", "lt":
		return "lt"
	case "<=", "lte":
		return "lte"
	case "=", "==", "eq":
		return "eq"
	case "!=", "<>", "neq":
		return "neq"
	default:
		return ""
	}
}

func isNumericRuleOperator(value string) bool {
	switch value {
	case "gt", "gte", "lt", "lte":
		return true
	default:
		return false
	}
}

func isSupportedDataType(value string) bool {
	switch value {
	case "int", "float", "double", "bool", "text", "enum", "object":
		return true
	default:
		return false
	}
}

func validateThingValues(product model.Product, values map[string]any) error {
	if len(values) == 0 {
		return nil
	}
	if len(product.ThingModel.Properties) == 0 {
		return nil
	}

	properties := make(map[string]model.ThingModelProperty, len(product.ThingModel.Properties))
	for _, property := range product.ThingModel.Properties {
		properties[property.Identifier] = property
	}

	for key, value := range values {
		property, exists := properties[key]
		if !exists {
			return fmt.Errorf("%w: %s", ErrUnknownThingProperty, key)
		}
		if !matchesThingDataType(property.DataType, value) {
			return fmt.Errorf("%w: property %s expects %s", ErrInvalidThingModel, key, property.DataType)
		}
	}
	return nil
}

func validateCommandName(product model.Product, name string) error {
	if len(product.ThingModel.Services) == 0 {
		return nil
	}

	for _, service := range product.ThingModel.Services {
		if service.Identifier == name {
			return nil
		}
	}

	return fmt.Errorf("%w: command %q not found in thing model services", ErrInvalidThingModel, name)
}

func matchesThingDataType(dataType string, value any) bool {
	switch dataType {
	case "bool":
		_, ok := value.(bool)
		return ok
	case "text":
		_, ok := value.(string)
		return ok
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "enum":
		switch value.(type) {
		case string, float64, int, int32, int64:
			return true
		default:
			return false
		}
	case "float", "double":
		return isNumber(value)
	case "int":
		return isIntegralNumber(value)
	default:
		return true
	}
}

func matchesRuleCondition(operator string, left, right any) bool {
	switch operator {
	case "eq":
		return valuesEqual(left, right)
	case "neq":
		return !valuesEqual(left, right)
	case "gt", "gte", "lt", "lte":
		leftNumber, leftOK := toFloat64(left)
		rightNumber, rightOK := toFloat64(right)
		if !leftOK || !rightOK {
			return false
		}
		switch operator {
		case "gt":
			return leftNumber > rightNumber
		case "gte":
			return leftNumber >= rightNumber
		case "lt":
			return leftNumber < rightNumber
		case "lte":
			return leftNumber <= rightNumber
		}
	}
	return false
}

func isNumber(value any) bool {
	switch value.(type) {
	case float64, float32, int, int32, int64, uint, uint32, uint64:
		return true
	default:
		return false
	}
}

func isIntegralNumber(value any) bool {
	switch number := value.(type) {
	case int, int32, int64, uint, uint32, uint64:
		return true
	case float64:
		return math.Mod(number, 1) == 0
	case float32:
		return math.Mod(float64(number), 1) == 0
	default:
		return false
	}
}

func toFloat64(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int32:
		return float64(number), true
	case int64:
		return float64(number), true
	case uint:
		return float64(number), true
	case uint32:
		return float64(number), true
	case uint64:
		return float64(number), true
	default:
		return 0, false
	}
}

func valuesEqual(left, right any) bool {
	switch leftValue := left.(type) {
	case float64:
		rightValue, ok := toFloat64(right)
		return ok && leftValue == rightValue
	case float32:
		rightValue, ok := toFloat64(right)
		return ok && float64(leftValue) == rightValue
	case bool:
		rightValue, ok := right.(bool)
		return ok && leftValue == rightValue
	case string:
		rightValue, ok := right.(string)
		return ok && leftValue == rightValue
	default:
		return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right)
	}
}

func makeRuleStateKey(ruleID, deviceID string) string {
	return ruleID + "\x00" + deviceID
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
