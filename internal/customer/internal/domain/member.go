package domain

import "github.com/shseooo/go-architecture/internal/shared"

// Member represents a customer.
type Member struct {
	ID      int64
	Name    string
	Address shared.Address
}
