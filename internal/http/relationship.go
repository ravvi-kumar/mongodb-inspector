package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
	"github.com/ravikumar/mongodb-inspector/internal/service"
	"github.com/ravikumar/mongodb-inspector/internal/store/pg"
)

type RelationshipHandler struct {
	relStore  *pg.RelationshipStore
	discovery *service.DiscoveryService
}

func NewRelationshipHandler(relStore *pg.RelationshipStore, discovery *service.DiscoveryService) *RelationshipHandler {
	return &RelationshipHandler{relStore: relStore, discovery: discovery}
}

func (h *RelationshipHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/discover", h.Discover)
	r.Get("/{id}", h.Get)
	r.Post("/{id}/approve", h.Approve)
	r.Post("/{id}/reject", h.Reject)

	return r
}

func (h *RelationshipHandler) List(w http.ResponseWriter, r *http.Request) {
	connectionID := r.URL.Query().Get("connection_id")
	if connectionID == "" {
		writeError(w, http.StatusBadRequest, "connection_id query param is required")
		return
	}

	var statusFilter *string
	if s := r.URL.Query().Get("status"); s != "" {
		statusFilter = &s
	}

	rels, err := h.relStore.List(r.Context(), connectionID, statusFilter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rels == nil {
		rels = []domain.Relationship{}
	}
	writeJSON(w, http.StatusOK, rels)
}

func (h *RelationshipHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rel, err := h.relStore.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "relationship not found")
		return
	}
	writeJSON(w, http.StatusOK, rel)
}

func (h *RelationshipHandler) Approve(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rel, err := h.relStore.UpdateStatus(r.Context(), id, domain.RelationshipStatusApproved)
	if err != nil {
		writeError(w, http.StatusNotFound, "relationship not found")
		return
	}
	writeJSON(w, http.StatusOK, rel)
}

func (h *RelationshipHandler) Reject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rel, err := h.relStore.UpdateStatus(r.Context(), id, domain.RelationshipStatusRejected)
	if err != nil {
		writeError(w, http.StatusNotFound, "relationship not found")
		return
	}
	writeJSON(w, http.StatusOK, rel)
}

type discoverRequest struct {
	ScanID string `json:"scan_id"`
}

func (h *RelationshipHandler) Discover(w http.ResponseWriter, r *http.Request) {
	var req discoverRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.ScanID == "" {
		writeError(w, http.StatusBadRequest, "scan_id is required")
		return
	}

	if err := h.discovery.DiscoverRelationships(r.Context(), req.ScanID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	connectionID, _ := h.relStore.GetByScanConnection(r.Context(), req.ScanID)

	statusFilter := string(domain.RelationshipStatusSuggested)
	rels, _ := h.relStore.List(r.Context(), connectionID, &statusFilter)
	if rels == nil {
		rels = []domain.Relationship{}
	}

	writeJSON(w, http.StatusOK, rels)
}
