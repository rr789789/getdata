package api

import (
	"errors"
	"net/http"
	"strings"

	"mvp-platform/internal/model"
	"mvp-platform/internal/store"
)

func (s *Server) handleProducts(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/products" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleListProducts(w, r)
	case http.MethodPost:
		s.handleCreateProduct(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleProductRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/products/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	parts := strings.Split(path, "/")
	productID := parts[0]

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleGetProduct(w, r, productID)
		return
	}

	switch parts[1] {
	case "thing-model":
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleUpdateThingModel(w, r, productID)
	case "access-profile":
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleUpdateAccessProfile(w, r, productID)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *Server) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name          string                    `json:"name"`
		Description   string                    `json:"description"`
		Metadata      map[string]string         `json:"metadata"`
		AccessProfile model.ProductAccessProfile `json:"access_profile"`
		ThingModel    model.ThingModel          `json:"thing_model"`
	}

	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	product, err := s.service.CreateProduct(r.Context(), request.Name, request.Description, request.Metadata, request.AccessProfile, request.ThingModel)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, product)
}

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	products, err := s.service.ListProducts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, products)
}

func (s *Server) handleGetProduct(w http.ResponseWriter, r *http.Request, productID string) {
	product, err := s.service.GetProduct(r.Context(), productID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrProductNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func (s *Server) handleUpdateThingModel(w http.ResponseWriter, r *http.Request, productID string) {
	var request struct {
		ThingModel model.ThingModel `json:"thing_model"`
	}

	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	product, err := s.service.UpdateProductThingModel(r.Context(), productID, request.ThingModel)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, store.ErrProductNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func (s *Server) handleUpdateAccessProfile(w http.ResponseWriter, r *http.Request, productID string) {
	var request struct {
		AccessProfile model.ProductAccessProfile `json:"access_profile"`
	}

	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	product, err := s.service.UpdateProductAccessProfile(r.Context(), productID, request.AccessProfile)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, store.ErrProductNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, product)
}
