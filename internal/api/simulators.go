package api

import (
	"errors"
	"net/http"
	"strings"

	"mvp-platform/internal/simulator"
)

func (s *Server) handleSimulators(w http.ResponseWriter, r *http.Request) {
	if s.simulators == nil {
		writeError(w, http.StatusServiceUnavailable, "simulator manager unavailable")
		return
	}

	if r.URL.Path != "/api/v1/simulators" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.simulators.List())
	case http.MethodPost:
		s.handleCreateSimulator(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleSimulatorRoutes(w http.ResponseWriter, r *http.Request) {
	if s.simulators == nil {
		writeError(w, http.StatusServiceUnavailable, "simulator manager unavailable")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/simulators/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	parts := strings.Split(path, "/")
	simulatorID := parts[0]

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleGetSimulator(w, r, simulatorID)
		case http.MethodDelete:
			s.handleDeleteSimulator(w, r, simulatorID)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	switch parts[1] {
	case "connect":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleConnectSimulator(w, r, simulatorID)
	case "disconnect":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleDisconnectSimulator(w, r, simulatorID)
	case "telemetry":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleSendSimulatorTelemetry(w, r, simulatorID)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *Server) handleCreateSimulator(w http.ResponseWriter, r *http.Request) {
	request := struct {
		Name                string            `json:"name"`
		ProductID           string            `json:"product_id"`
		Metadata            map[string]string `json:"metadata"`
		AutoConnect         bool              `json:"auto_connect"`
		AutoAck             bool              `json:"auto_ack"`
		AutoPing            bool              `json:"auto_ping"`
		AutoTelemetry       bool              `json:"auto_telemetry"`
		TelemetryIntervalMS int               `json:"telemetry_interval_ms"`
		DefaultValues       map[string]any    `json:"default_values"`
	}{
		AutoConnect:         true,
		AutoAck:             true,
		AutoPing:            true,
		AutoTelemetry:       true,
		TelemetryIntervalMS: 5000,
	}

	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	view, err := s.simulators.Create(r.Context(), simulator.CreateRequest{
		Name:                request.Name,
		ProductID:           request.ProductID,
		Metadata:            request.Metadata,
		AutoConnect:         request.AutoConnect,
		AutoAck:             request.AutoAck,
		AutoPing:            request.AutoPing,
		AutoTelemetry:       request.AutoTelemetry,
		TelemetryIntervalMS: request.TelemetryIntervalMS,
		DefaultValues:       request.DefaultValues,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, view)
}

func (s *Server) handleGetSimulator(w http.ResponseWriter, r *http.Request, simulatorID string) {
	view, err := s.simulators.Get(simulatorID)
	if err != nil {
		writeSimulatorError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleDeleteSimulator(w http.ResponseWriter, r *http.Request, simulatorID string) {
	if err := s.simulators.Remove(simulatorID); err != nil {
		writeSimulatorError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleConnectSimulator(w http.ResponseWriter, r *http.Request, simulatorID string) {
	view, err := s.simulators.Connect(simulatorID)
	if err != nil {
		writeSimulatorError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleDisconnectSimulator(w http.ResponseWriter, r *http.Request, simulatorID string) {
	view, err := s.simulators.Disconnect(simulatorID)
	if err != nil {
		writeSimulatorError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleSendSimulatorTelemetry(w http.ResponseWriter, r *http.Request, simulatorID string) {
	var request struct {
		Values map[string]any `json:"values"`
	}

	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	view, err := s.simulators.SendTelemetry(simulatorID, request.Values)
	if err != nil {
		writeSimulatorError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, view)
}

func writeSimulatorError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, simulator.ErrSimulatorNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, simulator.ErrSimulatorNotConnected):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, simulator.ErrSimulatorBusy):
		writeError(w, http.StatusTooManyRequests, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
