package api

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"mvp-platform/internal/setup"
	"mvp-platform/internal/store"
)

func (s *Server) handleInstallStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.installer == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"installed": true,
			"managed":   false,
			"node_id":   s.cfg.NodeID,
			"role":      normalizedNodeRole(s.cfg.NodeRole),
		})
		return
	}

	state := s.installer.Status()
	writeJSON(w, http.StatusOK, map[string]any{
		"installed":   state.InstallLock,
		"managed":     true,
		"setup_path":  s.installer.Path(),
		"state":       state,
		"node_id":     s.cfg.NodeID,
		"role":        normalizedNodeRole(s.cfg.NodeRole),
		"site_url":    state.SiteURL,
		"app_name":    state.AppName,
	})
}

func (s *Server) handleInstallBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.installer == nil {
		writeError(w, http.StatusServiceUnavailable, "install manager unavailable")
		return
	}
	if s.installer.Installed() {
		writeError(w, http.StatusConflict, "instance already installed")
		return
	}

	var request struct {
		AppName           string `json:"app_name"`
		SiteURL           string `json:"site_url"`
		AdminUsername     string `json:"admin_username"`
		AdminEmail        string `json:"admin_email"`
		DefaultTenantName string `json:"default_tenant_name"`
		DefaultTenantSlug string `json:"default_tenant_slug"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	state, err := s.installer.Bootstrap(setup.BootstrapRequest{
		AppName:           request.AppName,
		SiteURL:           request.SiteURL,
		AdminUsername:     request.AdminUsername,
		AdminEmail:        request.AdminEmail,
		DefaultTenantName: request.DefaultTenantName,
		DefaultTenantSlug: request.DefaultTenantSlug,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.seedInstallTenant(r, state); err != nil {
		s.logger.Warn("install bootstrap completed but default tenant seeding failed", "error", err)
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"installed":       true,
		"restart_required": false,
		"state":           state,
		"next_url":        "/",
	})
}

func (s *Server) seedInstallTenant(r *http.Request, state setup.State) error {
	if s.service == nil {
		return nil
	}

	name := strings.TrimSpace(state.DefaultTenantName)
	if name == "" {
		return nil
	}

	tenants, err := s.service.ListTenants(r.Context())
	if err != nil {
		return err
	}
	if len(tenants) > 0 {
		return nil
	}

	_, err = s.service.CreateTenant(
		r.Context(),
		name,
		state.DefaultTenantSlug,
		"created by install wizard",
		map[string]string{
			"source":         "install-wizard",
			"admin_username": state.AdminUsername,
			"admin_email":    state.AdminEmail,
		},
	)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, store.ErrTenantExists):
		return nil
	default:
		return err
	}
}

func (s *Server) handleReplicaSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.installer == nil {
		writeError(w, http.StatusServiceUnavailable, "replica setup endpoint unavailable")
		return
	}
	if strings.TrimSpace(s.cfg.ReplicaToken) == "" {
		writeError(w, http.StatusServiceUnavailable, "replica token not configured")
		return
	}
	if !subtleConstantCompare(strings.TrimSpace(r.Header.Get("X-Replica-Token")), s.cfg.ReplicaToken) {
		writeError(w, http.StatusForbidden, "replica token rejected")
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "unable to read setup payload")
		return
	}
	if err := s.installer.ApplyReplicaState(body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":    "setup replicated",
		"time":      time.Now().UTC(),
		"node_id":   s.cfg.NodeID,
		"node_role": normalizedNodeRole(s.cfg.NodeRole),
	})
}
