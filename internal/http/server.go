package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const scalarHTML = `<!DOCTYPE html>
<html>
<head>
  <title>MongoDB Investigation Engine - API Docs</title>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
</head>
<body>
  <script id="api-reference" data-url="/docs/json"></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`

type Server struct {
	router                chi.Router
	connectionHandler     *ConnectionHandler
	scanHandler           *ScanHandler
	relationshipHandler   *RelationshipHandler
	investigationHandler  *InvestigationHandler
	orphanHandler         *OrphanHandler
	swaggerJSON           json.RawMessage
}

func NewServer(connectionHandler *ConnectionHandler, scanHandler *ScanHandler, relationshipHandler *RelationshipHandler, investigationHandler *InvestigationHandler, orphanHandler *OrphanHandler, swaggerJSON json.RawMessage) *Server {
	s := &Server{
		connectionHandler:    connectionHandler,
		scanHandler:          scanHandler,
		relationshipHandler:  relationshipHandler,
		investigationHandler: investigationHandler,
		orphanHandler:        orphanHandler,
		swaggerJSON:          swaggerJSON,
	}

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Get("/docs", s.scalarUI)
	r.Get("/docs/json", s.swaggerJSONHandler)

	r.Route("/api/connections", func(r chi.Router) {
		r.Mount("/", s.connectionHandler.Routes())
	})

	r.Route("/api/scans", func(r chi.Router) {
		r.Mount("/", s.scanHandler.Routes())
	})

	r.Route("/api/relationships", func(r chi.Router) {
		r.Mount("/", s.relationshipHandler.Routes())
	})

	r.Route("/api/investigate", func(r chi.Router) {
		r.Mount("/", s.investigationHandler.Routes())
	})

	r.Route("/api/orphans", func(r chi.Router) {
		r.Mount("/", s.orphanHandler.Routes())
	})

	s.router = r
	return s
}

func (s *Server) Router() chi.Router {
	return s.router
}

func (s *Server) scalarUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(scalarHTML))
}

func (s *Server) swaggerJSONHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(s.swaggerJSON)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
