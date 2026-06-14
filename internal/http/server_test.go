package http

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHealthEndpoint(t *testing.T) {
	srv := NewServer(
		&ConnectionHandler{},
		&ScanHandler{},
		&RelationshipHandler{},
		&InvestigationHandler{},
		&OrphanHandler{},
		json.RawMessage(`{"openapi":"3.1.0"}`),
	)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("health status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("health body = %v, want status=ok", body)
	}
}

func TestWriteJSON(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"hello": "world"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["hello"] != "world" {
		t.Errorf("body = %v, want hello=world", body)
	}
}

func TestWriteError(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/err", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusBadRequest, "something went wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "something went wrong" {
		t.Errorf("error = %q, want 'something went wrong'", body["error"])
	}
}

func TestDecodeJSON(t *testing.T) {
	r := chi.NewRouter()

	type testReq struct {
		Name string `json:"name"`
	}

	r.Post("/decode", func(w http.ResponseWriter, r *http.Request) {
		var req testReq
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, req)
	})

	body := `{"name":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/decode", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var resp testReq
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "test" {
		t.Errorf("name = %q, want test", resp.Name)
	}
}

func TestDecodeJSON_UnknownFields(t *testing.T) {
	r := chi.NewRouter()

	type testReq struct {
		Name string `json:"name"`
	}

	r.Post("/decode", func(w http.ResponseWriter, r *http.Request) {
		var req testReq
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, req)
	})

	body := `{"name":"test","unknown":"field"}`
	req := httptest.NewRequest(http.MethodPost, "/decode", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("unknown fields should be rejected, got status %d", w.Code)
	}
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	r := chi.NewRouter()

	type testReq struct {
		Name string `json:"name"`
	}

	r.Post("/decode", func(w http.ResponseWriter, r *http.Request) {
		var req testReq
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, req)
	})

	req := httptest.NewRequest(http.MethodPost, "/decode", strings.NewReader(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON should return 400, got %d", w.Code)
	}
}

func TestInvestigateHandler_MissingFields(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"missing connection_id", `{"document_id":"123"}`, http.StatusBadRequest},
		{"missing document_id", `{"connection_id":"abc"}`, http.StatusBadRequest},
		{"missing both", `{}`, http.StatusBadRequest},
		{"empty body", ``, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &InvestigationHandler{}

			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/", body)
			if body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			handler.Investigate(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestOrphanHandler_Detect_MissingConnectionID(t *testing.T) {
	handler := &OrphanHandler{}

	req := httptest.NewRequest(http.MethodPost, "/detect", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Detect(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrphanHandler_List_MissingConnectionID(t *testing.T) {
	handler := &OrphanHandler{}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRelationshipHandler_List_MissingConnectionID(t *testing.T) {
	handler := &RelationshipHandler{}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRelationshipHandler_Discover_MissingScanID(t *testing.T) {
	handler := &RelationshipHandler{}

	req := httptest.NewRequest(http.MethodPost, "/discover", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Discover(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestScanHandler_Start_MissingConnectionID(t *testing.T) {
	handler := &ScanHandler{}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.StartScan(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestScanHandler_List_MissingConnectionID(t *testing.T) {
	handler := &ScanHandler{}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ListScans(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestScanHandler_Start_DefaultSampleSize(t *testing.T) {
	handler := &ScanHandler{
		scannerSvc: nil,
	}

	body := `{"connection_id":"test-conn"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic with nil service")
		}
	}()
	handler.StartScan(w, req)
}

func TestConnectionHandler_Create_MissingFields(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"missing both", `{}`, http.StatusBadRequest},
		{"missing name", `{"connection_string":"mongodb://localhost"}`, http.StatusBadRequest},
		{"missing connection_string", `{"name":"test"}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &ConnectionHandler{}

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Create(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestConnectionHandler_SelectDB_MissingDB(t *testing.T) {
	handler := &ConnectionHandler{}

	req := httptest.NewRequest(http.MethodPost, "/abc/select-db", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SelectDatabase(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSwaggerEndpoints(t *testing.T) {
	swaggerJSON := json.RawMessage(`{"openapi":"3.1.0","info":{"title":"test"}}`)

	srv := NewServer(
		&ConnectionHandler{},
		&ScanHandler{},
		&RelationshipHandler{},
		&InvestigationHandler{},
		&OrphanHandler{},
		swaggerJSON,
	)

	req := httptest.NewRequest(http.MethodGet, "/docs/json", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("swagger json status = %d, want %d", w.Code, http.StatusOK)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("swagger content-type = %q, want application/json", ct)
	}

	req = httptest.NewRequest(http.MethodGet, "/docs", nil)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("docs UI status = %d, want %d", w.Code, http.StatusOK)
	}
}
