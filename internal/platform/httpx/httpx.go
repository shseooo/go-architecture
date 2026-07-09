// Package httpx holds the HTTP conventions shared by every module's delivery
// layer: the response envelope, error mapping, and request parsing helpers.
package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/shseooo/go-architecture/internal/shared"
)

// Envelope is the standard success body: {data} for single/array, {data, meta}
// for paginated collections.
type Envelope[T any] struct {
	Data T     `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

type Meta struct {
	Pagination *Pagination `json:"pagination,omitempty"`
}

type Pagination struct {
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasNext bool `json:"has_next"`
}

// ErrorEnvelope is the standard error body.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Data writes a single resource: {"data": <v>}.
func Data[T any](w http.ResponseWriter, status int, data T) {
	writeJSON(w, status, Envelope[T]{Data: data})
}

// Paged writes a collection with pagination metadata.
func Paged[T any](w http.ResponseWriter, items []T, p Pagination) {
	writeJSON(w, http.StatusOK, Envelope[[]T]{Data: items, Meta: &Meta{Pagination: &p}})
}

// NoContent writes a 204.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error maps a (shared) domain error to an HTTP status + error envelope.
func Error(w http.ResponseWriter, err error) {
	status := statusFromError(err)
	if status == http.StatusInternalServerError {
		slog.Error("request failed", "error", err)
	}
	writeJSON(w, status, ErrorEnvelope{Error: ErrorBody{Code: errorCode(err), Message: err.Error()}})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func statusFromError(err error) int {
	switch {
	case errors.Is(err, shared.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, shared.ErrConflict),
		errors.Is(err, shared.ErrInsufficientStock),
		errors.Is(err, shared.ErrAlreadyCanceled):
		return http.StatusConflict
	case errors.Is(err, shared.ErrBadParamInput):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func errorCode(err error) string {
	switch {
	case errors.Is(err, shared.ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, shared.ErrConflict):
		return "CONFLICT"
	case errors.Is(err, shared.ErrInsufficientStock):
		return "INSUFFICIENT_STOCK"
	case errors.Is(err, shared.ErrAlreadyCanceled):
		return "ORDER_NOT_CANCELABLE"
	case errors.Is(err, shared.ErrBadParamInput):
		return "BAD_REQUEST"
	default:
		return "INTERNAL"
	}
}

// DecodeJSON reads the request body into dst, rejecting unknown fields.
func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return shared.ErrBadParamInput
	}
	return nil
}

// PathID parses a required int64 path parameter (Go 1.22 routing).
func PathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		return 0, shared.ErrBadParamInput
	}
	return id, nil
}

func QueryInt(r *http.Request, name string) *int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}
	return &v
}

func QueryInt64(r *http.Request, name string) *int64 {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}
