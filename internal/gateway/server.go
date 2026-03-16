package gateway

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"mvp-platform/internal/config"
	"mvp-platform/internal/core"
	"mvp-platform/internal/model"
	"mvp-platform/internal/util"
)

type Server struct {
	cfg     config.Config
	service *core.Service
	logger  *slog.Logger

	mu          sync.Mutex
	connections map[string]net.Conn
}

func NewServer(cfg config.Config, service *core.Service, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	return &Server{
		cfg:         cfg,
		service:     service,
		logger:      logger,
		connections: make(map[string]net.Conn),
	}
}

func (s *Server) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.cfg.GatewayAddr)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		_ = listener.Close()
		s.closeAllConnections()
	}()

	s.logger.Info("device gateway listening", "addr", s.cfg.GatewayAddr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}

			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Temporary() {
				s.logger.Warn("temporary accept error", "error", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return err
		}

		s.service.RecordConnectionAccepted()

		connID := util.NewID("conn")
		s.trackConnection(connID, conn)
		go s.handleConnection(ctx, connID, conn)
	}
}

func (s *Server) handleConnection(ctx context.Context, connID string, conn net.Conn) {
	defer func() {
		s.untrackConnection(connID)
		_ = conn.Close()
	}()

	logger := s.logger.With("conn_id", connID, "remote_addr", conn.RemoteAddr().String())
	session := newClientSession(connID, conn, s.cfg.DeviceQueueSize, s.cfg.DeviceWriteTimeout, logger)

	writerCtx, cancelWriter := context.WithCancel(ctx)
	defer cancelWriter()
	go session.runWriter(writerCtx)

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 4096), s.cfg.MaxMessageBytes)

	_ = conn.SetReadDeadline(time.Now().Add(s.cfg.DeviceAuthTimeout))

	var authenticated bool
	var deviceID string

	for scanner.Scan() {
		now := time.Now().UTC()
		line := append([]byte(nil), scanner.Bytes()...)

		var message model.DeviceMessage
		if err := json.Unmarshal(line, &message); err != nil {
			logger.Warn("invalid json message", "error", err)
			_ = session.Send(model.ServerMessage{Type: "error", Error: "invalid json"})
			continue
		}

		if !authenticated && message.Type != "auth" {
			s.service.RecordConnectionRejected()
			_ = session.Send(model.ServerMessage{Type: "error", Error: "auth required"})
			return
		}

		switch message.Type {
		case "auth":
			if authenticated {
				_ = session.Send(model.ServerMessage{Type: "error", Error: "already authenticated"})
				return
			}

			device, err := s.service.AuthenticateDevice(ctx, message.DeviceID, message.Token)
			if err != nil {
				s.service.RecordConnectionRejected()
				logger.Warn("authentication failed", "device_id", message.DeviceID, "error", err)
				_ = session.Send(model.ServerMessage{Type: "error", Error: "authentication failed"})
				return
			}

			authenticated = true
			deviceID = device.ID
			s.service.RegisterSession(device.ID, session)
			s.service.TouchDevice(device.ID, now)
			_ = session.Send(model.ServerMessage{
				Type:       "auth_ok",
				ServerTime: now.UnixMilli(),
				DeviceID:   device.ID,
			})
			logger.Info("device authenticated", "device_id", device.ID)
		case "ping":
			s.service.TouchDevice(deviceID, now)
			_ = session.Send(model.ServerMessage{
				Type:       "pong",
				ServerTime: now.UnixMilli(),
			})
		case "telemetry":
			timestamp := now
			if message.TS > 0 {
				timestamp = time.UnixMilli(message.TS).UTC()
			}

			if err := s.service.HandleTelemetry(ctx, deviceID, timestamp, message.Values); err != nil {
				logger.Warn("telemetry rejected", "device_id", deviceID, "error", err)
				_ = session.Send(model.ServerMessage{Type: "error", Error: "telemetry rejected"})
				continue
			}
			s.service.RecordTCPTelemetryAccepted(len(line), len(message.Values))
		case "ack":
			if err := s.service.HandleCommandAck(ctx, deviceID, message.CommandID, message.Status, message.Message); err != nil {
				logger.Warn("ack rejected", "device_id", deviceID, "command_id", message.CommandID, "error", err)
				_ = session.Send(model.ServerMessage{Type: "error", Error: "ack rejected"})
				continue
			}
			s.service.RecordCommandAckTransport("tcp")
		default:
			_ = session.Send(model.ServerMessage{Type: "error", Error: "unknown message type"})
		}

		_ = conn.SetReadDeadline(time.Now().Add(s.cfg.DeviceIdleTimeout))
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		logger.Warn("connection closed with read error", "error", err)
	}
	if !authenticated {
		s.service.RecordConnectionRejected()
	}
	if deviceID != "" {
		s.service.UnregisterSession(deviceID, session.SessionID())
		logger.Info("device disconnected", "device_id", deviceID)
	}
	_ = session.Close()
}

func (s *Server) trackConnection(connID string, conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connections[connID] = conn
}

func (s *Server) untrackConnection(connID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.connections, connID)
}

func (s *Server) closeAllConnections() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, conn := range s.connections {
		_ = conn.Close()
	}
}

type clientSession struct {
	id           string
	conn         net.Conn
	queue        chan model.ServerMessage
	writeTimeout time.Duration
	logger       *slog.Logger

	closed    chan struct{}
	closeOnce sync.Once
}

func newClientSession(id string, conn net.Conn, queueSize int, writeTimeout time.Duration, logger *slog.Logger) *clientSession {
	if queueSize <= 0 {
		queueSize = 128
	}

	return &clientSession{
		id:           id,
		conn:         conn,
		queue:        make(chan model.ServerMessage, queueSize),
		writeTimeout: writeTimeout,
		logger:       logger,
		closed:       make(chan struct{}),
	}
}

func (s *clientSession) SessionID() string {
	return s.id
}

func (s *clientSession) Transport() string {
	return "tcp"
}

func (s *clientSession) Send(message model.ServerMessage) error {
	select {
	case <-s.closed:
		return net.ErrClosed
	default:
	}

	select {
	case s.queue <- message:
		return nil
	default:
		return errors.New("session queue is full")
	}
}

func (s *clientSession) Close() error {
	var err error
	s.closeOnce.Do(func() {
		close(s.closed)
		err = s.conn.Close()
	})
	return err
}

func (s *clientSession) runWriter(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			_ = s.Close()
			return
		case <-s.closed:
			return
		case message := <-s.queue:
			if err := s.writeMessage(message); err != nil {
				s.logger.Warn("failed to write message", "error", err)
				_ = s.Close()
				return
			}
		}
	}
}

func (s *clientSession) writeMessage(message model.ServerMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')

	if err := s.conn.SetWriteDeadline(time.Now().Add(s.writeTimeout)); err != nil {
		return err
	}

	total := 0
	for total < len(payload) {
		written, err := s.conn.Write(payload[total:])
		if err != nil {
			return err
		}
		total += written
	}
	return nil
}

var _ core.Session = (*clientSession)(nil)
