package domain

import "errors"

var (
	// ErrInternalServerError is returned when an unexpected error happens.
	ErrInternalServerError = errors.New("internal server error")
	// ErrNotFound is returned when the requested resource does not exist.
	ErrNotFound = errors.New("requested resource is not found")
	// ErrConflict is returned when the resource already exists.
	ErrConflict = errors.New("resource already exists")
	// ErrBadParamInput is returned when the given request body or params are invalid.
	ErrBadParamInput = errors.New("given param is not valid")

	// ErrInsufficientStock is returned when an order requests more items than are in stock.
	ErrInsufficientStock = errors.New("not enough stock")
	// ErrAlreadyCanceled is returned when trying to cancel an order that is not in ORDER status.
	ErrAlreadyCanceled = errors.New("order cannot be canceled")
)
