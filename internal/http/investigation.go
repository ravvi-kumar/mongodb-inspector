package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ravikumar/mongodb-inspector/internal/service"
)

type InvestigationHandler struct {
	service *service.InvestigationService
}

func NewInvestigationHandler(service *service.InvestigationService) *InvestigationHandler {
	return &InvestigationHandler{service: service}
}

func (h *InvestigationHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Investigate)

	return r
}

type investigateRequest struct {
	ConnectionID string `json:"connection_id"`
	DocumentID   string `json:"document_id"`
}

func (h *InvestigationHandler) Investigate(w http.ResponseWriter, r *http.Request) {
	var req investigateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.ConnectionID == "" {
		writeError(w, http.StatusBadRequest, "connection_id is required")
		return
	}
	if req.DocumentID == "" {
		writeError(w, http.StatusBadRequest, "document_id is required")
		return
	}

	result, err := h.service.Investigate(r.Context(), req.ConnectionID, req.DocumentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
