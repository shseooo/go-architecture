package rest

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/shseooo/go-architecture/app/domain"
)

// Envelope is the standard success response body. Single resources use
// Envelope[T]; collections use Envelope[[]T]. Meta is present only for paginated
// responses.
type Envelope[T any] struct {
	Data T     `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

// Meta carries response metadata that is not part of the resource itself.
type Meta struct {
	Pagination *Pagination `json:"pagination,omitempty"`
}

// Pagination describes an offset-based page of a collection.
type Pagination struct {
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasNext bool `json:"has_next"`
}

// ErrorEnvelope is the standard error response body.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeData writes a single resource: {"data": <v>}.
func writeData[T any](w http.ResponseWriter, status int, data T) {
	writeJSON(w, status, Envelope[T]{Data: data})
}

// writePaged writes a collection with pagination metadata.
func writePaged[T any](w http.ResponseWriter, items []T, p Pagination) {
	writeJSON(w, http.StatusOK, Envelope[[]T]{Data: items, Meta: &Meta{Pagination: &p}})
}

// writeError maps a domain error to an HTTP status + error envelope.
func writeError(w http.ResponseWriter, err error) {
	status := statusFromError(err)
	if status == http.StatusInternalServerError {
		slog.Error("request failed", "error", err)
	}
	writeJSON(w, status, ErrorEnvelope{Error: ErrorBody{Code: errorCode(err), Message: err.Error()}})
}

// writeJSON is the low-level encoder used by the helpers above.
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

// statusFromError uses errors.Is so wrapped errors still map correctly.
func statusFromError(err error) int {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrConflict),
		errors.Is(err, domain.ErrInsufficientStock),
		errors.Is(err, domain.ErrAlreadyCanceled):
		return http.StatusConflict
	case errors.Is(err, domain.ErrBadParamInput):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// errorCode is the stable, machine-readable code clients can switch on.
func errorCode(err error) string {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, domain.ErrConflict):
		return "CONFLICT"
	case errors.Is(err, domain.ErrInsufficientStock):
		return "INSUFFICIENT_STOCK"
	case errors.Is(err, domain.ErrAlreadyCanceled):
		return "ORDER_NOT_CANCELABLE"
	case errors.Is(err, domain.ErrBadParamInput):
		return "BAD_REQUEST"
	default:
		return "INTERNAL"
	}
}

// decodeJSON reads the request body into dst, rejecting unknown fields.
func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return domain.ErrBadParamInput
	}
	return nil
}

// pathID parses a required int64 path parameter (Go 1.22 routing).
func pathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		return 0, domain.ErrBadParamInput
	}
	return id, nil
}

func queryInt(r *http.Request, name string) *int {
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

func queryInt64(r *http.Request, name string) *int64 {
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
