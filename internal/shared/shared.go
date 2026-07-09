// Package shared is the shared kernel: value objects and error taxonomy that
// every bounded context may depend on. Keep it small and stable — anything that
// belongs to a single context lives in that context instead.
package shared

import "errors"

// Address is a value object embedded by member and delivery.
type Address struct {
	City    string `json:"city"`
	Street  string `json:"street"`
	Zipcode string `json:"zipcode"`
}

// Cross-cutting sentinel errors. Modules return these; platform/httpx maps them
// to HTTP status codes and error codes in one place.
var (
	ErrNotFound          = errors.New("requested resource is not found")
	ErrConflict          = errors.New("resource already exists")
	ErrBadParamInput     = errors.New("given param is not valid")
	ErrInsufficientStock = errors.New("not enough stock")
	ErrAlreadyCanceled   = errors.New("order cannot be canceled")
)
