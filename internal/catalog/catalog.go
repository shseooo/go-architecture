// Package catalog is the public API of the catalog bounded context.
package catalog

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/shseooo/go-architecture/internal/catalog/internal/app"
	"github.com/shseooo/go-architecture/internal/catalog/internal/domain"
	"github.com/shseooo/go-architecture/internal/catalog/internal/repo"
	"github.com/shseooo/go-architecture/internal/catalog/internal/rest"
)

// ItemView is the read model other modules see for an item.
type ItemView struct {
	ID    int64
	Name  string
	Price int
}

// Catalog is the contract other modules use. Reserve/Restore run within the
// caller's transaction (via the shared context), enabling atomic order placement.
type Catalog interface {
	GetItem(ctx context.Context, id int64) (ItemView, error)
	Reserve(ctx context.Context, itemID int64, qty int) (ItemView, error)
	Restore(ctx context.Context, itemID int64, qty int) error
}

type Module struct {
	svc     *app.Service
	handler *rest.Handler
}

func New(db *sql.DB) *Module {
	svc := app.NewService(repo.New(db))
	return &Module{svc: svc, handler: rest.New(svc)}
}

func (m *Module) Routes(mux *http.ServeMux) {
	m.handler.Routes(mux)
}

func (m *Module) GetItem(ctx context.Context, id int64) (ItemView, error) {
	it, err := m.svc.GetByID(ctx, id)
	if err != nil {
		return ItemView{}, err
	}
	return ItemView{ID: it.ID, Name: it.Name, Price: it.Price}, nil
}

func (m *Module) Reserve(ctx context.Context, itemID int64, qty int) (ItemView, error) {
	it, err := m.svc.Reserve(ctx, itemID, qty)
	if err != nil {
		return ItemView{}, err
	}
	return ItemView{ID: it.ID, Name: it.Name, Price: it.Price}, nil
}

func (m *Module) Restore(ctx context.Context, itemID int64, qty int) error {
	return m.svc.Restore(ctx, itemID, qty)
}

// NewItem is the public input for creating an item (used by the CSV importer,
// which drives the same use-case as the HTTP API).
type NewItem struct {
	Name          string
	Price         int
	StockQuantity int
	Type          string
	Author        string
	ISBN          string
	Artist        string
	Etc           string
	Director      string
	Actor         string
	CategoryIDs   []int64
}

// CreateItem registers an item through the catalog use-case and returns its ID.
func (m *Module) CreateItem(ctx context.Context, in NewItem) (int64, error) {
	it := &domain.Item{
		Name: in.Name, Price: in.Price, StockQuantity: in.StockQuantity,
		Type:   domain.ItemType(in.Type),
		Author: in.Author, ISBN: in.ISBN, Artist: in.Artist, Etc: in.Etc,
		Director: in.Director, Actor: in.Actor, CategoryIDs: in.CategoryIDs,
	}
	if err := m.svc.Create(ctx, it); err != nil {
		return 0, err
	}
	return it.ID, nil
}
