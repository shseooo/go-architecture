// Package customer is the public API of the customer bounded context. Other
// modules depend only on this file — never on the module's internal packages
// (which the Go compiler enforces via the nested internal/ directory).
package customer

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/shseooo/go-architecture/internal/customer/internal/app"
	"github.com/shseooo/go-architecture/internal/customer/internal/repo"
	"github.com/shseooo/go-architecture/internal/customer/internal/rest"
)

// MemberView is the read model other modules see (no internal entity leaks out).
type MemberView struct {
	ID   int64
	Name string
}

// Customer is the contract other modules use to talk to this context.
type Customer interface {
	GetMember(ctx context.Context, id int64) (MemberView, error)
}

// Module bundles the customer context's construction and HTTP routes.
type Module struct {
	svc     *app.Service
	handler *rest.Handler
}

// New wires the module from a database handle.
func New(db *sql.DB) *Module {
	svc := app.NewService(repo.New(db))
	return &Module{svc: svc, handler: rest.New(svc)}
}

// Routes registers the module's HTTP endpoints.
func (m *Module) Routes(mux *http.ServeMux) {
	m.handler.Routes(mux)
}

// GetMember implements Customer for use by other modules.
func (m *Module) GetMember(ctx context.Context, id int64) (MemberView, error) {
	mem, err := m.svc.GetByID(ctx, id)
	if err != nil {
		return MemberView{}, err
	}
	return MemberView{ID: mem.ID, Name: mem.Name}, nil
}
