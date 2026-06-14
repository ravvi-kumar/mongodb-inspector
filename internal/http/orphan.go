package http

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
	"github.com/ravikumar/mongodb-inspector/internal/service"
)

type OrphanHandler struct {
	service *service.OrphanService
}

func NewOrphanHandler(service *service.OrphanService) *OrphanHandler {
	return &OrphanHandler{service: service}
}

func (h *OrphanHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/detect", h.Detect)
	r.Get("/", h.List)
	r.Get("/{id}/investigate", h.Investigate)

	return r
}

type detectOrphansRequest struct {
	ConnectionID string `json:"connection_id"`
}

func (h *OrphanHandler) Detect(w http.ResponseWriter, r *http.Request) {
	var req detectOrphansRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.ConnectionID == "" {
		writeError(w, http.StatusBadRequest, "connection_id is required")
		return
	}

	orphans, err := h.service.DetectOrphans(r.Context(), req.ConnectionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if orphans == nil {
		orphans = []domain.Orphan{}
	}

	writeJSON(w, http.StatusOK, orphans)
}

func (h *OrphanHandler) List(w http.ResponseWriter, r *http.Request) {
	connectionID := r.URL.Query().Get("connection_id")
	if connectionID == "" {
		writeError(w, http.StatusBadRequest, "connection_id query param is required")
		return
	}

	offset, limit := parsePagination(r)

	orphans, total, err := h.service.ListOrphansPaginated(r.Context(), connectionID, offset, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if orphans == nil {
		orphans = []domain.Orphan{}
	}

	writeJSON(w, http.StatusOK, domain.PaginatedResponse{
		Data:   orphans,
		Total:  int(total),
		Offset: offset,
		Limit:  limit,
	})
}

func (h *OrphanHandler) Investigate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := h.service.InvestigateOrphan(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
