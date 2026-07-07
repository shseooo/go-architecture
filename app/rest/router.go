package rest

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/shseooo/go-architecture/docs" // generated OpenAPI spec (swag init)
)

// NewRouter wires every route onto a net/http ServeMux using Go 1.22 method +
// path-parameter patterns. Middleware is applied by the caller.
func NewRouter(member *MemberHandler, item *ItemHandler, order *OrderHandler) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Swagger UI at /swagger/index.html (spec served from the generated docs).
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	// Member
	mux.HandleFunc("POST /members", member.Register)
	mux.HandleFunc("GET /members/{id}", member.GetByID)
	mux.HandleFunc("GET /members/{id}/orders", order.FindByMember)

	// Item
	mux.HandleFunc("POST /items", item.Create)
	mux.HandleFunc("GET /items", item.Search)
	mux.HandleFunc("GET /items/{id}", item.GetByID)
	mux.HandleFunc("PUT /items/{id}", item.Update)

	// Order
	mux.HandleFunc("POST /orders", order.Place)
	mux.HandleFunc("POST /orders/{id}/cancel", order.Cancel)

	return mux
}
