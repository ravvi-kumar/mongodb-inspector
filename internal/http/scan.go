package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
	"github.com/ravikumar/mongodb-inspector/internal/service"
	"github.com/ravikumar/mongodb-inspector/internal/worker"
)

type ScanHandler struct {
	scannerSvc *service.ScannerService
	worker     *worker.ScannerWorker
}

func NewScanHandler(scannerSvc *service.ScannerService, worker *worker.ScannerWorker) *ScanHandler {
	return &ScanHandler{scannerSvc: scannerSvc, worker: worker}
}

func (h *ScanHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.StartScan)
	r.Get("/", h.ListScans)
	r.Get("/{id}", h.GetScan)
	r.Get("/{id}/fields", h.GetFields)
	r.Get("/{id}/candidates", h.GetCandidates)

	return r
}

type startScanRequest struct {
	ConnectionID string `json:"connection_id"`
	SampleSize   int    `json:"sample_size"`
}

func (h *ScanHandler) StartScan(w http.ResponseWriter, r *http.Request) {
	var req startScanRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.ConnectionID == "" {
		writeError(w, http.StatusBadRequest, "connection_id is required")
		return
	}

	sampleSize := req.SampleSize
	if sampleSize <= 0 {
		sampleSize = 1000
	}

	scan, err := h.scannerSvc.StartScan(r.Context(), req.ConnectionID, sampleSize)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.worker.Enqueue(scan.ID)

	writeJSON(w, http.StatusAccepted, scan)
}

func (h *ScanHandler) ListScans(w http.ResponseWriter, r *http.Request) {
	connectionID := r.URL.Query().Get("connection_id")
	if connectionID == "" {
		writeError(w, http.StatusBadRequest, "connection_id query param is required")
		return
	}

	scans, err := h.scannerSvc.ListScans(r.Context(), connectionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if scans == nil {
		scans = []domain.Scan{}
	}
	writeJSON(w, http.StatusOK, scans)
}

func (h *ScanHandler) GetScan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	scan, err := h.scannerSvc.GetScan(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "scan not found")
		return
	}
	writeJSON(w, http.StatusOK, scan)
}

func (h *ScanHandler) GetFields(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	fields, err := h.scannerSvc.GetScanFields(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if fields == nil {
		fields = []domain.CollectionField{}
	}
	writeJSON(w, http.StatusOK, fields)
}

func (h *ScanHandler) GetCandidates(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	fields, err := h.scannerSvc.GetCandidateFields(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if fields == nil {
		fields = []domain.CollectionField{}
	}
	writeJSON(w, http.StatusOK, fields)
}
