package mqtt

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	mqttserver "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"

	"mvp-platform/internal/config"
	"mvp-platform/internal/core"
	"mvp-platform/internal/ingest"
	"mvp-platform/internal/model"
	"mvp-platform/internal/store"
)

type Server struct {
	cfg     config.Config
	service *core.Service
	logger  *slog.Logger
}

type routing struct {
	UpTopic   string
	DownTopic string
	AckTopic  string
}

type clientBinding struct {
	DeviceID      string
	AccessProfile model.ProductAccessProfile
	Routing       routing
}

type authHook struct {
	mqttserver.HookBase

	service *core.Service
	logger  *slog.Logger
	broker  *mqttserver.Server

	mu       sync.RWMutex
	bindings map[string]clientBinding
}

type session struct {
	broker  *mqttserver.Server
	deviceID string
	clientID string
	route   routing
}

func NewServer(cfg config.Config, service *core.Service, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		cfg:     cfg,
		service: service,
		logger:  logger,
	}
}

func (s *Server) Run(ctx context.Context) error {
	if strings.TrimSpace(s.cfg.MQTTAddr) == "" {
		return nil
	}

	broker := mqttserver.New(&mqttserver.Options{InlineClient: true})
	hook := &authHook{
		service:  s.service,
		logger:   s.logger.With("component", "mqtt-auth"),
		broker:   broker,
		bindings: make(map[string]clientBinding),
	}

	if err := broker.AddHook(hook, nil); err != nil {
		return err
	}

	listener := listeners.NewTCP(listeners.Config{ID: "mqtt", Address: s.cfg.MQTTAddr})
	if err := broker.AddListener(listener); err != nil {
		return err
	}

	if err := broker.Subscribe("#", 1, func(cl *mqttserver.Client, _ packets.Subscription, pk packets.Packet) {
		hook.handlePublish(ctx, cl, pk)
	}); err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		_ = broker.Close()
	}()

	s.logger.Info("mqtt broker listening", "addr", s.cfg.MQTTAddr)
	err := broker.Serve()
	if ctx.Err() != nil {
		return nil
	}
	return err
}

func (s *session) SessionID() string {
	return s.clientID
}

func (s *session) Transport() string {
	return "mqtt"
}

func (s *session) Send(message model.ServerMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return s.broker.Publish(s.route.DownTopic, payload, false, 0)
}

func (s *session) Close() error {
	return nil
}

func (h *authHook) ID() string {
	return "mvp-auth"
}

func (h *authHook) Provides(b byte) bool {
	switch b {
	case mqttserver.OnConnectAuthenticate, mqttserver.OnACLCheck, mqttserver.OnSessionEstablished, mqttserver.OnDisconnect:
		return true
	default:
		return false
	}
}

func (h *authHook) OnConnectAuthenticate(cl *mqttserver.Client, pk packets.Packet) bool {
	deviceID := strings.TrimSpace(string(cl.Properties.Username))
	if deviceID == "" {
		deviceID = strings.TrimSpace(cl.ID)
	}
	token := strings.TrimSpace(string(pk.Connect.Password))
	if deviceID == "" || token == "" {
		h.service.RecordConnectionAccepted()
		h.service.RecordConnectionRejected()
		h.service.RecordMQTTConnectionRejected()
		return false
	}

	device, err := h.service.AuthenticateDevice(context.Background(), deviceID, token)
	if err != nil {
		h.service.RecordConnectionAccepted()
		h.service.RecordConnectionRejected()
		h.service.RecordMQTTConnectionRejected()
		h.logger.Warn("mqtt authentication failed", "client_id", cl.ID, "device_id", deviceID, "error", err)
		return false
	}

	var accessProfile model.ProductAccessProfile
	if device.ProductID != "" {
		product, err := h.service.GetProduct(context.Background(), device.ProductID)
		if err != nil && !errors.Is(err, store.ErrProductNotFound) {
			h.service.RecordConnectionAccepted()
			h.service.RecordConnectionRejected()
			h.service.RecordMQTTConnectionRejected()
			h.logger.Warn("mqtt product lookup failed", "device_id", deviceID, "error", err)
			return false
		}
		if err == nil {
			accessProfile = product.Product.AccessProfile
		}
	}

	h.mu.Lock()
	h.bindings[cl.ID] = clientBinding{
		DeviceID:      device.ID,
		AccessProfile: accessProfile,
		Routing:       resolveRouting(accessProfile, device.ID),
	}
	h.mu.Unlock()

	h.service.RecordConnectionAccepted()
	h.service.RecordMQTTConnectionAccepted()
	return true
}

func (h *authHook) OnACLCheck(cl *mqttserver.Client, topic string, write bool) bool {
	binding, ok := h.binding(cl.ID)
	if !ok {
		return false
	}

	if write {
		return topic == binding.Routing.UpTopic || topic == binding.Routing.AckTopic
	}

	return mqttTopicMatch(topic, binding.Routing.DownTopic)
}

func (h *authHook) OnSessionEstablished(cl *mqttserver.Client, _ packets.Packet) {
	binding, ok := h.binding(cl.ID)
	if !ok {
		return
	}

	h.service.RegisterSession(binding.DeviceID, &session{
		broker:   h.broker,
		deviceID: binding.DeviceID,
		clientID: cl.ID,
		route:    binding.Routing,
	})
	h.service.TouchDevice(binding.DeviceID, time.Now().UTC())
}

func (h *authHook) OnDisconnect(cl *mqttserver.Client, err error, expire bool) {
	binding, ok := h.binding(cl.ID)
	if ok {
		h.service.UnregisterSession(binding.DeviceID, cl.ID)
	}

	h.mu.Lock()
	delete(h.bindings, cl.ID)
	h.mu.Unlock()
}

func (h *authHook) binding(clientID string) (clientBinding, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	binding, ok := h.bindings[clientID]
	return binding, ok
}

func (h *authHook) handlePublish(ctx context.Context, cl *mqttserver.Client, pk packets.Packet) {
	if cl == nil {
		return
	}

	binding, ok := h.binding(cl.ID)
	if !ok {
		return
	}

	switch pk.TopicName {
	case binding.Routing.UpTopic:
		h.handleTelemetry(ctx, binding, pk)
	case binding.Routing.AckTopic:
		h.handleAck(ctx, binding, pk)
	}
}

func (h *authHook) handleTelemetry(ctx context.Context, binding clientBinding, pk packets.Packet) {
	var payload map[string]any
	if err := json.Unmarshal(pk.Payload, &payload); err != nil {
		h.logger.Warn("invalid mqtt telemetry payload", "device_id", binding.DeviceID, "error", err)
		return
	}

	values, err := ingest.BuildValues(payload, binding.AccessProfile)
	if err != nil || len(values) == 0 {
		h.logger.Warn("unable to resolve mqtt telemetry values", "device_id", binding.DeviceID, "error", err)
		return
	}

	at := ingest.ExtractTimestamp(payload, time.Now().UTC())
	if err := h.service.HandleTelemetry(ctx, binding.DeviceID, at, values); err != nil {
		h.logger.Warn("mqtt telemetry rejected", "device_id", binding.DeviceID, "error", err)
		return
	}

	h.service.RecordMQTTMessageReceived(len(pk.Payload), len(values), true)
}

func (h *authHook) handleAck(ctx context.Context, binding clientBinding, pk packets.Packet) {
	var payload struct {
		CommandID string `json:"command_id"`
		Status    string `json:"status"`
		Message   string `json:"message"`
	}
	if err := json.Unmarshal(pk.Payload, &payload); err != nil {
		h.logger.Warn("invalid mqtt ack payload", "device_id", binding.DeviceID, "error", err)
		return
	}
	if strings.TrimSpace(payload.CommandID) == "" {
		return
	}

	if err := h.service.HandleCommandAck(ctx, binding.DeviceID, payload.CommandID, payload.Status, payload.Message); err != nil {
		h.logger.Warn("mqtt ack rejected", "device_id", binding.DeviceID, "command_id", payload.CommandID, "error", err)
		return
	}

	h.service.RecordMQTTMessageReceived(len(pk.Payload), 0, false)
	h.service.RecordCommandAckTransport("mqtt")
}

func resolveRouting(profile model.ProductAccessProfile, deviceID string) routing {
	topic := strings.TrimSpace(profile.Topic)
	if topic == "" {
		topic = "devices/{device_id}/up"
	}
	topic = strings.ReplaceAll(topic, "{device_id}", deviceID)
	if strings.ContainsAny(topic, "+#") {
		topic = "devices/" + deviceID + "/up"
	}
	if !strings.HasSuffix(topic, "/up") {
		topic = strings.TrimRight(topic, "/") + "/up"
	}

	base := strings.TrimSuffix(topic, "/up")
	return routing{
		UpTopic:   topic,
		DownTopic: base + "/down",
		AckTopic:  base + "/ack",
	}
}

func mqttTopicMatch(filter, topic string) bool {
	filterParts := strings.Split(strings.Trim(filter, "/"), "/")
	topicParts := strings.Split(strings.Trim(topic, "/"), "/")

	for index := 0; index < len(filterParts); index++ {
		if index >= len(topicParts) {
			return filterParts[index] == "#"
		}

		switch filterParts[index] {
		case "#":
			return true
		case "+":
			continue
		default:
			if filterParts[index] != topicParts[index] {
				return false
			}
		}
	}

	return len(filterParts) == len(topicParts)
}

var _ core.Session = (*session)(nil)
