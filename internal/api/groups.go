package api

import (
	"errors"
	"net/http"
	"strings"

	"mvp-platform/internal/core"
	"mvp-platform/internal/store"
)

func (s *Server) handleGroups(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/groups" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleListGroups(w, r)
	case http.MethodPost:
		s.handleCreateGroup(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleGroupRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/groups/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	parts := strings.Split(path, "/")
	groupID := parts[0]

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleGetGroup(w, r, groupID)
		return
	}

	if parts[1] != "devices" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	switch {
	case len(parts) == 2 && r.Method == http.MethodPost:
		s.handleAssignGroupDevice(w, r, groupID)
	case len(parts) == 3 && r.Method == http.MethodDelete:
		s.handleRemoveGroupDevice(w, r, groupID, parts[2])
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		ProductID   string            `json:"product_id"`
		Tags        map[string]string `json:"tags"`
	}

	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	group, err := s.service.CreateGroup(r.Context(), request.Name, request.Description, request.ProductID, request.Tags)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, group)
}

func (s *Server) handleListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := s.service.ListGroups(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (s *Server) handleGetGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	group, err := s.service.GetGroup(r.Context(), groupID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrGroupNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, group)
}

func (s *Server) handleAssignGroupDevice(w http.ResponseWriter, r *http.Request, groupID string) {
	var request struct {
		DeviceID string `json:"device_id"`
	}

	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	group, err := s.service.AssignDeviceToGroup(r.Context(), groupID, request.DeviceID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrGroupNotFound), errors.Is(err, store.ErrDeviceNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, core.ErrInvalidGroup):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, group)
}

func (s *Server) handleRemoveGroupDevice(w http.ResponseWriter, r *http.Request, groupID, deviceID string) {
	group, err := s.service.RemoveDeviceFromGroup(r.Context(), groupID, deviceID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrGroupNotFound), errors.Is(err, store.ErrDeviceNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, group)
}
