package api

import (
	"errors"
	"net/http"
	"strings"

	"mvp-platform/internal/store"
)

func (s *Server) handleFirmwareArtifacts(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/firmware" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		artifacts, err := s.service.ListFirmwareArtifacts(
			r.Context(),
			strings.TrimSpace(r.URL.Query().Get("tenant_id")),
			strings.TrimSpace(r.URL.Query().Get("product_id")),
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, artifacts)
	case http.MethodPost:
		var request struct {
			TenantID     string            `json:"tenant_id"`
			ProductID    string            `json:"product_id"`
			Name         string            `json:"name"`
			Version      string            `json:"version"`
			FileName     string            `json:"file_name"`
			URL          string            `json:"url"`
			Checksum     string            `json:"checksum"`
			ChecksumType string            `json:"checksum_type"`
			SizeBytes    int64             `json:"size_bytes"`
			Metadata     map[string]string `json:"metadata"`
			Notes        string            `json:"notes"`
		}
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		artifact, err := s.service.CreateFirmwareArtifact(
			r.Context(),
			request.TenantID,
			request.ProductID,
			request.Name,
			request.Version,
			request.FileName,
			request.URL,
			request.Checksum,
			request.ChecksumType,
			request.SizeBytes,
			request.Metadata,
			request.Notes,
		)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrTenantNotFound), errors.Is(err, store.ErrProductNotFound):
				writeError(w, http.StatusNotFound, err.Error())
			default:
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}
		writeJSON(w, http.StatusCreated, artifact)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleOTACampaigns(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/ota-campaigns" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		campaigns, err := s.service.ListOTACampaigns(r.Context(), strings.TrimSpace(r.URL.Query().Get("tenant_id")))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, campaigns)
	case http.MethodPost:
		var request struct {
			TenantID   string `json:"tenant_id"`
			Name       string `json:"name"`
			FirmwareID string `json:"firmware_id"`
			ProductID  string `json:"product_id"`
			GroupID    string `json:"group_id"`
			DeviceID   string `json:"device_id"`
		}
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		campaign, err := s.service.CreateOTACampaign(r.Context(), request.TenantID, request.Name, request.FirmwareID, request.ProductID, request.GroupID, request.DeviceID)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrTenantNotFound), errors.Is(err, store.ErrProductNotFound), errors.Is(err, store.ErrGroupNotFound), errors.Is(err, store.ErrDeviceNotFound), errors.Is(err, store.ErrFirmwareNotFound):
				writeError(w, http.StatusNotFound, err.Error())
			default:
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}
		writeJSON(w, http.StatusCreated, campaign)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
