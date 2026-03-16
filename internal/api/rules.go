package api

import (
	"errors"
	"net/http"
	"strings"

	"mvp-platform/internal/model"
	"mvp-platform/internal/store"
)

func (s *Server) handleRules(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/rules" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleListRules(w, r)
	case http.MethodPost:
		s.handleCreateRule(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/alerts" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	alerts, err := s.service.ListAlerts(
		r.Context(),
		parseLimit(r, 50, 500),
		strings.TrimSpace(r.URL.Query().Get("product_id")),
		strings.TrimSpace(r.URL.Query().Get("group_id")),
		strings.TrimSpace(r.URL.Query().Get("device_id")),
		strings.TrimSpace(r.URL.Query().Get("rule_id")),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, alerts)
}

func (s *Server) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	request := struct {
		Name            string              `json:"name"`
		Description     string              `json:"description"`
		ProductID       string              `json:"product_id"`
		GroupID         string              `json:"group_id"`
		DeviceID        string              `json:"device_id"`
		Enabled         bool                `json:"enabled"`
		Severity        model.AlertSeverity `json:"severity"`
		CooldownSeconds int                 `json:"cooldown_seconds"`
		Condition       model.RuleCondition `json:"condition"`
	}{
		Enabled:  true,
		Severity: model.AlertSeverityWarning,
	}

	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	rule, err := s.service.CreateRule(
		r.Context(),
		request.Name,
		request.Description,
		request.ProductID,
		request.GroupID,
		request.DeviceID,
		request.Enabled,
		request.Severity,
		request.CooldownSeconds,
		request.Condition,
	)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrProductNotFound), errors.Is(err, store.ErrGroupNotFound), errors.Is(err, store.ErrDeviceNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.service.ListRules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rules)
}
