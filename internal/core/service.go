package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mvp-platform/internal/model"
	"mvp-platform/internal/store"
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

type sessionTransport interface {
	Transport() string
}

type Service struct {
	tenants   store.TenantStore
	products  store.ProductStore
	devices   store.DeviceStore
	groups    store.GroupStore
	rules     store.RuleStore
	configs   store.ConfigStore
	firmware  store.FirmwareStore
	campaigns store.OTACampaignStore
	telemetry store.TelemetryStore
	shadows   store.ShadowStore
	commands  store.CommandStore
	alerts    store.AlertStore
	inspector store.Inspector
	logger    *slog.Logger
	startedAt time.Time

	mu         sync.RWMutex
	states     map[string]deviceState
	ruleStates map[string]ruleRuntimeState

	registeredDevices        atomic.Int64
	onlineDevices            atomic.Int64
	totalConnections         atomic.Int64
	rejectedConnections      atomic.Int64
	telemetryReceived        atomic.Int64
	commandsSent             atomic.Int64
	commandAcks              atomic.Int64
	httpRequests             atomic.Int64
	httpErrors               atomic.Int64
	httpIngestAccepted       atomic.Int64
	httpIngestRejected       atomic.Int64
	tcpTelemetryAccepted     atomic.Int64
	tcpCommandAcks           atomic.Int64
	tcpCommandsPublished     atomic.Int64
	mqttMessagesReceived     atomic.Int64
	mqttTelemetryAccepted    atomic.Int64
	mqttCommandAcks          atomic.Int64
	mqttCommandsPublished    atomic.Int64
	mqttConnectionsAccepted atomic.Int64
	mqttConnectionsRejected atomic.Int64
	bytesIngested            atomic.Int64
	telemetryValues          atomic.Int64
}

type deviceState struct {
	session     Session
	sessionID   string
	transport   string
	connectedAt time.Time
	lastSeen    time.Time
}

type ruleRuntimeState struct {
	TriggeredCount  int64
	LastTriggeredAt time.Time
}

func NewService(
	tenants store.TenantStore,
	products store.ProductStore,
	devices store.DeviceStore,
	groups store.GroupStore,
	rules store.RuleStore,
	configs store.ConfigStore,
	firmware store.FirmwareStore,
	campaigns store.OTACampaignStore,
	telemetry store.TelemetryStore,
	shadows store.ShadowStore,
	commands store.CommandStore,
	alerts store.AlertStore,
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	var inspector store.Inspector
	if candidate, ok := devices.(store.Inspector); ok {
		inspector = candidate
	}

	startedAt := time.Now().UTC()

	service := &Service{
		tenants:    tenants,
		products:   products,
		devices:    devices,
		groups:     groups,
		rules:      rules,
		configs:    configs,
		firmware:   firmware,
		campaigns:  campaigns,
		telemetry:  telemetry,
		shadows:    shadows,
		commands:   commands,
		alerts:     alerts,
		inspector:  inspector,
		logger:     logger,
		startedAt:  startedAt,
		states:     make(map[string]deviceState),
		ruleStates: make(map[string]ruleRuntimeState),
	}
	if inspector != nil {
		if stats, err := inspector.StorageStats(context.Background()); err == nil {
			service.registeredDevices.Store(stats.Devices)
		}
	}
	return service
}

func (s *Service) CreateProduct(ctx context.Context, name, description string, metadata map[string]string, accessProfile model.ProductAccessProfile, thingModel model.ThingModel) (model.Product, error) {
	return s.CreateProductWithTenant(ctx, "", name, description, metadata, accessProfile, thingModel)
}

func (s *Service) GetProduct(ctx context.Context, productID string) (model.ProductView, error) {
	product, err := s.products.GetProduct(ctx, strings.TrimSpace(productID))
	if err != nil {
		return model.ProductView{}, err
	}
	return s.buildProductView(ctx, product)
}

func (s *Service) ListProducts(ctx context.Context) ([]model.ProductView, error) {
	return s.ListProductsByTenant(ctx, "")
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
	return s.CreateGroupWithTenant(ctx, "", name, description, productID, tags)
}

func (s *Service) GetGroup(ctx context.Context, groupID string) (model.GroupView, error) {
	group, err := s.groups.GetGroup(ctx, strings.TrimSpace(groupID))
	if err != nil {
		return model.GroupView{}, err
	}
	return s.buildGroupView(ctx, group)
}

func (s *Service) ListGroups(ctx context.Context) ([]model.GroupView, error) {
	return s.ListGroupsByTenant(ctx, "")
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
	if group.TenantID != "" && device.TenantID != group.TenantID {
		return model.GroupView{}, fmt.Errorf("%w: group tenant scope mismatch", ErrInvalidGroup)
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
	return s.CreateDeviceWithTenant(ctx, "", name, productID, tags, metadata)
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
	return s.CreateConfigProfileWithTenant(ctx, "", name, description, productID, values)
}

func (s *Service) ListConfigProfiles(ctx context.Context) ([]model.ConfigProfileView, error) {
	return s.ListConfigProfilesByTenant(ctx, "")
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
	if profile.TenantID != "" && device.TenantID != profile.TenantID {
		return model.DeviceShadow{}, fmt.Errorf("%w: config profile tenant scope mismatch", ErrInvalidConfig)
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
	return s.ListDevicesByTenant(ctx, "", productID)
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
	return s.CreateRuleWithTenant(ctx, "", name, description, productID, groupID, deviceID, enabled, severity, cooldownSeconds, condition, nil)
}

func (s *Service) ListRules(ctx context.Context) ([]model.RuleView, error) {
	return s.ListRulesByTenant(ctx, "")
}

func (s *Service) ListAlerts(ctx context.Context, limit int, productID, groupID, deviceID, ruleID string) ([]model.AlertEvent, error) {
	return s.ListAlertsByTenant(ctx, "", limit, productID, groupID, deviceID, ruleID)
}

func (s *Service) ListAlertsByTenant(ctx context.Context, tenantID string, limit int, productID, groupID, deviceID, ruleID string) ([]model.AlertEvent, error) {
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
		if tenantID != "" && alert.TenantID != tenantID {
			continue
		}
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
	transport := sessionTransportName(session)

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
		transport:   transport,
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
		state.transport = ""
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
	return s.sendCommandWithCampaign(ctx, deviceID, "", name, params)
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
	if command.CampaignID != "" {
		if err := s.updateCampaignFromCommandAck(ctx, command.CampaignID, nextStatus); err != nil {
			s.logger.Warn("unable to update ota campaign status from command ack", "campaign_id", command.CampaignID, "command_id", commandID, "error", err)
		}
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

func (s *Service) RecordHTTPRequest(status int) {
	s.httpRequests.Add(1)
	if status >= 400 {
		s.httpErrors.Add(1)
	}
}

func (s *Service) RecordHTTPIngestResult(accepted bool, payloadBytes, valueCount int) {
	if accepted {
		s.httpIngestAccepted.Add(1)
	} else {
		s.httpIngestRejected.Add(1)
	}
	s.recordIngressVolume(payloadBytes, valueCount)
}

func (s *Service) RecordTCPTelemetryAccepted(payloadBytes, valueCount int) {
	s.tcpTelemetryAccepted.Add(1)
	s.recordIngressVolume(payloadBytes, valueCount)
}

func (s *Service) RecordCommandAckTransport(transport string) {
	switch strings.ToLower(strings.TrimSpace(transport)) {
	case "mqtt":
		s.mqttCommandAcks.Add(1)
	default:
		s.tcpCommandAcks.Add(1)
	}
}

func (s *Service) RecordMQTTConnectionAccepted() {
	s.mqttConnectionsAccepted.Add(1)
}

func (s *Service) RecordMQTTConnectionRejected() {
	s.mqttConnectionsRejected.Add(1)
}

func (s *Service) RecordMQTTMessageReceived(payloadBytes, valueCount int, telemetry bool) {
	s.mqttMessagesReceived.Add(1)
	if telemetry {
		s.mqttTelemetryAccepted.Add(1)
	}
	s.recordIngressVolume(payloadBytes, valueCount)
}

func (s *Service) recordIngressVolume(payloadBytes, valueCount int) {
	if payloadBytes > 0 {
		s.bytesIngested.Add(int64(payloadBytes))
	}
	if valueCount > 0 {
		s.telemetryValues.Add(int64(valueCount))
	}
}

func (s *Service) recordCommandPublished(transport string) {
	switch strings.ToLower(strings.TrimSpace(transport)) {
	case "mqtt":
		s.mqttCommandsPublished.Add(1)
	default:
		s.tcpCommandsPublished.Add(1)
	}
}

func sessionTransportName(session Session) string {
	if transportSession, ok := session.(sessionTransport); ok {
		value := strings.ToLower(strings.TrimSpace(transportSession.Transport()))
		if value != "" {
			return value
		}
	}
	return "tcp"
}

func (s *Service) Stats() model.Stats {
	storageStats := model.StorageStats{Backend: "memory"}
	if s.inspector != nil {
		if stats, err := s.inspector.StorageStats(context.Background()); err == nil {
			storageStats = stats
		}
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	var tcpOnlineDevices int64
	var mqttOnlineDevices int64
	s.mu.RLock()
	for _, state := range s.states {
		if state.session == nil {
			continue
		}
		switch state.transport {
		case "mqtt":
			mqttOnlineDevices++
		default:
			tcpOnlineDevices++
		}
	}
	s.mu.RUnlock()

	return model.Stats{
		RegisteredDevices:   s.registeredDevices.Load(),
		OnlineDevices:       s.onlineDevices.Load(),
		TotalConnections:    s.totalConnections.Load(),
		RejectedConnections: s.rejectedConnections.Load(),
		TelemetryReceived:   s.telemetryReceived.Load(),
		CommandsSent:        s.commandsSent.Load(),
		CommandAcks:         s.commandAcks.Load(),
		StartedAt:           s.startedAt,
		UptimeSeconds:       int64(time.Since(s.startedAt).Seconds()),
		Runtime: model.RuntimeStats{
			Goroutines:      runtime.NumGoroutine(),
			HeapAllocBytes:  memStats.HeapAlloc,
			HeapInuseBytes:  memStats.HeapInuse,
			StackInuseBytes: memStats.StackInuse,
			SysBytes:        memStats.Sys,
			NumGC:           memStats.NumGC,
		},
		Storage:             storageStats,
		Ingress: model.IngressStats{
			HTTPRequests:          s.httpRequests.Load(),
			HTTPErrors:            s.httpErrors.Load(),
			HTTPIngestAccepted:    s.httpIngestAccepted.Load(),
			HTTPIngestRejected:    s.httpIngestRejected.Load(),
			TCPTelemetryAccepted:  s.tcpTelemetryAccepted.Load(),
			TCPCommandAcks:        s.tcpCommandAcks.Load(),
			MQTTMessagesReceived:  s.mqttMessagesReceived.Load(),
			MQTTTelemetryAccepted: s.mqttTelemetryAccepted.Load(),
			MQTTCommandAcks:       s.mqttCommandAcks.Load(),
			BytesIngested:         s.bytesIngested.Load(),
			TelemetryValues:       s.telemetryValues.Load(),
		},
		Transport: model.TransportStats{
			TCPOnlineDevices:        tcpOnlineDevices,
			MQTTOnlineDevices:       mqttOnlineDevices,
			TCPCommandsPublished:    s.tcpCommandsPublished.Load(),
			MQTTCommandsPublished:   s.mqttCommandsPublished.Load(),
			MQTTConnectionsAccepted: s.mqttConnectionsAccepted.Load(),
			MQTTConnectionsRejected: s.mqttConnectionsRejected.Load(),
		},
	}
}

func (s *Service) buildProductView(ctx context.Context, product model.Product) (model.ProductView, error) {
	view := model.ProductView{Product: product}
	tenant, err := s.tenantSummary(ctx, product.TenantID)
	if err != nil {
		return model.ProductView{}, err
	}
	view.Tenant = tenant

	devices, err := s.devices.ListDevices(ctx)
	if err != nil {
		return model.ProductView{}, err
	}

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
	tenant, err := s.tenantSummary(ctx, group.TenantID)
	if err != nil {
		return model.GroupView{}, err
	}
	view.Tenant = tenant
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
	tenant, err := s.tenantSummary(ctx, rule.TenantID)
	if err != nil {
		return model.RuleView{}, err
	}
	view.Tenant = tenant

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
	tenant, err := s.tenantSummary(ctx, profile.TenantID)
	if err != nil {
		return model.ConfigProfileView{}, err
	}
	view.Tenant = tenant
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
	tenant, err := s.tenantSummary(ctx, device.TenantID)
	if err != nil {
		return model.DeviceView{}, err
	}
	view.Tenant = tenant
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
		if rule.TenantID != "" && rule.TenantID != device.TenantID {
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

		s.executeRuleActions(ctx, rule, device, telemetry, value)
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
		switch {
		case profile.IngestMode == "http_push":
			profile.Protocol = "http_json"
		case profile.IngestMode == "bridge_http", profile.IngestMode == "broker_mqtt", profile.Transport == "mqtt":
			profile.Protocol = "mqtt_json"
		case profile.Transport == "http":
			profile.Protocol = "http_json"
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
		case "mqtt_json":
			profile.IngestMode = "broker_mqtt"
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
	case "mqtt", "mqtt_broker", "broker_mqtt":
		return "broker_mqtt"
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
