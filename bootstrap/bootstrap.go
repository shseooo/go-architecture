// Package bootstrap wires the layers (repositories → services → handlers) into a
// single http.Handler. Keeping it separate lets both main and the E2E tests build
// the exact same application from a *sql.DB.
package bootstrap

import (
	"database/sql"
	"net/http"
	"time"

	mysqlRepo "github.com/shseooo/go-architecture/app/repository/mysql"
	"github.com/shseooo/go-architecture/app/rest"
	"github.com/shseooo/go-architecture/app/rest/middleware"
	"github.com/shseooo/go-architecture/app/service/item"
	"github.com/shseooo/go-architecture/app/service/member"
	"github.com/shseooo/go-architecture/app/service/order"
)

// NewHandler assembles the full application and returns the root HTTP handler.
func NewHandler(db *sql.DB, timeout time.Duration) http.Handler {
	// Repositories
	txManager := mysqlRepo.NewTxManager(db)
	memberRepo := mysqlRepo.NewMemberRepository(db)
	itemRepo := mysqlRepo.NewItemRepository(db)
	orderRepo := mysqlRepo.NewOrderRepository(db)
	deliveryRepo := mysqlRepo.NewDeliveryRepository(db)

	// Services
	memberSvc := member.NewService(memberRepo)
	itemSvc := item.NewService(itemRepo)
	orderSvc := order.NewService(orderRepo, itemRepo, deliveryRepo, memberRepo, txManager)

	// Router + middleware
	router := rest.NewRouter(
		rest.NewMemberHandler(memberSvc),
		rest.NewItemHandler(itemSvc),
		rest.NewOrderHandler(orderSvc),
	)
	return middleware.Chain(router, middleware.CORS, middleware.Timeout(timeout))
}
