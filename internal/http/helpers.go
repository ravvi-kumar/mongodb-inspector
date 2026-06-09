package http

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func decodeJSON(r *http.Request, v any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

const defaultLimit = 20
const maxLimit = 100

func parsePagination(r *http.Request) (int, int) {
	offset := 0
	limit := defaultLimit

	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return offset, limit
}
