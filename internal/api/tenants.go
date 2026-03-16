package api

import "net/http"

func (s *Server) handleTenants(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/tenants" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		tenants, err := s.service.ListTenants(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, tenants)
	case http.MethodPost:
		var request struct {
			Name        string            `json:"name"`
			Slug        string            `json:"slug"`
			Description string            `json:"description"`
			Metadata    map[string]string `json:"metadata"`
		}
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		tenant, err := s.service.CreateTenant(r.Context(), request.Name, request.Slug, request.Description, request.Metadata)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, tenant)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
