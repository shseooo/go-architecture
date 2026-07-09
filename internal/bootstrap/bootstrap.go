// Package bootstrap wires the bounded contexts (and the cross-context adapters)
// into a single http.Handler, shared by cmd/api and the e2e tests.
package bootstrap

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/shseooo/go-architecture/docs"
	"github.com/shseooo/go-architecture/internal/catalog"
	"github.com/shseooo/go-architecture/internal/customer"
	"github.com/shseooo/go-architecture/internal/ordering"
	"github.com/shseooo/go-architecture/internal/platform/httpx"
)

// Handler assembles the full application from a database handle.
func Handler(db *sql.DB, timeout time.Duration) http.Handler {
	catalogMod := catalog.New(db)
	customerMod := customer.New(db)
	// ordering depends on catalog and customer, wired via anti-corruption adapters.
	orderingMod := ordering.New(db, itemGateway{catalogMod}, memberGateway{customerMod})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)
	catalogMod.Routes(mux)
	customerMod.Routes(mux)
	orderingMod.Routes(mux)

	return httpx.Chain(mux, httpx.CORS, httpx.Timeout(timeout))
}

// --- cross-module adapters (anti-corruption layer) --------------------------

type itemGateway struct{ c *catalog.Module }

func (g itemGateway) Reserve(ctx context.Context, itemID int64, qty int) (ordering.Item, error) {
	v, err := g.c.Reserve(ctx, itemID, qty)
	if err != nil {
		return ordering.Item{}, err
	}
	return ordering.Item{ID: v.ID, Name: v.Name, Price: v.Price}, nil
}

func (g itemGateway) Restore(ctx context.Context, itemID int64, qty int) error {
	return g.c.Restore(ctx, itemID, qty)
}

type memberGateway struct{ c *customer.Module }

func (g memberGateway) EnsureExists(ctx context.Context, memberID int64) error {
	_, err := g.c.GetMember(ctx, memberID)
	return err
}
