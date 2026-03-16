package api

import (
	"errors"
	"net/http"
	"strings"

	"mvp-platform/internal/core"
	"mvp-platform/internal/store"
)

func (s *Server) handleConfigProfiles(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/config-profiles" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleListConfigProfiles(w, r)
	case http.MethodPost:
		s.handleCreateConfigProfile(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleConfigProfileRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/config-profiles/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	parts := strings.Split(path, "/")
	profileID := parts[0]
	if len(parts) != 2 || parts[1] != "apply" || r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	s.handleApplyConfigProfile(w, r, profileID)
}

func (s *Server) handleCreateConfigProfile(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		ProductID   string         `json:"product_id"`
		Values      map[string]any `json:"values"`
	}

	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	profile, err := s.service.CreateConfigProfile(r.Context(), request.Name, request.Description, request.ProductID, request.Values)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, profile)
}

func (s *Server) handleListConfigProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := s.service.ListConfigProfiles(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, profiles)
}

func (s *Server) handleApplyConfigProfile(w http.ResponseWriter, r *http.Request, profileID string) {
	var request struct {
		DeviceID string `json:"device_id"`
	}

	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	shadow, err := s.service.ApplyConfigProfile(r.Context(), profileID, request.DeviceID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrConfigNotFound), errors.Is(err, store.ErrDeviceNotFound), errors.Is(err, store.ErrProductNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, core.ErrInvalidConfig):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, shadow)
}
