package http

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
	"github.com/ravikumar/mongodb-inspector/internal/store/pg"
	mongostore "github.com/ravikumar/mongodb-inspector/internal/store/mongo"
)

type ConnectionHandler struct {
	connStore *pg.ConnectionStore
}

func NewConnectionHandler(connStore *pg.ConnectionStore) *ConnectionHandler {
	return &ConnectionHandler{connStore: connStore}
}

func (h *ConnectionHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Delete("/{id}", h.Delete)
	r.Get("/{id}/databases", h.ListDatabases)
	r.Post("/{id}/select-db", h.SelectDatabase)
	r.Get("/{id}/collections", h.ListCollections)

	return r
}

type createConnectionRequest struct {
	Name           string `json:"name"`
	ConnectionString string `json:"connection_string"`
}

func (h *ConnectionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createConnectionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Name == "" || req.ConnectionString == "" {
		writeError(w, http.StatusBadRequest, "name and connection_string are required")
		return
	}

	conn := &domain.Connection{
		Name:           req.Name,
		ConnectionString: req.ConnectionString,
	}

	if err := h.connStore.Create(r.Context(), conn); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, conn)
}

func (h *ConnectionHandler) List(w http.ResponseWriter, r *http.Request) {
	conns, err := h.connStore.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if conns == nil {
		conns = []domain.Connection{}
	}
	writeJSON(w, http.StatusOK, conns)
}

func (h *ConnectionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	conn, err := h.connStore.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}
	writeJSON(w, http.StatusOK, conn)
}

func (h *ConnectionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.connStore.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *ConnectionHandler) ListDatabases(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	conn, err := h.connStore.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	mongoConn, err := mongostore.NewConnector(r.Context(), conn.ConnectionString)
	if err != nil {
		writeError(w, http.StatusBadRequest, "cannot connect to MongoDB: "+err.Error())
		return
	}
	defer mongoConn.Close(context.Background())

	dbs, err := mongoConn.ListDatabases(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"databases": dbs})
}

type selectDBRequest struct {
	Database string `json:"database"`
}

func (h *ConnectionHandler) SelectDatabase(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req selectDBRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Database == "" {
		writeError(w, http.StatusBadRequest, "database is required")
		return
	}

	conn, err := h.connStore.UpdateDatabase(r.Context(), id, req.Database)
	if err != nil {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	writeJSON(w, http.StatusOK, conn)
}

func (h *ConnectionHandler) ListCollections(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	conn, err := h.connStore.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	if conn.Database == "" {
		writeError(w, http.StatusBadRequest, "no database selected for this connection")
		return
	}

	mongoConn, err := mongostore.NewConnector(r.Context(), conn.ConnectionString)
	if err != nil {
		writeError(w, http.StatusBadRequest, "cannot connect to MongoDB: "+err.Error())
		return
	}
	defer mongoConn.Close(context.Background())

	collections, err := mongoConn.ListCollections(r.Context(), conn.Database)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"collections": collections})
}
