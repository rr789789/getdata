package simulator

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"mvp-platform/internal/config"
	"mvp-platform/internal/core"
	"mvp-platform/internal/model"
	"mvp-platform/internal/util"
)

var (
	ErrSimulatorNotFound     = errors.New("simulator not found")
	ErrSimulatorNotConnected = errors.New("simulator not connected")
	ErrSimulatorBusy         = errors.New("simulator telemetry queue is full")
)

const (
	defaultTelemetryInterval = 5 * time.Second
	maxLogEntries            = 80
)

type CreateRequest struct {
	Name                string
	Metadata            map[string]string
	AutoConnect         bool
	AutoAck             bool
	AutoPing            bool
	AutoTelemetry       bool
	TelemetryIntervalMS int
	DefaultValues       map[string]any
}

type Manager struct {
	cfg     config.Config
	service *core.Service
	logger  *slog.Logger

	rootCtx context.Context
	cancel  context.CancelFunc

	mu         sync.RWMutex
	simulators map[string]*instance
}

type instance struct {
	id              string
	device          model.Device
	gatewayDialAddr string
	logger          *slog.Logger
	parentCtx       context.Context

	mu                  sync.RWMutex
	connected           bool
	autoAck             bool
	autoPing            bool
	autoTelemetry       bool
	telemetryInterval   time.Duration
	defaultValues       map[string]any
	lastConnectAt       time.Time
	lastDisconnectAt    time.Time
	lastPingAt          time.Time
	lastTelemetryAt     time.Time
	lastCommandAt       time.Time
	lastError           string
	logs                []model.SimulatorLogEntry
	cancel              context.CancelFunc
	manualTelemetryChan chan map[string]any
}

func NewManager(cfg config.Config, service *core.Service, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}

	rootCtx, cancel := context.WithCancel(context.Background())
	return &Manager{
		cfg:        cfg,
		service:    service,
		logger:     logger,
		rootCtx:    rootCtx,
		cancel:     cancel,
		simulators: make(map[string]*instance),
	}
}

func (m *Manager) Create(ctx context.Context, req CreateRequest) (model.SimulatorView, error) {
	deviceName := strings.TrimSpace(req.Name)
	if deviceName == "" {
		deviceName = "simulated-device"
	}

	device, err := m.service.CreateDevice(ctx, deviceName, req.Metadata)
	if err != nil {
		return model.SimulatorView{}, err
	}

	sim := &instance{
		id:                util.NewID("sim"),
		device:            device,
		gatewayDialAddr:   m.cfg.GatewayDialAddr,
		logger:            m.logger.With("device_id", device.ID),
		parentCtx:         m.rootCtx,
		autoAck:           req.AutoAck,
		autoPing:          req.AutoPing,
		autoTelemetry:     req.AutoTelemetry,
		telemetryInterval: sanitizeInterval(req.TelemetryIntervalMS),
		defaultValues:     normalizeDefaultValues(req.DefaultValues),
		logs:              make([]model.SimulatorLogEntry, 0, 16),
		manualTelemetryChan: make(chan map[string]any, 8),
	}

	sim.logger = m.logger.With("simulator_id", sim.id, "device_id", device.ID)
	sim.addLog("info", "simulator created")

	m.mu.Lock()
	m.simulators[sim.id] = sim
	m.mu.Unlock()

	if req.AutoConnect {
		if err := sim.connect(); err != nil {
			return sim.snapshot(), err
		}
	}

	return sim.snapshot(), nil
}

func (m *Manager) List() []model.SimulatorView {
	m.mu.RLock()
	items := make([]*instance, 0, len(m.simulators))
	for _, sim := range m.simulators {
		items = append(items, sim)
	}
	m.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].device.CreatedAt.After(items[j].device.CreatedAt)
	})

	result := make([]model.SimulatorView, 0, len(items))
	for _, sim := range items {
		result = append(result, sim.snapshot())
	}
	return result
}

func (m *Manager) Get(id string) (model.SimulatorView, error) {
	sim, err := m.lookup(id)
	if err != nil {
		return model.SimulatorView{}, err
	}
	return sim.snapshot(), nil
}

func (m *Manager) Connect(id string) (model.SimulatorView, error) {
	sim, err := m.lookup(id)
	if err != nil {
		return model.SimulatorView{}, err
	}
	if err := sim.connect(); err != nil {
		return sim.snapshot(), err
	}
	return sim.snapshot(), nil
}

func (m *Manager) Disconnect(id string) (model.SimulatorView, error) {
	sim, err := m.lookup(id)
	if err != nil {
		return model.SimulatorView{}, err
	}
	sim.disconnect()
	return sim.snapshot(), nil
}

func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	sim, exists := m.simulators[id]
	if exists {
		delete(m.simulators, id)
	}
	m.mu.Unlock()

	if !exists {
		return ErrSimulatorNotFound
	}

	sim.disconnect()
	return nil
}

func (m *Manager) SendTelemetry(id string, values map[string]any) (model.SimulatorView, error) {
	sim, err := m.lookup(id)
	if err != nil {
		return model.SimulatorView{}, err
	}
	if err := sim.sendTelemetry(values); err != nil {
		return sim.snapshot(), err
	}
	return sim.snapshot(), nil
}

func (m *Manager) Close() {
	m.cancel()

	m.mu.RLock()
	items := make([]*instance, 0, len(m.simulators))
	for _, sim := range m.simulators {
		items = append(items, sim)
	}
	m.mu.RUnlock()

	for _, sim := range items {
		sim.disconnect()
	}
}

func (m *Manager) lookup(id string) (*instance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sim, exists := m.simulators[strings.TrimSpace(id)]
	if !exists {
		return nil, ErrSimulatorNotFound
	}
	return sim, nil
}

func (s *instance) connect() error {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return nil
	}

	ctx, cancel := context.WithCancel(s.parentCtx)
	s.cancel = cancel
	s.lastError = ""
	s.mu.Unlock()

	s.addLog("info", "connect requested")
	go s.run(ctx)
	return nil
}

func (s *instance) disconnect() {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()

	if cancel != nil {
		s.addLog("info", "disconnect requested")
		cancel()
	}
}

func (s *instance) sendTelemetry(values map[string]any) error {
	s.mu.RLock()
	connected := s.connected
	s.mu.RUnlock()
	if !connected {
		return ErrSimulatorNotConnected
	}

	payload := normalizeDefaultValues(values)
	select {
	case s.manualTelemetryChan <- payload:
		return nil
	default:
		return ErrSimulatorBusy
	}
}

func (s *instance) run(ctx context.Context) {
	defer s.finishRun()

	conn, err := net.DialTimeout("tcp", s.gatewayDialAddr, 5*time.Second)
	if err != nil {
		s.fail(err, "dial failed")
		return
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	encoder.SetEscapeHTML(false)

	serverMessages := make(chan model.ServerMessage, 16)
	readErrs := make(chan error, 1)
	go scanServerMessages(conn, serverMessages, readErrs)

	if err := writeDeviceMessage(conn, encoder, model.DeviceMessage{
		Type:     "auth",
		DeviceID: s.device.ID,
		Token:    s.device.Token,
	}); err != nil {
		s.fail(err, "auth write failed")
		return
	}

	authDeadline := time.NewTimer(8 * time.Second)
	defer authDeadline.Stop()

	authenticated := false
	for !authenticated {
		select {
		case <-ctx.Done():
			return
		case err := <-readErrs:
			s.fail(err, "auth read failed")
			return
		case msg := <-serverMessages:
			switch msg.Type {
			case "auth_ok":
				authenticated = true
				s.setConnected(true)
				s.addLog("info", "device authenticated to gateway")
			case "error":
				s.fail(errors.New(msg.Error), "gateway rejected auth")
				return
			default:
				s.addLog("warn", fmt.Sprintf("unexpected auth response: %s", msg.Type))
			}
		case <-authDeadline.C:
			s.fail(errors.New("auth timeout"), "gateway auth timeout")
			return
		}
	}

	var pingTicker *time.Ticker
	var pingCh <-chan time.Time
	if s.autoPing {
		pingTicker = time.NewTicker(25 * time.Second)
		pingCh = pingTicker.C
		defer pingTicker.Stop()
	}

	var telemetryTicker *time.Ticker
	var telemetryCh <-chan time.Time
	if s.autoTelemetry {
		telemetryTicker = time.NewTicker(s.telemetryInterval)
		telemetryCh = telemetryTicker.C
		defer telemetryTicker.Stop()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-readErrs:
			s.fail(err, "gateway read failed")
			return
		case msg := <-serverMessages:
			if err := s.handleServerMessage(conn, encoder, msg); err != nil {
				s.fail(err, "server message handling failed")
				return
			}
		case <-pingCh:
			if err := writeDeviceMessage(conn, encoder, model.DeviceMessage{Type: "ping"}); err != nil {
				s.fail(err, "ping failed")
				return
			}
			s.notePing()
		case <-telemetryCh:
			if err := s.publishTelemetry(conn, encoder, s.snapshotDefaultValues()); err != nil {
				s.fail(err, "auto telemetry failed")
				return
			}
		case values := <-s.manualTelemetryChan:
			if err := s.publishTelemetry(conn, encoder, values); err != nil {
				s.fail(err, "manual telemetry failed")
				return
			}
		}
	}
}

func (s *instance) handleServerMessage(conn net.Conn, encoder *json.Encoder, msg model.ServerMessage) error {
	switch msg.Type {
	case "pong":
		s.addLog("info", "received pong from gateway")
		return nil
	case "command":
		s.noteCommand()
		s.addLog("info", fmt.Sprintf("received command %s (%s)", msg.Name, msg.CommandID))
		if s.autoAck {
			return writeDeviceMessage(conn, encoder, model.DeviceMessage{
				Type:      "ack",
				CommandID: msg.CommandID,
				Status:    "ok",
				Message:   "accepted by simulator",
			})
		}
		return nil
	case "error":
		s.addLog("warn", fmt.Sprintf("gateway error: %s", msg.Error))
		return nil
	default:
		s.addLog("info", fmt.Sprintf("received message type %s", msg.Type))
		return nil
	}
}

func (s *instance) publishTelemetry(conn net.Conn, encoder *json.Encoder, values map[string]any) error {
	if len(values) == 0 {
		values = map[string]any{
			"temperature": 23.5,
			"humidity":    60,
		}
	}

	if err := writeDeviceMessage(conn, encoder, model.DeviceMessage{
		Type:   "telemetry",
		TS:     time.Now().UTC().UnixMilli(),
		Values: values,
	}); err != nil {
		return err
	}

	s.noteTelemetry(values)
	return nil
}

func (s *instance) finishRun() {
	now := time.Now().UTC()

	s.mu.Lock()
	if s.connected {
		s.connected = false
		s.lastDisconnectAt = now
	}
	s.cancel = nil
	s.mu.Unlock()
}

func (s *instance) fail(err error, message string) {
	if err == nil {
		return
	}

	s.mu.Lock()
	s.connected = false
	s.lastDisconnectAt = time.Now().UTC()
	s.lastError = err.Error()
	s.mu.Unlock()

	s.addLog("error", fmt.Sprintf("%s: %v", message, err))
}

func (s *instance) setConnected(value bool) {
	now := time.Now().UTC()

	s.mu.Lock()
	s.connected = value
	if value {
		s.lastConnectAt = now
		s.lastError = ""
	} else {
		s.lastDisconnectAt = now
	}
	s.mu.Unlock()
}

func (s *instance) notePing() {
	now := time.Now().UTC()

	s.mu.Lock()
	s.lastPingAt = now
	s.mu.Unlock()
	s.addLog("info", "sent ping")
}

func (s *instance) noteTelemetry(values map[string]any) {
	now := time.Now().UTC()

	s.mu.Lock()
	s.lastTelemetryAt = now
	s.mu.Unlock()
	s.addLog("info", fmt.Sprintf("published telemetry %s", compactJSON(values)))
}

func (s *instance) noteCommand() {
	s.mu.Lock()
	s.lastCommandAt = time.Now().UTC()
	s.mu.Unlock()
}

func (s *instance) addLog(level, message string) {
	entry := model.SimulatorLogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   message,
	}

	s.mu.Lock()
	s.logs = append(s.logs, entry)
	if len(s.logs) > maxLogEntries {
		s.logs = append([]model.SimulatorLogEntry(nil), s.logs[len(s.logs)-maxLogEntries:]...)
	}
	s.mu.Unlock()

	s.logger.Info(message, "level", level)
}

func (s *instance) snapshot() model.SimulatorView {
	s.mu.RLock()
	defer s.mu.RUnlock()

	view := model.SimulatorView{
		ID:                  s.id,
		Device:              cloneDevice(s.device),
		Connected:           s.connected,
		AutoAck:             s.autoAck,
		AutoPing:            s.autoPing,
		AutoTelemetry:       s.autoTelemetry,
		TelemetryIntervalMS: int(s.telemetryInterval / time.Millisecond),
		DefaultValues:       cloneAnyMap(s.defaultValues),
		LastError:           s.lastError,
		Logs:                append([]model.SimulatorLogEntry(nil), s.logs...),
	}
	if !s.lastConnectAt.IsZero() {
		value := s.lastConnectAt
		view.LastConnectAt = &value
	}
	if !s.lastDisconnectAt.IsZero() {
		value := s.lastDisconnectAt
		view.LastDisconnectAt = &value
	}
	if !s.lastPingAt.IsZero() {
		value := s.lastPingAt
		view.LastPingAt = &value
	}
	if !s.lastTelemetryAt.IsZero() {
		value := s.lastTelemetryAt
		view.LastTelemetryAt = &value
	}
	if !s.lastCommandAt.IsZero() {
		value := s.lastCommandAt
		view.LastCommandAt = &value
	}
	return view
}

func (s *instance) snapshotDefaultValues() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneAnyMap(s.defaultValues)
}

func scanServerMessages(conn net.Conn, messages chan<- model.ServerMessage, errs chan<- error) {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)

	for scanner.Scan() {
		var msg model.ServerMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			select {
			case errs <- err:
			default:
			}
			return
		}

		select {
		case messages <- msg:
		default:
			select {
			case errs <- errors.New("server message queue overflow"):
			default:
			}
			return
		}
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		select {
		case errs <- err:
		default:
		}
		return
	}

	select {
	case errs <- io.EOF:
	default:
	}
}

func writeDeviceMessage(conn net.Conn, encoder *json.Encoder, message model.DeviceMessage) error {
	if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}
	return encoder.Encode(message)
}

func sanitizeInterval(ms int) time.Duration {
	if ms <= 0 {
		return defaultTelemetryInterval
	}

	interval := time.Duration(ms) * time.Millisecond
	if interval < time.Second {
		return time.Second
	}
	return interval
}

func normalizeDefaultValues(values map[string]any) map[string]any {
	if len(values) == 0 {
		return map[string]any{
			"temperature": 23.5,
			"humidity":    60,
			"voltage":     3.3,
		}
	}
	return cloneAnyMap(values)
}

func compactJSON(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(payload)
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
